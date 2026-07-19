package platform

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa -framework Carbon -framework ServiceManagement -framework ImageIO -framework QuickLookThumbnailing -framework CoreGraphics
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
	// ServicesAddFiles is called with file paths sent via the macOS Services menu.
	ServicesAddFiles func(paths []string)
	// DragEnded is called when a drag-out session finishes; completed is true
	// when the drop was accepted somewhere.
	DragEnded func(completed bool)
	// OpenSettings is called from the status item context menu.
	OpenSettings func()
	// GridVisibility reports native show/hide of the grid window.
	GridVisibility func(visible bool)
	// GridBeak reports the status icon's horizontal center relative to the
	// window's left edge, so the UI can point the popover beak at it.
	GridBeak func(x float64)
	// PopOutHotkey fires when the pop-out-Drop-Bar global hotkey is pressed.
	PopOutHotkey func()
	// DragActive reports whether a native file drag from Finder is
	// currently over the (already open) grid window, so the frontend can
	// show a drop-target overlay. Driven by the same global drag monitor
	// used for the menu-bar drag-reveal tab; gated by the drag-overlay
	// setting (SetDragOverlayEnabled).
	DragActive func(active bool)
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

// RegisterServices installs the app's macOS Services provider. Call after InitNative.
func RegisterServices() { C.dz_register_services() }

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

// FileThumbnailPNGBase64 returns a QuickLook content preview of the file
// (images, PDFs, videos, …) as base64 PNG data, or an error when the file
// has no preview; callers fall back to FileIconPNGBase64.
func FileThumbnailPNGBase64(path string, size int) (string, error) {
	cp := C.CString(path)
	defer C.free(unsafe.Pointer(cp))
	res := C.dz_file_thumbnail_png_base64(cp, C.int(size))
	if res == nil {
		return "", fmt.Errorf("no thumbnail for %s", path)
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

// Hotkey slots for SetHotkeyF.
const (
	HotkeySlotGrid   = 1 // toggle the grid
	HotkeySlotPopOut = 2 // pop out / dock the Drop Bar
)

// SetHotkeyF binds a global hotkey slot to F<n> (1-12); 0 disables the slot.
func SetHotkeyF(n, slot int) { C.dz_set_hotkey_f(C.int(n), C.int(slot)) }

// SetDragOverlayEnabled toggles the drag-target overlay behaviors: showing
// the grid when a file drag nears the menu bar, and the DragActive signal
// used for the "drop to add" overlay over an already-open grid.
func SetDragOverlayEnabled(enabled bool) { C.dz_set_drag_overlay_enabled(C.bool(enabled)) }

// StatusState values for SetStatusState, mirroring Dropzone's menu bar icon
// feedback.
const (
	StatusNormal  = 0
	StatusDrag    = 1
	StatusRunning = 2
	StatusSuccess = 3
	StatusFailure = 4
)

// SetStatusState switches the menu bar icon between its feedback states.
func SetStatusState(state int) { C.dz_set_status_state(C.int(state)) }

// ClipboardFilePaths returns file paths currently on the pasteboard, if any.
func ClipboardFilePaths() []string {
	res := C.dz_clipboard_file_paths()
	if res == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(res))
	var paths []string
	if err := json.Unmarshal([]byte(C.GoString(res)), &paths); err != nil {
		return nil
	}
	return paths
}

// OptionKeyDown reports whether the Option key is currently held, used to
// invert folder copy/move behavior and populate KEY_MODIFIERS for scripts.
func OptionKeyDown() bool { return bool(C.dz_option_key_down()) }

// SetPinned keeps the grid window visible across app deactivation, used by
// the popped-out Drop Bar mode.
func SetPinned(pinned bool) { C.dz_set_pinned(C.bool(pinned)) }

// SetPopoutFloating toggles the popped-out Drop Bar's floating, always-on-top
// window behavior and position memory across launches.
func SetPopoutFloating(on bool) { C.dz_set_popout_floating(C.bool(on)) }

// SetSettingsMode flips the shared window between the frameless popover grid
// chrome and a regular titled app window hosting the settings UI (centered,
// opaque, brought to the front). Hide-on-deactivate is suspended while on.
func SetSettingsMode(on bool) { C.dz_set_settings_mode(C.bool(on)) }

// SetDockVisible shows or hides the Dock icon (Regular vs Accessory
// activation policy).
func SetDockVisible(visible bool) { C.dz_set_dock_visible(C.bool(visible)) }

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

// HasScreenRecording reports whether the app currently has Screen Recording
// permission (macOS 10.15+). Pure check: never prompts the user.
func HasScreenRecording() bool { return bool(C.dz_has_screen_recording()) }

// RequestScreenRecording triggers the OS Screen Recording permission prompt
// if access hasn't already been granted or denied. Safe to call repeatedly;
// callers should re-check HasScreenRecording after the user responds.
func RequestScreenRecording() { C.dz_request_screen_recording() }

// OpenScreenRecordingSettings opens System Settings to the Screen Recording
// privacy pane, for use after the user dismisses or denies the prompt.
func OpenScreenRecordingSettings() { C.dz_open_screen_recording_settings() }

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

//export goServicesAddFiles
func goServicesAddFiles(cjson *C.char) {
	var paths []string
	if err := json.Unmarshal([]byte(C.GoString(cjson)), &paths); err != nil || len(paths) == 0 {
		return
	}
	handlersMu.RLock()
	fn := handlers.ServicesAddFiles
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

//export goGridBeak
func goGridBeak(x C.double) {
	handlersMu.RLock()
	fn := handlers.GridBeak
	handlersMu.RUnlock()
	if fn != nil {
		go fn(float64(x))
	}
}

//export goPopOutHotkey
func goPopOutHotkey() {
	handlersMu.RLock()
	fn := handlers.PopOutHotkey
	handlersMu.RUnlock()
	if fn != nil {
		go fn()
	}
}

//export goDragActive
func goDragActive(active C.bool) {
	handlersMu.RLock()
	fn := handlers.DragActive
	handlersMu.RUnlock()
	if fn != nil {
		go fn(bool(active))
	}
}
