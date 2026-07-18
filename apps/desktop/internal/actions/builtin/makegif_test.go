package builtin

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// writeTestPNG writes a tiny solid-color PNG at path.
func writeTestPNG(t *testing.T, path string, c color.Color) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestMakeGIFSpec(t *testing.T) {
	spec := MakeGIF{}.Spec()
	if spec.ID != "make-gif" {
		t.Errorf("ID = %q", spec.ID)
	}
	if spec.Icon != "film" {
		t.Errorf("Icon = %q", spec.Icon)
	}
	if len(spec.Events) != 1 || spec.Events[0] != model.EventDragged {
		t.Errorf("Events = %v", spec.Events)
	}
	if len(spec.Accepts) != 1 || spec.Accepts[0] != model.ItemFiles {
		t.Errorf("Accepts = %v", spec.Accepts)
	}
	if spec.Multi {
		t.Error("expected Multi = false")
	}
	byKey := map[string]model.OptionField{}
	for _, o := range spec.Options {
		byKey[o.Key] = o
	}
	if d, ok := byKey["delay"]; !ok || d.Default != "200" {
		t.Errorf("delay option = %+v", d)
	}
	if a, ok := byKey["after"]; !ok || a.Default != "dropbar" {
		t.Errorf("after option = %+v", a)
	}
}

func TestMakeGIFDroppedProducesAnimatedGIF(t *testing.T) {
	dir := t.TempDir()
	p1 := filepath.Join(dir, "one.png")
	p2 := filepath.Join(dir, "two.png")
	writeTestPNG(t, p1, color.RGBA{R: 255, A: 255})
	writeTestPNG(t, p2, color.RGBA{B: 255, A: 255})

	drop := &recDropBar{}
	inv := actions.Invocation{
		Target:     model.Target{Options: map[string]string{"delay": "300"}},
		Payload:    model.Payload{Kind: model.ItemFiles, Paths: []string{p1, p2}},
		Progress:   nullProgress{},
		Services:   &recServices{},
		AddDropBar: drop.add,
	}

	res, err := (MakeGIF{}).Dropped(context.Background(), inv)
	if err != nil {
		t.Fatalf("Dropped: %v", err)
	}
	if res.Message != "Created GIF from 2 image(s)" {
		t.Errorf("Message = %q", res.Message)
	}
	if len(drop.calls) != 1 || len(drop.calls[0]) != 1 {
		t.Fatalf("AddDropBar calls = %v", drop.calls)
	}
	dst := drop.calls[0][0]
	if filepath.Dir(dst) != dir {
		t.Errorf("dst dir = %q, want %q", filepath.Dir(dst), dir)
	}

	f, err := os.Open(dst)
	if err != nil {
		t.Fatalf("opening output gif: %v", err)
	}
	defer f.Close()
	g, err := gif.DecodeAll(f)
	if err != nil {
		t.Fatalf("DecodeAll: %v", err)
	}
	if len(g.Image) != 2 {
		t.Errorf("frame count = %d, want 2", len(g.Image))
	}
	for i, d := range g.Delay {
		if d != 30 { // 300ms / 10
			t.Errorf("Delay[%d] = %d, want 30", i, d)
		}
	}
}

func TestMakeGIFDroppedRevealAfter(t *testing.T) {
	dir := t.TempDir()
	p1 := filepath.Join(dir, "one.png")
	writeTestPNG(t, p1, color.RGBA{G: 255, A: 255})

	svc := &recServices{}
	inv := actions.Invocation{
		Target:   model.Target{Options: map[string]string{"after": "reveal"}},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{p1}},
		Progress: nullProgress{},
		Services: svc,
	}

	if _, err := (MakeGIF{}).Dropped(context.Background(), inv); err != nil {
		t.Fatalf("Dropped: %v", err)
	}
	if len(svc.Opened) != 1 {
		t.Errorf("expected Reveal to be called once, got %v", svc.Opened)
	}
}

func TestMakeGIFEmptyPayloadErrors(t *testing.T) {
	inv := actions.Invocation{
		Payload:  model.Payload{Kind: model.ItemFiles},
		Progress: nullProgress{},
		Services: &recServices{},
	}
	if _, err := (MakeGIF{}).Dropped(context.Background(), inv); err == nil {
		t.Error("empty payload should error")
	}
}

func TestMakeGIFNonImageFileErrors(t *testing.T) {
	dir := t.TempDir()
	garbage := filepath.Join(dir, "not-an-image.png")
	if err := os.WriteFile(garbage, []byte("this is not image data"), 0o644); err != nil {
		t.Fatal(err)
	}
	inv := actions.Invocation{
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{garbage}},
		Progress: nullProgress{},
		Services: &recServices{},
	}
	if _, err := (MakeGIF{}).Dropped(context.Background(), inv); err == nil {
		t.Error("non-decodable file should error")
	}
}
