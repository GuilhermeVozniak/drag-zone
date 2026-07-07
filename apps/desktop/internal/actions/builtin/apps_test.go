package builtin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppName(t *testing.T) {
	if got := appName("/Applications/Safari.app"); got != "Safari" {
		t.Errorf("appName = %q", got)
	}
	if got := appName("Notes"); got != "Notes" {
		t.Errorf("appName without .app = %q", got)
	}
}

func TestMountPointFromPlist(t *testing.T) {
	plist := `<plist><dict><key>system-entities</key><array><dict>
	<key>mount-point</key>
	<string>/Volumes/My App</string>
	</dict></array></dict></plist>`
	if got := mountPointFromPlist(plist); got != "/Volumes/My App" {
		t.Errorf("mountPointFromPlist = %q", got)
	}
	if got := mountPointFromPlist("<plist></plist>"); got != "" {
		t.Errorf("no mount point should be empty, got %q", got)
	}
}

func TestFindApp(t *testing.T) {
	// Top-level .app
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "Cool.app"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := findApp(dir); err != nil || filepath.Base(got) != "Cool.app" {
		t.Errorf("findApp top-level = %q err %v", got, err)
	}
	// Nested one level deep
	nest := t.TempDir()
	sub := filepath.Join(nest, "folder")
	if err := os.MkdirAll(filepath.Join(sub, "Deep.app"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := findApp(nest); err != nil || filepath.Base(got) != "Deep.app" {
		t.Errorf("findApp nested = %q err %v", got, err)
	}
	// None
	if _, err := findApp(t.TempDir()); err == nil {
		t.Error("findApp with no .app should error")
	}
}
