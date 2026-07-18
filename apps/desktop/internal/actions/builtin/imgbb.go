package builtin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// imgbbAPIURL is the upload endpoint; a package var so tests can point it at
// an httptest server.
var imgbbAPIURL = "https://api.imgbb.com/1/upload"

// ImgBBUpload uploads dropped image files to ImgBB and copies the resulting
// link(s) to the clipboard.
type ImgBBUpload struct{}

func (ImgBBUpload) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "imgbb",
		Name:        "ImgBB Upload",
		Description: "Upload dropped images to ImgBB and copy the link.",
		Icon:        "image-up",
		Category:    "Uploads",
		Events:      []string{model.EventDragged},
		Accepts:     []model.ItemKind{model.ItemFiles},
		Options: []model.OptionField{
			{Key: "api_key", Label: "ImgBB API Key", Type: "text", Required: true},
		},
	}
}

func (ImgBBUpload) Dropped(ctx context.Context, inv actions.Invocation) (actions.Result, error) {
	apiKey := inv.Target.Option("api_key", "")
	if apiKey == "" {
		return actions.Result{}, fmt.Errorf("no ImgBB API key configured")
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

	client := &http.Client{Timeout: 60 * time.Second}
	urls := make([]string, 0, len(paths))
	for i, p := range paths {
		inv.Progress.Detail(filepath.Base(p))
		inv.Progress.Percent(i * 100 / len(paths))
		link, err := uploadToImgBB(ctx, client, apiKey, p)
		if err != nil {
			return actions.Result{}, fmt.Errorf("uploading %s: %w", filepath.Base(p), err)
		}
		urls = append(urls, link)
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

// uploadToImgBB base64-encodes one image and posts it to the ImgBB API,
// returning its hosted URL.
func uploadToImgBB(ctx context.Context, client *http.Client, apiKey, path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	encoded := base64.StdEncoding.EncodeToString(data)

	form := url.Values{}
	form.Set("key", apiKey)
	form.Set("image", encoded)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, imgbbAPIURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("reading imgbb response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("imgbb returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var out struct {
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("parsing imgbb response: %w", err)
	}
	if out.Data.URL == "" {
		return "", fmt.Errorf("imgbb response contains no url")
	}
	return out.Data.URL, nil
}
