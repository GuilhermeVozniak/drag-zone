package builtin

// recServices is a recording actions.Services fake shared across builtin tests.
type recServices struct {
	Clipboard    string
	ClipboardErr error
	Trashed      [][]string
	AirDropped   [][]string
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
func (r *recServices) Notify(title, body string)      { r.Notes = append(r.Notes, title+"|"+body) }
func (r *recServices) PlaySound(name string)          { r.Sounds = append(r.Sounds, name) }
func (r *recServices) OpenURL(u string) error         { r.Opened = append(r.Opened, u); return nil }
func (r *recServices) OpenPath(p string) error        { r.Opened = append(r.Opened, p); return nil }
func (r *recServices) Reveal(p string) error          { r.Opened = append(r.Opened, p); return nil }
func (r *recServices) Trash(p []string) error         { r.Trashed = append(r.Trashed, p); return nil }
func (r *recServices) AirDrop(p []string) error       { r.AirDropped = append(r.AirDropped, p); return nil }
