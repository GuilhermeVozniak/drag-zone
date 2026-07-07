package tasks

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// fakeAction runs fn on Dropped/Clicked; both events supported.
type fakeAction struct {
	fn func(actions.Invocation) (actions.Result, error)
}

func (fakeAction) Spec() model.ActionSpec { return model.ActionSpec{ID: "fake", Name: "Fake"} }
func (a fakeAction) Dropped(_ context.Context, inv actions.Invocation) (actions.Result, error) {
	return a.fn(inv)
}
func (a fakeAction) Clicked(_ context.Context, inv actions.Invocation) (actions.Result, error) {
	return a.fn(inv)
}

type recSvc struct {
	mu     sync.Mutex
	notes  []string
	sounds []string
}

func (s *recSvc) CopyToClipboard(string) error   { return nil }
func (s *recSvc) ReadClipboard() (string, error) { return "", nil }
func (s *recSvc) Notify(t, b string)             { s.mu.Lock(); s.notes = append(s.notes, t); s.mu.Unlock() }
func (s *recSvc) PlaySound(n string)             { s.mu.Lock(); s.sounds = append(s.sounds, n); s.mu.Unlock() }
func (s *recSvc) OpenURL(string) error           { return nil }
func (s *recSvc) OpenPath(string) error          { return nil }
func (s *recSvc) Reveal(string) error            { return nil }
func (s *recSvc) Trash([]string) error           { return nil }
func (s *recSvc) AirDrop([]string) error         { return nil }

// newRunner returns a runner whose Emit signals `changed` on every publish, so
// tests can wait for the terminal state.
func newRunner(t *testing.T, svc actions.Services) (*Runner, chan struct{}) {
	t.Helper()
	changed := make(chan struct{}, 64)
	r := NewRunner(Config{
		Emit:          func(string, ...any) { changed <- struct{}{} },
		Services:      svc,
		NotifyEnabled: func() bool { return true },
		SoundsEnabled: func() bool { return true },
	})
	return r, changed
}

// waitDone polls List() until the single task reaches a terminal status.
func waitDone(t *testing.T, r *Runner) model.TaskState {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		list := r.List()
		if len(list) == 1 && list[0].Status != model.TaskRunning {
			return list[0]
		}
		select {
		case <-deadline:
			t.Fatalf("task did not finish; list=%+v", list)
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func TestRunSuccessNotifiesAndPlaysGlass(t *testing.T) {
	svc := &recSvc{}
	r, _ := newRunner(t, svc)
	act := fakeAction{fn: func(inv actions.Invocation) (actions.Result, error) {
		inv.Progress.Percent(50)
		return actions.Result{Message: "done", URL: "https://x/y"}, nil
	}}
	id, err := r.Run(context.Background(), act, model.Target{ID: "t", Label: "T"}, model.Payload{}, model.EventDragged)
	if err != nil || id == "" {
		t.Fatalf("Run: id=%q err=%v", id, err)
	}
	st := waitDone(t, r)
	if st.Status != model.TaskDone || st.Detail != "done" || st.Percent != 100 || st.ResultURL != "https://x/y" {
		t.Errorf("final state = %+v", st)
	}
	svc.mu.Lock()
	defer svc.mu.Unlock()
	if len(svc.notes) != 1 || len(svc.sounds) != 1 || svc.sounds[0] != "Glass" {
		t.Errorf("notes=%v sounds=%v", svc.notes, svc.sounds)
	}
}

func TestRunErrorNotifiesAndPlaysBasso(t *testing.T) {
	svc := &recSvc{}
	r, _ := newRunner(t, svc)
	act := fakeAction{fn: func(actions.Invocation) (actions.Result, error) {
		return actions.Result{}, errors.New("kaboom")
	}}
	if _, err := r.Run(context.Background(), act, model.Target{ID: "t", Label: "T"}, model.Payload{}, model.EventDragged); err != nil {
		t.Fatal(err)
	}
	st := waitDone(t, r)
	if st.Status != model.TaskError || st.Error != "kaboom" {
		t.Errorf("final state = %+v", st)
	}
	svc.mu.Lock()
	defer svc.mu.Unlock()
	if len(svc.sounds) != 1 || svc.sounds[0] != "Basso" {
		t.Errorf("sounds = %v, want [Basso]", svc.sounds)
	}
}

func TestRunRejectsUnsupportedEvent(t *testing.T) {
	r, _ := newRunner(t, &recSvc{})
	// A Dropper-only action cannot be clicked.
	dropOnly := struct {
		actions.Action
		actions.Dropper
	}{}
	_ = dropOnly
	// Use fakeAction but call an invalid event string.
	if _, err := r.Run(context.Background(), fakeAction{fn: func(actions.Invocation) (actions.Result, error) {
		return actions.Result{}, nil
	}}, model.Target{Label: "T"}, model.Payload{}, "bogus"); err == nil {
		t.Error("unknown event should error")
	}
}

func TestDismissRemovesFinishedTask(t *testing.T) {
	r, _ := newRunner(t, &recSvc{})
	id, _ := r.Run(context.Background(), fakeAction{fn: func(actions.Invocation) (actions.Result, error) {
		return actions.Result{Message: "ok"}, nil
	}}, model.Target{ID: "t", Label: "T"}, model.Payload{}, model.EventDragged)
	waitDone(t, r)
	r.Dismiss(id)
	if len(r.List()) != 0 {
		t.Errorf("dismiss left tasks: %+v", r.List())
	}
}

func TestCancelAbortsRunningTask(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{})
	r, _ := newRunner(t, &recSvc{})
	act := fakeAction{fn: func(inv actions.Invocation) (actions.Result, error) {
		close(started)
		<-release // block until cancelled context fires? we simply wait
		return actions.Result{Message: "late"}, nil
	}}
	id, _ := r.Run(context.Background(), act, model.Target{ID: "t", Label: "T"}, model.Payload{}, model.EventDragged)
	<-started
	r.Cancel(id)
	close(release)
	st := waitDone(t, r)
	if st.Status != model.TaskError || st.Error != "cancelled" {
		t.Errorf("cancelled task state = %+v", st)
	}
}
