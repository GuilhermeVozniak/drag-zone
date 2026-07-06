// Package bundles loads and runs Dropzone-style scriptable action bundles
// (.dzbundle directories containing action.rb or action.py plus icon.png).
//
// Metadata lives in a comment header at the top of the script, exactly like
// Dropzone 4:
//
//	# Dropzone Action Info
//	# Name: My Action
//	# Description: Does something
//	# Handles: Files
//	# Events: Dragged, Clicked
//	# OptionsNIB: Login
//	# SkipConfig: No
//	# UniqueID: 1234567890
package bundles

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"dragzone/internal/model"
)

// Meta is the parsed action header.
type Meta struct {
	Name                       string
	Description                string
	Handles                    []string // Files, Text
	Events                     []string // Dragged, Clicked (defaults to both)
	OptionsNIB                 string
	OptionsTitle               string
	SkipConfig                 bool
	KeyModifiers               []string
	UniqueID                   string
	Version                    string
	Creator                    string
	URL                        string
	RunsSandboxed              bool
	MinDropzoneVersion         string
	UseSelectedItemNameAndIcon bool
}

// ParseMeta reads the "# Dropzone Action Info" header from a script file.
func ParseMeta(scriptPath string) (Meta, error) {
	f, err := os.Open(scriptPath)
	if err != nil {
		return Meta{}, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	var meta Meta
	seenHeader := false
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if !strings.HasPrefix(line, "#") {
			if seenHeader {
				break
			}
			continue
		}
		content := strings.TrimSpace(strings.TrimPrefix(line, "#"))
		if content == "Dropzone Action Info" {
			seenHeader = true
			continue
		}
		if !seenHeader {
			continue
		}
		key, value, ok := strings.Cut(content, ":")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		switch strings.TrimSpace(key) {
		case "Name":
			meta.Name = value
		case "Description":
			meta.Description = value
		case "Handles":
			meta.Handles = splitList(value)
		case "Events":
			meta.Events = splitList(value)
		case "OptionsNIB":
			meta.OptionsNIB = value
		case "OptionsTitle":
			meta.OptionsTitle = value
		case "SkipConfig":
			meta.SkipConfig = isYes(value)
		case "KeyModifiers":
			meta.KeyModifiers = splitList(value)
		case "UniqueID":
			meta.UniqueID = value
		case "Version":
			meta.Version = value
		case "Creator":
			meta.Creator = value
		case "URL":
			meta.URL = value
		case "RunsSandboxed":
			meta.RunsSandboxed = isYes(value)
		case "MinDropzoneVersion":
			meta.MinDropzoneVersion = value
		case "UseSelectedItemNameAndIcon":
			meta.UseSelectedItemNameAndIcon = isYes(value)
		}
	}
	if err := sc.Err(); err != nil {
		return Meta{}, err
	}
	if !seenHeader {
		return Meta{}, fmt.Errorf("%s: missing '# Dropzone Action Info' header", scriptPath)
	}
	if meta.Name == "" {
		return Meta{}, fmt.Errorf("%s: header has no Name", scriptPath)
	}
	if len(meta.Events) == 0 {
		meta.Events = []string{"Dragged", "Clicked"}
	}
	if len(meta.Handles) == 0 {
		meta.Handles = []string{"Files"}
	}
	return meta, nil
}

func splitList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func isYes(s string) bool {
	return strings.EqualFold(strings.TrimSpace(s), "yes") || strings.EqualFold(strings.TrimSpace(s), "true")
}

// optionFields maps an OptionsNIB name to the form fields it collects,
// mirroring Dropzone's pre-built configuration panels.
func optionFields(nib string) []model.OptionField {
	switch nib {
	case "Login":
		return []model.OptionField{
			{Key: "username", Label: "Username", Type: "text", Required: true},
			{Key: "password", Label: "Password", Type: "password", Required: true},
		}
	case "ExtendedLogin":
		return []model.OptionField{
			{Key: "server", Label: "Server", Type: "text", Required: true},
			{Key: "port", Label: "Port", Type: "text"},
			{Key: "username", Label: "Username", Type: "text", Required: true},
			{Key: "password", Label: "Password", Type: "password", Required: true},
			{Key: "remote_path", Label: "Remote path", Type: "text"},
		}
	case "APIKey":
		return []model.OptionField{
			{Key: "api_key", Label: "API Key", Type: "password", Required: true},
		}
	case "UsernameAPIKey":
		return []model.OptionField{
			{Key: "username", Label: "Username", Type: "text", Required: true},
			{Key: "api_key", Label: "API Key", Type: "password", Required: true},
		}
	case "ChooseFolder":
		return []model.OptionField{
			{Key: "path", Label: "Folder", Type: "folder", Required: true},
		}
	case "ChooseApplication":
		return []model.OptionField{
			{Key: "app", Label: "Application", Type: "app", Required: true},
		}
	default:
		return nil
	}
}
