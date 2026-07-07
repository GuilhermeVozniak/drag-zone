package builtin

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
