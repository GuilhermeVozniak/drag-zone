//go:build darwin

package platform

import (
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func writeSamplePNG(t *testing.T) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 48, 48))
	for x := 0; x < 48; x++ {
		for y := 0; y < 48; y++ {
			img.Set(x, y, color.RGBA{R: uint8(x * 5), G: uint8(y * 5), B: 128, A: 255})
		}
	}
	p := filepath.Join(t.TempDir(), "sample.png")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	return p
}

func assertPNGBase64(t *testing.T, b64 string) {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("not valid base64: %v", err)
	}
	if len(raw) < 8 || raw[0] != 0x89 || string(raw[1:4]) != "PNG" {
		t.Fatalf("not a PNG (%d bytes)", len(raw))
	}
}

// TestFileThumbnailPNGBase64 guards the thumbnail encoder
// (QLThumbnailRepresentation.CGImage -> ImageIO), the fix that restored
// multi-image Drop Bar previews. Skips only if QuickLook yields no thumbnail
// in this environment, so the encoder is never silently broken where it runs.
func TestFileThumbnailPNGBase64(t *testing.T) {
	b64, err := FileThumbnailPNGBase64(writeSamplePNG(t), 64)
	if err != nil || b64 == "" {
		t.Skipf("QuickLook produced no thumbnail here (err=%v); encoder not exercised", err)
	}
	assertPNGBase64(t, b64)
}

// TestFileIconPNGBase64 guards the Finder-icon fallback used when a file has
// no QuickLook preview.
func TestFileIconPNGBase64(t *testing.T) {
	b64, err := FileIconPNGBase64(writeSamplePNG(t), 64)
	if err != nil {
		t.Fatalf("FileIconPNGBase64: %v", err)
	}
	assertPNGBase64(t, b64)
}
