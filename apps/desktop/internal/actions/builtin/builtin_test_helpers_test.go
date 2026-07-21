package builtin

import "testing"

// withScreenRecordingGranted stubs hasScreenRecording to report access is
// already granted, so Screenshot/ScreenshotSFTP tests can exercise the
// capture path without depending on the real (cgo, macOS-only) permission
// state. Restores the original seam on cleanup.
func withScreenRecordingGranted(t *testing.T) {
	t.Helper()
	origHas := hasScreenRecording
	hasScreenRecording = func() bool { return true }
	t.Cleanup(func() { hasScreenRecording = origHas })
}

// recServices is a recording actions.Services fake shared across builtin tests.
type recServices struct {
	Clipboard    string
	ClipboardErr error
	ClipFiles    []string
	Trashed      [][]string
	TrashErr     error
	AirDropped   [][]string
	AirDropErr   error
	Notes        []string
	Sounds       []string
	Opened       []string
	ReadClip     string
}

func (r *recServices) CopyToClipboard(s string) error {
	if r.ClipboardErr != nil {
		return r.ClipboardErr
	}
	r.Clipboard = s
	return nil
}
func (r *recServices) ReadClipboard() (string, error) { return r.ReadClip, nil }
func (r *recServices) CopyFilesToClipboard(paths []string) error {
	if r.ClipboardErr != nil {
		return r.ClipboardErr
	}
	r.ClipFiles = append(r.ClipFiles, paths...)
	return nil
}
func (r *recServices) Notify(title, body string) { r.Notes = append(r.Notes, title+"|"+body) }
func (r *recServices) PlaySound(name string)     { r.Sounds = append(r.Sounds, name) }
func (r *recServices) OpenURL(u string) error    { r.Opened = append(r.Opened, u); return nil }
func (r *recServices) OpenPath(p string) error   { r.Opened = append(r.Opened, p); return nil }
func (r *recServices) Reveal(p string) error     { r.Opened = append(r.Opened, p); return nil }
func (r *recServices) Trash(p []string) error {
	if r.TrashErr != nil {
		return r.TrashErr
	}
	r.Trashed = append(r.Trashed, p)
	return nil
}
func (r *recServices) AirDrop(p []string) error {
	if r.AirDropErr != nil {
		return r.AirDropErr
	}
	r.AirDropped = append(r.AirDropped, p)
	return nil
}
