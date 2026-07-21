package bundles

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeBundle(t *testing.T, dir, name, uid string) string {
	t.Helper()
	bundle := filepath.Join(dir, name+".dzbundle")
	if err := os.MkdirAll(bundle, 0o755); err != nil {
		t.Fatal(err)
	}
	script := "# Dropzone Action Info\n# Name: " + name + "\n# UniqueID: " + uid + "\n# Events: Dragged\n\ndef dragged\nend\n"
	if err := os.WriteFile(filepath.Join(bundle, "action.rb"), []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	return bundle
}

func TestCopyForEditingFreshUniqueID(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	bundle := writeBundle(t, src, "My Action", "111")

	act, err := LoadBundle(bundle, Host{})
	if err != nil {
		t.Fatal(err)
	}
	newBundle, newScript, err := act.CopyForEditing(dst)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(newBundle) != dst {
		t.Errorf("copy landed in %q, want %q", filepath.Dir(newBundle), dst)
	}
	dup, err := LoadBundle(newBundle, Host{})
	if err != nil {
		t.Fatal(err)
	}
	if dup.Spec().ID == act.Spec().ID {
		t.Error("copy kept the original UniqueID; it must register alongside it")
	}
	if dup.Spec().Name != "My Action" {
		t.Errorf("copy name = %q", dup.Spec().Name)
	}
	data, err := os.ReadFile(newScript)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "UniqueID: 111") {
		t.Error("copied script still carries the original UniqueID")
	}
}

func TestCopyForEditingUniqueDirOnNameClash(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	bundle := writeBundle(t, src, "My Action", "111")
	writeBundle(t, dst, "My Action", "222")

	act, err := LoadBundle(bundle, Host{})
	if err != nil {
		t.Fatal(err)
	}
	newBundle, _, err := act.CopyForEditing(dst)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(newBundle) == "My Action.dzbundle" {
		t.Error("copy overwrote the existing same-named bundle")
	}
}
