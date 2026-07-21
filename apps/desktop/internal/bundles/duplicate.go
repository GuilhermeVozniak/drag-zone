package bundles

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"dragzone/internal/fsutil"
)

// CopyForEditing duplicates the bundle into destDir as a new, user-editable
// action and returns the new bundle and script paths. The copy gets a fresh
// UniqueID so it registers alongside the original instead of colliding with
// it (the registry keys bundle actions by UniqueID).
func (s *ScriptAction) CopyForEditing(destDir string) (bundlePath, scriptPath string, err error) {
	name := s.meta.Name
	if name == "" {
		name = "Action"
	}
	if strings.ContainsAny(name, "/\\\n\r") {
		return "", "", fmt.Errorf("action name %q is not a valid file name", name)
	}
	dst := fsutil.UniqueDest(destDir, name+".dzbundle")
	if _, err := fsutil.CopyPathAs(s.bundlePath, dst, nil); err != nil {
		return "", "", fmt.Errorf("copying bundle: %w", err)
	}
	newScript := filepath.Join(dst, filepath.Base(s.scriptPath))
	if err := rewriteUniqueID(newScript); err != nil {
		os.RemoveAll(dst)
		return "", "", err
	}
	return dst, newScript, nil
}

// rewriteUniqueID replaces the UniqueID header line with a fresh random one.
// Scripts without a UniqueID line derive it from the (already unique) bundle
// directory name, so they need no edit.
func rewriteUniqueID(scriptPath string) error {
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		return err
	}
	uid := fmt.Sprintf("%d", 1000000000+rand.Intn(9000000000-1))
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "#"))
		if strings.HasPrefix(trimmed, "UniqueID:") {
			prefix := line[:strings.Index(line, "#")+1]
			lines[i] = prefix + " UniqueID: " + uid
			return os.WriteFile(scriptPath, []byte(strings.Join(lines, "\n")), 0o644)
		}
	}
	return nil
}
