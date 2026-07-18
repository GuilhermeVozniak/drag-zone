package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// shortioAPIURL is the link-creation endpoint; a package var so tests can
// point it at an httptest server.
var shortioAPIURL = "https://api.short.io/links"

// ShortIO shortens a dropped URL (or, on click, the clipboard URL) via
// Short.io and copies the short link to the clipboard.
type ShortIO struct{}

func (ShortIO) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "short-io",
		Name:        "Short.io",
		Description: "Shorten a dropped URL with Short.io and copy it. Click to shorten the clipboard URL.",
		Icon:        "link",
		Category:    "Uploads",
		Events:      []string{model.EventDragged, model.EventClicked},
		Accepts:     []model.ItemKind{model.ItemURL, model.ItemText},
		Options: []model.OptionField{
			{Key: "api_key", Label: "Short.io API Key", Type: "text", Required: true},
			{Key: "domain", Label: "Short.io Domain", Type: "text", Required: true, Placeholder: "short.domain"},
		},
	}
}

func (ShortIO) Dropped(ctx context.Context, inv actions.Invocation) (actions.Result, error) {
	return shortioShortenAndCopy(ctx, inv, inv.Payload.Text)
}

func (ShortIO) Clicked(ctx context.Context, inv actions.Invocation) (actions.Result, error) {
	text, err := inv.Services.ReadClipboard()
	if err != nil {
		return actions.Result{}, fmt.Errorf("reading clipboard: %w", err)
	}
	return shortioShortenAndCopy(ctx, inv, text)
}

// shortioShortenAndCopy validates raw as an http(s) URL, shortens it via
// Short.io, and copies the short link to the clipboard.
func shortioShortenAndCopy(ctx context.Context, inv actions.Invocation, raw string) (actions.Result, error) {
	longURL, err := parseHTTPURL(raw)
	if err != nil {
		return actions.Result{}, err
	}
	apiKey := inv.Target.Option("api_key", "")
	if apiKey == "" {
		return actions.Result{}, fmt.Errorf("no Short.io API key configured")
	}
	domain := inv.Target.Option("domain", "")
	if domain == "" {
		return actions.Result{}, fmt.Errorf("no Short.io domain configured")
	}
	inv.Progress.Detail(longURL)

	shortURL, err := shortenWithShortIO(ctx, apiKey, domain, longURL)
	if err != nil {
		return actions.Result{}, fmt.Errorf("shortening %s: %w", longURL, err)
	}
	if err := inv.Services.CopyToClipboard(shortURL); err != nil {
		return actions.Result{}, fmt.Errorf("copying URL to clipboard: %w", err)
	}
	return actions.Result{Message: "Shortened URL copied", URL: shortURL}, nil
}

// shortenWithShortIO asks the Short.io API to create a short link for
// longURL under domain.
func shortenWithShortIO(ctx context.Context, apiKey, domain, longURL string) (string, error) {
	reqBody, err := json.Marshal(struct {
		Domain      string `json:"domain"`
		OriginalURL string `json:"originalURL"`
	}{Domain: domain, OriginalURL: longURL})
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, shortioAPIURL, bytes.NewReader(reqBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return "", fmt.Errorf("reading short.io response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("short.io returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var out struct {
		ShortURL string `json:"shortURL"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("parsing short.io response: %w", err)
	}
	if out.ShortURL == "" {
		return "", fmt.Errorf("short.io response contains no shortURL")
	}
	return out.ShortURL, nil
}
