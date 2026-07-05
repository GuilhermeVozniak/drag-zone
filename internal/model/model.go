// Package model defines the shared domain types exchanged between the Go
// backend and the frontend.
package model

import "time"

// ItemKind identifies what kind of content a payload carries.
type ItemKind string

const (
	ItemFiles ItemKind = "files"
	ItemText  ItemKind = "text"
	ItemURL   ItemKind = "url"
)

// Payload is the content dropped onto (or sent to) an action target.
type Payload struct {
	Kind  ItemKind `json:"kind"`
	Paths []string `json:"paths,omitempty"`
	Text  string   `json:"text,omitempty"`
	// Modifiers lists modifier keys held during the drop (e.g. "Option").
	Modifiers []string `json:"modifiers,omitempty"`
}

// HasModifier reports whether the named modifier key was held.
func (p Payload) HasModifier(name string) bool {
	for _, m := range p.Modifiers {
		if m == name {
			return true
		}
	}
	return false
}

// IsEmpty reports whether the payload carries no content.
func (p Payload) IsEmpty() bool {
	return len(p.Paths) == 0 && p.Text == ""
}

// Event names for the two ways a target can be activated.
const (
	EventDragged = "dragged"
	EventClicked = "clicked"
)

// OptionField describes one configurable option of an action, rendered as a
// form field when the user adds or edits a target.
type OptionField struct {
	Key         string   `json:"key"`
	Label       string   `json:"label"`
	Type        string   `json:"type"` // text | password | folder | file | app | select | checkbox
	Placeholder string   `json:"placeholder,omitempty"`
	Choices     []string `json:"choices,omitempty"`
	Default     string   `json:"default,omitempty"`
	Required    bool     `json:"required,omitempty"`
}

// ActionSpec describes an installable action type shown in the action library.
type ActionSpec struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Icon        string        `json:"icon"` // lucide icon name, or a path for bundle icons
	Category    string        `json:"category"`
	Events      []string      `json:"events"`  // dragged, clicked
	Accepts     []ItemKind    `json:"accepts"` // payload kinds accepted for dragged
	Options     []OptionField `json:"options,omitempty"`
	Multi       bool          `json:"multi"` // may be placed in the grid more than once
}

// Target is an action instance placed in the user's grid.
type Target struct {
	ID       string            `json:"id"`
	ActionID string            `json:"actionId"`
	Label    string            `json:"label"`
	Options  map[string]string `json:"options,omitempty"`
	Position int               `json:"position"`
	// Shortcut is a single key that triggers the target while the grid is open.
	Shortcut string `json:"shortcut,omitempty"`
}

// Option returns the named option value, or def when unset.
func (t Target) Option(key, def string) string {
	if v, ok := t.Options[key]; ok && v != "" {
		return v
	}
	return def
}

// TaskStatus is the lifecycle state of a running action task.
type TaskStatus string

const (
	TaskRunning TaskStatus = "running"
	TaskDone    TaskStatus = "done"
	TaskError   TaskStatus = "error"
)

// TaskState is a snapshot of an action execution, streamed to the frontend.
type TaskState struct {
	ID         string     `json:"id"`
	TargetID   string     `json:"targetId"`
	Title      string     `json:"title"`
	Detail     string     `json:"detail,omitempty"`
	Percent    int        `json:"percent"` // 0-100, -1 = indeterminate
	Status     TaskStatus `json:"status"`
	Error      string     `json:"error,omitempty"`
	ResultURL  string     `json:"resultUrl,omitempty"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt time.Time  `json:"finishedAt,omitzero"`
}
