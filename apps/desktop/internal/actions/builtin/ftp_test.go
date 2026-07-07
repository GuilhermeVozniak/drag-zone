package builtin

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestCollectUploadEntriesFlattensDirs(t *testing.T) {
	root := t.TempDir()
	// A single loose file plus a directory tree.
	loose := filepath.Join(root, "loose.txt")
	if err := os.WriteFile(loose, []byte("12345"), 0o644); err != nil {
		t.Fatal(err)
	}
	tree := filepath.Join(root, "tree")
	if err := os.MkdirAll(filepath.Join(tree, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tree, "a.txt"), []byte("ab"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tree, "sub", "b.txt"), []byte("c"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, total, err := collectUploadEntries([]string{loose, tree})
	if err != nil {
		t.Fatal(err)
	}
	if total != 5+2+1 {
		t.Errorf("total bytes = %d, want 8", total)
	}
	var rels []string
	for _, e := range entries {
		rels = append(rels, e.rel)
	}
	sort.Strings(rels)
	want := []string{"loose.txt", "tree/a.txt", "tree/sub/b.txt"}
	if len(rels) != 3 || rels[0] != want[0] || rels[1] != want[1] || rels[2] != want[2] {
		t.Errorf("rels = %v, want %v", rels, want)
	}
}

func TestCollectUploadEntriesErrors(t *testing.T) {
	if _, _, err := collectUploadEntries([]string{"/does/not/exist/xyz"}); err == nil {
		t.Error("missing path should error")
	}
	if _, _, err := collectUploadEntries([]string{t.TempDir()}); err == nil {
		t.Error("empty dir (no files) should error with 'nothing to upload'")
	}
}
