package bundles

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateTemplateRubyAndPython(t *testing.T) {
	dir := t.TempDir()
	rb, err := CreateTemplate(dir, "My Action", "ruby")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(rb) != "My Action.dzbundle" {
		t.Errorf("bundle path = %q", rb)
	}
	script, err := os.ReadFile(filepath.Join(rb, "action.rb"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(script), "# Name: My Action") || !strings.Contains(string(script), "def dragged") {
		t.Errorf("ruby template missing header/handler")
	}

	py, err := CreateTemplate(dir, "Py Action", "python")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(py, "action.py")); err != nil {
		t.Errorf("python template not written: %v", err)
	}
}

func TestCreateTemplateRejectsBlankAndDuplicate(t *testing.T) {
	dir := t.TempDir()
	if _, err := CreateTemplate(dir, "   ", "ruby"); err == nil {
		t.Error("blank name should error")
	}
	if _, err := CreateTemplate(dir, "Dup", "ruby"); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateTemplate(dir, "Dup", "ruby"); err == nil {
		t.Error("duplicate name should error")
	}
}
