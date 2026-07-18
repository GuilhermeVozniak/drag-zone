package builtin

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// annotateCmd is a seam over `open` so tests stub the CleanShot X invocation
// without launching an app. `open` exits non-zero when no application handles
// the cleanshot:// scheme, which is how we detect CleanShot X isn't installed.
var annotateCmd = func(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "open", args...)
}

// annotateImageExts are the image types CleanShot X's annotator accepts.
var annotateImageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".heic": true,
	".heif": true, ".tiff": true, ".tif": true, ".bmp": true, ".webp": true,
}

// cleanShotRequiredMessage mirrors Dropzone 4: the action delegates annotation
// to the third-party CleanShot X app, so it is unavailable unless CleanShot X
// is installed.
const cleanShotRequiredMessage = "CleanShot X is required for this action. Install it from cleanshot.com, then try again."

// Annotate opens dropped images in CleanShot X's annotation editor via its
// cleanshot://open-annotate URL scheme (CleanShot 3.8.1+), matching Dropzone
// 4's "Annotate with CleanShot X" action — both delegate annotation to the
// separate CleanShot X app.
type Annotate struct{}

func (Annotate) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "annotate",
		Name:        "Annotate with CleanShot X",
		Description: "Open dropped images in CleanShot X's annotation editor.",
		Icon:        "pencil",
		Category:    "Capture",
		Events:      []string{model.EventDragged},
		Accepts:     []model.ItemKind{model.ItemFiles},
		Multi:       true,
	}
}

func (Annotate) Dropped(ctx context.Context, inv actions.Invocation) (actions.Result, error) {
	var images []string
	for _, p := range inv.Payload.Paths {
		if annotateImageExts[strings.ToLower(filepath.Ext(p))] {
			images = append(images, p)
		}
	}
	if len(images) == 0 {
		return actions.Result{Message: "No image files to annotate (PNG, JPEG, GIF, HEIC, …)."}, nil
	}

	for _, img := range images {
		abs, err := filepath.Abs(img)
		if err != nil {
			abs = img
		}
		if err := annotateCmd(ctx, cleanShotAnnotateURL(abs)).Run(); err != nil {
			// `open` exits non-zero when nothing handles the cleanshot:// scheme,
			// i.e. CleanShot X isn't installed — the same dependency Dropzone 4
			// has. Surface a clear message rather than a raw exec error.
			return actions.Result{Message: cleanShotRequiredMessage}, nil
		}
	}

	if len(images) == 1 {
		return actions.Result{Message: "Opened in CleanShot X for annotation"}, nil
	}
	return actions.Result{Message: fmt.Sprintf("Opened %d images in CleanShot X for annotation", len(images))}, nil
}

// cleanShotAnnotateURL builds a cleanshot://open-annotate URL for absPath,
// keeping path separators raw and encoding spaces/specials as %XX — the format
// CleanShot X documents (e.g. filepath=/Users/j/Desktop/my%20screenshot.png).
func cleanShotAnnotateURL(absPath string) string {
	enc := url.QueryEscape(absPath)
	enc = strings.ReplaceAll(enc, "%2F", "/")
	enc = strings.ReplaceAll(enc, "+", "%20")
	return "cleanshot://open-annotate?filepath=" + enc
}
