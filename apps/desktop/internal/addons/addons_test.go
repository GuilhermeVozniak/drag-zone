package addons

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"dragzone/internal/storage"
)

func TestListFiltersDzbundleDirs(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "application/vnd.github+json" {
			t.Errorf("missing Accept header: %q", r.Header.Get("Accept"))
		}
		w.Write([]byte(`[
			{"name":"Alpha.dzbundle","type":"dir"},
			{"name":"Beta.dzbundle","type":"dir"},
			{"name":"README.md","type":"file"},
			{"name":"NotABundle","type":"dir"}
		]`))
	}))
	defer ts.Close()
	old := contentsURL
	contentsURL = ts.URL
	defer func() { contentsURL = old }()

	names, err := List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(names) != 2 || names[0] != "Alpha" || names[1] != "Beta" {
		t.Errorf("names = %v, want [Alpha Beta]", names)
	}
}

func TestListServerErrorPropagates(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusForbidden)
	}))
	defer ts.Close()
	old := contentsURL
	contentsURL = ts.URL
	defer func() { contentsURL = old }()
	if _, err := List(context.Background()); err == nil {
		t.Error("403 should error")
	}
}

func TestListMalformedCatalogueErrors(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{not valid json`))
	}))
	defer ts.Close()
	old := contentsURL
	contentsURL = ts.URL
	defer func() { contentsURL = old }()
	if _, err := List(context.Background()); err == nil {
		t.Error("malformed catalogue should error")
	}
}

func TestListNoBundlesReturnsEmpty(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[{"name":"README.md","type":"file"}]`))
	}))
	defer ts.Close()
	old := contentsURL
	contentsURL = ts.URL
	defer func() { contentsURL = old }()
	names, err := List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("names = %v, want empty", names)
	}
}

// buildFakeArchive returns the bytes of a zip file laid out like the real
// dropzone4-actions GitHub archive: a top-level "dropzone4-actions-master"
// directory containing one "<bundleName>.dzbundle" bundle directory.
func buildFakeArchive(t *testing.T, bundleName string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	files := map[string]string{
		"dropzone4-actions-master/" + bundleName + ".dzbundle/Info.plist": "<plist/>",
		"dropzone4-actions-master/" + bundleName + ".dzbundle/default.py": "print('hi')",
	}
	for name, content := range files {
		fw, err := zw.Create(name)
		if err != nil {
			t.Fatalf("zip.Create(%q): %v", name, err)
		}
		if _, err := fw.Write([]byte(content)); err != nil {
			t.Fatalf("zip write %q: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip.Close: %v", err)
	}
	return buf.Bytes()
}

func TestFetchBundleDownloadsExtractsAndReturnsPath(t *testing.T) {
	t.Setenv(storage.EnvDataDir, t.TempDir())
	archive := buildFakeArchive(t, "Alpha")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(archive)
	}))
	defer ts.Close()
	old := archiveURL
	archiveURL = ts.URL
	defer func() { archiveURL = old }()

	cacheDir := t.TempDir()
	bundle, cleanup, err := FetchBundle(context.Background(), cacheDir, "Alpha")
	if err != nil {
		t.Fatalf("FetchBundle: %v", err)
	}
	defer cleanup()

	if filepath.Base(bundle) != "Alpha.dzbundle" {
		t.Errorf("bundle = %q, want basename Alpha.dzbundle", bundle)
	}
	if _, err := os.Stat(bundle); err != nil {
		t.Errorf("bundle path does not exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, "dropzone4-actions.zip")); err != nil {
		t.Errorf("archive was not cached in cacheDir: %v", err)
	}
}

func TestFetchBundleUsesCachedArchiveWithinTTL(t *testing.T) {
	t.Setenv(storage.EnvDataDir, t.TempDir())
	cacheDir := t.TempDir()
	zipPath := filepath.Join(cacheDir, "dropzone4-actions.zip")
	if err := os.WriteFile(zipPath, buildFakeArchive(t, "Cached"), 0o644); err != nil {
		t.Fatalf("seeding cache: %v", err)
	}

	called := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()
	old := archiveURL
	archiveURL = ts.URL
	defer func() { archiveURL = old }()

	bundle, cleanup, err := FetchBundle(context.Background(), cacheDir, "Cached")
	if err != nil {
		t.Fatalf("FetchBundle: %v", err)
	}
	defer cleanup()
	if called {
		t.Error("download should not be called while the cached archive is fresh")
	}
	if _, err := os.Stat(bundle); err != nil {
		t.Errorf("bundle missing: %v", err)
	}
}

func TestFetchBundleMissingNameErrors(t *testing.T) {
	t.Setenv(storage.EnvDataDir, t.TempDir())
	archive := buildFakeArchive(t, "Alpha")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(archive)
	}))
	defer ts.Close()
	old := archiveURL
	archiveURL = ts.URL
	defer func() { archiveURL = old }()

	if _, _, err := FetchBundle(context.Background(), t.TempDir(), "Beta"); err == nil {
		t.Error("expected error for a bundle name absent from the archive")
	}
}

func TestFetchBundleDownloadErrorPropagates(t *testing.T) {
	t.Setenv(storage.EnvDataDir, t.TempDir())
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusInternalServerError)
	}))
	defer ts.Close()
	old := archiveURL
	archiveURL = ts.URL
	defer func() { archiveURL = old }()

	if _, _, err := FetchBundle(context.Background(), t.TempDir(), "Alpha"); err == nil {
		t.Error("expected error when the archive download fails")
	}
}
