package builtin

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dragzone/internal/actions"
	"dragzone/internal/fsutil"
	"dragzone/internal/model"
)

// tinifyAPIURL is the compression endpoint; a package var so tests can point
// it at an httptest server.
var tinifyAPIURL = "https://api.tinify.com/shrink"

// Tinify compresses dropped images via the Tinify (TinyPNG) API and writes
// the result to a new file next to the original, leaving the original
// untouched.
type Tinify struct{}

func (Tinify) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "tinify",
		Name:        "Tinify (TinyPNG)",
		Description: "Compress dropped images with Tinify (TinyPNG) and save alongside the original.",
		Icon:        "minimize-2",
		Category:    "Image",
		Events:      []string{model.EventDragged},
		Accepts:     []model.ItemKind{model.ItemFiles},
		Multi:       true,
		Options: []model.OptionField{
			{Key: "api_key", Label: "Tinify API Key", Type: "text", Required: true},
		},
	}
}

func (Tinify) Dropped(ctx context.Context, inv actions.Invocation) (actions.Result, error) {
	apiKey := inv.Target.Option("api_key", "")
	if apiKey == "" {
		return actions.Result{}, fmt.Errorf("no Tinify API key configured")
	}
	paths := inv.Payload.Paths
	if len(paths) == 0 {
		return actions.Result{}, fmt.Errorf("nothing to compress")
	}
	for _, p := range paths {
		if !isImageFile(p) {
			return actions.Result{}, fmt.Errorf("%s is not a supported image (jpg, jpeg, png, gif, webp, heic, bmp, tiff)", filepath.Base(p))
		}
	}

	client := &http.Client{Timeout: 120 * time.Second}
	var originalTotal, compressedTotal int64
	var dests []string
	for i, p := range paths {
		inv.Progress.Detail(filepath.Base(p))
		inv.Progress.Percent(i * 100 / len(paths))
		dst, origSize, compSize, err := compressWithTinify(ctx, client, apiKey, p)
		if err != nil {
			return actions.Result{}, fmt.Errorf("compressing %s: %w", filepath.Base(p), err)
		}
		originalTotal += origSize
		compressedTotal += compSize
		dests = append(dests, dst)
	}
	inv.Progress.Percent(100)

	saved := originalTotal - compressedTotal
	return actions.Result{
		Message: fmt.Sprintf("Compressed %d image(s) — saved %s (%s → %s)",
			len(dests), humanBytes(saved), humanBytes(originalTotal), humanBytes(compressedTotal)),
	}, nil
}

// compressWithTinify uploads one image to the Tinify shrink endpoint, follows
// the returned Location to download the compressed bytes, and writes them to
// a new "<name>-tiny<ext>" file next to the original (never overwriting it).
// It returns the new file's path along with the original and compressed
// sizes.
func compressWithTinify(ctx context.Context, client *http.Client, apiKey, path string) (dst string, origSize, compSize int64, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", 0, 0, err
	}
	origSize = int64(len(data))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tinifyAPIURL, bytes.NewReader(data))
	if err != nil {
		return "", 0, 0, err
	}
	req.SetBasicAuth("api", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		return "", 0, 0, err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusCreated {
		return "", 0, 0, fmt.Errorf("tinify returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	location := resp.Header.Get("Location")
	if location == "" {
		return "", 0, 0, fmt.Errorf("tinify response contains no Location header")
	}

	getReq, err := http.NewRequestWithContext(ctx, http.MethodGet, location, nil)
	if err != nil {
		return "", 0, 0, err
	}
	getReq.SetBasicAuth("api", apiKey)

	getResp, err := client.Do(getReq)
	if err != nil {
		return "", 0, 0, err
	}
	defer getResp.Body.Close()

	compressed, err := io.ReadAll(io.LimitReader(getResp.Body, 50<<20))
	if err != nil {
		return "", 0, 0, fmt.Errorf("downloading compressed image: %w", err)
	}
	if getResp.StatusCode != http.StatusOK {
		return "", 0, 0, fmt.Errorf("tinify download returned %s: %s", getResp.Status, strings.TrimSpace(string(compressed)))
	}

	dir := filepath.Dir(path)
	name := filepath.Base(path)
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	dst = fsutil.UniqueDest(dir, stem+"-tiny"+ext)
	if err := os.WriteFile(dst, compressed, 0o644); err != nil {
		return "", 0, 0, fmt.Errorf("writing compressed image: %w", err)
	}

	return dst, origSize, int64(len(compressed)), nil
}

// humanBytes formats a byte count as a short human-readable size (e.g.
// "12.3 KiB").
func humanBytes(n int64) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(1024), 0
	for m := n / 1024; m >= 1024; m /= 1024 {
		div *= 1024
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
