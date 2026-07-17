package builtin

import (
	"context"
	"errors"
	"fmt"
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

func TestShortenURLSpec(t *testing.T) {
	spec := ShortenURL{}.Spec()
	if spec.ID != "shorten-url" {
		t.Errorf("ID = %q", spec.ID)
	}
	if len(spec.Events) != 2 || spec.Events[0] != model.EventDragged || spec.Events[1] != model.EventClicked {
		t.Errorf("Events = %v", spec.Events)
	}
	if len(spec.Accepts) != 2 || spec.Accepts[0] != model.ItemURL || spec.Accepts[1] != model.ItemText {
		t.Errorf("Accepts = %v", spec.Accepts)
	}
}

func TestShortenDroppedEmptyInputErrors(t *testing.T) {
	svc := &recServices{}
	if _, err := (ShortenURL{}).Dropped(context.Background(), actions.Invocation{
		Payload:  model.Payload{Kind: model.ItemText, Text: "   "},
		Progress: nullProgress{}, Services: svc,
	}); err == nil {
		t.Error("empty payload text should error")
	}
	if svc.Clipboard != "" {
		t.Errorf("clipboard should be untouched, got %q", svc.Clipboard)
	}
}

func TestShortenClickedEmptyClipboardErrors(t *testing.T) {
	svc := &recServices{ReadClip: ""}
	if _, err := (ShortenURL{}).Clicked(context.Background(), actions.Invocation{
		Progress: nullProgress{}, Services: svc,
	}); err == nil {
		t.Error("empty clipboard should error")
	}
}

func TestShortenDroppedMalformedURLErrors(t *testing.T) {
	// Fails net/url.Parse itself (invalid percent-escape), exercising the
	// url.Parse error branch of parseHTTPURL rather than the scheme/host checks.
	if _, err := (ShortenURL{}).Dropped(context.Background(), actions.Invocation{
		Payload:  model.Payload{Kind: model.ItemURL, Text: "http://a.com/%zz"},
		Progress: nullProgress{}, Services: &recServices{},
	}); err == nil {
		t.Error("malformed URL should error")
	}
}

func TestShortenAndCopyClipboardWriteErrorWrapped(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("https://tinyurl.com/abc"))
	}))
	defer ts.Close()
	old := tinyURLAPI
	tinyURLAPI = ts.URL
	defer func() { tinyURLAPI = old }()

	svc := &recServices{ClipboardErr: fmt.Errorf("clip locked")}
	_, err := (ShortenURL{}).Dropped(context.Background(), actions.Invocation{
		Payload:  model.Payload{Kind: model.ItemURL, Text: "https://example.com"},
		Progress: nullProgress{}, Services: svc,
	})
	if err == nil {
		t.Fatal("clipboard write failure should error")
	}
	if !errors.Is(err, svc.ClipboardErr) {
		t.Errorf("error should wrap clipboard err, got %v", err)
	}
}

func TestShortenWithTinyURLEmptyResponseErrors(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 200 OK with an empty body.
	}))
	defer ts.Close()
	old := tinyURLAPI
	tinyURLAPI = ts.URL
	defer func() { tinyURLAPI = old }()

	if _, err := (ShortenURL{}).Dropped(context.Background(), actions.Invocation{
		Payload:  model.Payload{Kind: model.ItemURL, Text: "https://example.com"},
		Progress: nullProgress{}, Services: &recServices{},
	}); err == nil {
		t.Error("empty tinyurl response should error")
	}
}

func TestShortenWithTinyURLRequestCreationErrorWrapped(t *testing.T) {
	// A control character in the endpoint makes http.NewRequestWithContext's
	// internal URL parse fail, exercising that error branch deterministically
	// (no network I/O at all).
	old := tinyURLAPI
	tinyURLAPI = "http://\x7f"
	defer func() { tinyURLAPI = old }()

	if _, err := (ShortenURL{}).Dropped(context.Background(), actions.Invocation{
		Payload:  model.Payload{Kind: model.ItemURL, Text: "https://example.com"},
		Progress: nullProgress{}, Services: &recServices{},
	}); err == nil {
		t.Error("malformed endpoint should error")
	}
}

func TestShortenWithTinyURLConnectionErrorWrapped(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	unreachable := ts.URL
	ts.Close() // closed before use: connection refused, no live network involved

	old := tinyURLAPI
	tinyURLAPI = unreachable
	defer func() { tinyURLAPI = old }()

	if _, err := (ShortenURL{}).Dropped(context.Background(), actions.Invocation{
		Payload:  model.Payload{Kind: model.ItemURL, Text: "https://example.com"},
		Progress: nullProgress{}, Services: &recServices{},
	}); err == nil {
		t.Error("connection failure should error")
	}
}
