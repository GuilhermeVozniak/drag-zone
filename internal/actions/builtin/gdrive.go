package builtin

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path"
	"time"

	"golang.org/x/oauth2"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// Google Drive endpoints and OAuth parameters. The endpoints are spelled out
// here instead of importing golang.org/x/oauth2/google to keep the dependency
// footprint at golang.org/x/oauth2 alone.
const (
	driveAuthURL     = "https://accounts.google.com/o/oauth2/auth"
	driveTokenURL    = "https://oauth2.googleapis.com/token"
	driveScope       = "https://www.googleapis.com/auth/drive.file"
	driveUploadURL   = "https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart"
	driveFilesURL    = "https://www.googleapis.com/drive/v3/files/"
	driveAuthWait    = 3 * time.Minute // how long to wait for the browser consent callback
	driveHTTPTimeout = 60 * time.Second
)

// GoogleDriveUpload uploads dropped files and folders to Google Drive and
// copies the web link of the first uploaded file to the clipboard.
//
// It authenticates with OAuth using the user's own Google Cloud credential:
// in the Google Cloud console, create an OAuth client ID of type "Desktop
// app" (APIs & Services > Credentials) with the Google Drive API enabled, and
// paste the resulting client ID and client secret into the target's options.
// On the first drop a browser window asks for consent; the granted refresh
// token is persisted via Invocation.SaveOption so later drops upload
// silently (or not at all when the host provides no persistence, in which
// case every drop re-authorizes).
type GoogleDriveUpload struct{}

func (GoogleDriveUpload) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "google-drive",
		Name:        "Google Drive",
		Description: "Upload dropped files to Google Drive and copy the link.",
		Icon:        "upload",
		Category:    "Uploads",
		Events:      []string{model.EventDragged},
		Accepts:     []model.ItemKind{model.ItemFiles},
		Multi:       true,
		Options: []model.OptionField{
			{Key: "client_id", Label: "OAuth Client ID", Type: "text", Required: true},
			{Key: "client_secret", Label: "OAuth Client Secret", Type: "password", Required: true},
			{Key: "folder_id", Label: "Drive folder ID (optional)", Type: "text"},
		},
	}
}

func (GoogleDriveUpload) Dropped(ctx context.Context, inv actions.Invocation) (actions.Result, error) {
	clientID := inv.Target.Option("client_id", "")
	clientSecret := inv.Target.Option("client_secret", "")
	if clientID == "" || clientSecret == "" {
		return actions.Result{}, fmt.Errorf("OAuth client ID and client secret must be configured")
	}
	if len(inv.Payload.Paths) == 0 {
		return actions.Result{}, fmt.Errorf("nothing to upload")
	}

	entries, total, err := collectUploadEntries(inv.Payload.Paths)
	if err != nil {
		return actions.Result{}, err
	}

	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{driveScope},
		Endpoint:     oauth2.Endpoint{AuthURL: driveAuthURL, TokenURL: driveTokenURL},
	}
	client, err := driveClient(ctx, cfg, inv)
	if err != nil {
		return actions.Result{}, err
	}

	folderID := inv.Target.Option("folder_id", "")
	var done int64
	var firstLink string
	for i, e := range entries {
		inv.Progress.Detail(path.Base(e.rel))
		id, err := driveUploadOne(ctx, client, e, folderID, total, &done, inv.Progress)
		if err != nil {
			if isDriveAuthError(err) {
				saveDriveRefreshToken(inv, "")
				return actions.Result{}, fmt.Errorf("uploading %s: Google Drive rejected the authorization; stored token cleared, drop again to re-authorize: %w", e.rel, err)
			}
			return actions.Result{}, fmt.Errorf("uploading %s: %w", e.rel, err)
		}
		if i == 0 {
			link, err := driveWebViewLink(ctx, client, id)
			if err != nil {
				return actions.Result{}, fmt.Errorf("fetching link for %s: %w", e.rel, err)
			}
			if err := inv.Services.CopyToClipboard(link); err != nil {
				return actions.Result{}, fmt.Errorf("copying URL to clipboard: %w", err)
			}
			firstLink = link
		}
	}
	return actions.Result{
		Message: fmt.Sprintf("Uploaded %d item(s) to Google Drive", len(inv.Payload.Paths)),
		URL:     firstLink,
	}, nil
}

// driveClient returns an HTTP client that attaches Drive access tokens to
// every request. It reuses the stored refresh token when present, otherwise
// it runs the interactive loopback OAuth flow, and it persists refresh-token
// changes via SaveTargetOption.
func driveClient(ctx context.Context, cfg *oauth2.Config, inv actions.Invocation) (*http.Client, error) {
	// Token-endpoint traffic (exchange and refreshes) is small, so a whole
	// request comfortably fits in the 60s budget.
	octx := context.WithValue(ctx, oauth2.HTTPClient, &http.Client{Timeout: driveHTTPTimeout})

	refresh := inv.Target.Option("refresh_token", "")
	tok := &oauth2.Token{RefreshToken: refresh}
	if refresh == "" {
		t, err := driveAuthorize(octx, cfg, inv)
		if err != nil {
			return nil, fmt.Errorf("authorizing with Google Drive: %w", err)
		}
		tok = t
		saveDriveRefreshToken(inv, tok.RefreshToken)
	}

	src := cfg.TokenSource(octx, tok)
	fresh, err := src.Token()
	if err != nil {
		saveDriveRefreshToken(inv, "")
		return nil, fmt.Errorf("refreshing Google Drive access token (stored authorization cleared, drop again to re-authorize): %w", err)
	}
	if fresh.RefreshToken != "" && fresh.RefreshToken != tok.RefreshToken {
		saveDriveRefreshToken(inv, fresh.RefreshToken)
	}

	// Uploads of large files legitimately run longer than 60s, so the API
	// client bounds each connection phase at 60s instead of capping the whole
	// request the way the token client above does.
	base := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: driveHTTPTimeout}).DialContext,
		TLSHandshakeTimeout:   driveHTTPTimeout,
		ResponseHeaderTimeout: driveHTTPTimeout,
	}
	return &http.Client{
		Transport: &oauth2.Transport{Source: oauth2.ReuseTokenSource(fresh, src), Base: base},
	}, nil
}

// driveAuthorize runs the loopback OAuth flow: it starts a listener on an
// ephemeral 127.0.0.1 port, opens the Google consent page in the browser and
// waits up to driveAuthWait for the redirect carrying the authorization code,
// which it exchanges (with PKCE) for tokens.
func driveAuthorize(ctx context.Context, cfg *oauth2.Config, inv actions.Invocation) (*oauth2.Token, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("starting local OAuth callback listener: %w", err)
	}
	defer ln.Close()

	state, err := randomURLToken()
	if err != nil {
		return nil, fmt.Errorf("generating OAuth state: %w", err)
	}
	verifier := oauth2.GenerateVerifier()

	local := *cfg // shallow copy: the redirect URL is per-attempt
	local.RedirectURL = "http://" + ln.Addr().String()
	authURL := local.AuthCodeURL(state,
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(verifier),
		// Force the consent screen so Google always returns a refresh token,
		// even when the user granted access in an earlier session.
		oauth2.SetAuthURLParam("prompt", "consent"),
	)

	type callback struct {
		code string
		err  error
	}
	ch := make(chan callback, 1)
	deliver := func(cb callback) {
		select {
		case ch <- cb:
		default: // a callback already arrived; ignore extras
		}
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		code, oauthErr := q.Get("code"), q.Get("error")
		if code == "" && oauthErr == "" {
			http.NotFound(w, r) // stray request, e.g. the browser fetching a favicon
			return
		}
		switch {
		case oauthErr != "":
			fmt.Fprintln(w, "Authorization failed. You can close this window.")
			deliver(callback{err: fmt.Errorf("google reported %q", oauthErr)})
		case q.Get("state") != state:
			http.Error(w, "State mismatch. You can close this window.", http.StatusBadRequest)
			deliver(callback{err: fmt.Errorf("OAuth state mismatch in callback")})
		default:
			fmt.Fprintln(w, "Google Drive connected. You can close this window and return to DragZone.")
			deliver(callback{code: code})
		}
	})}
	go srv.Serve(ln) //nolint:errcheck // Serve always returns a non-nil error on Close
	defer srv.Close()

	inv.Progress.Detail("Waiting for Google authorization in the browser")
	if err := inv.Services.OpenURL(authURL); err != nil {
		return nil, fmt.Errorf("opening Google authorization page: %w", err)
	}

	select {
	case cb := <-ch:
		if cb.err != nil {
			return nil, cb.err
		}
		tok, err := local.Exchange(ctx, cb.code, oauth2.VerifierOption(verifier))
		if err != nil {
			return nil, fmt.Errorf("exchanging authorization code: %w", err)
		}
		if tok.RefreshToken == "" {
			return nil, fmt.Errorf("google returned no refresh token; remove the app's access at myaccount.google.com/permissions and try again")
		}
		return tok, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("waiting for Google authorization: %w", ctx.Err())
	case <-time.After(driveAuthWait):
		return nil, fmt.Errorf("timed out after %s waiting for Google authorization", driveAuthWait)
	}
}

// driveUploadOne streams one local file to Drive as a multipart upload and
// returns the created file's ID, updating byte-based progress as it reads.
func driveUploadOne(ctx context.Context, client *http.Client, e uploadEntry, folderID string, total int64, done *int64, progress actions.Progress) (string, error) {
	f, err := os.Open(e.local)
	if err != nil {
		return "", fmt.Errorf("opening local file: %w", err)
	}
	defer f.Close()

	meta := map[string]any{"name": path.Base(e.rel)}
	if folderID != "" {
		meta["parents"] = []string{folderID}
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return "", fmt.Errorf("encoding file metadata: %w", err)
	}

	body := &progressReader{r: f, onBytes: func(n int64) {
		if total > 0 {
			*done += n
			progress.Percent(int(*done * 100 / total))
		}
	}}

	// Stream the multipart body through a pipe so large files are never
	// buffered in memory.
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		pw.CloseWithError(writeDriveMultipart(mw, metaJSON, body))
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, driveUploadURL, pr)
	if err != nil {
		return "", fmt.Errorf("building upload request: %w", err)
	}
	req.Header.Set("Content-Type", "multipart/related; boundary="+mw.Boundary())

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("sending upload request: %w", err)
	}
	defer resp.Body.Close()
	if err := driveCheckResponse(resp); err != nil {
		return "", err
	}

	var created struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return "", fmt.Errorf("decoding upload response: %w", err)
	}
	if created.ID == "" {
		return "", fmt.Errorf("upload response carried no file ID")
	}
	return created.ID, nil
}

// writeDriveMultipart writes the two parts of a Drive multipart upload (JSON
// metadata, then the raw file content) and closes the multipart writer.
func writeDriveMultipart(mw *multipart.Writer, metaJSON []byte, content io.Reader) error {
	metaHeader := textproto.MIMEHeader{}
	metaHeader.Set("Content-Type", "application/json; charset=UTF-8")
	metaPart, err := mw.CreatePart(metaHeader)
	if err != nil {
		return fmt.Errorf("creating metadata part: %w", err)
	}
	if _, err := metaPart.Write(metaJSON); err != nil {
		return fmt.Errorf("writing metadata part: %w", err)
	}

	fileHeader := textproto.MIMEHeader{}
	fileHeader.Set("Content-Type", "application/octet-stream")
	filePart, err := mw.CreatePart(fileHeader)
	if err != nil {
		return fmt.Errorf("creating content part: %w", err)
	}
	if _, err := io.Copy(filePart, content); err != nil {
		return fmt.Errorf("streaming file content: %w", err)
	}
	return mw.Close()
}

// driveWebViewLink fetches the browser link for an uploaded file.
func driveWebViewLink(ctx context.Context, client *http.Client, fileID string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, driveFilesURL+url.PathEscape(fileID)+"?fields=webViewLink", nil)
	if err != nil {
		return "", fmt.Errorf("building link request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("requesting web link: %w", err)
	}
	defer resp.Body.Close()
	if err := driveCheckResponse(resp); err != nil {
		return "", err
	}
	var info struct {
		WebViewLink string `json:"webViewLink"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", fmt.Errorf("decoding link response: %w", err)
	}
	if info.WebViewLink == "" {
		return "", fmt.Errorf("drive returned no webViewLink")
	}
	return info.WebViewLink, nil
}

// driveAuthFailed marks errors caused by Drive rejecting the access token, so
// the caller can clear the stored refresh token and ask for re-authorization.
type driveAuthFailed struct{ err error }

func (e *driveAuthFailed) Error() string { return e.err.Error() }
func (e *driveAuthFailed) Unwrap() error { return e.err }

// isDriveAuthError reports whether err stems from a 401 response.
func isDriveAuthError(err error) bool {
	var authErr *driveAuthFailed
	return errors.As(err, &authErr)
}

// driveCheckResponse turns a non-2xx Drive API response into an error carrying
// a snippet of the response body; 401s are wrapped as driveAuthFailed.
func driveCheckResponse(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
	err := fmt.Errorf("drive API returned %s: %s", resp.Status, string(snippet))
	if resp.StatusCode == http.StatusUnauthorized {
		return &driveAuthFailed{err: err}
	}
	return err
}

// saveDriveRefreshToken persists (or clears, for value "") the refresh token
// on the target, when the host provides option persistence.
func saveDriveRefreshToken(inv actions.Invocation, value string) {
	if inv.SaveOption != nil {
		inv.SaveOption("refresh_token", value)
	}
}

// randomURLToken returns a URL-safe random string for use as OAuth state.
func randomURLToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
