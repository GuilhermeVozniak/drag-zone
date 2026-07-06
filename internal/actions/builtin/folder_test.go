package builtin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// dropFolder copies srcFile into dstDir via the Folder action, resolving any
// conflict with prompt (nil = no UI available).
func dropFolder(t *testing.T, dstDir, srcFile string, prompt func(string, string, []string) (string, bool)) (actions.Result, error) {
	t.Helper()
	return Folder{}.Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Label: "Folder", Options: map[string]string{"path": dstDir, "mode": "copy"}},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{srcFile}},
		Progress: nullProgress{},
		Prompt:   prompt,
	})
}

// conflictSetup writes a source file ("new") and a colliding destination
// ("old"), returning the source path and destination dir.
func conflictSetup(t *testing.T) (srcFile, dstDir string) {
	t.Helper()
	srcDir, dstDir := t.TempDir(), t.TempDir()
	srcFile = filepath.Join(srcDir, "doc.txt")
	mustWrite(t, srcFile, "new")
	mustWrite(t, filepath.Join(dstDir, "doc.txt"), "old")
	return srcFile, dstDir
}

func alwaysChoose(choice string) func(string, string, []string) (string, bool) {
	return func(string, string, []string) (string, bool) { return choice, true }
}

func TestFolderConflictKeepBoth(t *testing.T) {
	srcFile, dstDir := conflictSetup(t)
	if _, err := dropFolder(t, dstDir, srcFile, alwaysChoose("Keep Both")); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dstDir, "doc.txt")); string(b) != "old" {
		t.Errorf("original should be untouched, got %q", b)
	}
	if b, err := os.ReadFile(filepath.Join(dstDir, "doc 2.txt")); err != nil || string(b) != "new" {
		t.Errorf("keep-both copy missing/wrong: %q err %v", b, err)
	}
}

func TestFolderConflictReplace(t *testing.T) {
	srcFile, dstDir := conflictSetup(t)
	if _, err := dropFolder(t, dstDir, srcFile, alwaysChoose("Replace")); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dstDir, "doc.txt")); string(b) != "new" {
		t.Errorf("destination should be replaced, got %q", b)
	}
	if _, err := os.Stat(filepath.Join(dstDir, "doc 2.txt")); !os.IsNotExist(err) {
		t.Error("replace must not leave a keep-both copy")
	}
}

func TestFolderConflictStop(t *testing.T) {
	srcFile, dstDir := conflictSetup(t)
	_, err := dropFolder(t, dstDir, srcFile, alwaysChoose("Stop"))
	if err == nil {
		t.Fatal("Stop should return an error")
	}
	if b, _ := os.ReadFile(filepath.Join(dstDir, "doc.txt")); string(b) != "old" {
		t.Errorf("Stop must not modify the destination, got %q", b)
	}
}

// With no prompt available (e.g. CLI runs), conflicts fall back to the safe,
// non-destructive keep-both behavior.
func TestFolderConflictNoPromptKeepsBoth(t *testing.T) {
	srcFile, dstDir := conflictSetup(t)
	if _, err := dropFolder(t, dstDir, srcFile, nil); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dstDir, "doc.txt")); string(b) != "old" {
		t.Errorf("original should be untouched, got %q", b)
	}
	if _, err := os.Stat(filepath.Join(dstDir, "doc 2.txt")); err != nil {
		t.Errorf("no-prompt conflict should keep both: %v", err)
	}
}
