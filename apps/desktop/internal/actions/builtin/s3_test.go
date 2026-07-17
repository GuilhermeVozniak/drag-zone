package builtin

import (
	"archive/zip"
	"bytes"
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

func TestS3PublicURL(t *testing.T) {
	// Default virtual-hosted-style URL.
	got := s3PublicURL("mybucket", "us-east-1", "", "uploads/a.png")
	want := "https://mybucket.s3.us-east-1.amazonaws.com/uploads/a.png"
	if got != want {
		t.Errorf("default = %q, want %q", got, want)
	}
	// Custom prefix wins and trailing slash is trimmed.
	got = s3PublicURL("b", "eu-west-1", "https://cdn.example.com/", "k/x.jpg")
	if got != "https://cdn.example.com/k/x.jpg" {
		t.Errorf("custom prefix = %q", got)
	}
}

// withS3TestServer points the S3 client at ts for the duration of the test.
func withS3TestServer(t *testing.T, ts *httptest.Server) {
	t.Helper()
	old := s3EndpointOverride
	s3EndpointOverride = ts.URL
	t.Cleanup(func() { s3EndpointOverride = old })
}

func TestS3UploadRoundTrip(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()
	withS3TestServer(t, ts)

	src := filepath.Join(t.TempDir(), "pic.txt")
	if err := os.WriteFile(src, []byte("hello s3"), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := &recServices{}
	res, err := S3Upload{}.Dropped(context.Background(), actions.Invocation{
		Target: model.Target{Options: map[string]string{
			"access_key": "AK", "secret_key": "SK", "region": "us-east-1", "bucket": "mybucket",
			"key_prefix": "uploads/",
		}},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{src}},
		Progress: nullProgress{},
		Services: svc,
	})
	if err != nil {
		t.Fatalf("Dropped: %v", err)
	}

	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	wantPath := "/mybucket/uploads/pic.txt"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if string(gotBody) != "hello s3" {
		t.Errorf("body = %q", gotBody)
	}

	wantURL := "https://mybucket.s3.us-east-1.amazonaws.com/uploads/pic.txt"
	if res.URL != wantURL {
		t.Errorf("result URL = %q, want %q", res.URL, wantURL)
	}
	if svc.Clipboard != wantURL {
		t.Errorf("clipboard = %q, want %q", svc.Clipboard, wantURL)
	}
}

func TestS3UploadOptionModifierZipsFirst(t *testing.T) {
	var putCount int
	var gotPath string
	var gotBody []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		putCount++
		gotPath = r.URL.Path
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()
	withS3TestServer(t, ts)

	dir := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("content-"+name), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	svc := &recServices{}
	_, err := S3Upload{}.Dropped(context.Background(), actions.Invocation{
		Target: model.Target{Options: map[string]string{
			"access_key": "AK", "secret_key": "SK", "region": "us-east-1", "bucket": "mybucket",
		}},
		Payload: model.Payload{
			Kind:      model.ItemFiles,
			Paths:     []string{filepath.Join(dir, "a.txt"), filepath.Join(dir, "b.txt")},
			Modifiers: []string{"Option"},
		},
		Progress: nullProgress{},
		Services: svc,
	})
	if err != nil {
		t.Fatalf("Dropped: %v", err)
	}

	if putCount != 1 {
		t.Fatalf("PUT count = %d, want 1 (single zip upload)", putCount)
	}
	if filepath.Ext(gotPath) != ".zip" {
		t.Errorf("uploaded key %q is not a .zip", gotPath)
	}

	zr, err := zip.NewReader(bytes.NewReader(gotBody), int64(len(gotBody)))
	if err != nil {
		t.Fatalf("uploaded body is not a valid zip: %v", err)
	}
	if len(zr.File) != 2 {
		t.Errorf("zip entries = %d, want 2", len(zr.File))
	}
}

func TestS3UploadMissingCredentials(t *testing.T) {
	base := map[string]string{
		"access_key": "AK", "secret_key": "SK", "region": "us-east-1", "bucket": "mybucket",
	}
	for _, missing := range []string{"access_key", "secret_key", "region", "bucket"} {
		opts := map[string]string{}
		for k, v := range base {
			if k != missing {
				opts[k] = v
			}
		}
		_, err := S3Upload{}.Dropped(context.Background(), actions.Invocation{
			Target:   model.Target{Options: opts},
			Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{"/tmp/whatever.txt"}},
			Progress: nullProgress{},
			Services: &recServices{},
		})
		if err == nil {
			t.Errorf("missing %s should error", missing)
		}
	}
}
