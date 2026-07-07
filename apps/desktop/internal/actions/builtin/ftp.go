package builtin

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

// FTPUpload uploads dropped files and folders to a remote server over SFTP or
// FTP, optionally copying a public URL for the first item to the clipboard.
type FTPUpload struct{}

func (FTPUpload) Spec() model.ActionSpec {
	return model.ActionSpec{
		ID:          "ftp-upload",
		Name:        "FTP / SFTP Upload",
		Description: "Upload dropped files to a server over SFTP or FTP.",
		Icon:        "upload",
		Category:    "Uploads",
		Events:      []string{model.EventDragged},
		Accepts:     []model.ItemKind{model.ItemFiles},
		Multi:       true,
		Options: []model.OptionField{
			{Key: "protocol", Label: "Protocol", Type: "select", Choices: []string{"sftp", "ftp"}, Default: "sftp"},
			{Key: "host", Label: "Host", Type: "text", Required: true},
			{Key: "port", Label: "Port", Type: "text", Placeholder: "22 / 21"},
			{Key: "username", Label: "Username", Type: "text", Required: true},
			{Key: "password", Label: "Password", Type: "password", Required: true},
			{Key: "remote_dir", Label: "Remote directory", Type: "text", Placeholder: "/var/www/uploads"},
			{Key: "url_prefix", Label: "URL prefix (optional)", Type: "text", Placeholder: "https://example.com/uploads"},
		},
	}
}

func (FTPUpload) Dropped(ctx context.Context, inv actions.Invocation) (actions.Result, error) {
	host := inv.Target.Option("host", "")
	user := inv.Target.Option("username", "")
	pass := inv.Target.Option("password", "")
	if host == "" || user == "" || pass == "" {
		return actions.Result{}, fmt.Errorf("host, username and password must be configured")
	}
	if len(inv.Payload.Paths) == 0 {
		return actions.Result{}, fmt.Errorf("nothing to upload")
	}

	protocol := inv.Target.Option("protocol", "sftp")
	port := inv.Target.Option("port", "")
	if port == "" {
		if protocol == "ftp" {
			port = "21"
		} else {
			port = "22"
		}
	}

	entries, total, err := collectUploadEntries(inv.Payload.Paths)
	if err != nil {
		return actions.Result{}, err
	}

	remote, err := connectRemote(ctx, protocol, host, port, user, pass)
	if err != nil {
		return actions.Result{}, fmt.Errorf("connecting to %s: %w", host, err)
	}
	defer remote.close()

	remoteDir := inv.Target.Option("remote_dir", "")
	var done int64
	made := map[string]bool{}
	for _, e := range entries {
		inv.Progress.Detail(path.Base(e.rel))
		rp := e.rel
		if remoteDir != "" {
			rp = path.Join(remoteDir, e.rel)
		}
		if dir := path.Dir(rp); dir != "." && dir != "/" && !made[dir] {
			if err := remote.mkdirAll(dir); err != nil {
				return actions.Result{}, fmt.Errorf("creating remote directory %s: %w", dir, err)
			}
			made[dir] = true
		}
		if err := uploadOne(remote, e, rp, total, &done, inv.Progress); err != nil {
			return actions.Result{}, fmt.Errorf("uploading %s: %w", e.rel, err)
		}
	}

	result := actions.Result{Message: fmt.Sprintf("Uploaded %d item(s) to %s", len(inv.Payload.Paths), host)}
	if prefix := inv.Target.Option("url_prefix", ""); prefix != "" {
		result.URL = strings.TrimRight(prefix, "/") + "/" + filepath.Base(inv.Payload.Paths[0])
		if err := inv.Services.CopyToClipboard(result.URL); err != nil {
			return actions.Result{}, fmt.Errorf("copying URL to clipboard: %w", err)
		}
	}
	return result, nil
}

// uploadOne streams one local file to the remote path, updating progress.
func uploadOne(remote remoteFS, e uploadEntry, remotePath string, total int64, done *int64, progress actions.Progress) error {
	f, err := os.Open(e.local)
	if err != nil {
		return err
	}
	defer f.Close()
	pr := &progressReader{r: f, onBytes: func(n int64) {
		if total > 0 {
			*done += n
			progress.Percent(int(*done * 100 / total))
		}
	}}
	return remote.upload(remotePath, pr)
}

// remoteFS abstracts the subset of remote operations FTPUpload needs, so SFTP
// and FTP share one upload loop.
type remoteFS interface {
	mkdirAll(dir string) error
	upload(remotePath string, r io.Reader) error
	close() error
}

// connectRemote dials the server using the selected protocol.
func connectRemote(ctx context.Context, protocol, host, port, user, pass string) (remoteFS, error) {
	addr := net.JoinHostPort(host, port)
	switch protocol {
	case "ftp":
		conn, err := ftp.Dial(addr, ftp.DialWithContext(ctx), ftp.DialWithTimeout(15*time.Second))
		if err != nil {
			return nil, err
		}
		if err := conn.Login(user, pass); err != nil {
			_ = conn.Quit()
			return nil, fmt.Errorf("logging in as %s: %w", user, err)
		}
		return &ftpFS{conn: conn}, nil
	default: // sftp
		sshConn, err := ssh.Dial("tcp", addr, &ssh.ClientConfig{
			User:            user,
			Auth:            []ssh.AuthMethod{ssh.Password(pass)},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         15 * time.Second,
		})
		if err != nil {
			return nil, err
		}
		client, err := sftp.NewClient(sshConn)
		if err != nil {
			sshConn.Close()
			return nil, fmt.Errorf("opening sftp session: %w", err)
		}
		return &sftpFS{ssh: sshConn, client: client}, nil
	}
}

// sftpFS implements remoteFS over an SSH/SFTP connection.
type sftpFS struct {
	ssh    *ssh.Client
	client *sftp.Client
}

func (s *sftpFS) mkdirAll(dir string) error { return s.client.MkdirAll(dir) }

func (s *sftpFS) upload(remotePath string, r io.Reader) error {
	dst, err := s.client.Create(remotePath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(dst, r); err != nil {
		dst.Close()
		return err
	}
	return dst.Close()
}

func (s *sftpFS) close() error {
	s.client.Close()
	return s.ssh.Close()
}

// ftpFS implements remoteFS over a plain FTP connection.
type ftpFS struct {
	conn *ftp.ServerConn
}

func (f *ftpFS) mkdirAll(dir string) error {
	cur := ""
	if strings.HasPrefix(dir, "/") {
		cur = "/"
	}
	for _, part := range strings.Split(dir, "/") {
		if part == "" {
			continue
		}
		cur = path.Join(cur, part)
		// MakeDir fails when the directory already exists; a genuinely missing
		// directory surfaces as an error from the subsequent upload.
		_ = f.conn.MakeDir(cur)
	}
	return nil
}

func (f *ftpFS) upload(remotePath string, r io.Reader) error {
	return f.conn.Stor(remotePath, r)
}

func (f *ftpFS) close() error { return f.conn.Quit() }

// uploadEntry is one local file queued for upload.
type uploadEntry struct {
	local string // absolute local path
	rel   string // slash-separated remote path relative to the destination root
	size  int64
}

// collectUploadEntries expands dropped paths into the flat list of files to
// upload (recursing into directories) and their total size in bytes.
func collectUploadEntries(roots []string) ([]uploadEntry, int64, error) {
	var entries []uploadEntry
	var total int64
	for _, root := range roots {
		info, err := os.Stat(root)
		if err != nil {
			return nil, 0, fmt.Errorf("reading %s: %w", filepath.Base(root), err)
		}
		if !info.IsDir() {
			entries = append(entries, uploadEntry{local: root, rel: filepath.Base(root), size: info.Size()})
			total += info.Size()
			continue
		}
		base := filepath.Dir(root)
		walkErr := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			fi, err := d.Info()
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(base, p)
			if err != nil {
				return err
			}
			entries = append(entries, uploadEntry{local: p, rel: filepath.ToSlash(rel), size: fi.Size()})
			total += fi.Size()
			return nil
		})
		if walkErr != nil {
			return nil, 0, fmt.Errorf("scanning %s: %w", filepath.Base(root), walkErr)
		}
	}
	if len(entries) == 0 {
		return nil, 0, fmt.Errorf("nothing to upload")
	}
	return entries, total, nil
}

// progressReader counts bytes as they are read, for byte-based progress.
type progressReader struct {
	r       io.Reader
	onBytes func(int64)
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		p.onBytes(int64(n))
	}
	return n, err
}
