package bundles

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

const testScript = `# Dropzone Action Info
# Name: Test Action
# Description: Echoes items
# Handles: Files, Text
# Events: Dragged, Clicked
# OptionsNIB: APIKey
# SkipConfig: Yes
# UniqueID: 9999test
# Version: 1.0

def dragged():
    dz.begin("working on " + items[0])
    dz.percent(50)
    dz.save_value("remembered", "yes")
    dz.finish("did " + str(len(items)) + " items")
    dz.url("https://example.com/out")

def clicked():
    dz.fail("clicked failure")
`

func writeTestBundle(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bundle := filepath.Join(dir, "Test.dzbundle")
	if err := os.MkdirAll(bundle, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, "action.py"), []byte(testScript), 0o644); err != nil {
		t.Fatal(err)
	}
	return bundle
}

func TestParseMeta(t *testing.T) {
	bundle := writeTestBundle(t)
	meta, err := ParseMeta(filepath.Join(bundle, "action.py"))
	if err != nil {
		t.Fatal(err)
	}
	if meta.Name != "Test Action" || meta.UniqueID != "9999test" || !meta.SkipConfig {
		t.Errorf("unexpected meta: %+v", meta)
	}
	if len(meta.Handles) != 2 || meta.Handles[1] != "Text" {
		t.Errorf("unexpected handles: %v", meta.Handles)
	}
}

type fakeServices struct{ clipboard string }

func (f *fakeServices) CopyToClipboard(text string) error { f.clipboard = text; return nil }
func (f *fakeServices) CopyFilesToClipboard(paths []string) error {
	return nil
}
func (f *fakeServices) ReadClipboard() (string, error) { return f.clipboard, nil }
func (f *fakeServices) Notify(title, body string)      {}
func (f *fakeServices) PlaySound(name string)          {}
func (f *fakeServices) OpenURL(url string) error       { return nil }
func (f *fakeServices) OpenPath(path string) error     { return nil }
func (f *fakeServices) Reveal(path string) error       { return nil }
func (f *fakeServices) Trash(paths []string) error     { return nil }
func (f *fakeServices) AirDrop(paths []string) error   { return nil }

type fakeProgress struct {
	details []string
	pcts    []int
}

func (p *fakeProgress) Detail(s string) { p.details = append(p.details, s) }
func (p *fakeProgress) Percent(n int)   { p.pcts = append(p.pcts, n) }

func TestScriptActionDragged(t *testing.T) {
	if _, err := os.Stat("/opt/homebrew/bin/python3"); err != nil {
		if _, err := os.Stat("/usr/bin/python3"); err != nil {
			t.Skip("python3 not available")
		}
	}
	bundle := writeTestBundle(t)

	saved := map[string]string{}
	act, err := LoadBundle(bundle, Host{
		SaveValue: func(targetID, name, value string) { saved[name] = value },
	})
	if err != nil {
		t.Fatal(err)
	}

	spec := act.Spec()
	if spec.ID != "bundle:9999test" || len(spec.Options) != 1 || spec.Options[0].Key != "api_key" {
		t.Errorf("unexpected spec: %+v", spec)
	}

	svc := &fakeServices{}
	prog := &fakeProgress{}
	res, err := act.Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{ID: "t1", Label: "Test"},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{"/tmp/a.txt", "/tmp/b.txt"}},
		Progress: prog,
		Services: svc,
	})
	if err != nil {
		t.Fatalf("dragged: %v", err)
	}
	if res.Message != "did 2 items" {
		t.Errorf("message = %q", res.Message)
	}
	if res.URL != "https://example.com/out" || svc.clipboard != "https://example.com/out" {
		t.Errorf("url = %q clipboard = %q", res.URL, svc.clipboard)
	}
	if saved["remembered"] != "yes" {
		t.Errorf("save_value not applied: %v", saved)
	}
	if len(prog.pcts) == 0 || prog.pcts[0] != 50 {
		t.Errorf("percent not reported: %v", prog.pcts)
	}

	// clicked path reports the script's fail message
	if _, err := act.Clicked(context.Background(), actions.Invocation{
		Target:   model.Target{ID: "t1"},
		Payload:  model.Payload{Kind: model.ItemText, Text: "hello"},
		Progress: prog,
		Services: svc,
	}); err == nil || err.Error() != "clicked failure" {
		t.Errorf("clicked err = %v, want 'clicked failure'", err)
	}
}
