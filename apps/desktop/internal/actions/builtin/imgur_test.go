package builtin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

func TestIsImageFile(t *testing.T) {
	for _, ok := range []string{"a.jpg", "a.JPEG", "b.png", "c.gif", "d.webp", "e.heic", "f.bmp", "g.tiff"} {
		if !isImageFile(ok) {
			t.Errorf("isImageFile(%q) = false", ok)
		}
	}
	for _, no := range []string{"a.txt", "b.pdf", "noext"} {
		if isImageFile(no) {
			t.Errorf("isImageFile(%q) = true", no)
		}
	}
}

func TestImgurUploadRoundTrip(t *testing.T) {
	var gotAuth, gotContentType string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		_ = r.ParseMultipartForm(1 << 20)
		w.Write([]byte(`{"data":{"link":"https://i.imgur.com/abc.png"}}`))
	}))
	defer ts.Close()
	old := imgurAPIURL
	imgurAPIURL = ts.URL
	defer func() { imgurAPIURL = old }()

	img := filepath.Join(t.TempDir(), "pic.png")
	if err := os.WriteFile(img, []byte("\x89PNG\r\n\x1a\nfake"), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := &recServices{}
	res, err := ImgurUpload{}.Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"client_id": "CID"}},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{img}},
		Progress: nullProgress{},
		Services: svc,
	})
	if err != nil {
		t.Fatalf("Dropped: %v", err)
	}
	if res.URL != "https://i.imgur.com/abc.png" || svc.Clipboard != res.URL {
		t.Errorf("url=%q clip=%q", res.URL, svc.Clipboard)
	}
	if gotAuth != "Client-ID CID" {
		t.Errorf("auth header = %q", gotAuth)
	}
	if len(gotContentType) < 9 || gotContentType[:9] != "multipart" {
		t.Errorf("content-type = %q", gotContentType)
	}
}

func TestImgurRejectsNonImageAndMissingID(t *testing.T) {
	svc := &recServices{}
	// missing client id
	if _, err := (ImgurUpload{}).Dropped(context.Background(), actions.Invocation{
		Target: model.Target{}, Payload: model.Payload{Paths: []string{"/a.png"}},
		Progress: nullProgress{}, Services: svc,
	}); err == nil {
		t.Error("missing client_id should error")
	}
	// non-image file
	txt := filepath.Join(t.TempDir(), "a.txt")
	os.WriteFile(txt, []byte("x"), 0o644)
	if _, err := (ImgurUpload{}).Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"client_id": "C"}},
		Payload:  model.Payload{Paths: []string{txt}},
		Progress: nullProgress{}, Services: svc,
	}); err == nil {
		t.Error("non-image should error")
	}
}
