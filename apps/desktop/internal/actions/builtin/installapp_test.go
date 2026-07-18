package builtin

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// recordedCmd captures one runCmd invocation for sequence assertions.
type recordedCmd struct {
	name string
	args []string
}

// fakeMountPlist builds a minimal hdiutil-attach-style plist whose mount
// point is the given directory. mountPointFromPlist only requires the line
// to start with "<string>/Volumes/", so "/Volumes/.." + dir (an absolute
// path) satisfies that prefix while the OS resolves ".." straight back to
// dir — letting the test point "the mount" at a real, writable temp
// directory without creating anything under the (SIP-protected) /Volumes.
func fakeMountPlist(dir string) string {
	mount := "/Volumes/.." + dir
	return "<plist><dict><key>system-entities</key><array><dict>\n" +
		"<key>mount-point</key>\n" +
		"<string>" + mount + "</string>\n" +
		"</dict></array></dict></plist>"
}

// withFakeRunCmd swaps runCmd for f and restores the original on cleanup.
func withFakeRunCmd(t *testing.T, f func(ctx context.Context, name string, args ...string) ([]byte, error)) *[]recordedCmd {
	t.Helper()
	var calls []recordedCmd
	orig := runCmd
	t.Cleanup(func() { runCmd = orig })
	runCmd = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		calls = append(calls, recordedCmd{name: name, args: append([]string(nil), args...)})
		return f(ctx, name, args...)
	}
	return &calls
}

// TestInstallAppDroppedDMGSequence covers the full happy path for a .dmg
// drop: mount, copy into /Applications, detach (the deferred cleanup inside
// installFromDMG runs before it returns to Dropped), launch, then trash the
// source dmg. It asserts the exact exec sequence and that no real hdiutil,
// ditto, or open ever runs.
func TestInstallAppDroppedDMGSequence(t *testing.T) {
	mountDir := t.TempDir()
	appBundle := "DragZoneSeamTest.app"
	if err := os.Mkdir(filepath.Join(mountDir, appBundle), 0o755); err != nil {
		t.Fatal(err)
	}
	plist := fakeMountPlist(mountDir)

	calls := withFakeRunCmd(t, func(_ context.Context, name string, args ...string) ([]byte, error) {
		switch {
		case name == "hdiutil" && args[0] == "attach":
			return []byte(plist), nil
		default:
			return nil, nil
		}
	})

	svc := &recServices{}
	src := "/fake/Installer.dmg"
	inv := actions.Invocation{
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{src}},
		Progress: nullProgress{},
		Services: svc,
	}

	res, err := (InstallApp{}).Dropped(context.Background(), inv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Message == "" {
		t.Error("expected a non-empty result message")
	}

	wantMount := "/Volumes/.." + mountDir
	wantDst := filepath.Join("/Applications", appBundle)
	wantSrcApp := filepath.Join(mountDir, appBundle)

	if len(*calls) != 4 {
		t.Fatalf("expected 4 exec calls, got %d: %+v", len(*calls), *calls)
	}

	// 1: mount.
	if got := (*calls)[0]; got.name != "hdiutil" || got.args[0] != "attach" {
		t.Errorf("call 0 = %+v, want hdiutil attach", got)
	}
	// 2: copy the .app into /Applications via ditto.
	if got := (*calls)[1]; got.name != "ditto" || len(got.args) != 2 || got.args[0] != wantSrcApp || got.args[1] != wantDst {
		t.Errorf("call 1 = %+v, want ditto %s %s", got, wantSrcApp, wantDst)
	}
	// 3: eject — the deferred detach runs before installFromDMG returns to
	// Dropped, i.e. before the launch step below.
	if got := (*calls)[2]; got.name != "hdiutil" || got.args[0] != "detach" || got.args[2] != wantMount {
		t.Errorf("call 2 = %+v, want hdiutil detach ... %s", got, wantMount)
	}
	// 4: launch the installed app.
	if got := (*calls)[3]; got.name != "open" || len(got.args) != 1 || got.args[0] != wantDst {
		t.Errorf("call 3 = %+v, want open %s", got, wantDst)
	}

	// 5: trash the source dmg (via Services, not exec).
	if len(svc.Trashed) != 1 || len(svc.Trashed[0]) != 1 || svc.Trashed[0][0] != src {
		t.Errorf("Trashed = %+v, want [[%s]]", svc.Trashed, src)
	}
}

// TestInstallAppMountFailureAbortsEarly covers a failing `hdiutil attach`:
// the error must be %w-wrapped and no copy/launch/detach/trash step may run.
func TestInstallAppMountFailureAbortsEarly(t *testing.T) {
	wantErr := errors.New("boom: no such image")
	calls := withFakeRunCmd(t, func(_ context.Context, name string, args ...string) ([]byte, error) {
		return nil, wantErr
	})

	svc := &recServices{}
	inv := actions.Invocation{
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{"/fake/Installer.dmg"}},
		Progress: nullProgress{},
		Services: svc,
	}

	_, err := (InstallApp{}).Dropped(context.Background(), inv)
	if err == nil {
		t.Fatal("expected an error when mounting fails")
	}
	if !errors.Is(err, wantErr) {
		t.Errorf("error %v does not wrap the mount failure %v", err, wantErr)
	}
	if len(*calls) != 1 {
		t.Fatalf("expected exactly one exec call (the failed attach), got %d: %+v", len(*calls), *calls)
	}
	if (*calls)[0].name != "hdiutil" || (*calls)[0].args[0] != "attach" {
		t.Errorf("the only call should be hdiutil attach, got %+v", (*calls)[0])
	}
	if len(svc.Trashed) != 0 {
		t.Errorf("mount failure must not reach the trash step, got %+v", svc.Trashed)
	}
}

// TestInstallAppRejectsBeforeExec covers inputs that must be rejected before
// any command is ever shelled out: an unsupported extension, an empty
// payload, and more than one dropped path.
func TestInstallAppRejectsBeforeExec(t *testing.T) {
	calls := withFakeRunCmd(t, func(_ context.Context, name string, args ...string) ([]byte, error) {
		return nil, nil
	})

	cases := []struct {
		name  string
		paths []string
	}{
		{"empty payload", nil},
		{"more than one path", []string{"/a.dmg", "/b.dmg"}},
		{"unsupported extension", []string{"/a.txt"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inv := actions.Invocation{
				Payload:  model.Payload{Kind: model.ItemFiles, Paths: tc.paths},
				Progress: nullProgress{},
				Services: &recServices{},
			}
			if _, err := (InstallApp{}).Dropped(context.Background(), inv); err == nil {
				t.Error("expected an error")
			}
		})
	}

	if len(*calls) != 0 {
		t.Errorf("none of these inputs should shell out, got %+v", *calls)
	}
}
