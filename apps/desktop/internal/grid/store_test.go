package grid

import (
	"os"
	"path/filepath"
	"testing"

	"dragzone/internal/model"
	"dragzone/internal/storage"
)

func load(t *testing.T, seed []model.Target) *Store {
	t.Helper()
	s, err := Load(seed)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestListNeverNil(t *testing.T) {
	t.Setenv(storage.EnvDataDir, t.TempDir())
	if got := load(t, nil).List(); got == nil {
		t.Fatal("List() on empty store returned nil")
	}
}

func TestSeedAndPersistence(t *testing.T) {
	t.Setenv(storage.EnvDataDir, t.TempDir())

	seed := []model.Target{
		{ActionID: "folder", Label: "A"},
		{ActionID: "zip", Label: "B"},
	}
	s := load(t, seed)
	got := s.List()
	if len(got) != 2 || got[0].Label != "A" || got[0].ID == "" {
		t.Fatalf("seeded targets = %+v", got)
	}

	// A reload must NOT re-seed: the file exists now.
	reloaded := load(t, []model.Target{{ActionID: "trash", Label: "other"}})
	if got := reloaded.List(); len(got) != 2 || got[0].Label != "A" {
		t.Fatalf("reload re-seeded: %+v", got)
	}
}

func TestAddUpdateRemoveMove(t *testing.T) {
	t.Setenv(storage.EnvDataDir, t.TempDir())
	s := load(t, nil)

	a, _ := s.Add("folder", "A", map[string]string{"path": "/tmp"})
	b, _ := s.Add("zip", "B", nil)
	c, _ := s.Add("trash", "C", nil)

	a.Label = "A2"
	if err := s.Update(a); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Get(a.ID); got.Label != "A2" || got.Option("path", "") != "/tmp" {
		t.Errorf("after update: %+v", got)
	}

	if err := s.Move(c.ID, 0); err != nil {
		t.Fatal(err)
	}
	if got := s.List(); got[0].ID != c.ID || got[0].Position != 0 || got[1].ID != a.ID {
		t.Errorf("after move: %+v", got)
	}

	if err := s.Remove(b.ID); err != nil {
		t.Fatal(err)
	}
	got := s.List()
	if len(got) != 2 {
		t.Fatalf("after remove: %+v", got)
	}
	for i, tgt := range got {
		if tgt.Position != i {
			t.Errorf("positions not compacted: %+v", got)
		}
	}
}

func TestSetOptionSetAndDelete(t *testing.T) {
	t.Setenv(storage.EnvDataDir, t.TempDir())
	s := load(t, nil)
	tg, err := s.Add("folder", "F", map[string]string{"path": "/tmp"})
	if err != nil {
		t.Fatal(err)
	}

	if err := s.SetOption(tg.ID, "token", "abc"); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Get(tg.ID)
	if got.Options["token"] != "abc" || got.Options["path"] != "/tmp" {
		t.Errorf("options after set = %+v", got.Options)
	}

	// Empty value deletes the key (credential rotation cleanup).
	if err := s.SetOption(tg.ID, "token", ""); err != nil {
		t.Fatal(err)
	}
	got, _ = s.Get(tg.ID)
	if _, ok := got.Options["token"]; ok {
		t.Errorf("token should be deleted: %+v", got.Options)
	}

	// SetOption on a target with nil Options must not panic.
	t2, _ := s.Add("trash", "T", nil)
	if err := s.SetOption(t2.ID, "k", "v"); err != nil {
		t.Fatal(err)
	}
	got, _ = s.Get(t2.ID)
	if got.Options["k"] != "v" {
		t.Errorf("options on previously-nil map = %+v", got.Options)
	}

	if err := s.SetOption("missing", "k", "v"); err == nil {
		t.Error("SetOption on unknown id should error")
	}
}

func TestAddRollsBackOnSaveFailure(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(storage.EnvDataDir, dir)
	s := load(t, nil)
	// Make persistence fail: a directory where targets.json should be.
	if err := os.Mkdir(filepath.Join(dir, fileName), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Add("folder", "F", nil); err == nil {
		t.Fatal("Add should report the save failure")
	}
	if got := s.List(); len(got) != 0 {
		t.Errorf("failed Add must not linger in memory: %+v", got)
	}
}
