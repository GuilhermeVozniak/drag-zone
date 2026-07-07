package builtin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

func TestParseHTTPURL(t *testing.T) {
	if _, err := parseHTTPURL("  https://a.com/x  "); err != nil {
		t.Errorf("valid url errored: %v", err)
	}
	for _, bad := range []string{"", "ftp://a.com", "notaurl", "http://"} {
		if _, err := parseHTTPURL(bad); err == nil {
			t.Errorf("parseHTTPURL(%q) should error", bad)
		}
	}
}

func TestShortenRoundTripAndClickReadsClipboard(t *testing.T) {
	var gotURL string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.Query().Get("url")
		w.Write([]byte("https://tinyurl.com/abc"))
	}))
	defer ts.Close()
	old := tinyURLAPI
	tinyURLAPI = ts.URL
	defer func() { tinyURLAPI = old }()

	// Dropped path
	svc := &recServices{}
	res, err := ShortenURL{}.Dropped(context.Background(), actions.Invocation{
		Payload:  model.Payload{Kind: model.ItemURL, Text: "https://example.com/very/long"},
		Progress: nullProgress{}, Services: svc,
	})
	if err != nil || res.URL != "https://tinyurl.com/abc" || svc.Clipboard != res.URL {
		t.Fatalf("dropped: res=%+v clip=%q err=%v", res, svc.Clipboard, err)
	}
	if gotURL != "https://example.com/very/long" {
		t.Errorf("forwarded url = %q", gotURL)
	}

	// Clicked path reads the clipboard
	svc2 := &recServices{ReadClip: "https://example.com/from-clip"}
	if _, err := (ShortenURL{}).Clicked(context.Background(), actions.Invocation{
		Progress: nullProgress{}, Services: svc2,
	}); err != nil {
		t.Fatalf("clicked: %v", err)
	}
	if gotURL != "https://example.com/from-clip" {
		t.Errorf("clicked forwarded url = %q", gotURL)
	}
}

func TestShortenServerErrorPropagates(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer ts.Close()
	old := tinyURLAPI
	tinyURLAPI = ts.URL
	defer func() { tinyURLAPI = old }()
	if _, err := (ShortenURL{}).Dropped(context.Background(), actions.Invocation{
		Payload:  model.Payload{Kind: model.ItemURL, Text: "https://a.com"},
		Progress: nullProgress{}, Services: &recServices{},
	}); err == nil {
		t.Error("500 from tinyurl should error")
	}
}
