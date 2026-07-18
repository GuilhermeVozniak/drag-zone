package builtin

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

func TestImgBBUploadRoundTrip(t *testing.T) {
	var gotKey, gotImage string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		gotKey = r.FormValue("key")
		gotImage = r.FormValue("image")
		w.Write([]byte(`{"data":{"url":"https://i.ibb.co/abc/pic.png"},"success":true}`))
	}))
	defer ts.Close()
	old := imgbbAPIURL
	imgbbAPIURL = ts.URL
	defer func() { imgbbAPIURL = old }()

	img := filepath.Join(t.TempDir(), "pic.png")
	content := []byte("\x89PNG\r\n\x1a\nfake")
	if err := os.WriteFile(img, content, 0o644); err != nil {
		t.Fatal(err)
	}
	svc := &recServices{}
	res, err := ImgBBUpload{}.Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"api_key": "KEY"}},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{img}},
		Progress: nullProgress{},
		Services: svc,
	})
	if err != nil {
		t.Fatalf("Dropped: %v", err)
	}
	if res.URL != "https://i.ibb.co/abc/pic.png" || svc.Clipboard != res.URL {
		t.Errorf("url=%q clip=%q", res.URL, svc.Clipboard)
	}
	if gotKey != "KEY" {
		t.Errorf("key = %q", gotKey)
	}
	wantImage := base64.StdEncoding.EncodeToString(content)
	if gotImage != wantImage {
		t.Errorf("image = %q, want %q", gotImage, wantImage)
	}
}

func TestImgBBServerErrorPropagates(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer ts.Close()
	old := imgbbAPIURL
	imgbbAPIURL = ts.URL
	defer func() { imgbbAPIURL = old }()

	img := filepath.Join(t.TempDir(), "pic.png")
	os.WriteFile(img, []byte("data"), 0o644)
	if _, err := (ImgBBUpload{}).Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"api_key": "KEY"}},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{img}},
		Progress: nullProgress{}, Services: &recServices{},
	}); err == nil {
		t.Error("500 from imgbb should error")
	}
}

func TestImgBBMissingAPIKeyAndEmptyInputErrors(t *testing.T) {
	svc := &recServices{}
	// missing api key
	img := filepath.Join(t.TempDir(), "pic.png")
	os.WriteFile(img, []byte("data"), 0o644)
	if _, err := (ImgBBUpload{}).Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{img}},
		Progress: nullProgress{}, Services: svc,
	}); err == nil {
		t.Error("missing api_key should error")
	}
	// empty input
	if _, err := (ImgBBUpload{}).Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"api_key": "KEY"}},
		Payload:  model.Payload{Kind: model.ItemFiles},
		Progress: nullProgress{}, Services: svc,
	}); err == nil {
		t.Error("empty payload should error")
	}
	// non-image file
	txt := filepath.Join(t.TempDir(), "a.txt")
	os.WriteFile(txt, []byte("x"), 0o644)
	if _, err := (ImgBBUpload{}).Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"api_key": "KEY"}},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{txt}},
		Progress: nullProgress{}, Services: svc,
	}); err == nil {
		t.Error("non-image should error")
	}
	if svc.Clipboard != "" {
		t.Errorf("clipboard should be untouched, got %q", svc.Clipboard)
	}
}

func TestImgBBSpec(t *testing.T) {
	spec := ImgBBUpload{}.Spec()
	if spec.ID != "imgbb" {
		t.Errorf("ID = %q", spec.ID)
	}
	if len(spec.Options) != 1 || !spec.Options[0].Required || spec.Options[0].Key != "api_key" {
		t.Errorf("Options = %+v", spec.Options)
	}
}
