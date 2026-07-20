package main

import (
	"os"
	"path/filepath"
	"testing"

	"dragzone/internal/model"
)

// tempFiles creates real files in a temp dir and returns their paths; the
// Drop Bar stages (copies) whatever it is given, so tests must hand it
// paths that actually exist.
func tempFiles(t *testing.T, names ...string) []string {
	t.Helper()
	dir := t.TempDir()
	paths := make([]string, 0, len(names))
	for _, n := range names {
		p := filepath.Join(dir, n)
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, p)
	}
	return paths
}

func TestDropBarAddRemoveClear(t *testing.T) {
	app := newTestApp(t)
	it, err := app.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: tempFiles(t, "a.txt")})
	if err != nil {
		t.Fatal(err)
	}
	if len(app.DropBarItems()) != 1 {
		t.Fatalf("add: %+v", app.DropBarItems())
	}
	if err := app.DropBarRemove(it.ID); err != nil {
		t.Fatal(err)
	}
	if len(app.DropBarItems()) != 0 {
		t.Error("remove failed")
	}
	app.DropBarAdd(model.Payload{Kind: model.ItemText, Text: "hi"})
	if err := app.DropBarClear(); err != nil || len(app.DropBarItems()) != 0 {
		t.Errorf("clear failed: %v", err)
	}
}

func TestDropBarConsumeHonorsLockAndSetting(t *testing.T) {
	app := newTestApp(t)

	// Locked item survives consume.
	locked, _ := app.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: tempFiles(t, "a")})
	if err := app.DropBarSetLocked(locked.ID, true); err != nil {
		t.Fatal(err)
	}
	if err := app.DropBarConsume(locked.ID); err != nil {
		t.Fatal(err)
	}
	if len(app.DropBarItems()) != 1 {
		t.Error("locked item should survive consume")
	}

	// Unlocked item is removed on consume.
	free, _ := app.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: tempFiles(t, "b")})
	if err := app.DropBarConsume(free.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := app.dropBar.Get(free.ID); ok {
		t.Error("unlocked item should be consumed")
	}

	// With the keep setting on, even unlocked items survive.
	s := app.settings.Get()
	s.DropBarKeepsItems = true
	app.settings.Set(s)
	kept, _ := app.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: tempFiles(t, "c")})
	app.DropBarConsume(kept.ID)
	if _, ok := app.dropBar.Get(kept.ID); !ok {
		t.Error("keep-items setting should preserve consumed item")
	}
}

func TestDropBarSeparateAndCombine(t *testing.T) {
	app := newTestApp(t)
	stack, _ := app.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: tempFiles(t, "a", "b", "c")})
	if err := app.DropBarSeparate(stack.ID); err != nil {
		t.Fatal(err)
	}
	if len(app.DropBarItems()) != 3 {
		t.Fatalf("separate: %+v", app.DropBarItems())
	}
	if err := app.DropBarCombineAll(); err != nil {
		t.Fatal(err)
	}
	if len(app.DropBarItems()) != 1 {
		t.Errorf("combine: %+v", app.DropBarItems())
	}
}
