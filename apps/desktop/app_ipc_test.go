package main

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"dragzone/internal/ipc"
	"dragzone/internal/model"
	"dragzone/internal/tasks"
)

func TestHandleIPCListAndAdd(t *testing.T) {
	app := newTestApp(t)
	// list reports every seeded grid target. handleIPC returns a
	// function-local named type, so a direct type assertion to an anonymous
	// struct never matches (Go type identity) — round-trip through JSON instead.
	rows, err := app.handleIPC(ipc.Request{Cmd: "list"})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	b, err := json.Marshal(rows)
	if err != nil {
		t.Fatalf("marshal list: %v", err)
	}
	var got []struct {
		Label  string `json:"label"`
		Action string `json:"action"`
		Events string `json:"events"`
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal list: %v", err)
	}
	if len(got) == 0 || len(got) != len(app.grid.List()) {
		t.Fatalf("list returned %d rows, want %d (the seeded grid)", len(got), len(app.grid.List()))
	}
	if got[0].Label == "" || got[0].Action == "" {
		t.Errorf("list row not populated: %+v", got[0])
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

// TestHandleIPCListItemsClearRemove covers the drop-bar-item IPC commands
// that TestHandleIPCListAndAdd/TestHandleIPCItemCommandsByIndex don't reach:
// "list-items" (round-trip through JSON, same reasoning as the "list" test:
// handleIPC's dropBar.List() return type isn't the anonymous struct we assert
// against), "remove" success (only the bad-index error path was covered
// before), and "clear".
func TestHandleIPCListItemsClearRemove(t *testing.T) {
	app := newTestApp(t)
	if _, err := app.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: []string{"/x/a.txt"}}); err != nil {
		t.Fatalf("seed item 1: %v", err)
	}
	if _, err := app.DropBarAdd(model.Payload{Kind: model.ItemFiles, Paths: []string{"/x/b.txt"}}); err != nil {
		t.Fatalf("seed item 2: %v", err)
	}

	rows, err := app.handleIPC(ipc.Request{Cmd: "list-items"})
	if err != nil {
		t.Fatalf("list-items: %v", err)
	}
	b, err := json.Marshal(rows)
	if err != nil {
		t.Fatalf("marshal list-items: %v", err)
	}
	var got []struct {
		Label string `json:"label"`
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal list-items: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("list-items returned %d rows, want 2", len(got))
	}

	// remove item 1 (1-based index): 2 items -> 1.
	if _, err := app.handleIPC(ipc.Request{Cmd: "remove", Args: []string{"1"}}); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if n := len(app.dropBar.List()); n != 1 {
		t.Fatalf("remove: %d items remain, want 1", n)
	}

	// clear empties whatever is left.
	if _, err := app.handleIPC(ipc.Request{Cmd: "clear"}); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if n := len(app.dropBar.List()); n != 0 {
		t.Fatalf("clear: %d items remain, want 0", n)
	}
}

// TestHandleIPCRunSuccess exercises the "run" command's success path via
// ipcRun: it matches the seeded "Clipboard" grid target (action
// copy-to-clipboard, see app.go's default grid) by label, requires a
// dragged/clicked event, and dispatches through the async runner (see
// app_grid.go trigger -> runner.Run). Following app_flow_test.go's pattern,
// a.ctx is seeded to context.Background() (trigger's parent context; startup
// normally supplies it from Wails) and a.onEmit is set before the call so
// a.emit (app.go) doesn't fall through to the real Wails runtime, which
// would abort the test process given the bare background context. Completion
// is awaited via tasks:changed rather than assumed synchronous.
func TestHandleIPCRunSuccess(t *testing.T) {
	app := newTestApp(t)
	app.ctx = context.Background()

	var mu sync.Mutex
	done := make(chan struct{})
	var closeOnce sync.Once
	app.onEmit = func(ev string, data ...any) {
		if ev != tasks.EventTasksChanged {
			return
		}
		states, ok := data[0].([]model.TaskState)
		if !ok || len(states) == 0 {
			return
		}
		if states[0].Status == model.TaskDone || states[0].Status == model.TaskError {
			mu.Lock()
			defer mu.Unlock()
			closeOnce.Do(func() { close(done) })
		}
	}

	res, err := app.handleIPC(ipc.Request{Cmd: "run", Args: []string{"Clipboard", "dragged", "/tmp/x.txt"}})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res == nil {
		t.Error("run: expected a non-nil result message")
	}

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("no tasks:changed emitted for task completion")
	}

	taskList := app.Tasks()
	if len(taskList) == 0 {
		t.Fatal("run: expected at least one task to be created")
	}
	if taskList[0].Status != model.TaskDone {
		t.Errorf("run: task status = %v, want %v (error %q)", taskList[0].Status, model.TaskDone, taskList[0].Error)
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
