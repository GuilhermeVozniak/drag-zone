package fsutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// skipIfRoot skips permission-based tests when running as root, since root
// bypasses Unix permission checks and the simulated failure would not occur.
func skipIfRoot(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "windows" && os.Geteuid() == 0 {
		t.Skip("skipping permission-based test: running as root")
	}
}

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

func TestUniqueDestMultipleCollisions(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(dir, "a 2.txt"), []byte("x"), 0o644)
	if got := UniqueDest(dir, "a.txt"); got != filepath.Join(dir, "a 3.txt") {
		t.Errorf("UniqueDest second collision = %q, want %q", got, filepath.Join(dir, "a 3.txt"))
	}
}

func TestTotalSize(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "one.txt"), []byte("12345"), 0o644) // 5 bytes
	sub := filepath.Join(dir, "sub")
	os.MkdirAll(sub, 0o755)
	os.WriteFile(filepath.Join(sub, "two.txt"), []byte("1234567890"), 0o644) // 10 bytes

	got := TotalSize([]string{filepath.Join(dir, "one.txt"), sub})
	if got != 15 {
		t.Errorf("TotalSize = %d, want 15", got)
	}
}

func TestTotalSizeIgnoresMissingPaths(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("abc"), 0o644)
	got := TotalSize([]string{filepath.Join(dir, "a.txt"), filepath.Join(dir, "missing")})
	if got != 3 {
		t.Errorf("TotalSize with missing path = %d, want 3", got)
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

func TestCopyPathAsExactDestination(t *testing.T) {
	srcDir, dstDir := t.TempDir(), t.TempDir()
	src := filepath.Join(srcDir, "src.txt")
	if err := os.WriteFile(src, []byte("exact"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dstDir, "renamed.txt")

	got, err := CopyPathAs(src, dst, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != dst {
		t.Errorf("CopyPathAs returned %q, want %q", got, dst)
	}
	data, err := os.ReadFile(dst)
	if err != nil || string(data) != "exact" {
		t.Fatalf("dst content = %q, err %v", data, err)
	}
	// Source must be untouched by a copy.
	if _, err := os.Stat(src); err != nil {
		t.Errorf("source missing after copy: %v", err)
	}
}

func TestCopyPathAsNonexistentSourceErrors(t *testing.T) {
	dstDir := t.TempDir()
	_, err := CopyPathAs(filepath.Join(t.TempDir(), "nope"), filepath.Join(dstDir, "out"), nil)
	if err == nil {
		t.Fatal("CopyPathAs with nonexistent source should error")
	}
}

func TestMovePathAsExactDestinationWithProgress(t *testing.T) {
	srcDir, dstDir := t.TempDir(), t.TempDir()
	src := filepath.Join(srcDir, "src.txt")
	if err := os.WriteFile(src, []byte("moved-exact"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dstDir, "exact-dst.txt")

	var reported int64
	got, err := MovePathAs(src, dst, func(n int64) { reported += n })
	if err != nil {
		t.Fatal(err)
	}
	if got != dst {
		t.Errorf("MovePathAs returned %q, want %q", got, dst)
	}
	if reported != int64(len("moved-exact")) {
		t.Errorf("reported bytes = %d, want %d", reported, len("moved-exact"))
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("source still exists after move")
	}
	data, err := os.ReadFile(dst)
	if err != nil || string(data) != "moved-exact" {
		t.Fatalf("dst content = %q, err %v", data, err)
	}
}

func TestMovePathAsNonexistentSourceErrors(t *testing.T) {
	_, err := MovePathAs(filepath.Join(t.TempDir(), "nope"), filepath.Join(t.TempDir(), "out"), nil)
	if err == nil {
		t.Fatal("MovePathAs with nonexistent source should error")
	}
}

// TestMovePathAsFallsBackToCopyWhenRenameFails exercises the copy+delete
// fallback: renaming a directory onto an existing non-empty directory fails
// (ENOTEMPTY), so MovePathAs must fall back to copyAny + os.RemoveAll(src).
func TestMovePathAsFallsBackToCopyWhenRenameFails(t *testing.T) {
	srcDir, dstParent := t.TempDir(), t.TempDir()
	src := filepath.Join(srcDir, "movedir")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "new.txt"), []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Pre-existing, non-empty destination directory: os.Rename(src, dst)
	// cannot succeed here, forcing the fallback path.
	dst := filepath.Join(dstParent, "movedir")
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dst, "existing.txt"), []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := MovePathAs(src, dst, nil)
	if err != nil {
		t.Fatalf("MovePathAs fallback: %v", err)
	}
	if got != dst {
		t.Errorf("MovePathAs returned %q, want %q", got, dst)
	}
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("source still exists after fallback move")
	}
	if data, err := os.ReadFile(filepath.Join(dst, "existing.txt")); err != nil || string(data) != "existing" {
		t.Errorf("pre-existing dst content lost: %q, err %v", data, err)
	}
	if data, err := os.ReadFile(filepath.Join(dst, "new.txt")); err != nil || string(data) != "new" {
		t.Errorf("moved content missing: %q, err %v", data, err)
	}
}

// TestMovePathAsFallbackErrorPropagates makes both os.Rename and the
// copyAny fallback fail: renaming a file onto an existing directory always
// fails, and copying a file onto an existing directory path also fails
// (os.OpenFile on a directory path errors), so MovePathAs must surface that
// error rather than silently succeeding.
func TestMovePathAsFallbackErrorPropagates(t *testing.T) {
	srcDir, dstParent := t.TempDir(), t.TempDir()
	src := filepath.Join(srcDir, "src.txt")
	if err := os.WriteFile(src, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dstParent, "already-a-dir")
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := MovePathAs(src, dst, nil)
	if err == nil {
		t.Fatal("MovePathAs should error when both rename and fallback copy fail")
	}
	// Source must be left intact since the move never completed.
	if _, statErr := os.Stat(src); statErr != nil {
		t.Errorf("source should remain after failed move: %v", statErr)
	}
}

func TestCopySymlink(t *testing.T) {
	srcDir, dstDir := t.TempDir(), t.TempDir()
	target := filepath.Join(srcDir, "target.txt")
	if err := os.WriteFile(target, []byte("linked"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(srcDir, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(dstDir, "link-copy")
	if _, err := CopyPathAs(link, dst, nil); err != nil {
		t.Fatal(err)
	}

	info, err := os.Lstat(dst)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("copied symlink is not a symlink")
	}
	resolved, err := os.Readlink(dst)
	if err != nil || resolved != target {
		t.Errorf("symlink target = %q, err %v, want %q", resolved, err, target)
	}
}

func TestCopyDirMkdirAllError(t *testing.T) {
	srcDir, dstParent := t.TempDir(), t.TempDir()
	src := filepath.Join(srcDir, "proj")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	// blocker is a regular file, so MkdirAll(blocker/sub, ...) must fail.
	blocker := filepath.Join(dstParent, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(blocker, "sub")

	if _, err := CopyPathAs(src, dst, nil); err == nil {
		t.Fatal("CopyPathAs should error when dst parent is a file")
	}
}

func TestCopyDirReadDirError(t *testing.T) {
	skipIfRoot(t)
	srcDir, dstDir := t.TempDir(), t.TempDir()
	src := filepath.Join(srcDir, "locked")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(src, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(src, 0o755) })

	dst := filepath.Join(dstDir, "copy-of-locked")
	if _, err := CopyPathAs(src, dst, nil); err == nil {
		t.Fatal("CopyPathAs should error when source dir is unreadable")
	}
}

func TestCopyFileOpenError(t *testing.T) {
	skipIfRoot(t)
	srcDir, dstDir := t.TempDir(), t.TempDir()
	src := filepath.Join(srcDir, "secret.txt")
	if err := os.WriteFile(src, []byte("x"), 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(src, 0o644) })

	dst := filepath.Join(dstDir, "out.txt")
	if _, err := CopyPathAs(src, dst, nil); err == nil {
		t.Fatal("CopyPathAs should error when source file is unreadable")
	}
}

func TestCopyFileOpenFileError(t *testing.T) {
	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "f.txt")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	// dst's parent directory does not exist, so OpenFile must fail.
	dst := filepath.Join(t.TempDir(), "missing-parent", "out.txt")
	if _, err := CopyPathAs(src, dst, nil); err == nil {
		t.Fatal("CopyPathAs should error when dst parent dir is missing")
	}
}
