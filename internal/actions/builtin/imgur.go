package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// ImgurUpload uploads dropped image files to Imgur and copies the resulting
// link(s) to the clipboard.
type ImgurUpload struct{}

func (ImgurUpload) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "imgur",
		Name:        "Imgur Upload",
		Description: "Upload dropped images to Imgur and copy the link.",
		Icon:        "upload",
		Category:    "Uploads",
		Events:      []string{model.EventDragged},
		Accepts:     []model.ItemKind{model.ItemFiles},
		Options: []model.OptionField{
			{Key: "client_id", Label: "Imgur Client ID", Type: "text", Required: true},
		},
	}
}

func (ImgurUpload) Dropped(ctx context.Context, inv actions.Invocation) (actions.Result, error) {
	clientID := inv.Target.Option("client_id", "")
	if clientID == "" {
		return actions.Result{}, fmt.Errorf("no Imgur client ID configured")
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
		link, err := uploadToImgur(ctx, client, clientID, p)
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

// isImageFile reports whether the file has an image extension Imgur accepts.
func isImageFile(path string) bool {
	switch strings.ToLower(strings.TrimPrefix(filepath.Ext(path), ".")) {
	case "jpg", "jpeg", "png", "gif", "webp", "heic", "bmp", "tiff":
		return true
	}
	return false
}

// uploadToImgur streams one image to the Imgur API and returns its link.
func uploadToImgur(ctx context.Context, client *http.Client, clientID, path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)
	go func() {
		part, err := mw.CreateFormFile("image", filepath.Base(path))
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

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.imgur.com/3/image", pr)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Client-ID "+clientID)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("reading imgur response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("imgur returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var out struct {
		Data struct {
			Link string `json:"link"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("parsing imgur response: %w", err)
	}
	if out.Data.Link == "" {
		return "", fmt.Errorf("imgur response contains no link")
	}
	return out.Data.Link, nil
}
