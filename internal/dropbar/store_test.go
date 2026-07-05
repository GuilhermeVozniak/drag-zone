package dropbar

import (
	"testing"

	"dragzone/internal/model"
	"dragzone/internal/storage"
)

func load(t *testing.T) *Store {
	t.Helper()
	s, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestListNeverNil(t *testing.T) {
	t.Setenv(storage.EnvDataDir, t.TempDir())
	// An empty store must return a non-nil slice so it marshals to a JSON
	// array, not null (which would crash the frontend on .length).
	if got := load(t).List(); got == nil {
		t.Fatal("List() on empty store returned nil")
	}
}

func TestAddLabelsAndStacks(t *testing.T) {
	t.Setenv(storage.EnvDataDir, t.TempDir())
	s := load(t)

	one, _ := s.Add(model.Payload{Kind: model.ItemFiles, Paths: []string{"/tmp/report.pdf"}})
	if one.Label != "report.pdf" {
		t.Errorf("single file label = %q", one.Label)
	}
	stack, _ := s.Add(model.Payload{Kind: model.ItemFiles, Paths: []string{"/tmp/a.txt", "/tmp/b.txt", "/tmp/c.txt"}})
	if stack.Label != "a.txt +2" {
		t.Errorf("stack label = %q", stack.Label)
	}
	url, _ := s.Add(model.Payload{Kind: model.ItemURL, Text: "https://example.com"})
	if url.Label != "https://example.com" {
		t.Errorf("url label = %q", url.Label)
	}
}

func TestLockRenameClearPersistence(t *testing.T) {
	t.Setenv(storage.EnvDataDir, t.TempDir())
	s := load(t)

	it, _ := s.Add(model.Payload{Kind: model.ItemFiles, Paths: []string{"/tmp/x.txt"}})
	if err := s.SetLocked(it.ID, true); err != nil {
		t.Fatal(err)
	}
	if err := s.Rename(it.ID, "custom"); err != nil {
		t.Fatal(err)
	}

	// A fresh store must see the persisted state.
	reloaded := load(t)
	got, ok := reloaded.Get(it.ID)
	if !ok || !got.Locked || got.Label != "custom" {
		t.Fatalf("persisted item = %+v ok=%v", got, ok)
	}

	// Empty rename resets to the derived label.
	if err := reloaded.Rename(it.ID, ""); err != nil {
		t.Fatal(err)
	}
	if got, _ := reloaded.Get(it.ID); got.Label != "x.txt" {
		t.Errorf("reset label = %q", got.Label)
	}

	if err := reloaded.Clear(); err != nil {
		t.Fatal(err)
	}
	if items := reloaded.List(); len(items) != 0 {
		t.Errorf("after clear: %+v", items)
	}
}
