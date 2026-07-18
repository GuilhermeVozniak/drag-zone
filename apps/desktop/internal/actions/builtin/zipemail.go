package builtin

import (
	"context"
	"fmt"
	"os/exec"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// mailCmd is a seam so tests stub email composition (osascript would
// otherwise open a real, visible Mail.app compose window).
var mailCmd = func(ctx context.Context, script string) *exec.Cmd {
	return exec.CommandContext(ctx, "osascript", "-e", script)
}

// ZipEmail compresses dropped files into a single archive and attaches it to
// a new Mail.app outgoing message, Dropzone-4 style.
type ZipEmail struct{}

func (ZipEmail) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "zip-email",
		Name:        "Zip & Email",
		Description: "Compress dropped files and attach them to a new email.",
		Icon:        "mail",
		Category:    "File Management",
		Events:      []string{model.EventDragged},
		Accepts:     []model.ItemKind{model.ItemFiles},
		Multi:       false,
	}
}

func (ZipEmail) Dropped(ctx context.Context, inv actions.Invocation) (actions.Result, error) {
	if len(inv.Payload.Paths) == 0 {
		return actions.Result{}, fmt.Errorf("nothing to zip and email")
	}

	zipPath, _, err := zipForUpload(inv.Payload.Paths)
	if err != nil {
		return actions.Result{}, fmt.Errorf("zipping before email: %w", err)
	}
	// The zip stays on disk in its fresh temp dir rather than being removed
	// here: Mail.app reads the attachment asynchronously after this function
	// returns, so an eager os.RemoveAll(tmpDir) would race the compose
	// window and can leave the attachment missing.

	script := fmt.Sprintf(`tell application "Mail"
	set msg to make new outgoing message with properties {subject:"Files", visible:true}
	tell content of msg to make new attachment with properties {file name:(POSIX file %s)} at after last paragraph
	activate
end tell`, appleScriptString(zipPath))

	if err := mailCmd(ctx, script).Run(); err != nil {
		return actions.Result{}, fmt.Errorf("composing email: %w", err)
	}

	return actions.Result{Message: fmt.Sprintf("Zipped %d file(s) into an email", len(inv.Payload.Paths))}, nil
}
