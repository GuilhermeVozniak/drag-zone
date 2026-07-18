package builtin

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// ConvertImages wraps the external `sips` tool (darwin-only). These tests
// cover the pure-Go branches only: Spec metadata and the empty-payload path,
// which returns before any subprocess is invoked. The one exception is
// TestConvertImagesDroppedUnsupportedFormat, which does invoke the real
// `sips` binary (not the cgo/ImageIO bridge used by metadata.go) but only to
// exercise the error-wrapping branch on a guaranteed-fast failure — it never
// performs a real image conversion, and is skipped on non-darwin.

func TestConvertImagesSpec(t *testing.T) {
	spec := ConvertImages{}.Spec()
	if spec.ID != "convert-images" {
		t.Errorf("ID = %q", spec.ID)
	}
	if !spec.Multi {
		t.Error("Multi = false, want true")
	}
	if len(spec.Accepts) != 1 || spec.Accepts[0] != model.ItemFiles {
		t.Errorf("Accepts = %+v", spec.Accepts)
	}
	if len(spec.Options) != 2 || spec.Options[0].Key != "format" || spec.Options[0].Default != "jpeg" {
		t.Errorf("Options = %+v", spec.Options)
	}
	if spec.Options[1].Key != "max_size" || spec.Options[1].Default != "original" {
		t.Errorf("Options[1] = %+v", spec.Options[1])
	}
}

func TestConvertImagesDroppedEmptyPayload(t *testing.T) {
	// No paths means the conversion loop never runs, so no subprocess is
	// spawned; this exercises option-reading and the result message only.
	res, err := ConvertImages{}.Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{},
		Payload:  model.Payload{Kind: model.ItemFiles},
		Progress: nullProgress{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Message != "Converted 0 image(s) to jpeg" {
		t.Errorf("message = %q", res.Message)
	}
}

func TestConvertImagesDroppedEmptyPayloadCustomFormat(t *testing.T) {
	res, err := ConvertImages{}.Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"format": "png"}},
		Payload:  model.Payload{Kind: model.ItemFiles},
		Progress: nullProgress{},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Message != "Converted 0 image(s) to png" {
		t.Errorf("message = %q", res.Message)
	}
}

func TestConvertImagesDroppedUnsupportedFormat(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("requires the macOS sips binary")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "not-an-image.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ConvertImages{}.Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{src}},
		Progress: nullProgress{},
	})
	if err == nil {
		t.Fatal("expected an error converting a non-image file")
	}
	if !strings.Contains(err.Error(), "converting not-an-image.txt") {
		t.Errorf("error = %q, want it to mention the file name", err.Error())
	}
}

func TestConvertImagesDroppedUnsupportedFormatWithResize(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("requires the macOS sips binary")
	}
	// Exercises the --resampleHeightWidthMax argument-building branch (taken
	// whenever max_size isn't "original"); the sips call still fails fast
	// since the source is not an image, so no real conversion happens.
	dir := t.TempDir()
	src := filepath.Join(dir, "not-an-image.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ConvertImages{}.Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"max_size": "1024"}},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{src}},
		Progress: nullProgress{},
	})
	if err == nil {
		t.Fatal("expected an error converting a non-image file")
	}
}
