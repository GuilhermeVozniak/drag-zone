package dropbar

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"dragzone/internal/fsutil"
	"dragzone/internal/storage"
)

// stageDirName is the subdirectory of the data dir holding app-owned copies
// of dropped files.
const stageDirName = "Staged"

// StageDir returns the directory holding app-owned copies of dropped files.
func StageDir() (string, error) {
	base, err := storage.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, stageDirName), nil
}

// StagePaths copies each path into the stage dir and returns the staged
// paths. The Drop Bar always references these app-owned copies — never the
// user's originals — so dragging an item back out moves DragZone's copy and
// leaves the source file untouched. Paths already staged (e.g. re-adding an
// existing item) are kept as-is. On any failure the copies made so far are
// rolled back.
func StagePaths(paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	dir, err := StageDir()
	if err != nil {
		return nil, fmt.Errorf("resolving stage dir: %w", err)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating stage dir: %w", err)
	}
	staged := make([]string, 0, len(paths))
	for _, p := range paths {
		if IsStaged(p) {
			staged = append(staged, p)
			continue
		}
		dest, err := fsutil.CopyPath(p, dir, nil)
		if err != nil {
			Unstage(staged)
			return nil, fmt.Errorf("staging %s: %w", p, err)
		}
		staged = append(staged, dest)
	}
	return staged, nil
}

// IsStaged reports whether p lives inside the stage dir, i.e. is an
// app-owned copy DragZone may delete freely.
func IsStaged(p string) bool {
	dir, err := StageDir()
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(dir, p)
	if err != nil {
		return false
	}
	return rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

// Unstage deletes app-owned copies, best-effort. Paths outside the stage dir
// (shouldn't happen, but a hand-edited dropbar.json could hold them) are
// never touched.
func Unstage(paths []string) {
	for _, p := range paths {
		if IsStaged(p) {
			_ = os.RemoveAll(p)
		}
	}
}
