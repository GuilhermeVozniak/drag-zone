package main

import (
	"context"
	"strings"
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

// addTargetAndDrop wires up app.onEmit *before* adding the target — app.emit
// (app.go) falls back to the real wails runtime.EventsEmit whenever onEmit
// is nil, and that call aborts the test process because a.ctx is a bare
// context.Background(), not the context Wails' lifecycle hooks supply. Task
// 3's test avoids this by setting onEmit first; AddTarget itself emits
// EventGridChanged, so the hook must already be in place before it runs. It
// then adds actionID as a target, drops payload onto it, and blocks until
// the runner reports the resulting task as finished (TaskDone or
// TaskError), returning its final state. DropOnTarget itself only errors on
// setup problems (bad target/action id, unsupported event, per
// app_grid.go trigger); an action's own rejection of a payload kind it
// doesn't accept surfaces asynchronously as TaskState.Error, since
// Runner.Run (internal/tasks/runner.go) executes Dropped in a goroutine. So
// kind-acceptance assertions must observe the task state, not DropOnTarget's
// return value.
func addTargetAndDrop(t *testing.T, app *App, actionID string, payload model.Payload) model.TaskState {
	t.Helper()

	var mu sync.Mutex
	var final model.TaskState
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
			final = states[0]
			mu.Unlock()
			closeOnce.Do(func() { close(done) })
		}
	}

	tgt, err := app.AddTarget(actionID, "Test Target", nil)
	if err != nil {
		t.Fatalf("AddTarget(%q): %v", actionID, err)
	}

	if _, err := app.DropOnTarget(tgt.ID, payload); err != nil {
		t.Fatalf("DropOnTarget: %v", err)
	}

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("no tasks:changed emitted for task completion")
	}

	mu.Lock()
	defer mu.Unlock()
	return final
}

// TestDropOnTargetRoutesPayloadKinds asserts that DropOnTarget routes text
// and URL payloads to a kind-agnostic action ("copy-to-clipboard", which
// accepts files, text, and url per its Spec) and that dropping a kind an
// action does not accept ("zip", files-only) is genuinely rejected — with
// a real error produced by the action's own validation
// (internal/actions/builtin/zip.go Dropped returns "nothing to compress"
// when Payload.Paths is empty), not a vacuous assertion. Neither
// DropOnTarget's routing (app_grid.go trigger) nor the runner
// (internal/tasks/runner.go Run) pre-validates Payload.Kind against
// Spec().Accepts, so the rejection must come from the action itself; the
// zip case demonstrates that a files-only action really does reject a
// non-file payload instead of silently succeeding.
func TestDropOnTargetRoutesPayloadKinds(t *testing.T) {
	cases := []struct {
		name          string
		actionID      string
		payload       model.Payload
		wantErr       bool
		wantErrSubstr string
		wantClipboard string
	}{
		{
			name:          "text payload routes to clipboard action",
			actionID:      "copy-to-clipboard",
			payload:       model.Payload{Kind: model.ItemText, Text: "hello text"},
			wantClipboard: "hello text",
		},
		{
			name:          "url payload routes to clipboard action",
			actionID:      "copy-to-clipboard",
			payload:       model.Payload{Kind: model.ItemURL, Text: "https://example.com"},
			wantClipboard: "https://example.com",
		},
		{
			name:          "text payload rejected by files-only zip action",
			actionID:      "zip",
			payload:       model.Payload{Kind: model.ItemText, Text: "hello"},
			wantErr:       true,
			wantErrSubstr: "nothing to compress",
		},
		{
			name:          "url payload rejected by files-only zip action",
			actionID:      "zip",
			payload:       model.Payload{Kind: model.ItemURL, Text: "https://example.com"},
			wantErr:       true,
			wantErrSubstr: "nothing to compress",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv(storage.EnvDataDir, t.TempDir())
			svc := &flowServices{}
			app, err := NewApp(svc)
			if err != nil {
				t.Fatalf("NewApp: %v", err)
			}
			app.ctx = context.Background()

			final := addTargetAndDrop(t, app, c.actionID, c.payload)

			if c.wantErr {
				if final.Status != model.TaskError {
					t.Fatalf("task status = %v, want %v (error %q)", final.Status, model.TaskError, final.Error)
				}
				if !strings.Contains(final.Error, c.wantErrSubstr) {
					t.Errorf("task error = %q, want substring %q", final.Error, c.wantErrSubstr)
				}
				if got := svc.clipboardText(); got != "" {
					t.Errorf("rejected drop had a clipboard side effect: %q", got)
				}
				return
			}

			if final.Status != model.TaskDone {
				t.Fatalf("task status = %v, want %v (error %q)", final.Status, model.TaskDone, final.Error)
			}
			if got := svc.clipboardText(); got != c.wantClipboard {
				t.Errorf("clipboard = %q, want %q", got, c.wantClipboard)
			}
		})
	}
}
