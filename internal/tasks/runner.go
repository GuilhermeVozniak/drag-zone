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

// Runner executes actions and tracks their task states.
type Runner struct {
	mu       sync.Mutex
	tasks    map[string]*model.TaskState
	order    []string
	emit     Emitter
	services actions.Services
	notify   func() bool // whether completion notifications are enabled
}

// NewRunner creates a Runner. notifyEnabled is consulted at completion time so
// settings changes apply to in-flight tasks.
func NewRunner(emit Emitter, services actions.Services, notifyEnabled func() bool) *Runner {
	return &Runner{
		tasks:    map[string]*model.TaskState{},
		emit:     emit,
		services: services,
		notify:   notifyEnabled,
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
	r.mu.Lock()
	r.tasks[id] = state
	r.order = append(r.order, id)
	r.mu.Unlock()
	r.publish()

	go r.execute(ctx, exec, actions.Invocation{
		Target:   target,
		Payload:  payload,
		Progress: &reporter{runner: r, id: id},
		Services: r.services,
	}, id)
	return id, nil
}

func (r *Runner) execute(ctx context.Context, exec func(context.Context, actions.Invocation) (actions.Result, error), inv actions.Invocation, id string) {
	res, err := exec(ctx, inv)

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
	r.mu.Unlock()
	r.publish()

	if err != nil {
		r.services.Notify(title+" failed", err.Error())
	} else if res.Message != "" && r.notify() {
		r.services.Notify(title, res.Message)
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
	r.emit(EventTasksChanged, r.List())
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
