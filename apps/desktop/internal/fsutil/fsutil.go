// Package fsutil implements file copy/move primitives with byte-level
// progress reporting for action implementations.
package fsutil

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// TotalSize returns the combined size in bytes of the given files/directories.
func TotalSize(paths []string) int64 {
	var total int64
	for _, p := range paths {
		filepath.WalkDir(p, func(_ string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if info, err := d.Info(); err == nil {
				total += info.Size()
			}
			return nil
		})
	}
	return total
}

// UniqueDest returns a non-colliding destination path for name inside dir,
// appending " 2", " 3", … before the extension when needed.
func UniqueDest(dir, name string) string {
	dst := filepath.Join(dir, name)
	if _, err := os.Lstat(dst); err != nil {
		return dst
	}
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	for i := 2; ; i++ {
		dst = filepath.Join(dir, fmt.Sprintf("%s %d%s", stem, i, ext))
		if _, err := os.Lstat(dst); err != nil {
			return dst
		}
	}
}

// CopyPath copies src into dstDir under a non-colliding name (see UniqueDest),
// reporting copied bytes via onBytes (may be nil). It returns the destination.
func CopyPath(src, dstDir string, onBytes func(int64)) (string, error) {
	return CopyPathAs(src, UniqueDest(dstDir, filepath.Base(src)), onBytes)
}

// CopyPathAs copies the file or directory src to the exact path dst. Callers
// that must not overwrite should use CopyPath; "Replace" conflict resolution
// uses this to write over an already-removed destination.
func CopyPathAs(src, dst string, onBytes func(int64)) (string, error) {
	info, err := os.Lstat(src)
	if err != nil {
		return "", err
	}
	if err := copyAny(src, dst, info, onBytes); err != nil {
		return "", err
	}
	return dst, nil
}

// MovePath moves src into dstDir under a non-colliding name and returns the
// destination path.
func MovePath(src, dstDir string, onBytes func(int64)) (string, error) {
	return MovePathAs(src, UniqueDest(dstDir, filepath.Base(src)), onBytes)
}

// MovePathAs moves src to the exact path dst, preferring rename and falling
// back to copy+delete across volumes. dst must not already exist (rename will
// not replace a non-empty directory); "Replace" callers remove it first.
func MovePathAs(src, dst string, onBytes func(int64)) (string, error) {
	if err := os.Rename(src, dst); err == nil {
		if onBytes != nil {
			onBytes(TotalSize([]string{dst}))
		}
		return dst, nil
	}
	info, err := os.Lstat(src)
	if err != nil {
		return "", err
	}
	if err := copyAny(src, dst, info, onBytes); err != nil {
		return "", err
	}
	return dst, os.RemoveAll(src)
}

func copyAny(src, dst string, info os.FileInfo, onBytes func(int64)) error {
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return os.Symlink(target, dst)
	case info.IsDir():
		return copyDir(src, dst, onBytes)
	default:
		return copyFile(src, dst, info, onBytes)
	}
}

func copyDir(src, dst string, onBytes func(int64)) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			return err
		}
		if err := copyAny(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name()), info, onBytes); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string, info os.FileInfo, onBytes func(int64)) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	w := io.Writer(out)
	if onBytes != nil {
		w = &countingWriter{w: out, onBytes: onBytes}
	}
	if _, err := io.Copy(w, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

type countingWriter struct {
	w       io.Writer
	onBytes func(int64)
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	if n > 0 {
		c.onBytes(int64(n))
	}
	return n, err
}
