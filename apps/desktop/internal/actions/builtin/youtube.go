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

// ytdlpCmd is a seam so tests stub the yt-dlp invocation.
var ytdlpCmd = func(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, "yt-dlp", args...)
}

// ytdlpLookPath is a seam so tests can simulate a missing/present yt-dlp
// binary without touching PATH.
var ytdlpLookPath = func() (string, error) {
	return exec.LookPath("yt-dlp")
}

// YouTubeDownloader downloads a dropped or clicked YouTube URL via the
// yt-dlp CLI (must be installed separately, e.g. `brew install yt-dlp`).
type YouTubeDownloader struct{}

func (YouTubeDownloader) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "youtube-dl",
		Name:        "YouTube Downloader",
		Description: "Download a YouTube video or audio track with yt-dlp.",
		Icon:        "youtube",
		Category:    "Downloads",
		Events:      []string{model.EventDragged, model.EventClicked},
		Accepts:     []model.ItemKind{model.ItemURL, model.ItemText},
		Options: []model.OptionField{
			{Key: "folder", Label: "Save to", Type: "folder", Placeholder: "~/Downloads"},
			{Key: "format", Label: "Format", Type: "select", Choices: []string{"video", "audio"}, Default: "video"},
		},
	}
}

func (y YouTubeDownloader) Dropped(ctx context.Context, inv actions.Invocation) (actions.Result, error) {
	return y.run(ctx, inv, inv.Payload.Text)
}

func (y YouTubeDownloader) Clicked(ctx context.Context, inv actions.Invocation) (actions.Result, error) {
	url, err := inv.Services.ReadClipboard()
	if err != nil {
		return actions.Result{}, fmt.Errorf("reading clipboard: %w", err)
	}
	return y.run(ctx, inv, url)
}

func (YouTubeDownloader) run(ctx context.Context, inv actions.Invocation, rawURL string) (actions.Result, error) {
	url := strings.TrimSpace(rawURL)
	if url == "" {
		return actions.Result{}, fmt.Errorf("no URL to download")
	}

	if _, err := ytdlpLookPath(); err != nil {
		return actions.Result{}, fmt.Errorf("yt-dlp not found — install it with: brew install yt-dlp")
	}

	dir := expandHome(inv.Target.Option("folder", ""))
	if dir == "" {
		dir = expandHome("~/Downloads")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return actions.Result{}, fmt.Errorf("creating download folder: %w", err)
	}

	args := []string{
		"-o", filepath.Join(dir, "%(title)s.%(ext)s"),
		"--print", "after_move:filepath",
	}
	if inv.Target.Option("format", "video") == "audio" {
		args = append(args, "-x", "--audio-format", "mp3")
	}
	args = append(args, url)

	out, err := ytdlpCmd(ctx, args...).Output()
	if err != nil {
		return actions.Result{}, fmt.Errorf("running yt-dlp: %w", err)
	}

	path := ""
	for _, line := range strings.Split(string(out), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			path = line
			break
		}
	}

	msg := "Downloaded"
	if path != "" {
		if _, statErr := os.Stat(path); statErr == nil {
			if inv.AddDropBar != nil {
				inv.AddDropBar([]string{path})
			}
			msg = fmt.Sprintf("Downloaded %s", filepath.Base(path))
		}
	}

	return actions.Result{Message: msg}, nil
}
