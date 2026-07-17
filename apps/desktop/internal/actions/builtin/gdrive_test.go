package builtin

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// driveTestServer wires a single httptest server that answers the three
// Google Drive endpoints the action talks to: token refresh, multipart
// upload, and the webViewLink lookup. rotateRefresh, when non-empty, is
// returned as a new refresh_token from the token endpoint so tests can
// exercise the rotation/SaveOption path. uploadStatus lets a test force the
// upload endpoint to fail.
func driveTestServer(t *testing.T, rotateRefresh string, uploadStatus int) (*httptest.Server, *string) {
	t.Helper()
	var gotAuthHeader string
	mux := http.NewServeMux()
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"access_token": "ACCESS-TOKEN",
			"token_type":   "Bearer",
			"expires_in":   3600,
		}
		if rotateRefresh != "" {
			resp["refresh_token"] = rotateRefresh
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		gotAuthHeader = r.Header.Get("Authorization")
		if uploadStatus != 0 && uploadStatus != http.StatusOK {
			w.WriteHeader(uploadStatus)
			w.Write([]byte("upload rejected"))
			return
		}
		_ = r.ParseMultipartForm(1 << 20)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"FILE-ID"}`))
	})
	mux.HandleFunc("/files/", func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "webViewLink") {
			t.Errorf("missing fields query: %q", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"webViewLink":"https://drive.google.com/file/d/FILE-ID/view"}`))
	})
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)
	return ts, &gotAuthHeader
}

// withDriveTestServer points the token, upload and files URLs at ts for the
// duration of the test.
func withDriveTestServer(t *testing.T, ts *httptest.Server) {
	t.Helper()
	oldToken, oldUpload, oldFiles := driveTokenURL, driveUploadURL, driveFilesURL
	driveTokenURL = ts.URL + "/token"
	driveUploadURL = ts.URL + "/upload"
	driveFilesURL = ts.URL + "/files/"
	t.Cleanup(func() {
		driveTokenURL, driveUploadURL, driveFilesURL = oldToken, oldUpload, oldFiles
	})
}

func TestRandomURLTokenUniqueAndURLSafe(t *testing.T) {
	a, err := randomURLToken()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := randomURLToken()
	if a == "" || a == b {
		t.Errorf("tokens not unique/non-empty: %q %q", a, b)
	}
	if strings.ContainsAny(a, "+/=") {
		t.Errorf("token not URL-safe: %q", a)
	}
}

func TestDriveCheckResponse(t *testing.T) {
	ok := &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(""))}
	if err := driveCheckResponse(ok); err != nil {
		t.Errorf("200 should be nil: %v", err)
	}
	unauth := &http.Response{StatusCode: 401, Status: "401 Unauthorized", Body: io.NopCloser(strings.NewReader("bad token"))}
	err := driveCheckResponse(unauth)
	if err == nil || !isDriveAuthError(err) {
		t.Errorf("401 should be a drive auth error, got %v", err)
	}
	serverErr := &http.Response{StatusCode: 500, Status: "500 err", Body: io.NopCloser(strings.NewReader("x"))}
	if err := driveCheckResponse(serverErr); err == nil || isDriveAuthError(err) {
		t.Errorf("500 should be a non-auth error, got %v", err)
	}
}

func TestDriveWebViewLink(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "webViewLink") {
			t.Errorf("missing fields query: %q", r.URL.RawQuery)
		}
		w.Write([]byte(`{"webViewLink":"https://drive.google.com/file/d/ID/view"}`))
	}))
	defer ts.Close()
	old := driveFilesURL
	driveFilesURL = ts.URL + "/"
	defer func() { driveFilesURL = old }()

	link, err := driveWebViewLink(context.Background(), ts.Client(), "ID")
	if err != nil || link != "https://drive.google.com/file/d/ID/view" {
		t.Errorf("link = %q err %v", link, err)
	}
}

// TestGoogleDriveUploadRoundTrip drives the whole Dropped path with a preset
// refresh_token (so driveClient takes the silent-refresh branch instead of
// running the interactive browser OAuth flow) against an httptest server
// standing in for the token, upload and files endpoints.
func TestGoogleDriveUploadRoundTrip(t *testing.T) {
	ts, gotAuthHeader := driveTestServer(t, "", 0)
	withDriveTestServer(t, ts)

	src := filepath.Join(t.TempDir(), "pic.png")
	if err := os.WriteFile(src, []byte("fake image bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	svc := &recServices{}
	res, err := GoogleDriveUpload{}.Dropped(context.Background(), actions.Invocation{
		Target: model.Target{Options: map[string]string{
			"client_id": "CID", "client_secret": "CSECRET", "refresh_token": "OLD-REFRESH",
		}},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{src}},
		Progress: nullProgress{},
		Services: svc,
	})
	if err != nil {
		t.Fatalf("Dropped: %v", err)
	}

	wantLink := "https://drive.google.com/file/d/FILE-ID/view"
	if res.URL != wantLink {
		t.Errorf("result URL = %q, want %q", res.URL, wantLink)
	}
	if svc.Clipboard != wantLink {
		t.Errorf("clipboard = %q, want %q", svc.Clipboard, wantLink)
	}
	if *gotAuthHeader != "Bearer ACCESS-TOKEN" {
		t.Errorf("upload Authorization header = %q", *gotAuthHeader)
	}
}

// TestGoogleDriveUploadRefreshTokenRotationSavesOption asserts that when the
// token endpoint returns a new refresh token (Google rotates it on some
// refreshes), driveClient persists it via Invocation.SaveOption.
func TestGoogleDriveUploadRefreshTokenRotationSavesOption(t *testing.T) {
	ts, _ := driveTestServer(t, "NEW-REFRESH", 0)
	withDriveTestServer(t, ts)

	src := filepath.Join(t.TempDir(), "pic.png")
	if err := os.WriteFile(src, []byte("fake image bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	saved := map[string]string{}
	_, err := GoogleDriveUpload{}.Dropped(context.Background(), actions.Invocation{
		Target: model.Target{Options: map[string]string{
			"client_id": "CID", "client_secret": "CSECRET", "refresh_token": "OLD-REFRESH",
		}},
		Payload:    model.Payload{Kind: model.ItemFiles, Paths: []string{src}},
		Progress:   nullProgress{},
		Services:   &recServices{},
		SaveOption: func(key, value string) { saved[key] = value },
	})
	if err != nil {
		t.Fatalf("Dropped: %v", err)
	}
	if saved["refresh_token"] != "NEW-REFRESH" {
		t.Errorf("SaveOption refresh_token = %q, want %q", saved["refresh_token"], "NEW-REFRESH")
	}
}

// TestGoogleDriveUploadNoRotationDoesNotSave asserts the counterpart: when
// the token endpoint does not rotate the refresh token, SaveOption is never
// invoked.
func TestGoogleDriveUploadNoRotationDoesNotSave(t *testing.T) {
	ts, _ := driveTestServer(t, "", 0)
	withDriveTestServer(t, ts)

	src := filepath.Join(t.TempDir(), "pic.png")
	if err := os.WriteFile(src, []byte("fake image bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	saveCalls := 0
	_, err := GoogleDriveUpload{}.Dropped(context.Background(), actions.Invocation{
		Target: model.Target{Options: map[string]string{
			"client_id": "CID", "client_secret": "CSECRET", "refresh_token": "OLD-REFRESH",
		}},
		Payload:    model.Payload{Kind: model.ItemFiles, Paths: []string{src}},
		Progress:   nullProgress{},
		Services:   &recServices{},
		SaveOption: func(key, value string) { saveCalls++ },
	})
	if err != nil {
		t.Fatalf("Dropped: %v", err)
	}
	if saveCalls != 0 {
		t.Errorf("SaveOption called %d times, want 0", saveCalls)
	}
}

// TestGoogleDriveUploadErrorWrapped asserts that an upload API failure is
// wrapped (with %w, preserving the underlying error for errors.Is/As) and
// identifies the file that failed.
func TestGoogleDriveUploadErrorWrapped(t *testing.T) {
	ts, _ := driveTestServer(t, "", http.StatusInternalServerError)
	withDriveTestServer(t, ts)

	src := filepath.Join(t.TempDir(), "pic.png")
	if err := os.WriteFile(src, []byte("fake image bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := GoogleDriveUpload{}.Dropped(context.Background(), actions.Invocation{
		Target: model.Target{Options: map[string]string{
			"client_id": "CID", "client_secret": "CSECRET", "refresh_token": "OLD-REFRESH",
		}},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{src}},
		Progress: nullProgress{},
		Services: &recServices{},
	})
	if err == nil {
		t.Fatal("expected an error from a failing upload")
	}
	if !strings.Contains(err.Error(), "uploading "+filepath.Base(src)) {
		t.Errorf("error should be wrapped with the file context: %v", err)
	}
	if isDriveAuthError(err) {
		t.Errorf("500 should not be classified as an auth error: %v", err)
	}
}

// TestGoogleDriveUploadMissingCredentials covers the early validation guards
// in Dropped that never touch the network.
func TestGoogleDriveUploadMissingCredentials(t *testing.T) {
	if _, err := (GoogleDriveUpload{}).Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"client_secret": "S"}},
		Payload:  model.Payload{Paths: []string{"/a.png"}},
		Progress: nullProgress{}, Services: &recServices{},
	}); err == nil {
		t.Error("missing client_id should error")
	}
	if _, err := (GoogleDriveUpload{}).Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: map[string]string{"client_id": "C", "client_secret": "S"}},
		Payload:  model.Payload{Paths: nil},
		Progress: nullProgress{}, Services: &recServices{},
	}); err == nil {
		t.Error("empty payload should error")
	}
}
