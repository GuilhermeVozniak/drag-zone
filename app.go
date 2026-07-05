package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"dragzone/internal/actions"
	"dragzone/internal/actions/builtin"
	"dragzone/internal/bundles"
	"dragzone/internal/config"
	"dragzone/internal/dropbar"
	"dragzone/internal/fsutil"
	"dragzone/internal/grid"
	"dragzone/internal/ipc"
	"dragzone/internal/model"
	"dragzone/internal/platform"
	"dragzone/internal/storage"
	"dragzone/internal/tasks"
)

// Events emitted to the frontend.
const (
	EventGridChanged      = "grid:changed"
	EventDropBarChanged   = "dropbar:changed"
	EventOpenSettings     = "settings:open"
	EventWindowVisibility = "window:visibility"
	EventSpecsChanged     = "specs:changed"
	EventDropBarPopOut    = "dropbar:popout"
	EventInputRequest     = "input:request"
)

// App wires the subsystems together and is the bindings facade exposed to the
// frontend.
type App struct {
	ctx      context.Context
	settings *config.Store
	grid     *grid.Store
	dropBar  *dropbar.Store
	registry *actions.Registry
	runner   *tasks.Runner
	services actions.Services

	dragMu       sync.Mutex
	draggingItem string // drop bar item ID of the in-flight drag-out

	iconMu    sync.Mutex
	iconCache map[string]string

	ipcServer *ipc.Server

	inputMu   sync.Mutex
	inputReqs map[string]chan inputAnswer
}

type inputAnswer struct {
	Value string
	OK    bool
}

// NewApp loads all stores and constructs the action engine.
func NewApp(services actions.Services) (*App, error) {
	settings, err := config.Load()
	if err != nil {
		return nil, err
	}
	gridStore, err := grid.Load(defaultTargets())
	if err != nil {
		return nil, err
	}
	barStore, err := dropbar.Load()
	if err != nil {
		return nil, err
	}

	reg := actions.NewRegistry()
	builtin.RegisterAll(reg)

	a := &App{
		settings: settings,
		grid:     gridStore,
		dropBar:  barStore,
		registry: reg,
		services: services,
	}
	a.runner = tasks.NewRunner(a.emit, services, func() bool { return settings.Get().NotifyOnComplete })
	a.iconCache = map[string]string{}
	a.inputReqs = map[string]chan inputAnswer{}

	// Google Drive persists rotated OAuth refresh tokens into target options.
	builtin.SaveTargetOption = func(targetID, key, value string) {
		a.bundleHost().SaveValue(targetID, key, value)
	}

	// User-installed scriptable action bundles.
	if dir, err := actionsDir(); err == nil {
		installed, err := bundles.LoadDir(dir, a.bundleHost())
		if err == nil {
			for _, act := range installed {
				_ = reg.TryRegister(act)
			}
		}
	}
	return a, nil
}

// actionsDir is where .dzbundle actions are installed.
func actionsDir() (string, error) {
	base, err := storage.Dir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "Actions")
	return dir, os.MkdirAll(dir, 0o755)
}

// bundleHost exposes app capabilities to running bundle scripts.
func (a *App) bundleHost() bundles.Host {
	return bundles.Host{
		SaveValue: func(targetID, name, value string) {
			t, err := a.grid.Get(targetID)
			if err != nil {
				return
			}
			if t.Options == nil {
				t.Options = map[string]string{}
			}
			t.Options[name] = value
			_ = a.UpdateTarget(t)
		},
		RemoveValue: func(targetID, name string) {
			t, err := a.grid.Get(targetID)
			if err != nil || t.Options == nil {
				return
			}
			delete(t.Options, name)
			_ = a.UpdateTarget(t)
		},
		AddDropBar: func(paths []string) {
			_, _ = a.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: paths})
		},
		RequestInput: a.requestInput,
	}
}

// requestInput shows an input dialog in the grid and blocks the calling
// script until the user answers or five minutes pass.
func (a *App) requestInput(title, prompt string) (string, bool) {
	id := uuid.NewString()
	ch := make(chan inputAnswer, 1)
	a.inputMu.Lock()
	a.inputReqs[id] = ch
	a.inputMu.Unlock()
	defer func() {
		a.inputMu.Lock()
		delete(a.inputReqs, id)
		a.inputMu.Unlock()
	}()

	platform.ShowGrid(true)
	a.emit(EventInputRequest, map[string]string{"id": id, "title": title, "prompt": prompt})

	select {
	case ans := <-ch:
		return ans.Value, ans.OK
	case <-time.After(5 * time.Minute):
		return "", false
	}
}

// AnswerInputRequest resolves a pending script input dialog.
func (a *App) AnswerInputRequest(id, value string, ok bool) {
	a.inputMu.Lock()
	ch := a.inputReqs[id]
	a.inputMu.Unlock()
	if ch != nil {
		ch <- inputAnswer{Value: value, OK: ok}
	}
}

// InstallBundle copies a .dzbundle into the actions directory and registers
// it immediately.
func (a *App) InstallBundle(path string) error {
	dir, err := actionsDir()
	if err != nil {
		return err
	}
	dst, err := fsutil.CopyPath(path, dir, nil)
	if err != nil {
		return fmt.Errorf("installing bundle: %w", err)
	}
	act, err := bundles.LoadBundle(dst, a.bundleHost())
	if err != nil {
		os.RemoveAll(dst)
		return err
	}
	if err := a.registry.TryRegister(act); err != nil {
		os.RemoveAll(dst)
		return fmt.Errorf("an action with this ID is already installed")
	}
	a.emit(EventSpecsChanged, a.registry.Specs())
	return nil
}

// DevelopAction creates a template .dzbundle, registers it, and reveals the
// script for editing — the "Develop Action…" workflow.
func (a *App) DevelopAction(name, language string) error {
	dir, err := actionsDir()
	if err != nil {
		return err
	}
	bundle, err := bundles.CreateTemplate(dir, name, language)
	if err != nil {
		return err
	}
	act, err := bundles.LoadBundle(bundle, a.bundleHost())
	if err != nil {
		return err
	}
	if err := a.registry.TryRegister(act); err != nil {
		return err
	}
	a.emit(EventSpecsChanged, a.registry.Specs())
	return a.services.OpenPath(bundle)
}

// OpenActionsFolder reveals the bundle installation directory in Finder.
func (a *App) OpenActionsFolder() error {
	dir, err := actionsDir()
	if err != nil {
		return err
	}
	return a.services.OpenPath(dir)
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	platform.SetHandlers(platform.Handlers{
		StatusDropped: func(paths []string) {
			if _, err := a.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: paths}); err == nil {
				platform.ShowGrid(true)
			}
		},
		DragEnded: func(completed bool) {
			a.dragMu.Lock()
			itemID := a.draggingItem
			a.draggingItem = ""
			a.dragMu.Unlock()
			if completed && itemID != "" {
				a.DropBarConsume(itemID)
			}
		},
		OpenSettings: func() {
			a.emit(EventOpenSettings)
		},
		GridVisibility: func(visible bool) {
			a.emit(EventWindowVisibility, visible)
		},
	})
	platform.InitNative("DragZone")
	platform.SetHotkeyF(parseFKey(a.settings.Get().GlobalShortcut))

	if srv, err := ipc.Serve(a.handleIPC); err == nil {
		a.ipcServer = srv
	}
}

func (a *App) shutdown(_ context.Context) {
	if a.ipcServer != nil {
		a.ipcServer.Close()
	}
}

// handleIPC dispatches `dz` CLI commands (see cmd/dz).
func (a *App) handleIPC(req ipc.Request) (any, error) {
	argAt := func(i int) string {
		if i < len(req.Args) {
			return req.Args[i]
		}
		return ""
	}
	switch req.Cmd {
	case "list":
		type row struct {
			Label  string `json:"label"`
			Action string `json:"action"`
			Events string `json:"events"`
		}
		var rows []row
		for _, t := range a.grid.List() {
			events := ""
			if act, err := a.registry.Get(t.ActionID); err == nil {
				events = strings.Join(act.Spec().Events, ", ")
			}
			rows = append(rows, row{Label: t.Label, Action: t.ActionID, Events: events})
		}
		return rows, nil
	case "run":
		name, event := argAt(0), argAt(1)
		if event != model.EventDragged && event != model.EventClicked {
			return nil, fmt.Errorf("event must be dragged or clicked")
		}
		var target *model.Target
		for _, t := range a.grid.List() {
			if strings.EqualFold(t.Label, name) {
				target = &t
				break
			}
		}
		if target == nil {
			return nil, fmt.Errorf("no grid target named %q", name)
		}
		payload := model.Payload{}
		if files := req.Args[2:]; len(files) > 0 {
			payload = model.Payload{Kind: model.ItemFiles, Paths: files}
		}
		if _, err := a.trigger(target.ID, payload, event); err != nil {
			return nil, err
		}
		return "running " + target.Label, nil
	case "list-items":
		return a.dropBar.List(), nil
	case "add":
		if len(req.Args) == 0 {
			return nil, fmt.Errorf("no files given")
		}
		if req.Flags["stack"] {
			if _, err := a.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: req.Args}); err != nil {
				return nil, err
			}
			return "added 1 stack", nil
		}
		for _, f := range req.Args {
			if _, err := a.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: []string{f}}); err != nil {
				return nil, err
			}
		}
		return fmt.Sprintf("added %d item(s)", len(req.Args)), nil
	case "rename":
		item, err := a.dropBarItemAt(argAt(0))
		if err != nil {
			return nil, err
		}
		name := argAt(1)
		if req.Flags["reset"] {
			name = ""
		}
		return nil, a.DropBarRename(item.ID, name)
	case "remove":
		item, err := a.dropBarItemAt(argAt(0))
		if err != nil {
			return nil, err
		}
		return nil, a.DropBarRemove(item.ID)
	case "lock", "unlock":
		item, err := a.dropBarItemAt(argAt(0))
		if err != nil {
			return nil, err
		}
		return nil, a.DropBarSetLocked(item.ID, req.Cmd == "lock")
	case "clear":
		return nil, a.DropBarClear()
	case "open":
		platform.ShowGrid(true)
		return nil, nil
	case "close":
		platform.HideGrid()
		return nil, nil
	case "open-dropbar":
		return nil, a.SetDropBarPopOut(true)
	case "close-dropbar":
		return nil, a.SetDropBarPopOut(false)
	default:
		return nil, fmt.Errorf("unknown command %q", req.Cmd)
	}
}

// dropBarItemAt resolves a 1-based CLI index into an item.
func (a *App) dropBarItemAt(arg string) (dropbar.Item, error) {
	idx, err := strconv.Atoi(arg)
	items := a.dropBar.List()
	if err != nil || idx < 1 || idx > len(items) {
		return dropbar.Item{}, fmt.Errorf("invalid item index %q (1-%d)", arg, len(items))
	}
	return items[idx-1], nil
}

// parseFKey converts "F3" to 3; unknown values disable the hotkey.
func parseFKey(shortcut string) int {
	s := strings.ToUpper(strings.TrimSpace(shortcut))
	if !strings.HasPrefix(s, "F") {
		return 0
	}
	n, err := strconv.Atoi(s[1:])
	if err != nil || n < 1 || n > 12 {
		return 0
	}
	return n
}

func (a *App) emit(event string, data ...any) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, event, data...)
	}
}

// defaultTargets seeds the grid on first launch.
func defaultTargets() []model.Target {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	return []model.Target{
		{ActionID: "folder", Label: "Desktop", Options: map[string]string{"path": filepath.Join(home, "Desktop"), "mode": "copy"}},
		{ActionID: "folder", Label: "Downloads", Options: map[string]string{"path": filepath.Join(home, "Downloads"), "mode": "move"}},
		{ActionID: "airdrop", Label: "AirDrop"},
		{ActionID: "zip", Label: "Zip Files"},
		{ActionID: "copy-to-clipboard", Label: "Clipboard"},
		{ActionID: "trash", Label: "Trash"},
	}
}

// --- Settings ---

func (a *App) GetSettings() config.Settings {
	return a.settings.Get()
}

func (a *App) SetSettings(s config.Settings) error {
	prev := a.settings.Get()
	if err := a.settings.Set(s); err != nil {
		return err
	}
	if s.GlobalShortcut != prev.GlobalShortcut {
		platform.SetHotkeyF(parseFKey(s.GlobalShortcut))
	}
	if s.LaunchAtLogin != prev.LaunchAtLogin {
		if err := platform.SetLoginItem(s.LaunchAtLogin); err != nil {
			return err
		}
	}
	return nil
}

// --- Action library ---

func (a *App) ActionSpecs() []model.ActionSpec {
	return a.registry.Specs()
}

// --- Grid targets ---

func (a *App) Targets() []model.Target {
	return a.grid.List()
}

func (a *App) AddTarget(actionID, label string, options map[string]string) (model.Target, error) {
	if _, err := a.registry.Get(actionID); err != nil {
		return model.Target{}, err
	}
	t, err := a.grid.Add(actionID, label, options)
	if err != nil {
		return model.Target{}, err
	}
	a.emit(EventGridChanged, a.grid.List())
	return t, nil
}

func (a *App) UpdateTarget(t model.Target) error {
	if err := a.grid.Update(t); err != nil {
		return err
	}
	a.emit(EventGridChanged, a.grid.List())
	return nil
}

func (a *App) RemoveTarget(id string) error {
	if err := a.grid.Remove(id); err != nil {
		return err
	}
	a.emit(EventGridChanged, a.grid.List())
	return nil
}

func (a *App) MoveTarget(id string, position int) error {
	if err := a.grid.Move(id, position); err != nil {
		return err
	}
	a.emit(EventGridChanged, a.grid.List())
	return nil
}

// --- Drops and clicks ---

func (a *App) DropOnTarget(targetID string, payload model.Payload) (string, error) {
	if platform.OptionKeyDown() {
		payload.Modifiers = append(payload.Modifiers, "Option")
	}
	return a.trigger(targetID, payload, model.EventDragged)
}

func (a *App) ClickTarget(targetID string) (string, error) {
	return a.trigger(targetID, model.Payload{}, model.EventClicked)
}

func (a *App) trigger(targetID string, payload model.Payload, event string) (string, error) {
	target, err := a.grid.Get(targetID)
	if err != nil {
		return "", err
	}
	act, err := a.registry.Get(target.ActionID)
	if err != nil {
		return "", err
	}
	return a.runner.Run(a.ctx, act, target, payload, event)
}

// --- Tasks ---

func (a *App) Tasks() []model.TaskState {
	return a.runner.List()
}

func (a *App) DismissTask(id string) {
	a.runner.Dismiss(id)
}

// --- Drop Bar ---

func (a *App) DropBarItems() []dropbar.Item {
	return a.dropBar.List()
}

func (a *App) DropBarAdd(p model.Payload) (dropbar.Item, error) {
	it, err := a.dropBar.Add(p)
	if err != nil {
		return dropbar.Item{}, err
	}
	a.emit(EventDropBarChanged, a.dropBar.List())
	return it, nil
}

// DropBarConsume removes an item after it was dragged out or dropped onto a
// target, honoring per-item locks and the global keep setting.
func (a *App) DropBarConsume(id string) error {
	item, ok := a.dropBar.Get(id)
	if !ok || item.Locked || a.settings.Get().DropBarKeepsItems {
		return nil
	}
	return a.DropBarRemove(id)
}

// DropBarRename sets a custom item label; empty resets to the derived label.
func (a *App) DropBarRename(id, name string) error {
	if err := a.dropBar.Rename(id, name); err != nil {
		return err
	}
	a.emit(EventDropBarChanged, a.dropBar.List())
	return nil
}

// SetDropBarPopOut pops the Drop Bar out into a pinned always-visible panel
// (or docks it back into the grid).
func (a *App) SetDropBarPopOut(popped bool) error {
	platform.SetPinned(popped)
	if popped {
		platform.ShowGrid(false)
	}
	a.emit(EventDropBarPopOut, popped)
	return nil
}

// QuickLook previews files with the system Quick Look panel.
func (a *App) QuickLook(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	return exec.Command("qlmanage", append([]string{"-p"}, paths...)...).Start()
}

// DropBarSetLocked toggles the lock state of a drop bar item.
func (a *App) DropBarSetLocked(id string, locked bool) error {
	if err := a.dropBar.SetLocked(id, locked); err != nil {
		return err
	}
	a.emit(EventDropBarChanged, a.dropBar.List())
	return nil
}

func (a *App) DropBarRemove(id string) error {
	if err := a.dropBar.Remove(id); err != nil {
		return err
	}
	a.emit(EventDropBarChanged, a.dropBar.List())
	return nil
}

func (a *App) DropBarClear() error {
	if err := a.dropBar.Clear(); err != nil {
		return err
	}
	a.emit(EventDropBarChanged, a.dropBar.List())
	return nil
}

// --- Drag-out & icons ---

// StartDragOut begins a native drag session for a drop bar item. The item is
// removed from the bar when the drop completes (unless configured otherwise).
func (a *App) StartDragOut(itemID string) error {
	item, ok := a.dropBar.Get(itemID)
	if !ok || len(item.Paths) == 0 {
		return nil
	}
	a.dragMu.Lock()
	a.draggingItem = itemID
	a.dragMu.Unlock()
	return platform.StartDrag(item.Paths)
}

// FileIcon returns the Finder icon for a path as a base64 PNG (cached).
func (a *App) FileIcon(path string) string {
	a.iconMu.Lock()
	defer a.iconMu.Unlock()
	if icon, ok := a.iconCache[path]; ok {
		return icon
	}
	icon, err := platform.FileIconPNGBase64(path, 64)
	if err != nil {
		icon = ""
	}
	a.iconCache[path] = icon
	return icon
}

// --- Dialogs & window ---

func (a *App) ChooseFolder() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "Choose Folder"})
}

func (a *App) ChooseApplication() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:                      "Choose Application",
		DefaultDirectory:           "/Applications",
		TreatPackagesAsDirectories: false,
		Filters:                    []runtime.FileFilter{{DisplayName: "Applications", Pattern: "*.app"}},
	})
}

func (a *App) HideWindow() {
	platform.HideGrid()
}

func (a *App) ShowWindow() {
	platform.ShowGrid(true)
}

func (a *App) Quit() {
	runtime.Quit(a.ctx)
}
