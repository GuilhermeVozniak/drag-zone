package main

import (
	"context"
	"sync"
	"testing"
	"time"

	"dragzone/internal/model"
	"dragzone/internal/storage"
	"dragzone/internal/tasks"
)

// flowServices is a recording actions.Services for this flow test. It lives
// here rather than reusing noopServices because App-package tests cannot see
// the builtin package's unexported recServices, and this test needs to
// observe the real side effect (clipboard write) an action performs.
type flowServices struct {
	mu        sync.Mutex
	clipboard string
}

func (s *flowServices) CopyToClipboard(text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clipboard = text
	return nil
}
func (s *flowServices) ReadClipboard() (string, error) { return "", nil }
func (s *flowServices) Notify(string, string)          {}
func (s *flowServices) PlaySound(string)               {}
func (s *flowServices) OpenURL(string) error           { return nil }
func (s *flowServices) OpenPath(string) error          { return nil }
func (s *flowServices) Reveal(string) error            { return nil }
func (s *flowServices) Trash([]string) error           { return nil }
func (s *flowServices) AirDrop([]string) error         { return nil }

func (s *flowServices) clipboardText() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.clipboard
}

// TestDropOnTargetRunsActionAndEmits drops text onto a Clipboard target and
// asserts the drop runs the action end-to-end (the recording service sees
// the clipboard write) and that the runner emits tasks.EventTasksChanged
// ("tasks:changed") with the finished task state. The runner executes the
// action in a goroutine (internal/tasks/runner.go Run), so completion is
// awaited via a channel closed from the onEmit hook, not assumed synchronous.
func TestDropOnTargetRunsActionAndEmits(t *testing.T) {
	t.Setenv(storage.EnvDataDir, t.TempDir())
	svc := &flowServices{}
	app, err := NewApp(svc)
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	// trigger (via DropOnTarget) runs the action with a.ctx as the parent
	// context; startup() normally supplies this from Wails, but tests never
	// call startup, so seed it directly.
	app.ctx = context.Background()

	var mu sync.Mutex
	var sawTasksChanged bool
	done := make(chan struct{})
	var closeOnce sync.Once
	app.onEmit = func(ev string, data ...any) {
		if ev != tasks.EventTasksChanged {
			return
		}
		mu.Lock()
		sawTasksChanged = true
		mu.Unlock()
		states, ok := data[0].([]model.TaskState)
		if !ok || len(states) == 0 {
			return
		}
		// states[0] is the most recent task (Runner.List reverses order).
		if states[0].Status == model.TaskDone || states[0].Status == model.TaskError {
			closeOnce.Do(func() { close(done) })
		}
	}

	tgt, err := app.AddTarget("copy-to-clipboard", "Copy", nil)
	if err != nil {
		t.Fatalf("AddTarget: %v", err)
	}
	if _, err := app.DropOnTarget(tgt.ID, model.Payload{Kind: model.ItemText, Text: "hello"}); err != nil {
		t.Fatalf("DropOnTarget: %v", err)
	}

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("no tasks:changed emitted for task completion")
	}

	if got := svc.clipboardText(); got != "hello" {
		t.Errorf("clipboard side effect = %q, want %q", got, "hello")
	}

	mu.Lock()
	defer mu.Unlock()
	if !sawTasksChanged {
		t.Error("expected tasks:changed to be emitted")
	}
}
