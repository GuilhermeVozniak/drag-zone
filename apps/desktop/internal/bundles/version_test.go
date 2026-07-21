package bundles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVersionNewer(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"1.0.0", "0.9.9", true},
		{"0.9.0", "1.0.0", false},
		{"1.0.0", "1.0.0", false},
		{"v1.2.3", "1.2.2", true},
		{"1.2", "1.1.9", true},
		{"1.0.0-beta.1", "1.0.0", false},
	}
	for _, c := range cases {
		if got := VersionNewer(c.a, c.b); got != c.want {
			t.Errorf("VersionNewer(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestLoadBundleMinVersionGate(t *testing.T) {
	dir := t.TempDir()
	bundle := writeBundle(t, dir, "Future Action", "42")
	// Ask for a version far in the future of any real release.
	script := "# Dropzone Action Info\n# Name: Future Action\n# UniqueID: 42\n# MinDropzoneVersion: 99.0\n\ndef dragged\nend\n"
	if err := os.WriteFile(filepath.Join(bundle, "action.rb"), []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}

	old := CurrentAppVersion
	defer func() { CurrentAppVersion = old }()
	CurrentAppVersion = "0.8.4"

	if _, err := LoadBundle(bundle, Host{}); err == nil {
		t.Fatal("bundle requiring a newer app version must not load")
	}

	CurrentAppVersion = "99.0"
	if _, err := LoadBundle(bundle, Host{}); err != nil {
		t.Fatalf("bundle at exact min version should load: %v", err)
	}
}
