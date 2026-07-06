// Package addons lists and fetches add-on actions from the official
// aptonic/dropzone4-actions repository, powering the Add-on Actions tab.
package addons

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	contentsURL = "https://api.github.com/repos/aptonic/dropzone4-actions/contents/"
	archiveURL  = "https://codeload.github.com/aptonic/dropzone4-actions/zip/refs/heads/master"
	cacheMaxAge = 24 * time.Hour
)

var client = &http.Client{Timeout: 60 * time.Second}

// List returns the .dzbundle names available in the official repository.
func List(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, contentsURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "DragZone")
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching add-on list: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching add-on list: %s", resp.Status)
	}
	var entries []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, fmt.Errorf("parsing add-on list: %w", err)
	}
	var names []string
	for _, e := range entries {
		if e.Type == "dir" && strings.HasSuffix(e.Name, ".dzbundle") {
			names = append(names, strings.TrimSuffix(e.Name, ".dzbundle"))
		}
	}
	return names, nil
}

// FetchBundle downloads (with a cached archive) the named bundle and returns
// the extracted .dzbundle path plus a cleanup function.
func FetchBundle(ctx context.Context, cacheDir, name string) (string, func(), error) {
	zipPath := filepath.Join(cacheDir, "dropzone4-actions.zip")
	if info, err := os.Stat(zipPath); err != nil || time.Since(info.ModTime()) > cacheMaxAge {
		if err := download(ctx, zipPath); err != nil {
			return "", nil, err
		}
	}

	tmp, err := os.MkdirTemp("", "dragzone-addon-*")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { os.RemoveAll(tmp) }
	if out, err := exec.CommandContext(ctx, "ditto", "-x", "-k", zipPath, tmp).CombinedOutput(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("extracting add-on archive: %s", strings.TrimSpace(string(out)))
	}
	bundle := filepath.Join(tmp, "dropzone4-actions-master", name+".dzbundle")
	if _, err := os.Stat(bundle); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("add-on %q not found in the repository archive", name)
	}
	return bundle, cleanup, nil
}

func download(ctx context.Context, dst string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, archiveURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "DragZone")
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("downloading add-on archive: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("downloading add-on archive: %s", resp.Status)
	}
	tmp, err := os.CreateTemp(filepath.Dir(dst), "addons-*.zip")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), dst)
}
