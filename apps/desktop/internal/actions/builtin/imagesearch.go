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

// imageSearchUploadURL is the ImgBB upload endpoint used to host the image
// before handing its URL to Google; a package var so tests can point it at
// an httptest server.
var imageSearchUploadURL = "https://api.imgbb.com/1/upload"

// ImageSearch uploads a dropped image to ImgBB, then opens a Google reverse
// image search for the uploaded URL.
type ImageSearch struct{}

func (ImageSearch) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "image-search",
		Name:        "Image Search",
		Description: "Upload a dropped image and open a reverse image search.",
		Icon:        "image-search",
		Category:    "Image",
		Events:      []string{model.EventDragged},
		Accepts:     []model.ItemKind{model.ItemFiles},
		Options: []model.OptionField{
			{Key: "api_key", Label: "ImgBB API key", Type: "text", Required: true},
		},
	}
}

func (ImageSearch) Dropped(ctx context.Context, inv actions.Invocation) (actions.Result, error) {
	apiKey := inv.Target.Option("api_key", "")
	if apiKey == "" {
		return actions.Result{}, fmt.Errorf("no ImgBB API key configured")
	}
	paths := inv.Payload.Paths
	if len(paths) == 0 {
		return actions.Result{}, fmt.Errorf("nothing to search")
	}
	if !isImageFile(paths[0]) {
		return actions.Result{}, fmt.Errorf("%s is not a supported image (jpg, jpeg, png, gif, webp, heic, bmp, tiff)", filepath.Base(paths[0]))
	}

	client := &http.Client{Timeout: 60 * time.Second}
	uploaded, err := uploadForImageSearch(ctx, client, apiKey, paths[0])
	if err != nil {
		return actions.Result{}, fmt.Errorf("uploading image: %w", err)
	}

	searchURL := "https://www.google.com/searchbyimage?image_url=" + url.QueryEscape(uploaded)
	if err := inv.Services.OpenURL(searchURL); err != nil {
		return actions.Result{}, fmt.Errorf("opening image search: %w", err)
	}

	return actions.Result{Message: "Opened image search", URL: searchURL}, nil
}

// uploadForImageSearch base64-encodes the image and posts it to the ImgBB
// API, returning its hosted URL (same request shape as imgbb.go's
// uploadToImgBB, against a distinct endpoint var).
func uploadForImageSearch(ctx context.Context, client *http.Client, apiKey, path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	encoded := base64.StdEncoding.EncodeToString(data)

	form := url.Values{}
	form.Set("key", apiKey)
	form.Set("image", encoded)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, imageSearchUploadURL, strings.NewReader(form.Encode()))
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
		return "", fmt.Errorf("reading upload response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upload returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var out struct {
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("parsing upload response: %w", err)
	}
	if out.Data.URL == "" {
		return "", fmt.Errorf("upload response contains no url")
	}
	return out.Data.URL, nil
}
