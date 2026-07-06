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

type nullProgress struct{}

func (nullProgress) Detail(string) {}
func (nullProgress) Percent(int)   {}

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
		t.Errorf("archive contents unexpected: %+v", zr.File)
	}
}
