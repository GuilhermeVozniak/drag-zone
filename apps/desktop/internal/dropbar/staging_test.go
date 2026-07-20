package dropbar

import (
	"os"
	"path/filepath"
	"testing"

	"dragzone/internal/storage"
)

func writeSource(t *testing.T, dir, name, content string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestStagePathsCopiesIntoStageDir(t *testing.T) {
	t.Setenv(storage.EnvDataDir, t.TempDir())
	src := writeSource(t, t.TempDir(), "photo.png", "png-bytes")

	staged, err := StagePaths([]string{src})
	if err != nil {
		t.Fatal(err)
	}
	if len(staged) != 1 {
		t.Fatalf("expected 1 staged path, got %d", len(staged))
	}
	if !IsStaged(staged[0]) {
		t.Errorf("staged path %q not inside stage dir", staged[0])
	}
	data, err := os.ReadFile(staged[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "png-bytes" {
		t.Errorf("staged copy content = %q, want %q", data, "png-bytes")
	}
	// The original must remain — drag-out moves DragZone's copy, never it.
	if _, err := os.Stat(src); err != nil {
		t.Errorf("original missing after staging: %v", err)
	}
}

func TestStagePathsAvoidsNameCollisions(t *testing.T) {
	t.Setenv(storage.EnvDataDir, t.TempDir())
	a := writeSource(t, filepath.Join(t.TempDir(), "a"), "same.txt", "a")
	b := writeSource(t, filepath.Join(t.TempDir(), "b"), "same.txt", "b")

	staged, err := StagePaths([]string{a, b})
	if err != nil {
		t.Fatal(err)
	}
	if staged[0] == staged[1] {
		t.Fatalf("colliding staged paths: %q", staged[0])
	}
	for _, p := range staged {
		if !IsStaged(p) {
			t.Errorf("staged path %q not inside stage dir", p)
		}
	}
}

func TestStagePathsKeepsAlreadyStagedPaths(t *testing.T) {
	t.Setenv(storage.EnvDataDir, t.TempDir())
	src := writeSource(t, t.TempDir(), "once.txt", "x")

	first, err := StagePaths([]string{src})
	if err != nil {
		t.Fatal(err)
	}
	second, err := StagePaths(first)
	if err != nil {
		t.Fatal(err)
	}
	if second[0] != first[0] {
		t.Errorf("re-staging produced a duplicate: %q vs %q", second[0], first[0])
	}
}

func TestStagePathsRollsBackOnFailure(t *testing.T) {
	t.Setenv(storage.EnvDataDir, t.TempDir())
	ok := writeSource(t, t.TempDir(), "ok.txt", "x")
	missing := filepath.Join(t.TempDir(), "missing.txt")

	if _, err := StagePaths([]string{ok, missing}); err == nil {
		t.Fatal("expected error for missing source")
	}
	dir, err := StageDir()
	if err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("stage dir not rolled back, holds %d entries", len(entries))
	}
}

func TestUnstageDeletesOnlyStagedCopies(t *testing.T) {
	t.Setenv(storage.EnvDataDir, t.TempDir())
	src := writeSource(t, t.TempDir(), "keep.txt", "x")
	staged, err := StagePaths([]string{src})
	if err != nil {
		t.Fatal(err)
	}

	Unstage([]string{staged[0], src})

	if _, err := os.Stat(staged[0]); !os.IsNotExist(err) {
		t.Errorf("staged copy still exists after Unstage")
	}
	if _, err := os.Stat(src); err != nil {
		t.Errorf("Unstage touched a non-staged path: %v", err)
	}
}

func TestIsStaged(t *testing.T) {
	t.Setenv(storage.EnvDataDir, t.TempDir())
	dir, err := StageDir()
	if err != nil {
		t.Fatal(err)
	}
	if !IsStaged(filepath.Join(dir, "a.png")) {
		t.Error("expected path inside stage dir to be staged")
	}
	if IsStaged(dir) {
		t.Error("the stage dir itself must not count as staged")
	}
	if IsStaged(filepath.Join(dir, "..", "outside.png")) {
		t.Error("path escaping the stage dir must not count as staged")
	}
	if IsStaged(filepath.Join(t.TempDir(), "elsewhere.png")) {
		t.Error("unrelated path must not count as staged")
	}
}
