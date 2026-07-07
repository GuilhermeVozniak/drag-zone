package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
