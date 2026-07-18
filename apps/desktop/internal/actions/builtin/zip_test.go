package builtin

import (
	"archive/zip"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

type nullProgress struct{}

func (nullProgress) Detail(string) {}
func (nullProgress) Percent(int)   {}

func TestZipFilesSpec(t *testing.T) {
	spec := ZipFiles{}.Spec()
	if spec.ID != "zip" {
		t.Errorf("ID = %q", spec.ID)
	}
	if len(spec.Accepts) != 1 || spec.Accepts[0] != model.ItemFiles {
		t.Errorf("Accepts = %+v", spec.Accepts)
	}
	if len(spec.Options) != 1 || spec.Options[0].Key != "dest" || spec.Options[0].Default != "same folder" {
		t.Errorf("Options = %+v", spec.Options)
	}
}

func TestZipFilesDropped(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "doc.txt")
	if err := os.WriteFile(src, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := ZipFiles{}.Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Label: "Zip"},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{src}},
		Progress: nullProgress{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Message != "Created doc.zip" {
		t.Errorf("message = %q", res.Message)
	}

	zr, err := zip.OpenReader(filepath.Join(dir, "doc.zip"))
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	if len(zr.File) != 1 || zr.File[0].Name != "doc.txt" {
		t.Fatalf("archive contents unexpected: %+v", zr.File)
	}

	// The archive must unzip back to identical bytes, not just the right name.
	rc, err := zr.File[0].Open()
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "content" {
		t.Errorf("unzipped content = %q, want %q", got, "content")
	}
}

func TestZipFilesDroppedRejectsEmptyPayload(t *testing.T) {
	if _, err := (ZipFiles{}).Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{},
		Payload:  model.Payload{Kind: model.ItemFiles},
		Progress: nullProgress{},
	}); err == nil {
		t.Error("empty payload should error")
	}
}

func TestZipFilesDroppedMultipleFilesNamedArchive(t *testing.T) {
	dir := t.TempDir()
	fileA := filepath.Join(dir, "a.txt")
	fileB := filepath.Join(dir, "b.txt")
	if err := os.WriteFile(fileA, []byte("aaa"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fileB, []byte("bbb"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := ZipFiles{}.Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{fileA, fileB}},
		Progress: nullProgress{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Message != "Created Archive.zip" {
		t.Errorf("message = %q, want %q", res.Message, "Created Archive.zip")
	}

	zr, err := zip.OpenReader(filepath.Join(dir, "Archive.zip"))
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	if len(zr.File) != 2 {
		t.Fatalf("archive entries = %d, want 2", len(zr.File))
	}
	want := map[string]string{"a.txt": "aaa", "b.txt": "bbb"}
	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		wantBytes, ok := want[f.Name]
		if !ok {
			t.Errorf("unexpected entry %q", f.Name)
			continue
		}
		if string(got) != wantBytes {
			t.Errorf("entry %q content = %q, want %q", f.Name, got, wantBytes)
		}
	}
}

func TestZipFilesDroppedDestDesktopHomeDirError(t *testing.T) {
	// An empty $HOME makes os.UserHomeDir fail, exercising the error branch
	// taken before any archive is written.
	t.Setenv("HOME", "")

	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "note.txt")
	if err := os.WriteFile(src, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := (ZipFiles{}).Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"dest": "Desktop"}},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{src}},
		Progress: nullProgress{},
	}); err == nil {
		t.Error("expected an error when $HOME cannot be resolved")
	}
}

func TestZipFilesDroppedNonexistentSourceErrors(t *testing.T) {
	// A source path that does not exist makes the WalkDir/writeZip step
	// fail, exercising the cleanup-and-propagate error branch.
	dir := t.TempDir()
	if _, err := (ZipFiles{}).Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{filepath.Join(dir, "missing.txt")}},
		Progress: nullProgress{},
	}); err == nil {
		t.Error("expected an error for a nonexistent source path")
	}
}

func TestZipFilesDroppedIncludesDirectoryContents(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "a.txt"), []byte("aaa"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "b.txt"), []byte("bbb"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := ZipFiles{}.Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{sub}},
		Progress: nullProgress{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Message != "Created sub.zip" {
		t.Errorf("message = %q, want %q", res.Message, "Created sub.zip")
	}

	zr, err := zip.OpenReader(filepath.Join(dir, "sub.zip"))
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	if len(zr.File) != 2 {
		t.Fatalf("archive entries = %d, want 2: %+v", len(zr.File), zr.File)
	}
	want := map[string]string{"sub/a.txt": "aaa", "sub/b.txt": "bbb"}
	for _, f := range zr.File {
		wantBytes, ok := want[f.Name]
		if !ok {
			t.Errorf("unexpected entry %q", f.Name)
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != wantBytes {
			t.Errorf("entry %q content = %q, want %q", f.Name, got, wantBytes)
		}
	}
}

func TestZipFilesDroppedDestDesktop(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "Desktop"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)

	srcDir := t.TempDir()
	src := filepath.Join(srcDir, "note.txt")
	if err := os.WriteFile(src, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := ZipFiles{}.Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"dest": "Desktop"}},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{src}},
		Progress: nullProgress{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Message != "Created note.zip" {
		t.Errorf("message = %q", res.Message)
	}
	if _, err := os.Stat(filepath.Join(home, "Desktop", "note.zip")); err != nil {
		t.Errorf("archive not written to Desktop: %v", err)
	}
}
