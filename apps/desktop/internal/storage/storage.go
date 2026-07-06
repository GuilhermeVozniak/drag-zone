// Package storage persists application state as JSON files in the app's
// data directory under ~/Library/Application Support.
package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const appDirName = "DragZone"

// EnvDataDir overrides the data directory when set; used by tests to keep
// store operations off the real user data.
const EnvDataDir = "DRAGZONE_DATA_DIR"

// Dir returns the application data directory, creating it if needed.
func Dir() (string, error) {
	dir := os.Getenv(EnvDataDir)
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolving home directory: %w", err)
		}
		dir = filepath.Join(home, "Library", "Application Support", appDirName)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating data directory: %w", err)
	}
	return dir, nil
}

// Load reads the named JSON file into v. A missing file is not an error and
// leaves v untouched, so callers can pre-populate v with defaults.
func Load(name string, v any) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(dir, name))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading %s: %w", name, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("parsing %s: %w", name, err)
	}
	return nil
}

// Save writes v as pretty-printed JSON to the named file atomically.
func Save(name string, v any) error {
	dir, err := Dir()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding %s: %w", name, err)
	}
	tmp, err := os.CreateTemp(dir, name+".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), filepath.Join(dir, name))
}
