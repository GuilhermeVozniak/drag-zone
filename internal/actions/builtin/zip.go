package builtin

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"dragzone/internal/actions"
	"dragzone/internal/fsutil"
	"dragzone/internal/model"
)

// ZipFiles compresses dropped files into a .zip archive next to the source.
type ZipFiles struct{}

func (ZipFiles) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "zip",
		Name:        "Zip Files",
		Description: "Compress dropped files into a .zip archive.",
		Icon:        "archive",
		Category:    "Utilities",
		Events:      []string{model.EventDragged},
		Accepts:     []model.ItemKind{model.ItemFiles},
		Options: []model.OptionField{
			{Key: "dest", Label: "Save archive in", Type: "select", Choices: []string{"same folder", "Desktop"}, Default: "same folder"},
		},
	}
}

func (ZipFiles) Dropped(_ context.Context, inv actions.Invocation) (actions.Result, error) {
	paths := inv.Payload.Paths
	if len(paths) == 0 {
		return actions.Result{}, fmt.Errorf("nothing to compress")
	}

	dir := filepath.Dir(paths[0])
	if inv.Target.Option("dest", "same folder") == "Desktop" {
		home, err := os.UserHomeDir()
		if err != nil {
			return actions.Result{}, err
		}
		dir = filepath.Join(home, "Desktop")
	}

	name := strings.TrimSuffix(filepath.Base(paths[0]), filepath.Ext(paths[0]))
	if len(paths) > 1 {
		name = "Archive"
	}
	dst := fsutil.UniqueDest(dir, name+".zip")

	total := fsutil.TotalSize(paths)
	var done int64
	if err := writeZip(dst, paths, func(n int64) {
		if total > 0 {
			done += n
			inv.Progress.Percent(int(done * 100 / total))
		}
	}); err != nil {
		os.Remove(dst)
		return actions.Result{}, err
	}
	return actions.Result{Message: "Created " + filepath.Base(dst)}, nil
}

func writeZip(dst string, paths []string, onBytes func(int64)) (err error) {
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	// The close error matters: it is the final flush of the archive.
	defer func() {
		if cerr := out.Close(); err == nil {
			err = cerr
		}
	}()
	zw := zip.NewWriter(out)

	for _, root := range paths {
		base := filepath.Dir(root)
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, err := filepath.Rel(base, path)
			if err != nil {
				return err
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			hdr, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}
			hdr.Name = filepath.ToSlash(rel)
			hdr.Method = zip.Deflate
			w, err := zw.CreateHeader(hdr)
			if err != nil {
				return err
			}
			in, err := os.Open(path)
			if err != nil {
				return err
			}
			defer in.Close()
			_, err = io.Copy(io.MultiWriter(w, byteCounter(onBytes)), in)
			return err
		})
		if err != nil {
			return err
		}
	}
	return zw.Close()
}

type byteCounter func(int64)

func (f byteCounter) Write(p []byte) (int, error) {
	f(int64(len(p)))
	return len(p), nil
}
