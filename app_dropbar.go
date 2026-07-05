package main

import (
	"os/exec"

	"dragzone/internal/dropbar"
	"dragzone/internal/model"
	"dragzone/internal/platform"
)

// Drop Bar bindings: the shelf of stashed items above the grid.

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

// DropBarConsume removes an item after it was dragged out or dropped onto a
// target, honoring per-item locks and the global keep setting.
func (a *App) DropBarConsume(id string) error {
	item, ok := a.dropBar.Get(id)
	if !ok || item.Locked || a.settings.Get().DropBarKeepsItems {
		return nil
	}
	return a.DropBarRemove(id)
}

// DropBarSetLocked toggles whether an item survives being dragged out.
func (a *App) DropBarSetLocked(id string, locked bool) error {
	if err := a.dropBar.SetLocked(id, locked); err != nil {
		return err
	}
	a.emit(EventDropBarChanged, a.dropBar.List())
	return nil
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

// StartDragOut begins a native drag session for a drop bar item. The item is
// consumed when the drop completes (unless locked or configured otherwise).
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

// QuickLook previews files with the system Quick Look panel.
func (a *App) QuickLook(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	return exec.Command("qlmanage", append([]string{"-p"}, paths...)...).Start()
}
