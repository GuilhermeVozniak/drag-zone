package main

import (
	"os"
	"strconv"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"dragzone/internal/config"
	"dragzone/internal/platform"
)

// Settings, native dialogs, window control, and icon bindings.

func (a *App) GetSettings() config.Settings {
	return a.settings.Get()
}

// SetSettings persists new settings and applies the side-effectful ones
// (global hotkey, login item) immediately.
func (a *App) SetSettings(s config.Settings) error {
	prev := a.settings.Get()
	if err := a.settings.Set(s); err != nil {
		return err
	}
	if s.GlobalShortcut != prev.GlobalShortcut {
		platform.SetHotkeyF(parseFKey(s.GlobalShortcut))
	}
	if s.LaunchAtLogin != prev.LaunchAtLogin {
		if err := platform.SetLoginItem(s.LaunchAtLogin); err != nil {
			return err
		}
	}
	return nil
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
