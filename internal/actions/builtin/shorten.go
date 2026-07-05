package builtin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// ShortenURL shortens a dropped URL (or, on click, the clipboard URL) via
// TinyURL and copies the short link to the clipboard.
type ShortenURL struct{}

func (ShortenURL) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "shorten-url",
		Name:        "Shorten URL",
		Description: "Shorten a dropped URL with TinyURL and copy it. Click to shorten the clipboard URL.",
		Icon:        "link",
		Category:    "Utilities",
		Events:      []string{model.EventDragged, model.EventClicked},
		Accepts:     []model.ItemKind{model.ItemURL, model.ItemText},
	}
}

func (ShortenURL) Dropped(ctx context.Context, inv actions.Invocation) (actions.Result, error) {
	return shortenAndCopy(ctx, inv, inv.Payload.Text)
}

func (ShortenURL) Clicked(ctx context.Context, inv actions.Invocation) (actions.Result, error) {
	text, err := inv.Services.ReadClipboard()
	if err != nil {
		return actions.Result{}, fmt.Errorf("reading clipboard: %w", err)
	}
	return shortenAndCopy(ctx, inv, text)
}

// shortenAndCopy validates raw as an http(s) URL, shortens it, and copies the
// short link to the clipboard.
func shortenAndCopy(ctx context.Context, inv actions.Invocation, raw string) (actions.Result, error) {
	longURL, err := parseHTTPURL(raw)
	if err != nil {
		return actions.Result{}, err
	}
	inv.Progress.Detail(longURL)

	shortURL, err := shortenWithTinyURL(ctx, longURL)
	if err != nil {
		return actions.Result{}, fmt.Errorf("shortening %s: %w", longURL, err)
	}
	if err := inv.Services.CopyToClipboard(shortURL); err != nil {
		return actions.Result{}, fmt.Errorf("copying URL to clipboard: %w", err)
	}
	return actions.Result{Message: "Shortened URL copied", URL: shortURL}, nil
}

// parseHTTPURL trims and validates raw as an absolute http or https URL.
func parseHTTPURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("no URL to shorten")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid URL %q: %w", raw, err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", fmt.Errorf("%q is not an http(s) URL", raw)
	}
	return raw, nil
}

// shortenWithTinyURL asks the TinyURL API for a short link.
func shortenWithTinyURL(ctx context.Context, longURL string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://tinyurl.com/api-create.php?url="+url.QueryEscape(longURL), nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return "", fmt.Errorf("reading tinyurl response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tinyurl returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	shortURL := strings.TrimSpace(string(body))
	if shortURL == "" {
		return "", fmt.Errorf("tinyurl returned an empty response")
	}
	return shortURL, nil
}
