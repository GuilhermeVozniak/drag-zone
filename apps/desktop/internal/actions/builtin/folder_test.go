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

// dropFolderMode drops srcFile into dstDir with an explicit configured mode
// and drop modifiers, bypassing dropFolder's fixed "copy" mode so Option-invert
// can be exercised in both directions.
func dropFolderMode(t *testing.T, dstDir, srcFile, mode string, modifiers []string, progress actions.Progress) (actions.Result, error) {
	t.Helper()
	return Folder{}.Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Label: "Folder", Options: map[string]string{"path": dstDir, "mode": mode}},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{srcFile}, Modifiers: modifiers},
		Progress: progress,
	})
}

// TestFolderOptionInvertsCopyToMove: dropping with Option held on a
// configured-copy target performs a move instead — the source must be gone
// afterward.
func TestFolderOptionInvertsCopyToMove(t *testing.T) {
	srcDir, dstDir := t.TempDir(), t.TempDir()
	srcFile := filepath.Join(srcDir, "doc.txt")
	mustWrite(t, srcFile, "payload")

	if _, err := dropFolderMode(t, dstDir, srcFile, "copy", []string{"Option"}, nullProgress{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(srcFile); !os.IsNotExist(err) {
		t.Errorf("Option should invert copy to move: source still exists (err=%v)", err)
	}
	if b, err := os.ReadFile(filepath.Join(dstDir, "doc.txt")); err != nil || string(b) != "payload" {
		t.Errorf("destination missing/wrong after inverted move: %q err %v", b, err)
	}
}

// TestFolderOptionInvertsMoveToCopy: dropping with Option held on a
// configured-move target performs a copy instead — the source must remain.
func TestFolderOptionInvertsMoveToCopy(t *testing.T) {
	srcDir, dstDir := t.TempDir(), t.TempDir()
	srcFile := filepath.Join(srcDir, "doc.txt")
	mustWrite(t, srcFile, "payload")

	if _, err := dropFolderMode(t, dstDir, srcFile, "move", []string{"Option"}, nullProgress{}); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(srcFile); err != nil || string(b) != "payload" {
		t.Errorf("Option should invert move to copy: source missing/changed (b=%q err=%v)", b, err)
	}
	if b, err := os.ReadFile(filepath.Join(dstDir, "doc.txt")); err != nil || string(b) != "payload" {
		t.Errorf("destination missing/wrong after inverted copy: %q err %v", b, err)
	}
}

// TestFolderMoveWithoutOptionRemovesSource is the non-inverted control for the
// two Option tests above: a plain move (no modifier) must also remove src.
func TestFolderMoveWithoutOptionRemovesSource(t *testing.T) {
	srcDir, dstDir := t.TempDir(), t.TempDir()
	srcFile := filepath.Join(srcDir, "doc.txt")
	mustWrite(t, srcFile, "payload")

	if _, err := dropFolderMode(t, dstDir, srcFile, "move", nil, nullProgress{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(srcFile); !os.IsNotExist(err) {
		t.Errorf("plain move should remove source (err=%v)", err)
	}
}

// recProgress records every Detail/Percent call so tests can assert progress
// is reported by bytes copied/moved.
type recProgress struct {
	details  []string
	percents []int
}

func (r *recProgress) Detail(text string) { r.details = append(r.details, text) }
func (r *recProgress) Percent(pct int)    { r.percents = append(r.percents, pct) }

// TestFolderProgressReportsBytes verifies Dropped reports the dropped file's
// name via Detail and drives Percent up to 100 as bytes are copied.
func TestFolderProgressReportsBytes(t *testing.T) {
	srcDir, dstDir := t.TempDir(), t.TempDir()
	srcFile := filepath.Join(srcDir, "big.bin")
	mustWrite(t, srcFile, string(make([]byte, 4096)))

	prog := &recProgress{}
	if _, err := dropFolderMode(t, dstDir, srcFile, "copy", nil, prog); err != nil {
		t.Fatal(err)
	}
	if len(prog.details) != 1 || prog.details[0] != "big.bin" {
		t.Errorf("Detail calls = %v, want [\"big.bin\"]", prog.details)
	}
	if len(prog.percents) == 0 {
		t.Fatal("expected at least one Percent report")
	}
	if last := prog.percents[len(prog.percents)-1]; last != 100 {
		t.Errorf("final Percent = %d, want 100", last)
	}
	for i := 1; i < len(prog.percents); i++ {
		if prog.percents[i] < prog.percents[i-1] {
			t.Errorf("Percent went backwards: %v", prog.percents)
			break
		}
	}
}
