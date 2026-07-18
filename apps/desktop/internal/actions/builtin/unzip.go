package builtin

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"dragzone/internal/actions"
	"dragzone/internal/fsutil"
	"dragzone/internal/model"
)

// Unzip extracts dropped .zip archives into a new folder next to each zip,
// keeping the original archive intact.
type Unzip struct{}

func (Unzip) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "unzip",
		Name:        "Unzip Files",
		Description: "Unzip dropped .zip archives (keeps the original zip).",
		Icon:        "package-open",
		Category:    "File Management",
		Events:      []string{model.EventDragged},
		Accepts:     []model.ItemKind{model.ItemFiles},
	}
}

func (Unzip) Dropped(_ context.Context, inv actions.Invocation) (actions.Result, error) {
	paths := inv.Payload.Paths
	if len(paths) == 0 {
		return actions.Result{}, fmt.Errorf("unzip: nothing to extract")
	}

	var folders []string
	var extracted int
	for _, p := range paths {
		if !strings.EqualFold(filepath.Ext(p), ".zip") {
			continue
		}
		folder, err := extractZip(p)
		if err != nil {
			return actions.Result{}, fmt.Errorf("unzip %s: %w", filepath.Base(p), err)
		}
		folders = append(folders, folder)
		extracted++
	}
	if extracted == 0 {
		return actions.Result{}, fmt.Errorf("unzip: no .zip archives in the dropped items")
	}

	if inv.AddDropBar != nil {
		inv.AddDropBar(folders)
	}

	return actions.Result{Message: fmt.Sprintf("Unzipped %d archive(s)", extracted)}, nil
}

// extractZip extracts the archive at src into a new, uniquely named folder
// next to src (named after the zip's stem) and returns that folder's path.
func extractZip(src string) (string, error) {
	zr, err := zip.OpenReader(src)
	if err != nil {
		return "", err
	}
	defer zr.Close()

	dir := filepath.Dir(src)
	stem := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	dest := fsutil.UniqueDest(dir, stem)
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return "", err
	}

	for _, f := range zr.File {
		if err := extractZipEntry(dest, f); err != nil {
			return "", err
		}
	}
	return dest, nil
}

// extractZipEntry writes a single zip entry under dest, rejecting entries
// whose cleaned path would escape dest (Zip Slip).
func extractZipEntry(dest string, f *zip.File) error {
	target := filepath.Join(dest, f.Name)
	if !isWithinDir(dest, target) {
		return fmt.Errorf("illegal file path in archive: %s", f.Name)
	}

	if f.FileInfo().IsDir() {
		return os.MkdirAll(target, 0o755)
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}

	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, f.Mode().Perm()|0o600)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}

// isWithinDir reports whether target is dir itself or a descendant of it,
// after cleaning both paths.
func isWithinDir(dir, target string) bool {
	dir = filepath.Clean(dir)
	target = filepath.Clean(target)
	rel, err := filepath.Rel(dir, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
