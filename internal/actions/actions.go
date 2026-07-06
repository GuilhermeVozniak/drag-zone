// Package actions defines the action engine: the interfaces every action
// implements, the host services actions may use, and the registry of
// available action types.
package actions

import (
	"context"
	"fmt"

	"dragzone/internal/model"
)

// Services are host capabilities provided to running actions.
type Services interface {
	CopyToClipboard(text string) error
	ReadClipboard() (string, error)
	Notify(title, body string)
	// PlaySound plays a named system sound (e.g. "Glass", "Basso").
	PlaySound(name string)
	OpenURL(url string) error
	OpenPath(path string) error
	Reveal(path string) error
	Trash(paths []string) error
	AirDrop(paths []string) error
}

// Progress lets a running action report status back to the task runner.
type Progress interface {
	Detail(text string)
	Percent(pct int)
}

// Invocation carries everything an action needs for one execution.
type Invocation struct {
	Target   model.Target
	Payload  model.Payload
	Progress Progress
	Services Services
	// SaveOption persists one option value on the invoked target (an empty
	// value deletes the key). Actions use it to store rotated credentials,
	// e.g. OAuth refresh tokens. May be nil when the host provides no
	// persistence.
	SaveOption func(key, value string)
	// Prompt asks the user to choose among options mid-run (e.g. resolving a
	// file conflict), returning the chosen option and whether the user
	// answered (false on cancel/timeout). Nil when the host provides no UI
	// (e.g. CLI runs); actions must fall back to a safe default when it is nil.
	Prompt func(title, message string, choices []string) (choice string, ok bool)
}

// Result is what an action produced, used for grid status and notifications.
type Result struct {
	Message string // human-readable completion message
	URL     string // e.g. an upload URL; shown and openable in the UI
}

// Action is the base interface all actions implement.
type Action interface {
	Spec() model.ActionSpec
}

// Dropper handles payloads dropped onto its target.
type Dropper interface {
	Action
	Dropped(ctx context.Context, inv Invocation) (Result, error)
}

// Clicker handles clicks on its target.
type Clicker interface {
	Action
	Clicked(ctx context.Context, inv Invocation) (Result, error)
}

// Registry holds the available action types in registration order.
type Registry struct {
	order []string
	byID  map[string]Action
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{byID: map[string]Action{}}
}

// Register adds an action type. Duplicate IDs panic: they are programmer error.
func (r *Registry) Register(a Action) {
	if err := r.TryRegister(a); err != nil {
		panic(err)
	}
}

// TryRegister adds an action type, reporting duplicates as an error. Use for
// user-installed actions where a collision is not a programmer error.
func (r *Registry) TryRegister(a Action) error {
	id := a.Spec().ID
	if _, dup := r.byID[id]; dup {
		return fmt.Errorf("actions: duplicate action id %q", id)
	}
	r.byID[id] = a
	r.order = append(r.order, id)
	return nil
}

// Get returns the action with the given ID.
func (r *Registry) Get(id string) (Action, error) {
	a, ok := r.byID[id]
	if !ok {
		return nil, fmt.Errorf("unknown action %q", id)
	}
	return a, nil
}

// Specs lists all registered action specs in registration order. It never
// returns nil so the value marshals to a JSON array for the frontend.
func (r *Registry) Specs() []model.ActionSpec {
	specs := make([]model.ActionSpec, 0, len(r.order))
	for _, id := range r.order {
		specs = append(specs, r.byID[id].Spec())
	}
	return specs
}
