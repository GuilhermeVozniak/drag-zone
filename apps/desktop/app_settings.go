package main

import (
	"context"
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
		// Width scales with the grid-size setting; height is owned by the
		// frontend's content-driven auto-resize (see ResizeWindow), so it is
		// preserved here rather than reset to the small startup default —
		// otherwise every settings save would collapse the window.
		scale := s.Scale()
		_, height := runtime.WindowGetSize(a.ctx)
		runtime.WindowSetSize(a.ctx, int(float64(windowWidth)*scale), height)
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
	if icon, ok := a.iconCache[path]; ok {
		a.iconMu.Unlock()
		return icon
	}
	a.iconMu.Unlock()

	// Generate outside the lock: QuickLook thumbnailing can take up to a
	// couple of seconds, and holding iconMu across it would serialize every
	// tile's icon load — including the several thumbnails of one multi-file
	// stack — behind a single slow file.
	var icon string
	if info, err := os.Stat(path); err == nil && info.Mode().IsRegular() {
		icon, _ = platform.FileThumbnailPNGBase64(path, 64)
	}
	if icon == "" {
		if fallback, err := platform.FileIconPNGBase64(path, 64); err == nil {
			icon = fallback
		}
	}

	a.iconMu.Lock()
	a.iconCache[path] = icon
	a.iconMu.Unlock()
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

// ResizeWindow fits the window to the frontend's measured content height,
// called by the panel's ResizeObserver (see useAutoResize.ts) whenever its
// natural height changes. Width is owned by the grid-size setting (see
// applySettings, which scales it) and is preserved here rather than reset to
// the windowWidth constant — otherwise every auto-resize would revert a
// non-default grid size's width scaling. Only the height grows/shrinks to
// the content, clamped to a sane range.
func (a *App) ResizeWindow(height int) {
	const minH, maxH = 120, 640
	if height < minH {
		height = minH
	}
	if height > maxH {
		height = maxH
	}
	width, _ := runtime.WindowGetSize(a.ctx)
	runtime.WindowSetSize(a.ctx, width, height)
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

// UpdateInfo describes the newest published release relative to this build.
type UpdateInfo struct {
	Version     string `json:"version"`     // running version
	Latest      string `json:"latest"`      // newest released version
	Available   bool   `json:"available"`   // latest is newer than running
	URL         string `json:"url"`         // release page
	DownloadURL string `json:"downloadUrl"` // direct DMG asset, when published
	PublishedAt string `json:"publishedAt"`
}

const latestReleaseAPI = "https://api.github.com/repos/GuilhermeVozniak/drag-zone/releases/latest"

// CheckForUpdates fetches the newest published release and reports whether
// it is newer than the running version.
func (a *App) CheckForUpdates() (UpdateInfo, error) {
	return a.checkForUpdates(latestReleaseAPI)
}

func (a *App) checkForUpdates(apiURL string) (UpdateInfo, error) {
	info := UpdateInfo{Version: appVersion, URL: "https://github.com/GuilhermeVozniak/drag-zone/releases"}
	ctx := a.ctx
	if ctx == nil { // before Wails startup (tests)
		ctx = context.Background()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return info, err
	}
	req.Header.Set("User-Agent", "DragZone")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return info, fmt.Errorf("checking for updates: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		// No published releases yet: not an error, just nothing to update to.
		return info, nil
	}
	if resp.StatusCode != http.StatusOK {
		return info, fmt.Errorf("checking for updates: %s", resp.Status)
	}
	var release struct {
		TagName     string `json:"tag_name"`
		HTMLURL     string `json:"html_url"`
		PublishedAt string `json:"published_at"`
		Assets      []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return info, err
	}
	info.Latest = strings.TrimPrefix(release.TagName, "v")
	info.PublishedAt = release.PublishedAt
	if release.HTMLURL != "" {
		info.URL = release.HTMLURL
	}
	for _, asset := range release.Assets {
		if strings.HasSuffix(asset.Name, ".dmg") {
			info.DownloadURL = asset.BrowserDownloadURL
			break
		}
	}
	info.Available = versionNewer(info.Latest, appVersion)
	return info, nil
}

// versionNewer reports whether version a is strictly newer than b, comparing
// up to three numeric components ("0.10.2" > "0.9.9"); a leading "v" and any
// non-numeric suffix per component are ignored.
func versionNewer(a, b string) bool {
	pa, pb := versionParts(a), versionParts(b)
	for i := 0; i < 3; i++ {
		if pa[i] != pb[i] {
			return pa[i] > pb[i]
		}
	}
	return false
}

func versionParts(v string) [3]int {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	var parts [3]int
	for i, s := range strings.SplitN(v, ".", 3) {
		digits := s
		for j, r := range s {
			if r < '0' || r > '9' {
				digits = s[:j]
				break
			}
		}
		parts[i], _ = strconv.Atoi(digits)
	}
	return parts
}

// autoUpdateLoop checks for updates shortly after launch and then daily,
// consulting the AutoUpdateCheck setting at every tick so toggling it takes
// effect without a restart. Runs until the app context is cancelled.
func (a *App) autoUpdateLoop() {
	timer := time.NewTimer(10 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-a.ctx.Done():
			return
		case <-timer.C:
		}
		a.autoUpdateCheck()
		timer.Reset(24 * time.Hour)
	}
}

// autoUpdateCheck notifies about a newer release, once per version.
func (a *App) autoUpdateCheck() {
	if !a.settings.Get().AutoUpdateCheck {
		return
	}
	info, err := a.CheckForUpdates()
	if err != nil || !info.Available {
		return
	}
	s := a.settings.Get()
	if s.LastUpdateNotified == info.Latest {
		return
	}
	s.LastUpdateNotified = info.Latest
	_ = a.settings.Set(s)
	a.services.Notify("DragZone "+info.Latest+" is available",
		"Open Settings › Updates to download the new version.")
}

// GetVersion returns the running app version.
func (a *App) GetVersion() string {
	return appVersion
}

// About shows a small native dialog with the app version, like the menu-bar
// app's About item.
func (a *App) About() {
	_, _ = runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.InfoDialog,
		Title:   "DragZone",
		Message: "DragZone " + appVersion + "\n\nA Dropzone 4–style menu bar app for macOS.\nBuilt with Wails, Go, and React.",
	})
}
