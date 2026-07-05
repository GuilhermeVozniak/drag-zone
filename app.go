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
		SaveTargetOption: a.saveTargetOption,
	})

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
				a.DropBarConsume(itemID)
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
