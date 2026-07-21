package bundles

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// sandboxProfile is the seatbelt profile for actions declaring
// RunsSandboxed: general operation is allowed, but the network is off-limits
// and writes are confined to the system temp area (matching Dropzone's
// sandbox contract: no network, writes only to the designated temp dirs).
const sandboxProfile = `(version 1)
(deny default)
(allow process*)
(allow signal)
(allow sysctl-read)
(allow mach-lookup)
(allow ipc-posix-shm)
(allow file-read*)
(allow file-write* (subpath %s) (literal "/dev/null"))
`

// sandboxedCommand wraps the interpreter invocation in sandbox-exec. ok is
// false when sandbox-exec is unavailable (the caller falls back to running
// unsandboxed with a console warning).
func sandboxedCommand(ctx context.Context, interpreter string, args []string) (cmd *exec.Cmd, ok bool) {
	if _, err := exec.LookPath("sandbox-exec"); err != nil {
		return nil, false
	}
	profile := fmt.Sprintf(sandboxProfile, strconv.Quote(os.TempDir()))
	full := append([]string{"-p", profile, interpreter}, args...)
	return exec.CommandContext(ctx, "sandbox-exec", full...), true
}

// finderSelection returns the paths of the current Finder selection, for
// actions declaring UseSelectedItemNameAndIcon that run without a drop.
// Empty when Finder has no selection or isn't running.
func finderSelection(ctx context.Context) []string {
	const script = `tell application "Finder"
	try
		set sel to selection as alias list
	on error
		return ""
	end try
	set out to ""
	repeat with f in sel
		set out to out & POSIX path of f & linefeed
	end repeat
	return out
end tell`
	out, err := exec.CommandContext(ctx, "osascript", "-e", script).Output()
	if err != nil {
		return nil
	}
	var paths []string
	for line := range strings.Lines(string(out)) {
		if p := strings.TrimSpace(line); p != "" {
			paths = append(paths, p)
		}
	}
	return paths
}
