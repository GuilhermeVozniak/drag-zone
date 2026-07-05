package bundles

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
)

const rubyTemplate = `# Dropzone Action Info
# Name: %s
# Description: Describe what your action does
# Handles: Files
# Creator: You
# URL: https://example.com
# Events: Dragged, Clicked
# SkipConfig: Yes
# RunsSandboxed: No
# Version: 1.0
# UniqueID: %s

def dragged
  $dz.begin("Processing #{$items.length} item(s)...")
  $dz.determinate(true)
  $dz.percent(50)
  # Your code here. $items holds the dropped file paths.
  $dz.finish("Done")
  $dz.url(false)
end

def clicked
  $dz.finish("Clicked!")
  $dz.url(false)
end
`

const pythonTemplate = `# Dropzone Action Info
# Name: %s
# Description: Describe what your action does
# Handles: Files
# Creator: You
# URL: https://example.com
# Events: Dragged, Clicked
# SkipConfig: Yes
# RunsSandboxed: No
# Version: 1.0
# UniqueID: %s

def dragged():
    dz.begin("Processing %%d item(s)..." %% len(items))
    dz.determinate(True)
    dz.percent(50)
    # Your code here. items holds the dropped file paths.
    dz.finish("Done")
    dz.url(False)

def clicked():
    dz.finish("Clicked!")
    dz.url(False)
`

// CreateTemplate writes a new template .dzbundle into dir and returns its
// path. language is "ruby" or "python".
func CreateTemplate(dir, name, language string) (string, error) {
	if strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("action name is required")
	}
	bundle := filepath.Join(dir, name+".dzbundle")
	if _, err := os.Stat(bundle); err == nil {
		return "", fmt.Errorf("an action named %q already exists", name)
	}
	if err := os.MkdirAll(bundle, 0o755); err != nil {
		return "", err
	}

	uid := fmt.Sprintf("%d", 1000000000+rand.Intn(9000000000-1))
	script, tmpl := "action.rb", rubyTemplate
	if language == "python" {
		script, tmpl = "action.py", pythonTemplate
	}
	content := fmt.Sprintf(tmpl, name, uid)
	if err := os.WriteFile(filepath.Join(bundle, script), []byte(content), 0o644); err != nil {
		os.RemoveAll(bundle)
		return "", err
	}
	return bundle, nil
}
