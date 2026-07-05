package grid

import (
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
