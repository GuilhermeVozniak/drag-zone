package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUniqueDest(t *testing.T) {
	dir := t.TempDir()
	if got := UniqueDest(dir, "a.txt"); got != filepath.Join(dir, "a.txt") {
		t.Errorf("UniqueDest fresh = %q", got)
	}
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644)
	if got := UniqueDest(dir, "a.txt"); got != filepath.Join(dir, "a 2.txt") {
		t.Errorf("UniqueDest collision = %q", got)
	}
}

func TestCopyAndMovePath(t *testing.T) {
	srcDir, dstDir := t.TempDir(), t.TempDir()
	sub := filepath.Join(srcDir, "proj", "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(sub, "f.txt"), []byte("hello world"), 0o644)

	var copied int64
	dst, err := CopyPath(filepath.Join(srcDir, "proj"), dstDir, func(n int64) { copied += n })
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dst, "nested", "f.txt"))
	if err != nil || string(data) != "hello world" {
		t.Fatalf("copied content = %q, err %v", data, err)
	}
	if copied != int64(len("hello world")) {
		t.Errorf("progress bytes = %d", copied)
	}

	moveDst, err := MovePath(dst, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Errorf("source still exists after move")
	}
	if _, err := os.Stat(filepath.Join(moveDst, "nested", "f.txt")); err != nil {
		t.Errorf("moved file missing: %v", err)
	}
}
