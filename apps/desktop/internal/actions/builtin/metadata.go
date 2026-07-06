package builtin

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"dragzone/internal/actions"
	"dragzone/internal/fsutil"
	"dragzone/internal/model"
	"dragzone/internal/platform"
)

// RemoveImageMetadata rewrites dropped images without EXIF/GPS/IPTC metadata
// via the native ImageIO bridge, keeping the originals untouched.
type RemoveImageMetadata struct{}

func (RemoveImageMetadata) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "remove-metadata",
		Name:        "Remove Image Metadata",
		Description: "Save copies of dropped images with EXIF, GPS, and other hidden metadata removed.",
		Icon:        "file",
		Category:    "Images",
		Events:      []string{model.EventDragged},
		Accepts:     []model.ItemKind{model.ItemFiles},
	}
}

func (RemoveImageMetadata) Dropped(_ context.Context, inv actions.Invocation) (actions.Result, error) {
	for i, src := range inv.Payload.Paths {
		name := filepath.Base(src)
		inv.Progress.Detail(name)
		inv.Progress.Percent(i * 100 / len(inv.Payload.Paths))

		ext := filepath.Ext(name)
		stem := strings.TrimSuffix(name, ext)
		dst := fsutil.UniqueDest(filepath.Dir(src), stem+" clean"+ext)
		if err := platform.StripImageMetadata(src, dst); err != nil {
			return actions.Result{}, fmt.Errorf("%s: %w", name, err)
		}
	}
	return actions.Result{Message: fmt.Sprintf("Cleaned %d image(s)", len(inv.Payload.Paths))}, nil
}
