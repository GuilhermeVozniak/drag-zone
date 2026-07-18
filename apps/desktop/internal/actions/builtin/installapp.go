package builtin

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// runCmd executes an external command and returns its standard output. It is
// a package variable purely as a test seam: tests substitute a fake to cover
// the mount/copy/launch/eject/trash sequence without invoking real
// hdiutil/ditto/open. The default is behavior-identical to calling
// exec.CommandContext(ctx, name, args...).Output() directly.
var runCmd = func(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

// InstallApp installs a dropped .dmg or .zip: mounts/extracts it, copies the
// contained .app to /Applications, launches it, and cleans up.
type InstallApp struct{}

func (InstallApp) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "install-app",
		Name:        "Install Application",
		Description: "Drop a .dmg or .zip to install the app inside into /Applications and launch it.",
		Icon:        "app-window",
		Category:    "Utilities",
		Events:      []string{model.EventDragged},
		Accepts:     []model.ItemKind{model.ItemFiles},
	}
}

func (InstallApp) Dropped(ctx context.Context, inv actions.Invocation) (actions.Result, error) {
	if len(inv.Payload.Paths) != 1 {
		return actions.Result{}, fmt.Errorf("drop a single .dmg or .zip file")
	}
	src := inv.Payload.Paths[0]

	var appName string
	var err error
	switch strings.ToLower(filepath.Ext(src)) {
	case ".dmg":
		appName, err = installFromDMG(ctx, src, inv.Progress)
	case ".zip":
		appName, err = installFromZip(ctx, src, inv.Progress)
	default:
		return actions.Result{}, fmt.Errorf("unsupported file type %s (need .dmg or .zip)", filepath.Ext(src))
	}
	if err != nil {
		return actions.Result{}, err
	}

	inv.Progress.Detail("Launching " + appName)
	installed := filepath.Join("/Applications", appName)
	if _, err := runCmd(ctx, "open", installed); err != nil {
		return actions.Result{}, fmt.Errorf("launching %s: %w", appName, err)
	}
	if err := inv.Services.Trash([]string{src}); err == nil {
		return actions.Result{Message: "Installed " + appName + " (installer moved to Trash)"}, nil
	}
	return actions.Result{Message: "Installed " + appName}, nil
}

func installFromDMG(ctx context.Context, dmg string, progress actions.Progress) (string, error) {
	progress.Detail("Mounting disk image")
	out, err := runCmd(ctx, "hdiutil", "attach", "-nobrowse", "-readonly", "-plist", dmg)
	if err != nil {
		return "", fmt.Errorf("mounting %s: %w", filepath.Base(dmg), err)
	}
	mount := mountPointFromPlist(string(out))
	if mount == "" {
		return "", fmt.Errorf("could not find mount point for %s", filepath.Base(dmg))
	}
	// Detach uses a background context (not ctx) so cleanup still runs even
	// if the invocation's context is done, matching the prior exec.Command
	// (no context) behavior.
	defer func() { _, _ = runCmd(context.Background(), "hdiutil", "detach", "-quiet", mount) }()

	app, err := findApp(mount)
	if err != nil {
		return "", err
	}
	return copyAppToApplications(ctx, app, progress)
}

func installFromZip(ctx context.Context, zipPath string, progress actions.Progress) (string, error) {
	progress.Detail("Extracting archive")
	tmp, err := os.MkdirTemp("", "dragzone-install-*")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)

	if _, err := runCmd(ctx, "ditto", "-x", "-k", zipPath, tmp); err != nil {
		return "", fmt.Errorf("extracting %s: %w", filepath.Base(zipPath), err)
	}
	app, err := findApp(tmp)
	if err != nil {
		return "", err
	}
	return copyAppToApplications(ctx, app, progress)
}

// mountPointFromPlist extracts the first /Volumes mount point from hdiutil's
// plist output without a full plist parser.
func mountPointFromPlist(plist string) string {
	for _, line := range strings.Split(plist, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "<string>/Volumes/") {
			return strings.TrimSuffix(strings.TrimPrefix(line, "<string>"), "</string>")
		}
	}
	return ""
}

func findApp(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".app") {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	// One level deep covers zips that extract into a folder.
	for _, e := range entries {
		if e.IsDir() {
			sub, err := os.ReadDir(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			for _, s := range sub {
				if strings.HasSuffix(s.Name(), ".app") {
					return filepath.Join(dir, e.Name(), s.Name()), nil
				}
			}
		}
	}
	return "", fmt.Errorf("no .app found inside")
}

// copyAppToApplications uses ditto to preserve bundle metadata, signatures,
// and resource forks that a plain file copy would lose.
func copyAppToApplications(ctx context.Context, app string, progress actions.Progress) (string, error) {
	name := filepath.Base(app)
	progress.Detail("Copying " + name + " to Applications")
	dst := filepath.Join("/Applications", name)
	if _, err := os.Stat(dst); err == nil {
		if err := os.RemoveAll(dst); err != nil {
			return "", fmt.Errorf("replacing existing %s: %w", name, err)
		}
	}
	if _, err := runCmd(ctx, "ditto", app, dst); err != nil {
		return "", fmt.Errorf("copying %s to /Applications: %w", name, err)
	}
	return name, nil
}
