package bundles

import (
	"os"
	"path/filepath"
	"testing"
)

func writeScript(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "action.py")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestParseMetaMissingHeaderOrName(t *testing.T) {
	if _, err := ParseMeta(writeScript(t, "print('hi')\n")); err == nil {
		t.Error("missing header should error")
	}
	noName := "# Dropzone Action Info\n# Description: x\n"
	if _, err := ParseMeta(writeScript(t, noName)); err == nil {
		t.Error("missing Name should error")
	}
}

func TestParseMetaDefaultsHandlesAndEvents(t *testing.T) {
	body := "# Dropzone Action Info\n# Name: Minimal\n\nprint('x')\n"
	meta, err := ParseMeta(writeScript(t, body))
	if err != nil {
		t.Fatal(err)
	}
	if len(meta.Events) != 2 || meta.Handles[0] != "Files" {
		t.Errorf("defaults not applied: %+v", meta)
	}
}

func TestOptionFieldsByNIB(t *testing.T) {
	if f := optionFields("Login"); len(f) != 2 || f[0].Key != "username" || f[1].Type != "password" {
		t.Errorf("Login fields = %+v", f)
	}
	if f := optionFields("APIKey"); len(f) != 1 || f[0].Key != "api_key" {
		t.Errorf("APIKey fields = %+v", f)
	}
	if f := optionFields("Unknown"); f != nil {
		t.Errorf("unknown NIB should yield nil, got %+v", f)
	}
}
