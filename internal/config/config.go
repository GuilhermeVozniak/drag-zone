// Package config holds user preferences and their persistence.
package config

import (
	"sync"

	"dragzone/internal/storage"
)

const fileName = "settings.json"

// Settings are the user-configurable preferences.
type Settings struct {
	LaunchAtLogin     bool   `json:"launchAtLogin"`
	GlobalShortcut    string `json:"globalShortcut"` // e.g. "F3"
	GridColumns       int    `json:"gridColumns"`
	Theme             string `json:"theme"` // system | light | dark
	DropBarKeepsItems bool   `json:"dropBarKeepsItems"`
	NotifyOnComplete  bool   `json:"notifyOnComplete"`
}

// Defaults returns the settings used on first launch.
func Defaults() Settings {
	return Settings{
		GlobalShortcut:   "F3",
		GridColumns:      4,
		Theme:            "system",
		NotifyOnComplete: true,
	}
}

// Store provides concurrency-safe access to persisted settings.
type Store struct {
	mu sync.Mutex
	s  Settings
}

// Load reads settings from disk, falling back to defaults.
func Load() (*Store, error) {
	s := Defaults()
	if err := storage.Load(fileName, &s); err != nil {
		return nil, err
	}
	return &Store{s: s}, nil
}

// Get returns the current settings snapshot.
func (st *Store) Get() Settings {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.s
}

// Set replaces the settings and persists them.
func (st *Store) Set(s Settings) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.s = s
	return storage.Save(fileName, s)
}
