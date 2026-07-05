#ifndef DZ_BRIDGE_DARWIN_H
#define DZ_BRIDGE_DARWIN_H

#include <stdbool.h>

// Initializes the native layer: accessory activation policy, status item,
// window observers, and global drag monitors. windowTitle identifies the
// Wails grid window. Safe to call once after the app has started.
void dz_init(const char *windowTitle);

void dz_show_grid(bool activate);
void dz_hide_grid(void);
void dz_toggle_grid(void);
bool dz_grid_visible(void);

// Begins a native dragging session for the given file paths (JSON array of
// strings). Must be triggered from a recent mouse event in the webview.
void dz_start_drag(const char *pathsJSON);

// Shares files via AirDrop. pathsJSON is a JSON array of absolute paths.
// Returns 0 on success.
int dz_airdrop(const char *pathsJSON);

// Returns a malloc'd base64 PNG of the file's Finder icon at size*size
// points, or NULL. Caller frees.
char *dz_file_icon_png_base64(const char *path, int size);

// Registers/unregisters the app as a login item. Returns 0 on success,
// -1 when unsupported, -2 on failure.
int dz_set_login_item(bool enabled);

// Registers a global hotkey on the given function key (1-12); 0 unregisters.
void dz_set_hotkey_f(int fkey);

// Reports whether the Option key is currently held.
bool dz_option_key_down(void);

// Pinned mode keeps the grid window visible when the app deactivates
// (used by the popped-out Drop Bar).
void dz_set_pinned(bool pinned);

// Rewrites the image at src to dst with EXIF/GPS/TIFF/IPTC metadata removed.
// Returns 0 on success.
int dz_strip_image_metadata(const char *src, const char *dst);

#endif
