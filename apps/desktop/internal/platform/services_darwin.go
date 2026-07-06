// Package platform provides macOS host integrations. This file implements the
// exec-based services; deeper integrations (status item, drag-out, AirDrop)
// live in the cgo bridge.
package platform

import (
	"fmt"
	"os/exec"
	"strings"
)

// Services implements actions.Services on macOS.
type Services struct{}

// CopyToClipboard places text on the general pasteboard.
func (Services) CopyToClipboard(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

// ReadClipboard returns the general pasteboard's text contents.
func (Services) ReadClipboard() (string, error) {
	out, err := exec.Command("pbpaste").Output()
	return string(out), err
}

// Notify shows a user notification.
func (Services) Notify(title, body string) {
	script := fmt.Sprintf("display notification %s with title %s", appleString(body), appleString(title))
	exec.Command("osascript", "-e", script).Run()
}

// PlaySound plays a named system sound asynchronously.
func (Services) PlaySound(name string) {
	exec.Command("afplay", "/System/Library/Sounds/"+name+".aiff").Start()
}

// OpenURL opens a URL in the default browser.
func (Services) OpenURL(url string) error {
	return exec.Command("open", url).Run()
}

// OpenPath opens a file or folder with its default application.
func (Services) OpenPath(path string) error {
	return exec.Command("open", path).Run()
}

// Reveal shows the path in a Finder window.
func (Services) Reveal(path string) error {
	return exec.Command("open", "-R", path).Run()
}

// Trash moves paths to the Finder trash (recoverable, unlike os.Remove).
func (Services) Trash(paths []string) error {
	if len(paths) == 0 {
		return nil
	}
	items := make([]string, len(paths))
	for i, p := range paths {
		items[i] = "POSIX file " + appleString(p)
	}
	script := fmt.Sprintf("tell application \"Finder\" to delete {%s}", strings.Join(items, ", "))
	out, err := exec.Command("osascript", "-e", script).CombinedOutput()
	if err != nil {
		return fmt.Errorf("moving to trash: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// AirDrop shares files via AirDrop through the native sharing service.
func (Services) AirDrop(paths []string) error {
	return airDrop(paths)
}

// appleString quotes s as an AppleScript string literal.
func appleString(s string) string {
	return "\"" + strings.NewReplacer("\\", "\\\\", "\"", "\\\"").Replace(s) + "\""
}
