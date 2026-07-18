package builtin

import (
	"context"
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strconv"

	"dragzone/internal/actions"
	"dragzone/internal/fsutil"
	"dragzone/internal/model"
)

// MakeGIF combines dropped images into a single animated GIF using only the
// Go standard library (image/gif + image/draw) — no external dependency.
type MakeGIF struct{}

func (MakeGIF) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "make-gif",
		Name:        "Create GIF",
		Description: "Combine dropped images into an animated GIF.",
		Icon:        "film",
		Category:    "Image",
		Events:      []string{model.EventDragged},
		Accepts:     []model.ItemKind{model.ItemFiles},
		Options: []model.OptionField{
			{Key: "delay", Label: "Frame delay (ms)", Type: "text", Placeholder: "200", Default: "200"},
			{Key: "after", Label: "After creating", Type: "select", Choices: []string{"dropbar", "reveal"}, Default: "dropbar"},
		},
	}
}

func (MakeGIF) Dropped(_ context.Context, inv actions.Invocation) (actions.Result, error) {
	paths := inv.Payload.Paths
	if len(paths) == 0 {
		return actions.Result{}, fmt.Errorf("nothing to animate")
	}

	delayMS, err := strconv.Atoi(inv.Target.Option("delay", "200"))
	if err != nil || delayMS <= 0 {
		delayMS = 200
	}
	delay := delayMS / 10 // gif.GIF.Delay is in hundredths of a second.

	g := &gif.GIF{}
	for i, p := range paths {
		inv.Progress.Detail(filepath.Base(p))
		inv.Progress.Percent(i * 100 / len(paths))

		frame, err := decodeFrame(p)
		if err != nil {
			return actions.Result{}, fmt.Errorf("decoding %s: %w", filepath.Base(p), err)
		}
		g.Image = append(g.Image, frame)
		g.Delay = append(g.Delay, delay)
	}
	inv.Progress.Percent(100)

	dir := filepath.Dir(paths[0])
	if dir == "" || dir == "." {
		dir = expandHome("~/")
	}
	dst := fsutil.UniqueDest(dir, "animation.gif")

	out, err := os.Create(dst)
	if err != nil {
		return actions.Result{}, fmt.Errorf("creating %s: %w", filepath.Base(dst), err)
	}
	defer out.Close()
	if err := gif.EncodeAll(out, g); err != nil {
		return actions.Result{}, fmt.Errorf("encoding gif: %w", err)
	}

	switch inv.Target.Option("after", "dropbar") {
	case "reveal":
		if err := inv.Services.Reveal(dst); err != nil {
			return actions.Result{}, fmt.Errorf("revealing gif: %w", err)
		}
	default:
		if inv.AddDropBar != nil {
			inv.AddDropBar([]string{dst})
		}
	}

	return actions.Result{Message: fmt.Sprintf("Created GIF from %d image(s)", len(paths))}, nil
}

// decodeFrame decodes a source image and quantizes it to a paletted image
// suitable for a GIF frame, dithering with Floyd-Steinberg for quality.
func decodeFrame(path string) (*image.Paletted, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	src, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}

	bounds := src.Bounds()
	paletted := image.NewPaletted(bounds, palette.Plan9)
	draw.FloydSteinberg.Draw(paletted, bounds, src, bounds.Min)
	return paletted, nil
}
