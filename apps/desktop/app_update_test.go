package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseTeamID(t *testing.T) {
	out := "Executable=/a/b\nIdentifier=com.x.y\nFormat=app bundle\nTeamIdentifier=AB12CD34EF\nSealed Resources=yes\n"
	if got := parseTeamID(out); got != "AB12CD34EF" {
		t.Errorf("parseTeamID = %q", got)
	}
	if got := parseTeamID("Identifier=com.x.y\nTeamIdentifier=not set\n"); got != "not set" {
		t.Errorf("parseTeamID = %q", got)
	}
	if got := parseTeamID("Identifier=com.x.y\n"); got != "" {
		t.Errorf("parseTeamID without team = %q", got)
	}
}

func TestAllowedDownloadHost(t *testing.T) {
	ok := []string{
		"https://github.com/GuilhermeVozniak/drag-zone/releases/download/v1/a.dmg",
		"https://objects.githubusercontent.com/x/y.dmg",
	}
	bad := []string{
		"http://github.com/x.dmg", // plain http
		"https://evil.com/x.dmg",
		"https://github.com.evil.com/x.dmg",
		"not a url",
		"https://user:pw@/x.dmg",
	}
	for _, u := range ok {
		if !allowedDownloadHost(u) {
			t.Errorf("allowedDownloadHost(%q) = false, want true", u)
		}
	}
	for _, u := range bad {
		if allowedDownloadHost(u) {
			t.Errorf("allowedDownloadHost(%q) = true, want false", u)
		}
	}
}

func TestDownloadUpdate(t *testing.T) {
	payload := strings.Repeat("0123456789", 1000) // 10 KB
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprint(len(payload)))
		_, _ = w.Write([]byte(payload))
	}))
	defer srv.Close()

	dst := filepath.Join(t.TempDir(), "x.dmg")
	var pcts []int
	if err := downloadUpdate(context.Background(), srv.URL, dst, func(p int) { pcts = append(pcts, p) }); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil || string(got) != payload {
		t.Fatalf("downloaded content mismatch (err=%v)", err)
	}
	if len(pcts) == 0 || pcts[len(pcts)-1] != 100 {
		t.Errorf("progress callbacks = %v, want ...100", pcts)
	}
}

func TestDownloadUpdateHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	if err := downloadUpdate(context.Background(), srv.URL, filepath.Join(t.TempDir(), "x"), func(int) {}); err == nil {
		t.Error("404 download should fail")
	}
}

// makeFakeApp builds a minimal .app bundle with a marker file, ad-hoc
// signed so it passes `codesign --verify --deep --strict`.
func makeFakeApp(t *testing.T, dir, marker string) string {
	t.Helper()
	app := filepath.Join(dir, "dragzone.app")
	macOS := filepath.Join(app, "Contents", "MacOS")
	if err := os.MkdirAll(macOS, 0o755); err != nil {
		t.Fatal(err)
	}
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><dict>
<key>CFBundleExecutable</key><string>dragzone</string>
<key>CFBundleIdentifier</key><string>dev.vozniak.dragzone</string>
</dict></plist>`
	if err := os.WriteFile(filepath.Join(app, "Contents", "Info.plist"), []byte(plist), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(macOS, "dragzone")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(app, "Contents", marker), []byte(marker), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("codesign", "--force", "--deep", "--sign", "-", app).CombinedOutput(); err != nil {
		t.Skipf("codesign ad-hoc unavailable: %s", out)
	}
	return app
}

// createUpdateDMG packs a fake dragzone.app into a real DMG via hdiutil.
func createUpdateDMG(t *testing.T, marker string) (dmgPath, srcDir string) {
	t.Helper()
	if _, err := exec.LookPath("hdiutil"); err != nil {
		t.Skip("hdiutil unavailable")
	}
	srcDir = t.TempDir()
	makeFakeApp(t, srcDir, marker)
	dmgPath = filepath.Join(t.TempDir(), "update.dmg")
	out, err := exec.Command("hdiutil", "create", "-volname", "DragZone", "-srcfolder", srcDir,
		"-ov", "-format", "UDZO", dmgPath).CombinedOutput()
	if err != nil {
		t.Fatalf("hdiutil create: %s", out)
	}
	return dmgPath, srcDir
}

func TestAttachDMGRoundTrip(t *testing.T) {
	dmgPath, _ := createUpdateDMG(t, "v9")
	mountPoint, detach, err := attachDMG(dmgPath)
	if err != nil {
		t.Fatal(err)
	}
	defer detach()
	got, err := os.ReadFile(filepath.Join(mountPoint, "dragzone.app", "Contents", "v9"))
	if err != nil || string(got) != "v9" {
		t.Errorf("mounted app marker = %q, err=%v", got, err)
	}
	detach()
	if _, err := os.Stat(filepath.Join(mountPoint, "dragzone.app")); !os.IsNotExist(err) {
		t.Errorf("mount point should be gone after detach (err=%v)", err)
	}
}

func TestReplaceAppSwapsAndCleansUp(t *testing.T) {
	dir := t.TempDir()
	newApp := makeFakeApp(t, dir, "new")
	current := makeFakeApp(t, filepath.Join(dir, "installed"), "old")

	if err := replaceApp(newApp, current); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(current, "Contents", "new")); err != nil {
		t.Error("new bundle content not in place")
	}
	if _, err := os.Stat(filepath.Join(current, "Contents", "old")); !os.IsNotExist(err) {
		t.Error("old bundle content survived the swap")
	}
	// No backup left behind.
	matches, _ := filepath.Glob(current + ".old-*")
	if len(matches) != 0 {
		t.Errorf("backup bundles left behind: %v", matches)
	}
}

func TestReplaceAppRollsBackOnFailure(t *testing.T) {
	dir := t.TempDir()
	current := makeFakeApp(t, dir, "old")
	missing := filepath.Join(dir, "no-such.app")

	err := replaceApp(missing, current)
	if err == nil {
		t.Fatal("replaceApp with a missing source should fail")
	}
	// The original bundle must be restored intact.
	if _, err := os.Stat(filepath.Join(current, "Contents", "old")); err != nil {
		t.Errorf("original app not restored after failed swap: %v", err)
	}
}

func TestCurrentAppBundleRejectsUnpackaged(t *testing.T) {
	// The test binary never runs from a .app bundle.
	if _, err := currentAppBundle(); err == nil {
		t.Error("expected unpackaged executable to be rejected")
	}
}

func TestVerifyUpdateCandidateRejectsUnsigned(t *testing.T) {
	dir := t.TempDir()
	current := makeFakeApp(t, dir, "old")
	// An unsigned (invalid) candidate must fail verification.
	bad := filepath.Join(dir, "dragzone.app")
	if err := os.MkdirAll(filepath.Join(bad, "Contents", "MacOS"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bad, "Contents", "Info.plist"), []byte("<plist/>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyUpdateCandidate(bad, current); err == nil {
		t.Error("unsigned candidate should fail verification")
	}
}

// TestInstallFromDMGFlow drives the full verify→swap→record pipeline with a
// real DMG, the way installUpdate does after the download.
func TestInstallFromDMGFlow(t *testing.T) {
	app := newTestApp(t)
	var stages []string
	app.onEmit = func(ev string, data ...any) {
		if ev != EventUpdateProgress {
			return
		}
		if p, ok := data[0].(UpdateProgress); ok {
			stages = append(stages, p.Stage)
		}
	}

	dmgPath, _ := createUpdateDMG(t, "v9.9.9")
	current := makeFakeApp(t, t.TempDir(), "old")

	if err := app.installFromDMG(dmgPath, current, "9.9.9"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(current, "Contents", "v9.9.9")); err != nil {
		t.Error("updated app not in place")
	}
	if got := app.settings.Get().LastUpdateNotified; got != "9.9.9" {
		t.Errorf("LastUpdateNotified = %q", got)
	}
	joined := strings.Join(stages, ",")
	for _, want := range []string{"verifying", "installing"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q progress stage in %v", want, stages)
		}
	}
}

func TestInstallFromDMGMissingApp(t *testing.T) {
	if _, err := exec.LookPath("hdiutil"); err != nil {
		t.Skip("hdiutil unavailable")
	}
	app := newTestApp(t)
	// A DMG without dragzone.app inside.
	src := t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "readme.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	dmg := filepath.Join(t.TempDir(), "empty.dmg")
	if out, err := exec.Command("hdiutil", "create", "-srcfolder", src, "-ov", "-format", "UDZO", dmg).CombinedOutput(); err != nil {
		t.Fatalf("hdiutil create: %s", out)
	}
	if err := app.installFromDMG(dmg, filepath.Join(t.TempDir(), "x.app"), "1"); err == nil {
		t.Error("DMG without dragzone.app should fail")
	}
}

func TestUpdateProgressEmits(t *testing.T) {
	app := newTestApp(t)
	var got UpdateProgress
	app.onEmit = func(ev string, data ...any) {
		if ev == EventUpdateProgress {
			got = data[0].(UpdateProgress)
		}
	}
	app.updateProgress("downloading", 42, "1.2.3", "")
	if got.Stage != "downloading" || got.Percent != 42 || got.Version != "1.2.3" {
		t.Errorf("progress = %+v", got)
	}
}

func TestInstallUpdateConcurrentGuard(t *testing.T) {
	// Hold the lock; a second InstallUpdate must refuse immediately.
	if !updateMu.TryLock() {
		t.Fatal("lock should be free")
	}
	defer updateMu.Unlock()
	app := newTestApp(t)
	if err := app.InstallUpdate(); err == nil {
		t.Error("concurrent InstallUpdate should be rejected")
	}
}
