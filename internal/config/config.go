// Package config holds user preferences and their persistence.
package config

import (
	"sync"

	"dragzone/internal/storage"
)

const fileName = "settings.json"

// Settings are the user-configurable preferences, mirroring Dropzone 4's
// General settings tab.
type Settings struct {
	LaunchAtLogin     bool   `json:"launchAtLogin"`
	GlobalShortcut    string `json:"globalShortcut"` // open grid, e.g. "F3"
	PopOutShortcut    string `json:"popOutShortcut"` // pop out Drop Bar, e.g. "F4"
	GridColumns       int    `json:"gridColumns"`
	GridSize          int    `json:"gridSize"` // 0-100 slider; ~33 = 100% scale
	Theme             string `json:"theme"`    // system | dark ("always use dark mode")
	AnimateGrid       bool   `json:"animateGrid"`
	ShowKeyOverlays   bool   `json:"showKeyOverlays"`
	PlaySounds        bool   `json:"playSounds"`
	DragOverlay       bool   `json:"dragOverlay"` // show grid when dragging near the menu bar
	DropBarKeepsItems bool   `json:"dropBarKeepsItems"`
	NotifyOnComplete  bool   `json:"notifyOnComplete"`
	AutoUpdateCheck   bool   `json:"autoUpdateCheck"`
	OnboardingSeen    bool   `json:"onboardingSeen"` // first-run carousel dismissed
}

// Defaults returns the settings used on first launch.
func Defaults() Settings {
	return Settings{
		GlobalShortcut:   "F3",
		PopOutShortcut:   "F4",
		GridColumns:      4,
		GridSize:         33,
		Theme:            "system",
		AnimateGrid:      true,
		ShowKeyOverlays:  true,
		PlaySounds:       true,
		DragOverlay:      true,
		NotifyOnComplete: true,
		AutoUpdateCheck:  true,
	}
}

// Scale converts the GridSize slider value into a UI scale factor.
func (s Settings) Scale() float64 {
	pct := s.GridSize
	if pct < 0 {
		pct = 0
	} else if pct > 100 {
		pct = 100
	}
	return 0.8 + float64(pct)/100*0.6 // 0.8x .. 1.4x, ~1.0 at 33
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
