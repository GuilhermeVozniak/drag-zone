package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/mac"

	"dragzone/internal/platform"
)

//go:embed all:frontend/dist
var assets embed.FS

const (
	windowWidth  = 360
	windowHeight = 160 // small starting height; the frontend resizes it to fit content
)

// appVersion is shown in the Updates tab. Release builds inject the git tag
// via -ldflags "-X main.appVersion=<version>" (see .github/workflows/release.yml).
var appVersion = "0.2.0"

func main() {
	app, err := NewApp(platform.Services{})
	if err != nil {
		log.Fatalf("initializing app: %v", err)
	}

	err = wails.Run(&options.App{
		Title:            "DragZone",
		Width:            windowWidth,
		Height:           windowHeight,
		Frameless:        true,
		DisableResize:    true,
		AlwaysOnTop:      true,
		StartHidden:      true,
		BackgroundColour: &options.RGBA{R: 0, G: 0, B: 0, A: 0},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		Mac: &mac.Options{
			WebviewIsTransparent: true,
			// NOTE: no WindowIsTranslucent — it makes Wails insert an
			// NSVisualEffectView behind the webview, a frosted dark square
			// that shows behind the panel's rounded corners.
		},
		DragAndDrop: &options.DragAndDrop{
			EnableFileDrop: true,
		},
		OnStartup: app.startup,
		// The close button only exists in settings mode (the popover is
		// frameless); beforeClose turns it into "close settings" there.
		OnBeforeClose: app.beforeClose,
		OnShutdown:    app.shutdown,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		log.Fatalf("running app: %v", err)
	}
}
