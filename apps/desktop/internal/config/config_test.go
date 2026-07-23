package config

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"dragzone/internal/storage"
)

func TestDefaults(t *testing.T) {
	d := Defaults()
	if d.GlobalShortcut != "F3" || d.PopOutShortcut != "F4" {
		t.Errorf("shortcut defaults wrong: %+v", d)
	}
	if d.GridColumns != 4 || d.GridSize != 33 || d.Theme != "system" {
		t.Errorf("grid defaults wrong: %+v", d)
	}
	if !d.AnimateGrid || !d.ShowKeyOverlays || !d.PlaySounds || !d.DragOverlay ||
		!d.NotifyOnComplete || !d.AutoUpdateCheck {
		t.Errorf("boolean defaults wrong: %+v", d)
	}
}

func TestScaleClamp(t *testing.T) {
	cases := []struct {
		grid int
		want float64
	}{
		{0, 0.8}, {100, 1.4}, {33, 0.8 + 33.0/100*0.6},
		{-50, 0.8}, {250, 1.4}, // clamped
	}
	for _, c := range cases {
		got := Settings{GridSize: c.grid}.Scale()
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("Scale(gridSize=%d) = %v, want %v", c.grid, got, c.want)
		}
	}
}

func TestLoadDefaultsThenSetPersists(t *testing.T) {
	t.Setenv(storage.EnvDataDir, t.TempDir())
	// Fresh load with no file returns defaults.
	st, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if st.Get().GridColumns != 4 {
		t.Errorf("fresh Load did not return defaults: %+v", st.Get())
	}
	// Set persists; a fresh Load sees the change.
	s := st.Get()
	s.GridColumns = 6
	s.Theme = "dark"
	if err := st.Set(s); err != nil {
		t.Fatalf("Set: %v", err)
	}
	st2, err := Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if st2.Get().GridColumns != 6 || st2.Get().Theme != "dark" {
		t.Errorf("persisted settings = %+v", st2.Get())
	}
	// Unset fields keep their default (merge over Defaults()).
	if st2.Get().GlobalShortcut != "F3" {
		t.Errorf("defaults not preserved on reload: %+v", st2.Get())
	}
}

func TestSetRollsBackOnSaveFailure(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(storage.EnvDataDir, dir)
	st, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	// Make persistence fail: a directory where settings.json should be.
	if err := os.Mkdir(filepath.Join(dir, fileName), 0o755); err != nil {
		t.Fatal(err)
	}
	s := st.Get()
	s.GridColumns = 8
	if err := st.Set(s); err == nil {
		t.Fatal("Set should report the save failure")
	}
	if got := st.Get().GridColumns; got != 4 {
		t.Errorf("failed Set must roll back in-memory settings; GridColumns = %d", got)
	}
}
