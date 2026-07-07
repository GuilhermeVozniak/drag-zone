package main

import (
	"testing"

	"dragzone/internal/model"
)

func TestDropBarAddRemoveClear(t *testing.T) {
	app := newTestApp(t)
	it, err := app.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: []string{"/tmp/a.txt"}})
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
	locked, _ := app.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: []string{"/a"}})
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
	free, _ := app.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: []string{"/b"}})
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
	kept, _ := app.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: []string{"/c"}})
	app.DropBarConsume(kept.ID)
	if _, ok := app.dropBar.Get(kept.ID); !ok {
		t.Error("keep-items setting should preserve consumed item")
	}
}

func TestDropBarSeparateAndCombine(t *testing.T) {
	app := newTestApp(t)
	stack, _ := app.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: []string{"/a", "/b", "/c"}})
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
