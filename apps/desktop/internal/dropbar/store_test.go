package dropbar

import (
	"os"
	"path/filepath"
	"testing"
	"unicode/utf8"

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

	url, _ := s.Add(model.Payload{Kind: model.ItemURL, Text: "https://example.com"})
	if url.Label != "https://example.com" {
		t.Errorf("url label = %q", url.Label)
	}
	one, _ := s.Add(model.Payload{Kind: model.ItemFiles, Paths: []string{"/tmp/report.pdf"}})
	if one.Label != "report.pdf" {
		t.Errorf("single file label = %q", one.Label)
	}
	stack, _ := s.Add(model.Payload{Kind: model.ItemFiles, Paths: []string{"/tmp/a.txt", "/tmp/b.txt", "/tmp/c.txt"}})
	if stack.Label != "3 Items" {
		t.Errorf("stack label = %q", stack.Label)
	}

	// Separate splits the stack into singles; CombineAll re-merges them.
	if err := s.Separate(stack.ID); err != nil {
		t.Fatal(err)
	}
	if items := s.List(); len(items) != 5 { // url + single + three separated
		t.Fatalf("after separate: %d items", len(items))
	}
	if err := s.CombineAll(); err != nil {
		t.Fatal(err)
	}
	items := s.List()
	if len(items) != 2 { // url item + one merged stack of 4 files
		t.Fatalf("after combine: %+v", items)
	}
}

func TestMoveReorders(t *testing.T) {
	t.Setenv(storage.EnvDataDir, t.TempDir())
	s := load(t)
	a, _ := s.Add(model.Payload{Kind: model.ItemText, Text: "a"})
	b, _ := s.Add(model.Payload{Kind: model.ItemText, Text: "b"})
	c, _ := s.Add(model.Payload{Kind: model.ItemText, Text: "c"})

	// Move the last item to the front.
	if err := s.Move(c.ID, 0); err != nil {
		t.Fatal(err)
	}
	items := s.List()
	if items[0].ID != c.ID || items[1].ID != a.ID || items[2].ID != b.ID {
		t.Fatalf("after move-to-front: %+v", items)
	}

	// Move it back to the end (index clamps to len).
	if err := s.Move(c.ID, 99); err != nil {
		t.Fatal(err)
	}
	items = s.List()
	if items[2].ID != c.ID {
		t.Fatalf("after move-to-end: %+v", items)
	}

	if err := s.Move("missing", 0); err == nil {
		t.Error("Move with unknown id should error")
	}

	// Order persists across reload.
	s2 := load(t)
	if got := s2.List(); got[0].ID != a.ID || got[1].ID != b.ID || got[2].ID != c.ID {
		t.Fatalf("order not persisted: %+v", got)
	}
}

func TestCombine(t *testing.T) {
	t.Setenv(storage.EnvDataDir, t.TempDir())
	s := load(t)

	target, _ := s.Add(model.Payload{Kind: model.ItemFiles, Paths: []string{"/tmp/a.txt"}})
	source, _ := s.Add(model.Payload{Kind: model.ItemFiles, Paths: []string{"/tmp/b.txt"}})

	if err := s.Combine(target.ID, source.ID); err != nil {
		t.Fatal(err)
	}

	items := s.List()
	if len(items) != 1 {
		t.Fatalf("after combine: %d items, want 1: %+v", len(items), items)
	}
	got := items[0]
	if got.ID != target.ID {
		t.Errorf("combined item id = %q, want target id %q (target keeps its slot)", got.ID, target.ID)
	}
	if len(got.Paths) != 2 || got.Paths[0] != "/tmp/a.txt" || got.Paths[1] != "/tmp/b.txt" {
		t.Errorf("combined paths = %+v", got.Paths)
	}
	if got.Label != "2 Items" {
		t.Errorf("combined label = %q, want %q", got.Label, "2 Items")
	}
	if _, ok := s.Get(source.ID); ok {
		t.Error("source item still present after Combine")
	}
}

func TestCombineNoOpCases(t *testing.T) {
	t.Setenv(storage.EnvDataDir, t.TempDir())
	s := load(t)

	files, _ := s.Add(model.Payload{Kind: model.ItemFiles, Paths: []string{"/tmp/a.txt"}})
	url, _ := s.Add(model.Payload{Kind: model.ItemURL, Text: "https://example.com"})

	// Unknown target or source: no-op, no error.
	if err := s.Combine("missing", files.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Combine(files.ID, "missing"); err != nil {
		t.Fatal(err)
	}
	// Same item: no-op.
	if err := s.Combine(files.ID, files.ID); err != nil {
		t.Fatal(err)
	}
	// Non-files item on either side: no-op.
	if err := s.Combine(files.ID, url.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Combine(url.ID, files.ID); err != nil {
		t.Fatal(err)
	}
	if items := s.List(); len(items) != 2 {
		t.Fatalf("no-op cases mutated the store: %+v", items)
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

func TestLabelForTruncatesByRunes(t *testing.T) {
	t.Setenv(storage.EnvDataDir, t.TempDir())
	s := load(t)

	// 45 CJK characters: 135 bytes. A byte-slice cut at 40 would split a
	// rune and produce invalid UTF-8; a rune-safe cut stays valid.
	long := ""
	for len([]rune(long)) < 45 {
		long += "界"
	}
	it, err := s.Add(model.Payload{Kind: model.ItemText, Text: long})
	if err != nil {
		t.Fatal(err)
	}
	if !utf8.ValidString(it.Label) {
		t.Errorf("label is not valid UTF-8: %q", it.Label)
	}
	if got := len([]rune(it.Label)); got != 41 { // 40 runes + ellipsis
		t.Errorf("label length = %d runes, want 41: %q", got, it.Label)
	}
}

func TestCombinePropagatesLocked(t *testing.T) {
	t.Setenv(storage.EnvDataDir, t.TempDir())
	s := load(t)

	target, _ := s.Add(model.Payload{Kind: model.ItemFiles, Paths: []string{"/tmp/a.txt"}})
	source, _ := s.Add(model.Payload{Kind: model.ItemFiles, Paths: []string{"/tmp/b.txt"}})
	if err := s.SetLocked(source.ID, true); err != nil {
		t.Fatal(err)
	}
	if err := s.Combine(target.ID, source.ID); err != nil {
		t.Fatal(err)
	}
	got, ok := s.Get(target.ID)
	if !ok {
		t.Fatal("combined item missing")
	}
	if !got.Locked {
		t.Error("combining a locked source must keep the merged stack locked")
	}
}

func TestCombineAllPropagatesLocked(t *testing.T) {
	t.Setenv(storage.EnvDataDir, t.TempDir())
	s := load(t)

	a, _ := s.Add(model.Payload{Kind: model.ItemFiles, Paths: []string{"/tmp/a.txt"}})
	_, _ = s.Add(model.Payload{Kind: model.ItemFiles, Paths: []string{"/tmp/b.txt"}})
	if err := s.SetLocked(a.ID, true); err != nil {
		t.Fatal(err)
	}
	if err := s.CombineAll(); err != nil {
		t.Fatal(err)
	}
	items := s.List()
	if len(items) != 1 {
		t.Fatalf("after CombineAll: %d items, want 1", len(items))
	}
	if !items[0].Locked {
		t.Error("CombineAll with a locked constituent must lock the stack")
	}
}

func TestAddRollsBackOnSaveFailure(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(storage.EnvDataDir, dir)
	s := load(t)
	// Make persistence fail: a directory where dropbar.json should be.
	if err := os.Mkdir(filepath.Join(dir, fileName), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Add(model.Payload{Kind: model.ItemText, Text: "x"}); err == nil {
		t.Fatal("Add should report the save failure")
	}
	if got := s.List(); len(got) != 0 {
		t.Errorf("failed Add must not linger in memory: %+v", got)
	}
}
