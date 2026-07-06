package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"dragzone/internal/config"
	"dragzone/internal/platform"
)

// Settings, native dialogs, window control, and icon bindings.

func (a *App) GetSettings() config.Settings {
	return a.settings.Get()
}

// SetSettings persists new settings and applies the side-effectful ones
// (global hotkeys, login item, drag overlay, window scale) immediately.
func (a *App) SetSettings(s config.Settings) error {
	prev := a.settings.Get()
	if err := a.settings.Set(s); err != nil {
		return err
	}
	a.applySettings(s)
	if s.LaunchAtLogin != prev.LaunchAtLogin {
		if err := platform.SetLoginItem(s.LaunchAtLogin); err != nil {
			return err
		}
	}
	return nil
}

// applySettings pushes setting-dependent state into the native layer.
func (a *App) applySettings(s config.Settings) {
	platform.SetHotkeyF(parseFKey(s.GlobalShortcut), platform.HotkeySlotGrid)
	platform.SetHotkeyF(parseFKey(s.PopOutShortcut), platform.HotkeySlotPopOut)
	platform.SetDragOverlayEnabled(s.DragOverlay)
	if a.ctx != nil {
		scale := s.Scale()
		runtime.WindowSetSize(a.ctx, int(float64(windowWidth)*scale), int(float64(windowHeight)*scale))
	}
}

// parseFKey converts "F3" to 3; unknown values disable the hotkey.
func parseFKey(shortcut string) int {
	s := strings.ToUpper(strings.TrimSpace(shortcut))
	if !strings.HasPrefix(s, "F") {
		return 0
	}
	n, err := strconv.Atoi(s[1:])
	if err != nil || n < 1 || n > 12 {
		return 0
	}
	return n
}

// FileIcon returns an image for a path as a base64 PNG (cached for the
// app's lifetime): a QuickLook content preview for regular files that have
// one (images, PDFs, videos), otherwise the Finder icon.
func (a *App) FileIcon(path string) string {
	a.iconMu.Lock()
	defer a.iconMu.Unlock()
	if icon, ok := a.iconCache[path]; ok {
		return icon
	}
	var icon string
	if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
		icon, _ = platform.FileThumbnailPNGBase64(path, 64)
	}
	if icon == "" {
		if fallback, err := platform.FileIconPNGBase64(path, 64); err == nil {
			icon = fallback
		}
	}
	a.iconCache[path] = icon
	return icon
}

func (a *App) ChooseFolder() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "Choose Folder"})
}

func (a *App) ChooseApplication() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:                      "Choose Application",
		DefaultDirectory:           "/Applications",
		TreatPackagesAsDirectories: false,
		Filters:                    []runtime.FileFilter{{DisplayName: "Applications", Pattern: "*.app"}},
	})
}

func (a *App) HideWindow() {
	platform.HideGrid()
}

func (a *App) ShowWindow() {
	platform.ShowGrid(true)
}

func (a *App) Quit() {
	runtime.Quit(a.ctx)
}

// --- Command line tool & updates ---

const cliInstallPath = "/usr/local/bin/dz"

// CLIInstalled reports whether the dz command line tool is installed.
func (a *App) CLIInstalled() bool {
	_, err := os.Stat(cliInstallPath)
	return err == nil
}

// InstallCLI copies the bundled dz binary to /usr/local/bin, prompting for
// administrator rights when needed.
func (a *App) InstallCLI() error {
	src, err := findCLIBinary()
	if err != nil {
		return err
	}
	// Try a plain copy first; fall back to an admin-privileged copy.
	if err := exec.Command("cp", "-f", src, cliInstallPath).Run(); err == nil {
		return nil
	}
	script := fmt.Sprintf(
		"do shell script \"mkdir -p /usr/local/bin && cp -f '%s' '%s' && chmod 755 '%s'\" with administrator privileges",
		src, cliInstallPath, cliInstallPath)
	if out, err := exec.Command("osascript", "-e", script).CombinedOutput(); err != nil {
		return fmt.Errorf("installing dz: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// findCLIBinary locates the dz binary shipped alongside the app: in the app
// bundle's Resources, next to the executable, or in the dev build tree.
func findCLIBinary() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(exe)
	candidates := []string{
		filepath.Join(dir, "..", "Resources", "dz"),
		filepath.Join(dir, "dz"),
		filepath.Join(dir, "..", "..", "..", "dz"), // build/bin/dz next to dragzone.app
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return filepath.Clean(c), nil
		}
	}
	return "", fmt.Errorf("dz binary not found next to the app; build it with: go build -o build/bin/dz ./cmd/dz")
}

// UpdateInfo describes the newest available code revision.
type UpdateInfo struct {
	Version   string `json:"version"`   // running version
	LatestSHA string `json:"latestSha"` // newest commit on main (short)
	LatestAt  string `json:"latestAt"`  // newest commit date
	URL       string `json:"url"`
}

// CheckForUpdates fetches the newest commit of the project repository.
func (a *App) CheckForUpdates() (UpdateInfo, error) {
	info := UpdateInfo{Version: appVersion, URL: "https://github.com/GuilhermeVozniak/drag-zone"}
	req, err := http.NewRequestWithContext(a.ctx, http.MethodGet,
		"https://api.github.com/repos/GuilhermeVozniak/drag-zone/commits/main", nil)
	if err != nil {
		return info, err
	}
	req.Header.Set("User-Agent", "DragZone")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return info, fmt.Errorf("checking for updates: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return info, fmt.Errorf("checking for updates: %s", resp.Status)
	}
	var commit struct {
		SHA    string `json:"sha"`
		Commit struct {
			Committer struct {
				Date string `json:"date"`
			} `json:"committer"`
		} `json:"commit"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&commit); err != nil {
		return info, err
	}
	if len(commit.SHA) >= 7 {
		info.LatestSHA = commit.SHA[:7]
	}
	info.LatestAt = commit.Commit.Committer.Date
	return info, nil
}

// GetVersion returns the running app version.
func (a *App) GetVersion() string {
	return appVersion
}
