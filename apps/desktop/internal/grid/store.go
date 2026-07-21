// Package grid persists the user's arrangement of action targets.
package grid

import (
	"fmt"
	"slices"
	"sync"

	"github.com/google/uuid"

	"dragzone/internal/model"
	"dragzone/internal/storage"
)

const fileName = "targets.json"

// Store provides concurrency-safe access to the persisted grid targets.
type Store struct {
	mu      sync.Mutex
	targets []model.Target
}

// Load reads targets from disk. seed is applied (and persisted) when no
// targets file exists yet, i.e. on first launch.
func Load(seed []model.Target) (*Store, error) {
	var targets []model.Target
	if err := storage.Load(fileName, &targets); err != nil {
		return nil, err
	}
	s := &Store{targets: targets}
	if targets == nil && len(seed) > 0 {
		for i := range seed {
			seed[i].ID = uuid.NewString()
			seed[i].Position = i
		}
		s.targets = seed
		if err := s.save(); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// List returns all targets ordered by position.
func (s *Store) List() []model.Target {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sorted()
}

// sorted returns the targets ordered by position. It never returns nil, so
// the value marshals to a JSON array (not null) for the frontend. Options
// maps are cloned so callers never alias the store's mutable state.
func (s *Store) sorted() []model.Target {
	if len(s.targets) == 0 {
		return []model.Target{}
	}
	out := slices.Clone(s.targets)
	slices.SortStableFunc(out, func(a, b model.Target) int { return a.Position - b.Position })
	for i := range out {
		out[i].Options = cloneOptions(out[i].Options)
	}
	return out
}

// Get returns the target with the given ID.
func (s *Store) Get(id string) (model.Target, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.targets {
		if t.ID == id {
			t.Options = cloneOptions(t.Options)
			return t, nil
		}
	}
	return model.Target{}, fmt.Errorf("no target with id %q", id)
}

// Add appends a new target to the grid and returns it with its assigned ID.
func (s *Store) Add(actionID, label string, options map[string]string) (model.Target, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := model.Target{
		ID:       uuid.NewString(),
		ActionID: actionID,
		Label:    label,
		Options:  cloneOptions(options),
		Position: len(s.targets),
	}
	s.targets = append(s.targets, t)
	t.Options = cloneOptions(t.Options)
	return t, s.save()
}

// Update replaces the stored target with the same ID.
func (s *Store) Update(t model.Target) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.targets {
		if s.targets[i].ID == t.ID {
			t.Position = s.targets[i].Position
			t.Options = cloneOptions(t.Options)
			s.targets[i] = t
			return s.save()
		}
	}
	return fmt.Errorf("no target with id %q", t.ID)
}

// SetOption sets (or, when value is empty, deletes) one option on a target.
// It is the concurrency-safe way to persist rotated credentials without a
// read-modify-write through Get/Update.
func (s *Store) SetOption(id, key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.targets {
		if s.targets[i].ID == id {
			if s.targets[i].Options == nil {
				s.targets[i].Options = map[string]string{}
			}
			if value == "" {
				delete(s.targets[i].Options, key)
			} else {
				s.targets[i].Options[key] = value
			}
			return s.save()
		}
	}
	return fmt.Errorf("no target with id %q", id)
}

// cloneOptions copies an options map so store state is never aliased by
// callers (concurrent JSON marshaling vs mutation is a fatal data race).
func cloneOptions(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// Remove deletes a target and compacts positions.
func (s *Store) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.targets = slices.DeleteFunc(s.targets, func(t model.Target) bool { return t.ID == id })
	s.renumber()
	return s.save()
}

// Move places the target at the given position, shifting the others.
func (s *Store) Move(id string, position int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	ordered := s.sorted()
	idx := slices.IndexFunc(ordered, func(t model.Target) bool { return t.ID == id })
	if idx < 0 {
		return fmt.Errorf("no target with id %q", id)
	}
	t := ordered[idx]
	ordered = slices.Delete(ordered, idx, idx+1)
	position = max(0, min(position, len(ordered)))
	ordered = slices.Insert(ordered, position, t)
	s.targets = ordered
	s.renumber()
	return s.save()
}

func (s *Store) renumber() {
	for i := range s.targets {
		s.targets[i].Position = i
	}
}

func (s *Store) save() error {
	return storage.Save(fileName, s.targets)
}
