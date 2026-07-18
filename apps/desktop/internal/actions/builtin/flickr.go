package builtin

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// flickrUploadURL is the OAuth 1.0a-signed upload endpoint; a package var so
// tests can point it at an httptest server.
var flickrUploadURL = "https://up.flickr.com/services/upload/"

// flickrNonce returns a fresh OAuth nonce. A package var so tests can seam it
// to a fixed value.
var flickrNonce = func() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand.Read only fails if the OS entropy source is broken;
		// fall back to a timestamp-derived value rather than panicking.
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(b)
}

// flickrNow returns the current Unix time for the oauth_timestamp
// parameter. A package var so tests can seam it to a fixed value.
var flickrNow = func() int64 { return time.Now().Unix() }

// FlickrUpload uploads dropped image files to Flickr via an OAuth 1.0a
// signed multipart POST and copies the resulting link(s) to the clipboard.
type FlickrUpload struct{}

func (FlickrUpload) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "flickr",
		Name:        "Flickr Upload",
		Description: "Upload dropped images to Flickr and copy the link.",
		Icon:        "aperture",
		Category:    "Uploads",
		Events:      []string{model.EventDragged},
		Accepts:     []model.ItemKind{model.ItemFiles},
		Multi:       true,
		Options: []model.OptionField{
			{Key: "api_key", Label: "Flickr API Key", Type: "text", Required: true},
			{Key: "api_secret", Label: "Flickr API Secret", Type: "password", Required: true},
			{Key: "oauth_token", Label: "OAuth Token", Type: "text", Required: true},
			{Key: "oauth_token_secret", Label: "OAuth Token Secret", Type: "password", Required: true},
		},
	}
}

func (FlickrUpload) Dropped(ctx context.Context, inv actions.Invocation) (actions.Result, error) {
	apiKey := inv.Target.Option("api_key", "")
	apiSecret := inv.Target.Option("api_secret", "")
	oauthToken := inv.Target.Option("oauth_token", "")
	oauthTokenSecret := inv.Target.Option("oauth_token_secret", "")
	if apiKey == "" || apiSecret == "" || oauthToken == "" || oauthTokenSecret == "" {
		return actions.Result{}, fmt.Errorf("flickr API key, API secret, OAuth token and OAuth token secret must be configured")
	}
	paths := inv.Payload.Paths
	if len(paths) == 0 {
		return actions.Result{}, fmt.Errorf("nothing to upload")
	}
	for _, p := range paths {
		if !isImageFile(p) {
			return actions.Result{}, fmt.Errorf("%s is not a supported image (jpg, jpeg, png, gif, webp, heic, bmp, tiff)", filepath.Base(p))
		}
	}

	client := &http.Client{Timeout: 120 * time.Second}
	urls := make([]string, 0, len(paths))
	for i, p := range paths {
		inv.Progress.Detail(filepath.Base(p))
		inv.Progress.Percent(i * 100 / len(paths))
		photoID, err := uploadToFlickr(ctx, client, apiKey, apiSecret, oauthToken, oauthTokenSecret, p)
		if err != nil {
			return actions.Result{}, fmt.Errorf("uploading %s: %w", filepath.Base(p), err)
		}
		urls = append(urls, "https://www.flickr.com/photo.gne?id="+photoID)
	}
	inv.Progress.Percent(100)

	if err := inv.Services.CopyToClipboard(strings.Join(urls, "\n")); err != nil {
		return actions.Result{}, fmt.Errorf("copying URL to clipboard: %w", err)
	}
	return actions.Result{
		Message: fmt.Sprintf("Uploaded %d image(s) — URL copied", len(urls)),
		URL:     urls[0],
	}, nil
}

// uploadToFlickr streams one image to the Flickr upload API, authenticated
// via an OAuth 1.0a Authorization header, and returns the new photo's id.
//
// Flickr's upload endpoint signs only the OAuth protocol parameters — the
// multipart photo body is excluded from the signature base string.
func uploadToFlickr(ctx context.Context, client *http.Client, apiKey, apiSecret, oauthToken, oauthTokenSecret, path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	params := map[string]string{
		"oauth_consumer_key":     apiKey,
		"oauth_nonce":            flickrNonce(),
		"oauth_signature_method": "HMAC-SHA1",
		"oauth_timestamp":        strconv.FormatInt(flickrNow(), 10),
		"oauth_token":            oauthToken,
		"oauth_version":          "1.0",
	}
	params["oauth_signature"] = signOAuth1(http.MethodPost, flickrUploadURL, params, apiSecret, oauthTokenSecret)

	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		part, err := mw.CreateFormFile("photo", filepath.Base(path))
		if err != nil {
			pw.CloseWithError(err)
			return
		}
		if _, err := io.Copy(part, f); err != nil {
			pw.CloseWithError(err)
			return
		}
		pw.CloseWithError(mw.Close())
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, flickrUploadURL, pr)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", oauthAuthorizationHeader(params))
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("reading flickr response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("flickr returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var out struct {
		XMLName xml.Name `xml:"rsp"`
		Stat    string   `xml:"stat,attr"`
		PhotoID string   `xml:"photoid"`
		Err     struct {
			Msg string `xml:"msg,attr"`
		} `xml:"err"`
	}
	if err := xml.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("parsing flickr response: %w", err)
	}
	if out.Stat != "ok" {
		return "", fmt.Errorf("flickr upload failed: %s", out.Err.Msg)
	}
	if out.PhotoID == "" {
		return "", fmt.Errorf("flickr response contains no photo id")
	}
	return out.PhotoID, nil
}

// oauthAuthorizationHeader builds the "OAuth k=\"v\", ..." Authorization
// header value from a set of already-final (unsigned or signed) OAuth
// parameters, percent-encoding each value per RFC 3986.
func oauthAuthorizationHeader(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = fmt.Sprintf(`%s="%s"`, percentEncode(k), percentEncode(params[k]))
	}
	return "OAuth " + strings.Join(parts, ", ")
}

// signOAuth1 computes an OAuth 1.0a HMAC-SHA1 signature per RFC 5849 §3.4:
// build the signature base string from the method, base URL and the sorted,
// percent-encoded parameters, then HMAC-SHA1 it with a key derived from the
// consumer and token secrets.
func signOAuth1(method, baseURL string, params map[string]string, consumerSecret, tokenSecret string) string {
	type kv struct{ k, v string }
	pairs := make([]kv, 0, len(params))
	for k, v := range params {
		pairs = append(pairs, kv{percentEncode(k), percentEncode(v)})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].k != pairs[j].k {
			return pairs[i].k < pairs[j].k
		}
		return pairs[i].v < pairs[j].v
	})

	parts := make([]string, len(pairs))
	for i, p := range pairs {
		parts[i] = p.k + "=" + p.v
	}
	paramString := strings.Join(parts, "&")

	baseString := strings.ToUpper(method) + "&" + percentEncode(baseURL) + "&" + percentEncode(paramString)
	signingKey := percentEncode(consumerSecret) + "&" + percentEncode(tokenSecret)

	mac := hmac.New(sha1.New, []byte(signingKey))
	mac.Write([]byte(baseString))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// percentEncode RFC3986-percent-encodes s: everything except unreserved
// characters (A-Za-z0-9-._~) is escaped as %XX. This differs from
// net/url.QueryEscape, which encodes spaces as "+" rather than "%20" and is
// therefore not usable for OAuth 1.0a signing.
func percentEncode(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') ||
			c == '-' || c == '.' || c == '_' || c == '~' {
			b.WriteByte(c)
		} else {
			fmt.Fprintf(&b, "%%%02X", c)
		}
	}
	return b.String()
}
