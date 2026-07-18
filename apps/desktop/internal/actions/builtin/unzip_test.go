package builtin

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"testing"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

func TestUnzipSpec(t *testing.T) {
	spec := Unzip{}.Spec()
	if spec.ID != "unzip" {
		t.Errorf("ID = %q", spec.ID)
	}
	if len(spec.Accepts) != 1 || spec.Accepts[0] != model.ItemFiles {
		t.Errorf("Accepts = %+v", spec.Accepts)
	}
	if len(spec.Events) != 1 || spec.Events[0] != model.EventDragged {
		t.Errorf("Events = %+v", spec.Events)
	}
}

// buildZip creates a zip archive at dst containing the given name->content
// entries (a "/" suffix on a name creates an empty directory entry).
func buildZip(t *testing.T, dst string, entries map[string]string) {
	t.Helper()
	out, err := os.Create(dst)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestUnzipDropped(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "archive.zip")
	buildZip(t, zipPath, map[string]string{
		"a.txt":        "aaa",
		"nested/b.txt": "bbb",
	})

	rec := &recServices{}
	var addedToDropBar [][]string
	res, err := Unzip{}.Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{zipPath}},
		Progress: nullProgress{},
		Services: rec,
		AddDropBar: func(paths []string) {
			addedToDropBar = append(addedToDropBar, paths)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Message != "Unzipped 1 archive(s)" {
		t.Errorf("message = %q", res.Message)
	}

	extractedDir := filepath.Join(dir, "archive")
	gotA, err := os.ReadFile(filepath.Join(extractedDir, "a.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(gotA) != "aaa" {
		t.Errorf("a.txt content = %q", gotA)
	}
	gotB, err := os.ReadFile(filepath.Join(extractedDir, "nested", "b.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(gotB) != "bbb" {
		t.Errorf("nested/b.txt content = %q", gotB)
	}

	// The original zip must still exist.
	if _, err := os.Stat(zipPath); err != nil {
		t.Errorf("original zip removed: %v", err)
	}

	if len(addedToDropBar) != 1 || len(addedToDropBar[0]) != 1 || addedToDropBar[0][0] != extractedDir {
		t.Errorf("AddDropBar got %+v, want [[%s]]", addedToDropBar, extractedDir)
	}
}

func TestUnzipDroppedRejectsZipSlip(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "evil.zip")
	// evil.zip extracts into a dest folder named "evil"; an entry of "../evil"
	// would collapse right back onto that same dest path, defeating the
	// escape check by accident. Use a name that lands on a distinct sibling
	// path instead, so an unblocked escape is actually observable.
	buildZip(t, zipPath, map[string]string{
		"../outside.txt": "pwned",
	})

	if _, err := (Unzip{}).Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{zipPath}},
		Progress: nullProgress{},
	}); err == nil {
		t.Error("expected an error for a zip-slip entry")
	}

	// The escape target must not have been written.
	if _, err := os.Stat(filepath.Join(dir, "outside.txt")); err == nil {
		t.Error("zip-slip entry was written outside the destination")
	}
}

func TestUnzipDroppedRejectsEmptyPayload(t *testing.T) {
	if _, err := (Unzip{}).Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{},
		Payload:  model.Payload{Kind: model.ItemFiles},
		Progress: nullProgress{},
	}); err == nil {
		t.Error("empty payload should error")
	}
}

func TestUnzipDroppedRejectsNonZipInput(t *testing.T) {
	dir := t.TempDir()
	txt := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(txt, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := (Unzip{}).Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{txt}},
		Progress: nullProgress{},
	}); err == nil {
		t.Error("non-zip input should error")
	}
}
