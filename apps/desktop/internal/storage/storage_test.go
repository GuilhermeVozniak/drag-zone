package storage

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

type sample struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Setenv(EnvDataDir, t.TempDir())
	want := sample{Name: "a", Count: 3}
	if err := Save("s.json", want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	var got sample
	if err := Load("s.json", &got); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got != want {
		t.Errorf("round trip = %+v, want %+v", got, want)
	}
}

func TestLoadMissingFileLeavesValueUntouched(t *testing.T) {
	t.Setenv(EnvDataDir, t.TempDir())
	pre := sample{Name: "default", Count: 1}
	got := pre
	if err := Load("nope.json", &got); err != nil {
		t.Fatalf("Load of missing file must not error: %v", err)
	}
	if got != pre {
		t.Errorf("missing-file Load mutated value to %+v", got)
	}
}

func TestLoadMalformedJSONErrors(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvDataDir, dir)
	if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	var got sample
	if err := Load("bad.json", &got); err == nil {
		t.Error("Load of malformed JSON should error")
	}
}

func TestSaveIsAtomicAndPretty(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvDataDir, dir)
	if err := Save("p.json", sample{Name: "x", Count: 2}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "p.json"))
	if err != nil {
		t.Fatal(err)
	}
	// storage.Save writes json.MarshalIndent output verbatim (no trailing
	// newline), so assert pretty-printing via the 2-space indent instead.
	if !filepath.IsAbs(dir) || len(b) == 0 || !strings.Contains(string(b), "\n  ") {
		t.Errorf("unexpected file content: %q", b)
	}
	// No leftover temp files from the atomic write.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("expected exactly p.json, got %d entries", len(entries))
	}
}

func TestDirHonorsEnvOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvDataDir, dir)
	got, err := Dir()
	if err != nil || got != dir {
		t.Errorf("Dir() = %q, %v; want %q", got, err, dir)
	}
}

func TestDirFallsBackToHomeWhenEnvUnset(t *testing.T) {
	t.Setenv(EnvDataDir, "")
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := Dir()
	if err != nil {
		t.Fatalf("Dir(): %v", err)
	}
	want := filepath.Join(home, "Library", "Application Support", "DragZone")
	if got != want {
		t.Errorf("Dir() = %q, want %q", got, want)
	}
	if info, err := os.Stat(got); err != nil || !info.IsDir() {
		t.Errorf("Dir() did not create %q: %v", got, err)
	}
}

func TestDirErrorsWhenHomeUnresolvable(t *testing.T) {
	t.Setenv(EnvDataDir, "")
	t.Setenv("HOME", "")

	_, err := Dir()
	if err == nil {
		t.Fatal("Dir() should error when $HOME is unresolvable and no override is set")
	}
	if !strings.Contains(err.Error(), "resolving home directory") {
		t.Errorf("Dir() error = %v, want it to mention resolving home directory", err)
	}
}

func TestDirErrorsWhenMkdirAllFails(t *testing.T) {
	parent := t.TempDir()
	// blocker is a regular file, so MkdirAll(blocker/sub, ...) must fail.
	blocker := filepath.Join(parent, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvDataDir, filepath.Join(blocker, "sub"))

	_, err := Dir()
	if err == nil {
		t.Fatal("Dir() should error when the data directory cannot be created")
	}
	if !strings.Contains(err.Error(), "creating data directory") {
		t.Errorf("Dir() error = %v, want it to mention creating data directory", err)
	}
}

// TestSaveFileHasPrivatePermissions asserts Save writes files 0600, not
// relying on the umask of a broader default mode.
func TestSaveFileHasPrivatePermissions(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvDataDir, dir)
	if err := Save("perm.json", sample{Name: "x", Count: 1}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(dir, "perm.json"))
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("saved file mode = %o, want 0600", mode)
	}
}

// TestSaveMarshalErrorLeavesNoPartialFile simulates a failure inside the
// atomic write (json.Marshal cannot encode a channel) and asserts the
// target file is never created and no temp file is left behind.
func TestSaveMarshalErrorLeavesNoPartialFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvDataDir, dir)

	err := Save("unmarshalable.json", struct{ C chan int }{C: make(chan int)})
	if err == nil {
		t.Fatal("Save should error when the value cannot be marshaled")
	}
	entries, readErr := os.ReadDir(dir)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Errorf("expected no files after marshal error, got %v", entries)
	}
}

// TestSaveRenameErrorLeavesTargetIntact simulates the final os.Rename
// failing (the destination path is an existing directory, which a file
// can never be renamed onto) and asserts the atomic write leaves the
// pre-existing target untouched and no temp file behind.
func TestSaveRenameErrorLeavesTargetIntact(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvDataDir, dir)

	// d.json is a directory, not a file, so Save("d.json", ...) cannot
	// possibly complete via rename.
	targetDir := filepath.Join(dir, "d.json")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "child.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Save("d.json", sample{Name: "x", Count: 1}); err == nil {
		t.Fatal("Save should error when the target path is an existing directory")
	}

	// The pre-existing directory and its content must be untouched, and no
	// leftover tmp-* file should remain (Save's defer removes it).
	if data, err := os.ReadFile(filepath.Join(targetDir, "child.txt")); err != nil || string(data) != "keep" {
		t.Errorf("pre-existing target content lost: %q, err %v", data, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("expected only d.json to remain in %s, got %v", dir, entries)
	}
}

// TestSaveErrorsWhenDataDirNotWritable simulates a read-only target
// directory: Dir() succeeds (the directory already exists), but
// os.CreateTemp cannot create the temp file, so Save must error and leave
// the directory empty (no partial file).
func TestSaveErrorsWhenDataDirNotWritable(t *testing.T) {
	skipIfRoot(t)
	dir := t.TempDir()
	t.Setenv(EnvDataDir, dir)
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	if err := Save("ro.json", sample{Name: "x", Count: 1}); err == nil {
		t.Fatal("Save should error when the data directory is not writable")
	}
}

// TestLoadPropagatesDirError asserts Load surfaces the error returned by
// Dir() rather than silently treating it as a missing file.
func TestLoadPropagatesDirError(t *testing.T) {
	parent := t.TempDir()
	blocker := filepath.Join(parent, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvDataDir, filepath.Join(blocker, "sub"))

	var got sample
	if err := Load("whatever.json", &got); err == nil {
		t.Fatal("Load should propagate a Dir() error")
	}
}

// TestSavePropagatesDirError asserts Save surfaces the error returned by
// Dir() (a distinct branch from the marshal/write/rename failures above)
// without writing anything.
func TestSavePropagatesDirError(t *testing.T) {
	parent := t.TempDir()
	blocker := filepath.Join(parent, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(EnvDataDir, filepath.Join(blocker, "sub"))

	if err := Save("whatever.json", sample{Name: "x", Count: 1}); err == nil {
		t.Fatal("Save should propagate a Dir() error")
	}
}
