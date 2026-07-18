package builtin

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

func TestTinifyRoundTrip(t *testing.T) {
	original := []byte("\x89PNG\r\n\x1a\noriginal-bytes-are-longer-than-compressed")
	compressed := []byte("tiny")

	var gotUser, gotPass string
	var gotUploadBody []byte
	var downloadPath string
	// tsURL is filled in once the server is listening; the /shrink handler
	// closes over it so the Location header it returns can point back at ts.
	var tsURL string
	mux := http.NewServeMux()
	mux.HandleFunc("/shrink", func(w http.ResponseWriter, r *http.Request) {
		u, p, _ := r.BasicAuth()
		gotUser, gotPass = u, p
		body, _ := io.ReadAll(r.Body)
		gotUploadBody = body
		w.Header().Set("Location", tsURL+"/output/compressed.png")
		w.WriteHeader(http.StatusCreated)
	})
	mux.HandleFunc("/output/compressed.png", func(w http.ResponseWriter, r *http.Request) {
		downloadPath = r.URL.Path
		w.Write(compressed)
	})
	ts := httptest.NewServer(mux)
	defer ts.Close()
	tsURL = ts.URL

	old := tinifyAPIURL
	tinifyAPIURL = ts.URL + "/shrink"
	defer func() { tinifyAPIURL = old }()

	dir := t.TempDir()
	img := filepath.Join(dir, "pic.png")
	if err := os.WriteFile(img, original, 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Tinify{}.Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"api_key": "KEY"}},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{img}},
		Progress: nullProgress{},
		Services: &recServices{},
	})
	if err != nil {
		t.Fatalf("Dropped: %v", err)
	}
	if gotUser != "api" || gotPass != "KEY" {
		t.Errorf("basic auth = %q/%q", gotUser, gotPass)
	}
	if string(gotUploadBody) != string(original) {
		t.Errorf("uploaded body = %q, want %q", gotUploadBody, original)
	}
	if downloadPath != "/output/compressed.png" {
		t.Errorf("download path = %q", downloadPath)
	}

	// Original must be untouched.
	origData, err := os.ReadFile(img)
	if err != nil || string(origData) != string(original) {
		t.Fatalf("original modified: %q, err %v", origData, err)
	}

	// A new file must be written with the downloaded bytes.
	dst := filepath.Join(dir, "pic-tiny.png")
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("reading compressed file: %v", err)
	}
	if string(data) != string(compressed) {
		t.Errorf("compressed file content = %q, want %q", data, compressed)
	}
	if res.Message == "" {
		t.Errorf("expected a non-empty result message")
	}
}

func TestTinifyServerErrorPropagates(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusUnauthorized)
	}))
	defer ts.Close()
	old := tinifyAPIURL
	tinifyAPIURL = ts.URL
	defer func() { tinifyAPIURL = old }()

	img := filepath.Join(t.TempDir(), "pic.png")
	os.WriteFile(img, []byte("data"), 0o644)
	if _, err := (Tinify{}).Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"api_key": "KEY"}},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{img}},
		Progress: nullProgress{}, Services: &recServices{},
	}); err == nil {
		t.Error("401 from tinify should error")
	}
}

func TestTinifyDownloadErrorPropagates(t *testing.T) {
	mux := http.NewServeMux()
	ts := httptest.NewServer(mux)
	defer ts.Close()
	mux.HandleFunc("/shrink", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", ts.URL+"/output/missing.png")
		w.WriteHeader(http.StatusCreated)
	})
	mux.HandleFunc("/output/missing.png", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "gone", http.StatusNotFound)
	})

	old := tinifyAPIURL
	tinifyAPIURL = ts.URL + "/shrink"
	defer func() { tinifyAPIURL = old }()

	img := filepath.Join(t.TempDir(), "pic.png")
	os.WriteFile(img, []byte("data"), 0o644)
	if _, err := (Tinify{}).Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"api_key": "KEY"}},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{img}},
		Progress: nullProgress{}, Services: &recServices{},
	}); err == nil {
		t.Error("404 download should error")
	}
}

func TestTinifyMissingAPIKeyAndEmptyInputErrors(t *testing.T) {
	// missing api key
	img := filepath.Join(t.TempDir(), "pic.png")
	os.WriteFile(img, []byte("data"), 0o644)
	if _, err := (Tinify{}).Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{img}},
		Progress: nullProgress{}, Services: &recServices{},
	}); err == nil {
		t.Error("missing api_key should error")
	}
	// empty input
	if _, err := (Tinify{}).Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"api_key": "KEY"}},
		Payload:  model.Payload{Kind: model.ItemFiles},
		Progress: nullProgress{}, Services: &recServices{},
	}); err == nil {
		t.Error("empty payload should error")
	}
	// non-image file
	txt := filepath.Join(t.TempDir(), "a.txt")
	os.WriteFile(txt, []byte("x"), 0o644)
	if _, err := (Tinify{}).Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"api_key": "KEY"}},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{txt}},
		Progress: nullProgress{}, Services: &recServices{},
	}); err == nil {
		t.Error("non-image should error")
	}
}

func TestTinifySpec(t *testing.T) {
	spec := Tinify{}.Spec()
	if spec.ID != "tinify" {
		t.Errorf("ID = %q", spec.ID)
	}
	if len(spec.Options) != 1 || !spec.Options[0].Required || spec.Options[0].Key != "api_key" {
		t.Errorf("Options = %+v", spec.Options)
	}
}

func TestHumanBytes(t *testing.T) {
	cases := map[int64]string{
		500:             "500 B",
		2048:            "2.0 KiB",
		5 * 1024 * 1024: "5.0 MiB",
	}
	for n, want := range cases {
		if got := humanBytes(n); got != want {
			t.Errorf("humanBytes(%d) = %q, want %q", n, got, want)
		}
	}
}
