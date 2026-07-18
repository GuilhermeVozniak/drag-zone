package builtin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

func TestImageSearchSpec(t *testing.T) {
	spec := ImageSearch{}.Spec()
	if spec.ID != "image-search" {
		t.Errorf("ID = %q", spec.ID)
	}
	if len(spec.Events) != 1 || spec.Events[0] != model.EventDragged {
		t.Errorf("Events = %v", spec.Events)
	}
	if len(spec.Accepts) != 1 || spec.Accepts[0] != model.ItemFiles {
		t.Errorf("Accepts = %v", spec.Accepts)
	}
	if len(spec.Options) != 1 || !spec.Options[0].Required || spec.Options[0].Key != "api_key" {
		t.Errorf("Options = %+v", spec.Options)
	}
}

func TestImageSearchUploadsAndOpensGoogleReverseSearch(t *testing.T) {
	const uploadedURL = "https://i.ibb.co/abc/pic.png"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.FormValue("key") != "KEY" {
			t.Errorf("key = %q", r.FormValue("key"))
		}
		w.Write([]byte(`{"data":{"url":"` + uploadedURL + `"}}`))
	}))
	defer ts.Close()
	old := imageSearchUploadURL
	imageSearchUploadURL = ts.URL
	defer func() { imageSearchUploadURL = old }()

	img := filepath.Join(t.TempDir(), "pic.png")
	if err := os.WriteFile(img, []byte("\x89PNG\r\n\x1a\nfake"), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := &recServices{}

	res, err := (ImageSearch{}).Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"api_key": "KEY"}},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{img}},
		Services: svc,
	})
	if err != nil {
		t.Fatalf("Dropped: %v", err)
	}
	if res.Message != "Opened image search" {
		t.Errorf("Message = %q", res.Message)
	}
	if len(svc.Opened) != 1 {
		t.Fatalf("Opened = %v", svc.Opened)
	}
	wantSuffix := "google.com/searchbyimage?image_url=" + url.QueryEscape(uploadedURL)
	if !strings.Contains(svc.Opened[0], wantSuffix) {
		t.Errorf("opened URL = %q, want to contain %q", svc.Opened[0], wantSuffix)
	}
}

func TestImageSearchMissingAPIKeyErrors(t *testing.T) {
	img := filepath.Join(t.TempDir(), "pic.png")
	os.WriteFile(img, []byte("data"), 0o644)
	svc := &recServices{}
	if _, err := (ImageSearch{}).Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{img}},
		Services: svc,
	}); err == nil {
		t.Error("missing api_key should error")
	}
	if len(svc.Opened) != 0 {
		t.Errorf("OpenURL should not be called, got %v", svc.Opened)
	}
}

func TestImageSearchNonImageErrors(t *testing.T) {
	txt := filepath.Join(t.TempDir(), "a.txt")
	os.WriteFile(txt, []byte("x"), 0o644)
	svc := &recServices{}
	if _, err := (ImageSearch{}).Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"api_key": "KEY"}},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{txt}},
		Services: svc,
	}); err == nil {
		t.Error("non-image should error")
	}
}

func TestImageSearchEmptyPayloadErrors(t *testing.T) {
	svc := &recServices{}
	if _, err := (ImageSearch{}).Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"api_key": "KEY"}},
		Payload:  model.Payload{Kind: model.ItemFiles},
		Services: svc,
	}); err == nil {
		t.Error("empty payload should error")
	}
}

func TestImageSearchUploadServerErrorPropagates(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer ts.Close()
	old := imageSearchUploadURL
	imageSearchUploadURL = ts.URL
	defer func() { imageSearchUploadURL = old }()

	img := filepath.Join(t.TempDir(), "pic.png")
	os.WriteFile(img, []byte("data"), 0o644)
	svc := &recServices{}
	if _, err := (ImageSearch{}).Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"api_key": "KEY"}},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{img}},
		Services: svc,
	}); err == nil {
		t.Error("500 from upload should error")
	}
	if len(svc.Opened) != 0 {
		t.Errorf("OpenURL should not be called on upload failure, got %v", svc.Opened)
	}
}
