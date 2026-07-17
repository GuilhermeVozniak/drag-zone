package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVersionNewer(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"0.3.6", "0.2.0", true},
		{"0.2.0", "0.3.6", false},
		{"0.3.6", "0.3.6", false},
		{"0.10.0", "0.9.9", true}, // numeric, not lexicographic
		{"0.9.9", "0.10.0", false},
		{"1.0.0", "0.99.99", true},
		{"v0.4.0", "0.3.6", true},      // leading v ignored
		{"0.4.0-beta", "0.4.0", false}, // suffix ignored, equal
		{"0.4", "0.3.9", true},         // missing patch = 0
		{"", "0.1.0", false},           // unparseable never wins
	}
	for _, c := range cases {
		if got := versionNewer(c.a, c.b); got != c.want {
			t.Errorf("versionNewer(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

// TestCheckForUpdates guards the releases/latest parsing and the
// newer-version decision against a canned GitHub API response.
func TestCheckForUpdates(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"tag_name": "v9.9.9",
			"html_url": "https://example.com/releases/tag/v9.9.9",
			"published_at": "2026-07-06T21:36:49Z",
			"assets": [
				{"name": "checksums.txt", "browser_download_url": "https://example.com/checksums.txt"},
				{"name": "DragZone-9.9.9.dmg", "browser_download_url": "https://example.com/DragZone-9.9.9.dmg"}
			]
		}`))
	}))
	defer ts.Close()

	app := newTestApp(t)
	info, err := app.checkForUpdates(ts.URL)
	if err != nil {
		t.Fatalf("checkForUpdates: %v", err)
	}
	if info.Latest != "9.9.9" {
		t.Errorf("Latest = %q, want 9.9.9", info.Latest)
	}
	if !info.Available {
		t.Errorf("Available = false for %s -> 9.9.9", info.Version)
	}
	if info.DownloadURL != "https://example.com/DragZone-9.9.9.dmg" {
		t.Errorf("DownloadURL = %q, want the .dmg asset", info.DownloadURL)
	}
	if info.URL != "https://example.com/releases/tag/v9.9.9" {
		t.Errorf("URL = %q", info.URL)
	}
}

// TestCheckForUpdates404IsUpToDate covers repos with no published releases:
// GitHub's releases/latest returns 404, which must be reported as up to date,
// not as an error.
func TestCheckForUpdates404IsUpToDate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message":"Not Found"}`, http.StatusNotFound)
	}))
	defer ts.Close()
	app := newTestApp(t)
	info, err := app.checkForUpdates(ts.URL)
	if err != nil {
		t.Fatalf("404 should not be an error, got %v", err)
	}
	if info.Available {
		t.Error("404 (no releases) must report no update available")
	}
}

// TestCheckForUpdatesUpToDate covers the no-update path: same version.
func TestCheckForUpdatesUpToDate(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name": "v` + appVersion + `", "assets": []}`))
	}))
	defer ts.Close()

	app := newTestApp(t)
	info, err := app.checkForUpdates(ts.URL)
	if err != nil {
		t.Fatalf("checkForUpdates: %v", err)
	}
	if info.Available {
		t.Errorf("Available = true for equal versions (%s)", appVersion)
	}
}
