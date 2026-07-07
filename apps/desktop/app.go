// DragZone is a Dropzone 4-style menu bar drag-and-drop utility. The main
// package holds the Wails bootstrap (main.go) and the App bindings facade,
// split by domain: app.go (construction and wiring), app_grid.go (targets,
// drops, tasks), app_dropbar.go (the shelf), app_bundles.go (scriptable
// actions), app_ipc.go (the dz CLI channel), and app_settings.go (settings,
// dialogs, window).
package main

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"dragzone/internal/actions"
	"dragzone/internal/actions/builtin"
	"dragzone/internal/bundles"
	"dragzone/internal/config"
	"dragzone/internal/dropbar"
	"dragzone/internal/grid"
	"dragzone/internal/ipc"
	"dragzone/internal/model"
	"dragzone/internal/platform"
	"dragzone/internal/storage"
	"dragzone/internal/tasks"
)

// Events emitted to the frontend. The frontend subscribes to these names in
// lib/backend.ts; keep the two lists in sync.
const (
	EventGridChanged      = "grid:changed"
	EventDropBarChanged   = "dropbar:changed"
	EventOpenSettings     = "settings:open"
	EventWindowVisibility = "window:visibility"
	EventSpecsChanged     = "specs:changed"
	EventDropBarPopOut    = "dropbar:popout"
	EventInputRequest     = "input:request"
	EventWindowBeak       = "window:beak"
	EventSharesChanged    = "shares:changed"
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
	poppedOut    bool   // Drop Bar pop-out mode is active

	iconMu    sync.Mutex
	iconCache map[string]string

	ipcServer *ipc.Server

	inputMu   sync.Mutex
	inputReqs map[string]chan inputAnswer

	taskMu       sync.Mutex
	runningTasks int
	recentShares []Share
}

// Share is one entry of the "Recently Shared" menu.
type Share struct {
	Title string    `json:"title"`
	URL   string    `json:"url"`
	At    time.Time `json:"at"`
}

const recentsFile = "recents.json"

// taskFeedback drives the menu bar icon through Dropzone's task states.
func (a *App) taskFeedback(status model.TaskStatus) {
	a.taskMu.Lock()
	defer a.taskMu.Unlock()
	switch status {
	case model.TaskRunning:
		a.runningTasks++
		platform.SetStatusState(platform.StatusRunning)
	default:
		if a.runningTasks > 0 {
			a.runningTasks--
		}
		if a.runningTasks > 0 {
			return // still busy, keep the running state
		}
		if status == model.TaskError {
			platform.SetStatusState(platform.StatusFailure)
		} else {
			platform.SetStatusState(platform.StatusSuccess)
		}
		time.AfterFunc(2*time.Second, func() {
			a.taskMu.Lock()
			defer a.taskMu.Unlock()
			if a.runningTasks == 0 {
				platform.SetStatusState(platform.StatusNormal)
			}
		})
	}
}

// addRecentShare records a shared URL for the Recently Shared menu.
func (a *App) addRecentShare(title, url string) {
	a.taskMu.Lock()
	a.recentShares = append([]Share{{Title: title, URL: url, At: time.Now()}}, a.recentShares...)
	if len(a.recentShares) > 10 {
		a.recentShares = a.recentShares[:10]
	}
	shares := append([]Share(nil), a.recentShares...)
	a.taskMu.Unlock()
	_ = storage.Save(recentsFile, shares)
	a.emit(EventSharesChanged, shares)
}

// RecentShares lists the most recently shared URLs, newest first.
func (a *App) RecentShares() []Share {
	a.taskMu.Lock()
	defer a.taskMu.Unlock()
	if len(a.recentShares) == 0 {
		return []Share{}
	}
	return append([]Share(nil), a.recentShares...)
}

// ClearRecentShares empties the Recently Shared menu.
func (a *App) ClearRecentShares() error {
	a.taskMu.Lock()
	a.recentShares = []Share{}
	a.taskMu.Unlock()
	a.emit(EventSharesChanged, []Share{})
	return storage.Save(recentsFile, []Share{})
}

// OpenURL opens a URL in the default browser.
func (a *App) OpenURL(url string) error {
	return a.services.OpenURL(url)
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
		settings:  settings,
		grid:      gridStore,
		dropBar:   barStore,
		registry:  reg,
		services:  services,
		iconCache: map[string]string{},
		inputReqs: map[string]chan inputAnswer{},
	}
	a.runner = tasks.NewRunner(tasks.Config{
		Emit:             a.emit,
		Services:         services,
		NotifyEnabled:    func() bool { return settings.Get().NotifyOnComplete },
		SoundsEnabled:    func() bool { return settings.Get().PlaySounds },
		SaveTargetOption: a.saveTargetOption,
		OnTask:           a.taskFeedback,
		OnResultURL:      a.addRecentShare,
		Prompt:           a.requestChoice,
	})
	if err := storage.Load(recentsFile, &a.recentShares); err == nil && a.recentShares == nil {
		a.recentShares = []Share{}
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

// startup receives the Wails context and brings up the native layer and the
// CLI control socket.
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
				_ = a.DropBarConsume(itemID)
			}
		},
		OpenSettings: func() {
			a.emit(EventOpenSettings)
		},
		GridVisibility: func(visible bool) {
			a.emit(EventWindowVisibility, visible)
		},
		GridBeak: func(x float64) {
			a.emit(EventWindowBeak, x)
		},
		PopOutHotkey: func() {
			a.dragMu.Lock()
			popped := a.poppedOut
			a.dragMu.Unlock()
			_ = a.SetDropBarPopOut(!popped)
		},
	})
	platform.InitNative("DragZone")
	a.applySettings(a.settings.Get())

	// Background update checks (startup + daily), gated on the setting.
	go a.autoUpdateLoop()

	if srv, err := ipc.Serve(a.handleIPC); err == nil {
		a.ipcServer = srv
	}
}

func (a *App) shutdown(_ context.Context) {
	if a.ipcServer != nil {
		a.ipcServer.Close()
	}
}

// emit publishes an event to the frontend; safe to call before startup.
func (a *App) emit(event string, data ...any) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, event, data...)
	}
}

// saveTargetOption persists one option value on a grid target. It backs both
// script save_value calls and actions that rotate credentials (Google Drive).
func (a *App) saveTargetOption(targetID, key, value string) {
	t, err := a.grid.Get(targetID)
	if err != nil {
		return
	}
	if t.Options == nil {
		t.Options = map[string]string{}
	}
	if value == "" {
		delete(t.Options, key)
	} else {
		t.Options[key] = value
	}
	_ = a.UpdateTarget(t)
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
