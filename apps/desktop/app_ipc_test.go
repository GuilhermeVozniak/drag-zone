package main

import (
	"testing"

	"dragzone/internal/ipc"
	"dragzone/internal/model"
)

func TestHandleIPCListAndAdd(t *testing.T) {
	app := newTestApp(t)
	// The default grid seeds 6 targets.
	if rows, err := app.handleIPC(ipc.Request{Cmd: "list"}); err != nil {
		t.Fatalf("list: %v", err)
	} else if rs, ok := rows.([]struct {
		Label  string `json:"label"`
		Action string `json:"action"`
		Events string `json:"events"`
	}); ok && len(rs) == 0 {
		t.Error("list returned no rows")
	}

	// add two files individually.
	if _, err := app.handleIPC(ipc.Request{Cmd: "add", Args: []string{"/x/a.txt", "/x/b.txt"}}); err != nil {
		t.Fatal(err)
	}
	if len(app.dropBar.List()) != 2 {
		t.Errorf("add: %d items", len(app.dropBar.List()))
	}
	// add --stack keeps them as one item.
	app.DropBarClear()
	if _, err := app.handleIPC(ipc.Request{Cmd: "add", Args: []string{"/x/a", "/x/b"}, Flags: map[string]bool{"stack": true}}); err != nil {
		t.Fatal(err)
	}
	if len(app.dropBar.List()) != 1 {
		t.Errorf("add --stack: %d items", len(app.dropBar.List()))
	}
}

func TestHandleIPCItemCommandsByIndex(t *testing.T) {
	app := newTestApp(t)
	app.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: []string{"/x/a.txt"}})
	// rename item 1
	if _, err := app.handleIPC(ipc.Request{Cmd: "rename", Args: []string{"1", "custom"}}); err != nil {
		t.Fatal(err)
	}
	if app.dropBar.List()[0].Label != "custom" {
		t.Errorf("rename failed: %+v", app.dropBar.List()[0])
	}
	// lock / unlock
	if _, err := app.handleIPC(ipc.Request{Cmd: "lock", Args: []string{"1"}}); err != nil {
		t.Fatal(err)
	}
	if !app.dropBar.List()[0].Locked {
		t.Error("lock failed")
	}
	// bad index
	if _, err := app.handleIPC(ipc.Request{Cmd: "remove", Args: []string{"99"}}); err == nil {
		t.Error("out-of-range index should error")
	}
}

func TestHandleIPCUnknownAndRunErrors(t *testing.T) {
	app := newTestApp(t)
	if _, err := app.handleIPC(ipc.Request{Cmd: "frobnicate"}); err == nil {
		t.Error("unknown command should error")
	}
	// run with a bad event
	if _, err := app.handleIPC(ipc.Request{Cmd: "run", Args: []string{"Desktop", "sideways"}}); err == nil {
		t.Error("bad event should error")
	}
	// run with an unknown target label
	if _, err := app.handleIPC(ipc.Request{Cmd: "run", Args: []string{"NoSuchTarget", "dragged"}}); err == nil {
		t.Error("unknown target should error")
	}
}
