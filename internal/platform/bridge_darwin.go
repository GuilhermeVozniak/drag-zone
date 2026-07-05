package platform

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework Carbon -framework ServiceManagement -framework ImageIO
#include <stdlib.h>
#include "bridge_darwin.h"
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"sync"
	"unsafe"
)

// Handlers receive callbacks from the native layer.
type Handlers struct {
	// StatusDropped is called with file paths dropped onto the menu bar icon.
	StatusDropped func(paths []string)
	// DragEnded is called when a drag-out session finishes; completed is true
	// when the drop was accepted somewhere.
	DragEnded func(completed bool)
	// OpenSettings is called from the status item context menu.
	OpenSettings func()
	// GridVisibility reports native show/hide of the grid window.
	GridVisibility func(visible bool)
}

var (
	handlersMu sync.RWMutex
	handlers   Handlers
)

// SetHandlers installs the native callback handlers. Call before InitNative.
func SetHandlers(h Handlers) {
	handlersMu.Lock()
	defer handlersMu.Unlock()
	handlers = h
}

// InitNative sets up the status item, activation policy, and monitors.
// windowTitle must match the Wails window title.
func InitNative(windowTitle string) {
	ct := C.CString(windowTitle)
	defer C.free(unsafe.Pointer(ct))
	C.dz_init(ct)
}

// ShowGrid shows the grid window under the status item. activate gives it
// keyboard focus; passive display is used during drags.
func ShowGrid(activate bool) { C.dz_show_grid(C.bool(activate)) }

// HideGrid hides the grid window.
func HideGrid() { C.dz_hide_grid() }

// ToggleGrid toggles grid visibility.
func ToggleGrid() { C.dz_toggle_grid() }

// StartDrag begins a native drag session for the given files. The result is
// reported through Handlers.DragEnded.
func StartDrag(paths []string) error {
	data, err := json.Marshal(paths)
	if err != nil {
		return err
	}
	cj := C.CString(string(data))
	defer C.free(unsafe.Pointer(cj))
	C.dz_start_drag(cj)
	return nil
}

// FileIconPNGBase64 returns the Finder icon for path as base64 PNG data.
func FileIconPNGBase64(path string, size int) (string, error) {
	cp := C.CString(path)
	defer C.free(unsafe.Pointer(cp))
	res := C.dz_file_icon_png_base64(cp, C.int(size))
	if res == nil {
		return "", fmt.Errorf("no icon for %s", path)
	}
	defer C.free(unsafe.Pointer(res))
	return C.GoString(res), nil
}

// SetLoginItem registers or unregisters the app as a login item.
func SetLoginItem(enabled bool) error {
	switch C.dz_set_login_item(C.bool(enabled)) {
	case 0:
		return nil
	case -1:
		return fmt.Errorf("login items require macOS 13 or later")
	default:
		return fmt.Errorf("updating login item failed (app may need to be installed in /Applications)")
	}
}

// SetHotkeyF binds the global toggle-grid hotkey to F<n> (1-12); 0 disables.
func SetHotkeyF(n int) { C.dz_set_hotkey_f(C.int(n)) }

// OptionKeyDown reports whether the Option key is currently held, used to
// invert folder copy/move behavior and populate KEY_MODIFIERS for scripts.
func OptionKeyDown() bool { return bool(C.dz_option_key_down()) }

// SetPinned keeps the grid window visible across app deactivation, used by
// the popped-out Drop Bar mode.
func SetPinned(pinned bool) { C.dz_set_pinned(C.bool(pinned)) }

// StripImageMetadata rewrites src to dst without EXIF/GPS/TIFF/IPTC metadata.
func StripImageMetadata(src, dst string) error {
	cs, cd := C.CString(src), C.CString(dst)
	defer C.free(unsafe.Pointer(cs))
	defer C.free(unsafe.Pointer(cd))
	if rc := C.dz_strip_image_metadata(cs, cd); rc != 0 {
		return fmt.Errorf("stripping metadata failed (code %d)", int(rc))
	}
	return nil
}

// airDrop shares files via AirDrop using the native sharing service.
func airDrop(paths []string) error {
	data, err := json.Marshal(paths)
	if err != nil {
		return err
	}
	cj := C.CString(string(data))
	defer C.free(unsafe.Pointer(cj))
	if C.dz_airdrop(cj) != 0 {
		return fmt.Errorf("nothing to share")
	}
	return nil
}

//export goStatusDropped
func goStatusDropped(cjson *C.char) {
	var paths []string
	if err := json.Unmarshal([]byte(C.GoString(cjson)), &paths); err != nil || len(paths) == 0 {
		return
	}
	handlersMu.RLock()
	fn := handlers.StatusDropped
	handlersMu.RUnlock()
	if fn != nil {
		go fn(paths)
	}
}

//export goDragSessionEnded
func goDragSessionEnded(completed C.bool) {
	handlersMu.RLock()
	fn := handlers.DragEnded
	handlersMu.RUnlock()
	if fn != nil {
		go fn(bool(completed))
	}
}

//export goOpenSettings
func goOpenSettings() {
	handlersMu.RLock()
	fn := handlers.OpenSettings
	handlersMu.RUnlock()
	if fn != nil {
		go fn()
	}
}

//export goGridVisibility
func goGridVisibility(visible C.bool) {
	handlersMu.RLock()
	fn := handlers.GridVisibility
	handlersMu.RUnlock()
	if fn != nil {
		go fn(bool(visible))
	}
}
