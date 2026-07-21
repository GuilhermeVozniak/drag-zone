// Package dropbar holds the shelf of stashed items shown above the grid.
package dropbar

import (
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"

	"dragzone/internal/model"
	"dragzone/internal/storage"
)

const fileName = "dropbar.json"

// Item is one stashed entry. Multiple paths in one item form a stack.
type Item struct {
	ID      string         `json:"id"`
	Kind    model.ItemKind `json:"kind"`
	Paths   []string       `json:"paths,omitempty"`
	Text    string         `json:"text,omitempty"`
	Label   string         `json:"label"`
	Locked  bool           `json:"locked"` // locked items stay after drag-out
	AddedAt time.Time      `json:"addedAt"`
}

// Payload converts the item back into a droppable payload.
func (it Item) Payload() model.Payload {
	return model.Payload{Kind: it.Kind, Paths: it.Paths, Text: it.Text}
}

// Store provides concurrency-safe access to persisted drop bar items.
type Store struct {
	mu    sync.Mutex
	items []Item
}

// Load reads the drop bar from disk.
func Load() (*Store, error) {
	var items []Item
	if err := storage.Load(fileName, &items); err != nil {
		return nil, err
	}
	return &Store{items: items}, nil
}

// List returns all items, oldest first. It never returns nil, so the value
// marshals to a JSON array (not null) for the frontend. Paths slices are
// cloned so callers never alias the store's mutable state.
func (s *Store) List() []Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.items) == 0 {
		return []Item{}
	}
	out := slices.Clone(s.items)
	for i := range out {
		out[i].Paths = slices.Clone(out[i].Paths)
	}
	return out
}

// Get returns the item with the given ID.
func (s *Store) Get(id string) (Item, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, it := range s.items {
		if it.ID == id {
			it.Paths = slices.Clone(it.Paths)
			return it, true
		}
	}
	return Item{}, false
}

// Add stashes a payload and returns the created item.
func (s *Store) Add(p model.Payload) (Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	it := Item{
		ID:      uuid.NewString(),
		Kind:    p.Kind,
		Paths:   slices.Clone(p.Paths),
		Text:    p.Text,
		Label:   labelFor(p),
		AddedAt: time.Now(),
	}
	s.items = append(s.items, it)
	it.Paths = slices.Clone(it.Paths)
	return it, s.save()
}

// Separate splits a stack into one item per file, in place of the stack.
func (s *Store) Separate(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, it := range s.items {
		if it.ID != id || len(it.Paths) < 2 {
			continue
		}
		singles := make([]Item, 0, len(it.Paths))
		for _, p := range it.Paths {
			payload := model.Payload{Kind: model.ItemFiles, Paths: []string{p}}
			singles = append(singles, Item{
				ID:      uuid.NewString(),
				Kind:    model.ItemFiles,
				Paths:   payload.Paths,
				Label:   labelFor(payload),
				Locked:  it.Locked,
				AddedAt: it.AddedAt,
			})
		}
		s.items = append(s.items[:i], append(singles, s.items[i+1:]...)...)
		return s.save()
	}
	return nil
}

// CombineAll merges every file item into a single stack; text items stay.
func (s *Store) CombineAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var paths []string
	var rest []Item
	for _, it := range s.items {
		if it.Kind == model.ItemFiles {
			paths = append(paths, it.Paths...)
		} else {
			rest = append(rest, it)
		}
	}
	if len(paths) == 0 {
		return nil
	}
	payload := model.Payload{Kind: model.ItemFiles, Paths: paths}
	stack := Item{
		ID:      uuid.NewString(),
		Kind:    model.ItemFiles,
		Paths:   paths,
		Label:   labelFor(payload),
		AddedAt: time.Now(),
	}
	s.items = append(rest, stack)
	return s.save()
}

// Combine merges sourceID's paths into targetID, forming (or growing) a
// stack: targetID keeps its position and gets a relabeled "N Items" name,
// sourceID is removed. No-op if either ID is missing, they're the same item,
// or either item isn't a file item (stacks only ever hold files).
func (s *Store) Combine(targetID, sourceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if targetID == sourceID {
		return nil
	}
	targetIdx := -1
	var target, source Item
	foundSource := false
	for i, it := range s.items {
		if it.ID == targetID {
			targetIdx = i
			target = it
		}
		if it.ID == sourceID {
			source = it
			foundSource = true
		}
	}
	if targetIdx == -1 || !foundSource {
		return nil
	}
	if target.Kind != model.ItemFiles || source.Kind != model.ItemFiles {
		return nil
	}
	merged := append(append([]string{}, target.Paths...), source.Paths...)
	payload := model.Payload{Kind: model.ItemFiles, Paths: merged}
	s.items[targetIdx].Paths = merged
	s.items[targetIdx].Label = labelFor(payload)
	s.items = slices.DeleteFunc(s.items, func(it Item) bool { return it.ID == sourceID })
	return s.save()
}

// Rename sets a custom label; an empty name resets to the derived label.
func (s *Store) Rename(id, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.items {
		if s.items[i].ID == id {
			if name == "" {
				name = labelFor(s.items[i].Payload())
			}
			s.items[i].Label = name
			return s.save()
		}
	}
	return nil
}

// SetLocked toggles whether an item survives being dragged out.
func (s *Store) SetLocked(id string, locked bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.items {
		if s.items[i].ID == id {
			s.items[i].Locked = locked
			return s.save()
		}
	}
	return nil
}

// Remove deletes one item.
func (s *Store) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = slices.DeleteFunc(s.items, func(it Item) bool { return it.ID == id })
	return s.save()
}

// Move places the item at the given index, shifting the others. The index
// refers to the list after the item is removed (mirroring grid.Move).
func (s *Store) Move(id string, index int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := slices.IndexFunc(s.items, func(it Item) bool { return it.ID == id })
	if idx < 0 {
		return fmt.Errorf("no item with id %q", id)
	}
	it := s.items[idx]
	s.items = slices.Delete(s.items, idx, idx+1)
	index = max(0, min(index, len(s.items)))
	s.items = slices.Insert(s.items, index, it)
	return s.save()
}

// Clear removes all items.
func (s *Store) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = nil
	return s.save()
}

func (s *Store) save() error {
	return storage.Save(fileName, s.items)
}

func labelFor(p model.Payload) string {
	switch {
	case len(p.Paths) == 1:
		return filepath.Base(p.Paths[0])
	case len(p.Paths) > 1:
		// Stacks are labeled like Dropzone: "3 Items".
		return strconv.Itoa(len(p.Paths)) + " Items"
	case p.Kind == model.ItemURL:
		return p.Text
	default:
		if len(p.Text) > 40 {
			return p.Text[:40] + "…"
		}
		return p.Text
	}
}
