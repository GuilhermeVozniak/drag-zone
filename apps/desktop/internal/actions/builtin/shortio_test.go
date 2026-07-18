package builtin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

func TestShortIORoundTripAndClickReadsClipboard(t *testing.T) {
	var gotAuth, gotContentType string
	var gotBody struct {
		Domain      string `json:"domain"`
		OriginalURL string `json:"originalURL"`
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Write([]byte(`{"shortURL":"https://short.domain/abc"}`))
	}))
	defer ts.Close()
	old := shortioAPIURL
	shortioAPIURL = ts.URL
	defer func() { shortioAPIURL = old }()

	// Dropped path
	svc := &recServices{}
	res, err := ShortIO{}.Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"api_key": "KEY", "domain": "short.domain"}},
		Payload:  model.Payload{Kind: model.ItemURL, Text: "https://example.com/very/long"},
		Progress: nullProgress{}, Services: svc,
	})
	if err != nil || res.URL != "https://short.domain/abc" || svc.Clipboard != res.URL {
		t.Fatalf("dropped: res=%+v clip=%q err=%v", res, svc.Clipboard, err)
	}
	if gotAuth != "KEY" {
		t.Errorf("auth header = %q", gotAuth)
	}
	if gotContentType != "application/json" {
		t.Errorf("content-type = %q", gotContentType)
	}
	if gotBody.Domain != "short.domain" || gotBody.OriginalURL != "https://example.com/very/long" {
		t.Errorf("body = %+v", gotBody)
	}

	// Clicked path reads the clipboard
	svc2 := &recServices{ReadClip: "https://example.com/from-clip"}
	if _, err := (ShortIO{}).Clicked(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"api_key": "KEY", "domain": "short.domain"}},
		Progress: nullProgress{}, Services: svc2,
	}); err != nil {
		t.Fatalf("clicked: %v", err)
	}
	if gotBody.OriginalURL != "https://example.com/from-clip" {
		t.Errorf("clicked forwarded url = %q", gotBody.OriginalURL)
	}
}

func TestShortIOServerErrorPropagates(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer ts.Close()
	old := shortioAPIURL
	shortioAPIURL = ts.URL
	defer func() { shortioAPIURL = old }()
	if _, err := (ShortIO{}).Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"api_key": "KEY", "domain": "short.domain"}},
		Payload:  model.Payload{Kind: model.ItemURL, Text: "https://a.com"},
		Progress: nullProgress{}, Services: &recServices{},
	}); err == nil {
		t.Error("500 from short.io should error")
	}
}

func TestShortIOMissingOptionsAndEmptyInputErrors(t *testing.T) {
	svc := &recServices{}
	// missing api_key
	if _, err := (ShortIO{}).Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"domain": "short.domain"}},
		Payload:  model.Payload{Kind: model.ItemURL, Text: "https://a.com"},
		Progress: nullProgress{}, Services: svc,
	}); err == nil {
		t.Error("missing api_key should error")
	}
	// missing domain
	if _, err := (ShortIO{}).Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"api_key": "KEY"}},
		Payload:  model.Payload{Kind: model.ItemURL, Text: "https://a.com"},
		Progress: nullProgress{}, Services: svc,
	}); err == nil {
		t.Error("missing domain should error")
	}
	// empty input
	if _, err := (ShortIO{}).Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"api_key": "KEY", "domain": "short.domain"}},
		Payload:  model.Payload{Kind: model.ItemText, Text: "   "},
		Progress: nullProgress{}, Services: svc,
	}); err == nil {
		t.Error("empty payload text should error")
	}
	if svc.Clipboard != "" {
		t.Errorf("clipboard should be untouched, got %q", svc.Clipboard)
	}
}

func TestShortIOSpec(t *testing.T) {
	spec := ShortIO{}.Spec()
	if spec.ID != "short-io" {
		t.Errorf("ID = %q", spec.ID)
	}
	if len(spec.Events) != 2 || spec.Events[0] != model.EventDragged || spec.Events[1] != model.EventClicked {
		t.Errorf("Events = %v", spec.Events)
	}
	if len(spec.Options) != 2 {
		t.Errorf("Options = %+v", spec.Options)
	}
}
