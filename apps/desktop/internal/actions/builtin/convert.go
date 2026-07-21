package builtin

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"dragzone/internal/actions"
	"dragzone/internal/fsutil"
	"dragzone/internal/model"
)

// ConvertImages converts dropped images to another format and optionally
// resizes them, using the system sips tool.
type ConvertImages struct{}

// convertFormats caches the base format list plus webp when the host's
// ImageIO can write it (sips gained webp encode only on recent macOS).
var convertFormats = sync.OnceValue(func() []string {
	formats := []string{"jpeg", "png", "tiff", "heic"}
	out, err := exec.Command("sips", "--formats").Output()
	if err != nil {
		return formats
	}
	for line := range strings.Lines(string(out)) {
		if strings.Contains(line, "webp") && strings.Contains(line, "Writable") {
			return append(formats, "webp")
		}
	}
	return formats
})

func (ConvertImages) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "convert-images",
		Name:        "Convert Images",
		Description: "Convert dropped images to another format, optionally resizing them.",
		Icon:        "file",
		Category:    "Images",
		Events:      []string{model.EventDragged},
		Accepts:     []model.ItemKind{model.ItemFiles},
		Multi:       true,
		Options: []model.OptionField{
			{Key: "format", Label: "Format", Type: "select", Choices: convertFormats(), Default: "jpeg"},
			{Key: "max_size", Label: "Max dimension", Type: "select", Choices: []string{"original", "2048", "1024", "800", "512"}, Default: "original"},
		},
	}
}

func (ConvertImages) Dropped(ctx context.Context, inv actions.Invocation) (actions.Result, error) {
	format := inv.Target.Option("format", "jpeg")
	maxSize := inv.Target.Option("max_size", "original")
	ext := map[string]string{"jpeg": ".jpg", "png": ".png", "tiff": ".tiff", "heic": ".heic", "webp": ".webp"}[format]

	for i, src := range inv.Payload.Paths {
		name := filepath.Base(src)
		inv.Progress.Detail(name)
		inv.Progress.Percent(i * 100 / len(inv.Payload.Paths))

		stem := strings.TrimSuffix(name, filepath.Ext(name))
		dst := fsutil.UniqueDest(filepath.Dir(src), stem+ext)
		args := []string{"-s", "format", format}
		if maxSize != "original" {
			args = append(args, "--resampleHeightWidthMax", maxSize)
		}
		args = append(args, src, "--out", dst)
		if out, err := exec.CommandContext(ctx, "sips", args...).CombinedOutput(); err != nil {
			return actions.Result{}, fmt.Errorf("converting %s: %s", name, strings.TrimSpace(string(out)))
		}
	}
	return actions.Result{Message: fmt.Sprintf("Converted %d image(s) to %s", len(inv.Payload.Paths), format)}, nil
}
