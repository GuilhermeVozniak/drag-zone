package main

import (
	"os"
	"path/filepath"
	"strings"

	"dragzone/internal/model"
	"dragzone/internal/platform"
)

// Grid targets, drops/clicks, and task bindings.

func (a *App) ActionSpecs() []model.ActionSpec {
	return a.registry.Specs()
}

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

// DuplicateTarget adds a copy of an existing target to the grid, like
// Dropzone's tile "Duplicate" command. The copy gets a fresh ID and its own
// position; the single-key shortcut is deliberately not carried over so two
// tiles never claim the same key.
func (a *App) DuplicateTarget(id string) (model.Target, error) {
	src, err := a.grid.Get(id)
	if err != nil {
		return model.Target{}, err
	}
	opts := make(map[string]string, len(src.Options))
	for k, v := range src.Options {
		opts[k] = v
	}
	return a.AddTarget(src.ActionID, src.Label, opts)
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

// AddTargetsFromPaths adds grid targets inferred from paths dropped on the
// "Add to Grid" area: .dzbundle installs as an action, .app becomes an Open
// Application target, plain directories become Folder targets.
func (a *App) AddTargetsFromPaths(paths []string) error {
	for _, p := range paths {
		switch {
		case strings.HasSuffix(p, ".dzbundle"):
			if err := a.InstallBundle(p); err != nil {
				return err
			}
		case strings.HasSuffix(p, ".app"):
			name := strings.TrimSuffix(filepath.Base(p), ".app")
			if _, err := a.AddTarget("open-app", name, map[string]string{"app": p}); err != nil {
				return err
			}
		default:
			if info, err := os.Stat(p); err == nil && info.IsDir() {
				if _, err := a.AddTarget("folder", filepath.Base(p), map[string]string{"path": p, "mode": "copy"}); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// DropOnTarget runs a target's dragged event with the given payload. Holding
// Option is recorded as a payload modifier (folders invert copy/move, scripts
// see KEY_MODIFIERS).
func (a *App) DropOnTarget(targetID string, payload model.Payload) (string, error) {
	if platform.OptionKeyDown() {
		payload.Modifiers = append(payload.Modifiers, "Option")
	}
	return a.trigger(targetID, payload, model.EventDragged)
}

// ClickTarget runs a target's clicked event.
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

func (a *App) Tasks() []model.TaskState {
	return a.runner.List()
}

func (a *App) DismissTask(id string) {
	a.runner.Dismiss(id)
}

// CancelTask aborts a running task.
func (a *App) CancelTask(id string) {
	a.runner.Cancel(id)
}

// PlayDropSound gives audio feedback for a drop, honoring the sound setting.
func (a *App) PlayDropSound() {
	if a.settings.Get().PlaySounds {
		a.services.PlaySound("Pop")
	}
}
