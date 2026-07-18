package builtin

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// ScreenshotSFTP captures a screenshot the same way Screenshot does, then
// uploads it to a remote server over SFTP/FTP — reusing FTPUpload's
// connectRemote/uploadOne transport — and copies the resulting URL to the
// clipboard, Dropzone-4 style.
type ScreenshotSFTP struct{}

func (ScreenshotSFTP) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "screenshot-sftp",
		Name:        "Screenshot & Upload",
		Description: "Capture a screenshot and upload it to a server over SFTP or FTP.",
		Icon:        "cloud-upload",
		Category:    "Capture",
		Events:      []string{model.EventClicked},
		Multi:       true,
		Options: []model.OptionField{
			{Key: "mode", Label: "Capture", Type: "select", Choices: []string{"interactive", "window", "screen"}, Default: "interactive"},
			{Key: "protocol", Label: "Protocol", Type: "select", Choices: []string{"sftp", "ftp"}, Default: "sftp"},
			{Key: "host", Label: "Host", Type: "text", Required: true},
			{Key: "port", Label: "Port", Type: "text", Placeholder: "22 / 21"},
			{Key: "username", Label: "Username", Type: "text", Required: true},
			{Key: "password", Label: "Password", Type: "password", Required: true},
			{Key: "remote_dir", Label: "Remote directory", Type: "text", Placeholder: "/var/www/uploads"},
			{Key: "url_prefix", Label: "URL prefix (optional)", Type: "text", Placeholder: "https://example.com/uploads"},
		},
	}
}

func (ScreenshotSFTP) Clicked(ctx context.Context, inv actions.Invocation) (actions.Result, error) {
	host := inv.Target.Option("host", "")
	user := inv.Target.Option("username", "")
	pass := inv.Target.Option("password", "")
	if host == "" || user == "" || pass == "" {
		return actions.Result{}, fmt.Errorf("host, username and password must be configured")
	}

	if !hasScreenRecording() {
		requestScreenRecording()
		return actions.Result{Message: screenRecordingPermissionMessage}, nil
	}

	dst, err := captureScreenshotToTemp(ctx, inv.Target.Option("mode", "interactive"))
	if err != nil {
		return actions.Result{}, err
	}
	if dst == "" {
		// No file means the user cancelled the capture (e.g. pressed Esc).
		return actions.Result{Message: "Screenshot cancelled"}, nil
	}
	defer os.RemoveAll(filepath.Dir(dst))

	protocol := inv.Target.Option("protocol", "sftp")
	port := inv.Target.Option("port", "")
	if port == "" {
		if protocol == "ftp" {
			port = "21"
		} else {
			port = "22"
		}
	}

	entries, total, err := collectUploadEntries([]string{dst})
	if err != nil {
		return actions.Result{}, fmt.Errorf("preparing upload: %w", err)
	}

	remote, err := connectRemote(ctx, protocol, host, port, user, pass)
	if err != nil {
		return actions.Result{}, fmt.Errorf("connecting to %s: %w", host, err)
	}
	defer remote.close()

	remoteDir := inv.Target.Option("remote_dir", "")
	var done int64
	for _, e := range entries {
		rp := e.rel
		if remoteDir != "" {
			rp = path.Join(remoteDir, e.rel)
			if err := remote.mkdirAll(remoteDir); err != nil {
				return actions.Result{}, fmt.Errorf("creating remote directory %s: %w", remoteDir, err)
			}
		}
		if err := uploadOne(remote, e, rp, total, &done, inv.Progress); err != nil {
			return actions.Result{}, fmt.Errorf("uploading %s: %w", e.rel, err)
		}
	}

	result := actions.Result{Message: "Screenshot uploaded to " + host}
	if prefix := inv.Target.Option("url_prefix", ""); prefix != "" {
		result.URL = strings.TrimRight(prefix, "/") + "/" + filepath.Base(dst)
		if err := inv.Services.CopyToClipboard(result.URL); err != nil {
			return actions.Result{}, fmt.Errorf("copying URL to clipboard: %w", err)
		}
	}
	return result, nil
}

// captureScreenshotToTemp captures a screenshot via the same screenshotCmd/
// screencaptureArgs/screenshotNow seams Screenshot uses, saving it to a
// timestamped .png inside a fresh temp directory. It returns an empty path
// (and nil error) when the capture produced no file, i.e. the user cancelled.
func captureScreenshotToTemp(ctx context.Context, mode string) (string, error) {
	dir, err := os.MkdirTemp("", "dragzone-screenshot-sftp")
	if err != nil {
		return "", fmt.Errorf("creating temp folder: %w", err)
	}

	name := "Screenshot " + screenshotNow().Format("2006-01-02 at 15.04.05") + ".png"
	dst := filepath.Join(dir, name)

	if err := screenshotCmd(ctx, screencaptureArgs(mode, dst)...).Run(); err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("capturing screenshot: %w", err)
	}

	if _, err := os.Stat(dst); err != nil {
		os.RemoveAll(dir)
		return "", nil
	}
	return dst, nil
}
