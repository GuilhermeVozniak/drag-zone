package main

// In-place auto-update: downloads the newest release's DMG from GitHub,
// verifies the bundled app's code signature, swaps it over the running
// installation (with rollback), and relaunches. This reuses the existing
// release pipeline (signed + notarized universal DMG on GitHub Releases)
// instead of integrating the Sparkle framework, which would need a bundled
// native framework, an EdDSA-signed appcast feed, and cgo glue.
//
// Quarantine is not a concern: files fetched with a plain HTTP client never
// receive the com.apple.quarantine xattr (only LaunchServices downloads get
// it), and the replacement app is signed and notarized regardless.

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// UpdateProgress is streamed to the frontend while an update installs.
type UpdateProgress struct {
	Stage   string `json:"stage"` // checking | downloading | verifying | installing | relaunching | done | error
	Percent int    `json:"percent"`
	Version string `json:"version"`
	Error   string `json:"error,omitempty"`
}

// updateMu serializes update installs (the button is disabled in the UI,
// but the dz CLI or a double event could re-trigger).
var updateMu sync.Mutex

// InstallUpdate downloads and installs the newest release, then relaunches.
// It runs asynchronously; progress is streamed via EventUpdateProgress. The
// returned error is only for failures before the install starts (e.g. no
// update available); later failures surface as progress events.
func (a *App) InstallUpdate() error {
	if !updateMu.TryLock() {
		return fmt.Errorf("an update is already being installed")
	}
	ctx := a.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	go func() {
		defer updateMu.Unlock()
		a.installUpdate(ctx)
	}()
	return nil
}

func (a *App) updateProgress(stage string, percent int, version, errMsg string) {
	a.emit(EventUpdateProgress, UpdateProgress{Stage: stage, Percent: percent, Version: version, Error: errMsg})
}

func (a *App) installUpdate(ctx context.Context) {
	fail := func(format string, args ...any) {
		a.updateProgress("error", -1, "", fmt.Sprintf(format, args...))
	}

	a.updateProgress("checking", -1, "", "")
	info, err := a.checkForUpdates(latestReleaseAPI)
	if err != nil {
		fail("checking for updates: %v", err)
		return
	}
	if !info.Available {
		fail("no update available")
		return
	}
	if info.DownloadURL == "" {
		fail("release %s has no DMG asset", info.Latest)
		return
	}
	if !allowedDownloadHost(info.DownloadURL) {
		fail("refusing download from unexpected host: %s", info.DownloadURL)
		return
	}

	// Where are we installed? Only a packaged .app can be swapped — dev
	// builds (wails dev, go run, go test) are left alone.
	currentApp, err := currentAppBundle()
	if err != nil {
		fail("%v", err)
		return
	}

	tmp, err := os.MkdirTemp("", "dragzone-update-*")
	if err != nil {
		fail("%v", err)
		return
	}
	defer os.RemoveAll(tmp)

	dmgPath := filepath.Join(tmp, "update.dmg")
	if err := downloadUpdate(ctx, info.DownloadURL, dmgPath, func(pct int) {
		a.updateProgress("downloading", pct, info.Latest, "")
	}); err != nil {
		fail("downloading update: %v", err)
		return
	}

	if err := a.installFromDMG(dmgPath, currentApp, info.Latest); err != nil {
		fail("%v", err)
		return
	}

	a.updateProgress("done", 100, info.Latest, "")
	// Relaunch only inside a real Wails runtime (a.ctx is nil in tests).
	if a.ctx != nil {
		a.relaunchApp(currentApp)
	}
}

// installFromDMG verifies and installs the app in dmgPath over currentApp,
// emitting verifying/installing progress. On success it records the new
// version so the daily check stays quiet.
func (a *App) installFromDMG(dmgPath, currentApp, version string) error {
	mountPoint, detach, err := attachDMG(dmgPath)
	if err != nil {
		return fmt.Errorf("mounting update: %w", err)
	}
	defer detach()

	newApp := filepath.Join(mountPoint, "dragzone.app")
	if _, err := os.Stat(newApp); err != nil {
		return fmt.Errorf("update DMG does not contain dragzone.app")
	}

	a.updateProgress("verifying", -1, version, "")
	if err := verifyUpdateCandidate(newApp, currentApp); err != nil {
		return fmt.Errorf("verifying update: %w", err)
	}

	a.updateProgress("installing", -1, version, "")
	if err := replaceApp(newApp, currentApp); err != nil {
		return fmt.Errorf("installing update: %w", err)
	}

	// Record that this version is installed so the daily check doesn't
	// notify about it again after relaunch.
	s := a.settings.Get()
	s.LastUpdateNotified = version
	_ = a.settings.Set(s)
	return nil
}

// currentAppBundle returns the path of the .app bundle the process runs
// from, or an error for unpackaged (dev/test) executables.
func currentAppBundle() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	// <name>.app/Contents/MacOS/<binary> → three levels up.
	app := filepath.Dir(filepath.Dir(filepath.Dir(exe)))
	if !strings.HasSuffix(app, ".app") {
		return "", fmt.Errorf("not running from a packaged .app (%s); updates need the released build", exe)
	}
	return app, nil
}

// allowedDownloadHost restricts updates to GitHub hosts so a poisoned
// release response can't point the installer at an arbitrary server.
func allowedDownloadHost(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" {
		return false
	}
	h := u.Hostname()
	return h == "github.com" || strings.HasSuffix(h, ".github.com") ||
		h == "githubusercontent.com" || strings.HasSuffix(h, ".githubusercontent.com")
}

// downloadUpdate fetches url into dst, reporting 0-100 via onPercent. The
// caller is responsible for host validation (see allowedDownloadHost); this
// function stays host-agnostic so tests can use local servers.
func downloadUpdate(ctx context.Context, rawURL, dst string, onPercent func(int)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "DragZone")
	resp, err := (&http.Client{Timeout: 30 * time.Minute}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %s", resp.Status)
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	total := resp.ContentLength
	var got int64
	buf := make([]byte, 256*1024)
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := out.Write(buf[:n]); werr != nil {
				out.Close()
				return werr
			}
			got += int64(n)
			if total > 0 {
				onPercent(int(got * 100 / total))
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			out.Close()
			return rerr
		}
	}
	return out.Close()
}

// attachDMG mounts a disk image read-only and returns the mount point and a
// detach function (safe to call on error paths; best-effort).
func attachDMG(dmgPath string) (string, func(), error) {
	mountPoint, err := os.MkdirTemp("", "dragzone-dmg-*")
	if err != nil {
		return "", nil, err
	}
	out, err := exec.Command("hdiutil", "attach", "-nobrowse", "-readonly", "-mountpoint", mountPoint, dmgPath).CombinedOutput()
	if err != nil {
		os.RemoveAll(mountPoint)
		return "", nil, fmt.Errorf("hdiutil attach: %s", strings.TrimSpace(string(out)))
	}
	detach := func() {
		_ = exec.Command("hdiutil", "detach", "-quiet", mountPoint).Run()
		os.RemoveAll(mountPoint)
	}
	return mountPoint, detach, nil
}

// verifyUpdateCandidate checks the downloaded app before it replaces the
// running one: the signature must validate, and when the running app carries
// a Developer ID Team ID the candidate must match it (so a validly signed
// app from a *different* developer can never be swapped in).
func verifyUpdateCandidate(newApp, currentApp string) error {
	if out, err := exec.Command("codesign", "--verify", "--deep", "--strict", newApp).CombinedOutput(); err != nil {
		return fmt.Errorf("signature check failed: %s", strings.TrimSpace(string(out)))
	}
	currentTeam, err := appTeamID(currentApp)
	if err != nil || currentTeam == "" {
		// Dev/ad-hoc-signed installs have no Team ID to enforce; the
		// signature validity check above still applies.
		return nil
	}
	newTeam, err := appTeamID(newApp)
	if err != nil {
		return fmt.Errorf("reading update signature: %w", err)
	}
	if newTeam != currentTeam {
		return fmt.Errorf("update is signed by a different team (%s, want %s)", newTeam, currentTeam)
	}
	return nil
}

// appTeamID extracts the TeamIdentifier from a signed bundle, "" for
// unsigned or ad-hoc-signed bundles.
func appTeamID(appPath string) (string, error) {
	out, err := exec.Command("codesign", "-dv", appPath).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("codesign -dv: %s", strings.TrimSpace(string(out)))
	}
	return parseTeamID(string(out)), nil
}

func parseTeamID(codesignOutput string) string {
	for line := range strings.Lines(codesignOutput) {
		if id, ok := strings.CutPrefix(strings.TrimSpace(line), "TeamIdentifier="); ok {
			return strings.TrimSpace(id)
		}
	}
	return ""
}

// replaceApp swaps currentApp for newApp atomically-ish: the current bundle
// moves aside, the new one copies in, and any failure restores the original.
// Replacing a running app is safe on macOS — the running process keeps its
// open files; the next launch picks up the new bundle.
func replaceApp(newApp, currentApp string) error {
	backup := currentApp + ".old-" + time.Now().Format("20060102150405")
	if err := os.Rename(currentApp, backup); err != nil {
		return replaceAppPrivileged(newApp, currentApp)
	}
	// ditto preserves bundle metadata (code signature lives in the bundle,
	// resource forks and xattrs included) better than cp -R.
	if out, err := exec.Command("ditto", newApp, currentApp).CombinedOutput(); err != nil {
		// Roll back: never leave the user without a working app.
		_ = os.RemoveAll(currentApp)
		_ = os.Rename(backup, currentApp)
		return fmt.Errorf("copying update into place: %s", strings.TrimSpace(string(out)))
	}
	os.RemoveAll(backup)
	return nil
}

// replaceAppPrivileged retries the swap through an admin-privileged shell
// when /Applications isn't writable by the current user.
func replaceAppPrivileged(newApp, currentApp string) error {
	shQuote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'" }
	backup := currentApp + ".old-update"
	shell := fmt.Sprintf(
		"rm -rf %s && mv %s %s && ditto %s %s && rm -rf %s",
		shQuote(backup), shQuote(currentApp), shQuote(backup), shQuote(newApp), shQuote(currentApp), shQuote(backup))
	script := fmt.Sprintf("do shell script %q with administrator privileges", shell)
	if out, err := exec.Command("osascript", "-e", script).CombinedOutput(); err != nil {
		return fmt.Errorf("permission denied and admin install failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// relaunchApp starts the freshly installed copy and quits this process.
// A detached shell waits for this process to exit before opening the new
// copy: two instances must never run at once (each registers its own
// menu-bar icon). No-op outside a running Wails app (tests).
func (a *App) relaunchApp(appPath string) {
	if a.ctx == nil {
		return
	}
	// Small delay so the final progress event reaches the frontend first.
	time.Sleep(300 * time.Millisecond)
	script := `while kill -0 "$1" 2>/dev/null; do sleep 0.1; done; open -n "$2"`
	// Never Wait: this process is about to exit and the shell reparents to
	// launchd, opening the new copy as soon as the old one is gone.
	if err := exec.Command("/bin/sh", "-c", script, "sh", strconv.Itoa(os.Getpid()), appPath).Start(); err != nil {
		a.updateProgress("error", 0, "", fmt.Sprintf("relaunching: %v", err))
		return
	}
	// runtime.Quit routes through beforeClose, which swallows the request
	// while settings is open — and updates install from the Settings tab,
	// so without this the old process stays alive next to the new one.
	a.dragMu.Lock()
	a.settingsOpen = false
	a.dragMu.Unlock()
	runtime.Quit(a.ctx)
}
