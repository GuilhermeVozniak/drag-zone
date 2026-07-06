// Package tasks executes actions asynchronously and streams task state to the
// frontend via events.
package tasks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// EventTasksChanged is emitted with the full task list on every change.
const EventTasksChanged = "tasks:changed"

// Emitter publishes an event to the frontend.
type Emitter func(event string, data ...any)

// Config carries the Runner's collaborators.
type Config struct {
	// Emit publishes task-state events to the frontend.
	Emit Emitter
	// Services are the host capabilities handed to running actions.
	Services actions.Services
	// NotifyEnabled is consulted at completion time so settings changes
	// apply to in-flight tasks.
	NotifyEnabled func() bool
	// SoundsEnabled gates completion/failure sounds the same way.
	SoundsEnabled func() bool
	// SaveTargetOption persists one option value on a grid target; it backs
	// Invocation.SaveOption for actions that store credentials. Optional.
	SaveTargetOption func(targetID, key, value string)
	// OnTask is called when a task starts (TaskRunning) and when it finishes
	// (TaskDone/TaskError); used for menu bar icon feedback. Optional.
	OnTask func(status model.TaskStatus)
	// OnResultURL is called when a task produces a shareable URL. Optional.
	OnResultURL func(title, url string)
}

// Runner executes actions and tracks their task states.
type Runner struct {
	mu      sync.Mutex
	tasks   map[string]*model.TaskState
	order   []string
	cancels map[string]context.CancelFunc
	cfg     Config
}

// NewRunner creates a Runner from its configuration.
func NewRunner(cfg Config) *Runner {
	if cfg.NotifyEnabled == nil {
		cfg.NotifyEnabled = func() bool { return true }
	}
	return &Runner{
		tasks:   map[string]*model.TaskState{},
		cancels: map[string]context.CancelFunc{},
		cfg:     cfg,
	}
}

// Run starts executing an action for the given event and returns the task ID.
func (r *Runner) Run(ctx context.Context, act actions.Action, target model.Target, payload model.Payload, event string) (string, error) {
	var exec func(context.Context, actions.Invocation) (actions.Result, error)
	switch event {
	case model.EventClicked:
		c, ok := act.(actions.Clicker)
		if !ok {
			return "", fmt.Errorf("action %q does not support clicks", target.Label)
		}
		exec = c.Clicked
	case model.EventDragged:
		d, ok := act.(actions.Dropper)
		if !ok {
			return "", fmt.Errorf("action %q does not accept drops", target.Label)
		}
		exec = d.Dropped
	default:
		return "", fmt.Errorf("unknown event %q", event)
	}

	id := uuid.NewString()
	state := &model.TaskState{
		ID:        id,
		TargetID:  target.ID,
		Title:     target.Label,
		Percent:   -1,
		Status:    model.TaskRunning,
		StartedAt: time.Now(),
	}
	ctx, cancel := context.WithCancel(ctx)
	r.mu.Lock()
	r.tasks[id] = state
	r.order = append(r.order, id)
	r.cancels[id] = cancel
	r.mu.Unlock()
	r.publish()
	if r.cfg.OnTask != nil {
		r.cfg.OnTask(model.TaskRunning)
	}

	inv := actions.Invocation{
		Target:   target,
		Payload:  payload,
		Progress: &reporter{runner: r, id: id},
		Services: r.cfg.Services,
	}
	if save := r.cfg.SaveTargetOption; save != nil {
		inv.SaveOption = func(key, value string) { save(target.ID, key, value) }
	}
	go r.execute(ctx, exec, inv, id)
	return id, nil
}

func (r *Runner) execute(ctx context.Context, exec func(context.Context, actions.Invocation) (actions.Result, error), inv actions.Invocation, id string) {
	res, err := exec(ctx, inv)
	if ctx.Err() == context.Canceled {
		err = fmt.Errorf("cancelled")
	}

	r.mu.Lock()
	state := r.tasks[id]
	state.FinishedAt = time.Now()
	if err != nil {
		state.Status = model.TaskError
		state.Error = err.Error()
	} else {
		state.Status = model.TaskDone
		state.Percent = 100
		if res.Message != "" {
			state.Detail = res.Message
		}
		state.ResultURL = res.URL
	}
	title := state.Title
	delete(r.cancels, id)
	r.mu.Unlock()
	r.publish()
	if r.cfg.OnTask != nil {
		r.cfg.OnTask(state.Status)
	}
	if err == nil && res.URL != "" && r.cfg.OnResultURL != nil {
		r.cfg.OnResultURL(title, res.URL)
	}

	if err != nil {
		r.cfg.Services.Notify(title+" failed", err.Error())
		if r.cfg.SoundsEnabled == nil || r.cfg.SoundsEnabled() {
			r.cfg.Services.PlaySound("Basso")
		}
	} else if res.Message != "" && r.cfg.NotifyEnabled() {
		r.cfg.Services.Notify(title, res.Message)
		if r.cfg.SoundsEnabled == nil || r.cfg.SoundsEnabled() {
			r.cfg.Services.PlaySound("Glass")
		}
	}
}

// List returns task states, most recent first.
func (r *Runner) List() []model.TaskState {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]model.TaskState, 0, len(r.order))
	for i := len(r.order) - 1; i >= 0; i-- {
		out = append(out, *r.tasks[r.order[i]])
	}
	return out
}

// Cancel aborts a running task; its action sees a cancelled context.
func (r *Runner) Cancel(id string) {
	r.mu.Lock()
	cancel := r.cancels[id]
	r.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// Dismiss removes a finished task from the list.
func (r *Runner) Dismiss(id string) {
	r.mu.Lock()
	if t, ok := r.tasks[id]; ok && t.Status != model.TaskRunning {
		delete(r.tasks, id)
		for i, oid := range r.order {
			if oid == id {
				r.order = append(r.order[:i], r.order[i+1:]...)
				break
			}
		}
	}
	r.mu.Unlock()
	r.publish()
}

func (r *Runner) publish() {
	r.cfg.Emit(EventTasksChanged, r.List())
}

// reporter adapts progress calls onto a task state.
type reporter struct {
	runner *Runner
	id     string
}

func (p *reporter) Detail(text string) {
	p.runner.mu.Lock()
	if t, ok := p.runner.tasks[p.id]; ok {
		t.Detail = text
	}
	p.runner.mu.Unlock()
	p.runner.publish()
}

func (p *reporter) Percent(pct int) {
	p.runner.mu.Lock()
	if t, ok := p.runner.tasks[p.id]; ok {
		t.Percent = pct
	}
	p.runner.mu.Unlock()
	p.runner.publish()
}
