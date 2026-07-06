package main

import (
	"testing"

	"dragzone/internal/storage"
)

// noopServices satisfies actions.Services for tests that don't exercise host
// side effects (clipboard, notifications, file ops).
type noopServices struct{}

func (noopServices) CopyToClipboard(string) error   { return nil }
func (noopServices) ReadClipboard() (string, error) { return "", nil }
func (noopServices) Notify(string, string)          {}
func (noopServices) PlaySound(string)               {}
func (noopServices) OpenURL(string) error           { return nil }
func (noopServices) OpenPath(string) error          { return nil }
func (noopServices) Reveal(string) error            { return nil }
func (noopServices) Trash([]string) error           { return nil }
func (noopServices) AirDrop([]string) error         { return nil }

func newTestApp(t *testing.T) *App {
	t.Helper()
	t.Setenv(storage.EnvDataDir, t.TempDir())
	app, err := NewApp(noopServices{})
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	return app
}

// TestDuplicateTarget guards the tile "Duplicate" command: the copy gets a
// fresh ID and independent options, keeps the action/label, and deliberately
// drops the single-key shortcut so two tiles never claim the same key.
func TestDuplicateTarget(t *testing.T) {
	app := newTestApp(t)

	orig, err := app.AddTarget("folder", "My Folder", map[string]string{"path": "/tmp", "mode": "copy"})
	if err != nil {
		t.Fatalf("AddTarget: %v", err)
	}
	orig.Shortcut = "f"
	if err := app.UpdateTarget(orig); err != nil {
		t.Fatalf("UpdateTarget: %v", err)
	}
	before := len(app.Targets())

	dup, err := app.DuplicateTarget(orig.ID)
	if err != nil {
		t.Fatalf("DuplicateTarget: %v", err)
	}

	if dup.ID == "" || dup.ID == orig.ID {
		t.Errorf("duplicate must get a fresh ID, got %q (orig %q)", dup.ID, orig.ID)
	}
	if dup.ActionID != orig.ActionID || dup.Label != orig.Label {
		t.Errorf("duplicate lost action/label: %+v", dup)
	}
	if dup.Options["path"] != "/tmp" || dup.Options["mode"] != "copy" {
		t.Errorf("duplicate lost options: %+v", dup.Options)
	}
	if dup.Shortcut != "" {
		t.Errorf("duplicate must not carry the shortcut, got %q", dup.Shortcut)
	}
	if got := len(app.Targets()); got != before+1 {
		t.Errorf("grid target count = %d, want %d", got, before+1)
	}

	// The copied options must be independent of the original's map.
	dup.Options["path"] = "/changed"
	orig2, err := app.grid.Get(orig.ID)
	if err != nil {
		t.Fatalf("Get orig: %v", err)
	}
	if orig2.Options["path"] != "/tmp" {
		t.Error("mutating the duplicate changed the original's options (shared map)")
	}
}

func TestDuplicateTargetUnknownID(t *testing.T) {
	app := newTestApp(t)
	if _, err := app.DuplicateTarget("does-not-exist"); err == nil {
		t.Error("DuplicateTarget on an unknown ID should return an error")
	}
}
