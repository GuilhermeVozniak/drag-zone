// Package dropbar holds the shelf of stashed items shown above the grid.
package dropbar

import (
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
// marshals to a JSON array (not null) for the frontend.
func (s *Store) List() []Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.items) == 0 {
		return []Item{}
	}
	return slices.Clone(s.items)
}

// Get returns the item with the given ID.
func (s *Store) Get(id string) (Item, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, it := range s.items {
		if it.ID == id {
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
		Paths:   p.Paths,
		Text:    p.Text,
		Label:   labelFor(p),
		AddedAt: time.Now(),
	}
	s.items = append(s.items, it)
	return it, s.save()
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
		return filepath.Base(p.Paths[0]) + " +" + strconv.Itoa(len(p.Paths)-1)
	case p.Kind == model.ItemURL:
		return p.Text
	default:
		if len(p.Text) > 40 {
			return p.Text[:40] + "…"
		}
		return p.Text
	}
}
