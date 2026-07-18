package builtin

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"testing"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// withFakeYtdlpCmd swaps ytdlpCmd for a fake that records the args it was
// invoked with and runs a shell one-liner that prints resultPath to stdout
// and creates the file there, simulating yt-dlp's
// `--print after_move:filepath` output plus the downloaded file landing on
// disk.
func withFakeYtdlpCmd(t *testing.T, resultPath string) *[][]string {
	t.Helper()
	var calls [][]string
	orig := ytdlpCmd
	t.Cleanup(func() { ytdlpCmd = orig })
	ytdlpCmd = func(ctx context.Context, args ...string) *exec.Cmd {
		calls = append(calls, append([]string(nil), args...))
		return exec.CommandContext(ctx, "sh", "-c", `printf '%s\n' "$1"; touch "$1"`, "--", resultPath)
	}
	return &calls
}

func withFakeYtdlpLookPath(t *testing.T, found bool) {
	t.Helper()
	orig := ytdlpLookPath
	t.Cleanup(func() { ytdlpLookPath = orig })
	ytdlpLookPath = func() (string, error) {
		if found {
			return "/usr/local/bin/yt-dlp", nil
		}
		return "", errors.New("not found")
	}
}

func TestYouTubeDownloaderSpec(t *testing.T) {
	spec := YouTubeDownloader{}.Spec()
	if spec.ID != "youtube-dl" {
		t.Errorf("ID = %q", spec.ID)
	}
	if spec.Icon != "youtube" {
		t.Errorf("Icon = %q", spec.Icon)
	}
	if len(spec.Events) != 2 || spec.Events[0] != model.EventDragged || spec.Events[1] != model.EventClicked {
		t.Errorf("Events = %v", spec.Events)
	}
	if len(spec.Accepts) != 2 || spec.Accepts[0] != model.ItemURL || spec.Accepts[1] != model.ItemText {
		t.Errorf("Accepts = %v", spec.Accepts)
	}
	byKey := map[string]model.OptionField{}
	for _, o := range spec.Options {
		byKey[o.Key] = o
	}
	if f, ok := byKey["format"]; !ok || f.Default != "video" {
		t.Errorf("format option = %+v", f)
	}
}

func TestYouTubeDownloaderDroppedBuildsArgsAndAddsToDropBar(t *testing.T) {
	dir := t.TempDir()
	withFakeYtdlpLookPath(t, true)
	resultPath := filepath.Join(dir, "My Video.mp4")
	calls := withFakeYtdlpCmd(t, resultPath)
	drop := &recDropBar{}

	inv := actions.Invocation{
		Target: model.Target{Options: map[string]string{
			"folder": dir,
		}},
		Payload:    model.Payload{Kind: model.ItemURL, Text: "https://youtube.com/watch?v=abc"},
		Services:   &recServices{},
		AddDropBar: drop.add,
	}

	res, err := (YouTubeDownloader{}).Dropped(context.Background(), inv)
	if err != nil {
		t.Fatalf("Dropped: %v", err)
	}

	if len(*calls) != 1 {
		t.Fatalf("expected 1 ytdlpCmd call, got %d", len(*calls))
	}
	args := (*calls)[0]
	wantOutputTpl := filepath.Join(dir, "%(title)s.%(ext)s")
	wantArgs := []string{"-o", wantOutputTpl, "--print", "after_move:filepath", "https://youtube.com/watch?v=abc"}
	if len(args) != len(wantArgs) {
		t.Fatalf("args = %v, want %v", args, wantArgs)
	}
	for i, a := range args {
		if a != wantArgs[i] {
			t.Errorf("args[%d] = %q, want %q (full %v)", i, a, wantArgs[i], args)
		}
	}

	if len(drop.calls) != 1 || len(drop.calls[0]) != 1 || drop.calls[0][0] != resultPath {
		t.Errorf("AddDropBar calls = %v, want [[%s]]", drop.calls, resultPath)
	}
	wantMsg := fmt.Sprintf("Downloaded %s", filepath.Base(resultPath))
	if res.Message != wantMsg {
		t.Errorf("Message = %q, want %q", res.Message, wantMsg)
	}
}

func TestYouTubeDownloaderAudioFormatAddsFlags(t *testing.T) {
	dir := t.TempDir()
	withFakeYtdlpLookPath(t, true)
	resultPath := filepath.Join(dir, "song.mp3")
	calls := withFakeYtdlpCmd(t, resultPath)

	inv := actions.Invocation{
		Target: model.Target{Options: map[string]string{
			"folder": dir,
			"format": "audio",
		}},
		Payload:  model.Payload{Kind: model.ItemURL, Text: "https://youtube.com/watch?v=xyz"},
		Services: &recServices{},
	}

	if _, err := (YouTubeDownloader{}).Dropped(context.Background(), inv); err != nil {
		t.Fatalf("Dropped: %v", err)
	}

	args := (*calls)[0]
	wantArgs := []string{
		"-o", filepath.Join(dir, "%(title)s.%(ext)s"),
		"--print", "after_move:filepath",
		"-x", "--audio-format", "mp3",
		"https://youtube.com/watch?v=xyz",
	}
	if len(args) != len(wantArgs) {
		t.Fatalf("args = %v, want %v", args, wantArgs)
	}
	for i, a := range args {
		if a != wantArgs[i] {
			t.Errorf("args[%d] = %q, want %q (full %v)", i, a, wantArgs[i], args)
		}
	}
}

func TestYouTubeDownloaderClickedReadsClipboard(t *testing.T) {
	dir := t.TempDir()
	withFakeYtdlpLookPath(t, true)
	resultPath := filepath.Join(dir, "clip.mp4")
	withFakeYtdlpCmd(t, resultPath)

	svc := &recServices{ReadClip: "https://youtube.com/watch?v=clip"}
	inv := actions.Invocation{
		Target:   model.Target{Options: map[string]string{"folder": dir}},
		Services: svc,
	}

	if _, err := (YouTubeDownloader{}).Clicked(context.Background(), inv); err != nil {
		t.Fatalf("Clicked: %v", err)
	}
}

func TestYouTubeDownloaderMissingBinaryErrorsFriendly(t *testing.T) {
	withFakeYtdlpLookPath(t, false)
	inv := actions.Invocation{
		Payload:  model.Payload{Kind: model.ItemURL, Text: "https://youtube.com/watch?v=abc"},
		Services: &recServices{},
	}
	_, err := (YouTubeDownloader{}).Dropped(context.Background(), inv)
	if err == nil {
		t.Fatal("expected error when yt-dlp is missing")
	}
	if err.Error() != "yt-dlp not found — install it with: brew install yt-dlp" {
		t.Errorf("error = %q", err.Error())
	}
}

func TestYouTubeDownloaderEmptyURLErrors(t *testing.T) {
	withFakeYtdlpLookPath(t, true)
	inv := actions.Invocation{
		Payload:  model.Payload{Kind: model.ItemURL, Text: "   "},
		Services: &recServices{},
	}
	if _, err := (YouTubeDownloader{}).Dropped(context.Background(), inv); err == nil {
		t.Error("empty URL should error")
	}
}
