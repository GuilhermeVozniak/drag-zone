package builtin

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/jlaffaye/ftp"
	"golang.org/x/crypto/ssh"

	"dragzone/internal/actions"
	"dragzone/internal/model"
)

func TestCollectUploadEntriesFlattensDirs(t *testing.T) {
	root := t.TempDir()
	// A single loose file plus a directory tree.
	loose := filepath.Join(root, "loose.txt")
	if err := os.WriteFile(loose, []byte("12345"), 0o644); err != nil {
		t.Fatal(err)
	}
	tree := filepath.Join(root, "tree")
	if err := os.MkdirAll(filepath.Join(tree, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tree, "a.txt"), []byte("ab"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tree, "sub", "b.txt"), []byte("c"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, total, err := collectUploadEntries([]string{loose, tree})
	if err != nil {
		t.Fatal(err)
	}
	if total != 5+2+1 {
		t.Errorf("total bytes = %d, want 8", total)
	}
	var rels []string
	for _, e := range entries {
		rels = append(rels, e.rel)
	}
	sort.Strings(rels)
	want := []string{"loose.txt", "tree/a.txt", "tree/sub/b.txt"}
	if len(rels) != 3 || rels[0] != want[0] || rels[1] != want[1] || rels[2] != want[2] {
		t.Errorf("rels = %v, want %v", rels, want)
	}
}

func TestCollectUploadEntriesErrors(t *testing.T) {
	if _, _, err := collectUploadEntries([]string{"/does/not/exist/xyz"}); err == nil {
		t.Error("missing path should error")
	}
	if _, _, err := collectUploadEntries([]string{t.TempDir()}); err == nil {
		t.Error("empty dir (no files) should error with 'nothing to upload'")
	}
}

// fakeRemote records mkdirAll/upload calls in place of a live FTP/SFTP
// connection, so tests can exercise FTPUpload.Dropped end to end without any
// network I/O.
type fakeRemote struct {
	mkdirs  []string
	uploads []string // remote paths, in upload order
	bodies  map[string][]byte
	closed  bool
}

func (f *fakeRemote) mkdirAll(dir string) error {
	f.mkdirs = append(f.mkdirs, dir)
	return nil
}

func (f *fakeRemote) upload(remotePath string, r io.Reader) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	if f.bodies == nil {
		f.bodies = map[string][]byte{}
	}
	f.uploads = append(f.uploads, remotePath)
	f.bodies[remotePath] = b
	return nil
}

func (f *fakeRemote) close() error {
	f.closed = true
	return nil
}

// withFakeRemote overrides connectRemote to return fr instead of dialing a
// live server, restoring the original (network-backed) implementation when
// the test ends.
func withFakeRemote(t *testing.T, fr *fakeRemote) {
	t.Helper()
	old := connectRemote
	connectRemote = func(ctx context.Context, protocol, host, port, user, pass string) (remoteFS, error) {
		return fr, nil
	}
	t.Cleanup(func() { connectRemote = old })
}

func ftpBaseOptions() map[string]string {
	return map[string]string{"host": "example.com", "username": "bob", "password": "secret"}
}

func TestFTPUploadMissingCredentials(t *testing.T) {
	base := ftpBaseOptions()
	for _, missing := range []string{"host", "username", "password"} {
		opts := map[string]string{}
		for k, v := range base {
			if k != missing {
				opts[k] = v
			}
		}
		_, err := FTPUpload{}.Dropped(context.Background(), actions.Invocation{
			Target:   model.Target{Options: opts},
			Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{"/tmp/whatever.txt"}},
			Progress: nullProgress{},
			Services: &recServices{},
		})
		if err == nil {
			t.Errorf("missing %s should error", missing)
		}
	}
}

func TestFTPUploadNoPaths(t *testing.T) {
	_, err := FTPUpload{}.Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: ftpBaseOptions()},
		Payload:  model.Payload{Kind: model.ItemFiles},
		Progress: nullProgress{},
		Services: &recServices{},
	})
	if err == nil {
		t.Error("empty payload should error")
	}
}

func TestFTPUploadURLPrefixAndClipboard(t *testing.T) {
	src := filepath.Join(t.TempDir(), "pic.txt")
	if err := os.WriteFile(src, []byte("hello ftp"), 0o644); err != nil {
		t.Fatal(err)
	}

	fr := &fakeRemote{}
	withFakeRemote(t, fr)

	svc := &recServices{}
	opts := ftpBaseOptions()
	opts["url_prefix"] = "https://cdn.example.com/uploads/"
	res, err := FTPUpload{}.Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: opts},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{src}},
		Progress: nullProgress{},
		Services: svc,
	})
	if err != nil {
		t.Fatalf("Dropped: %v", err)
	}

	wantURL := "https://cdn.example.com/uploads/pic.txt"
	if res.URL != wantURL {
		t.Errorf("result URL = %q, want %q", res.URL, wantURL)
	}
	if svc.Clipboard != wantURL {
		t.Errorf("clipboard = %q, want %q", svc.Clipboard, wantURL)
	}
	if len(fr.uploads) != 1 || fr.uploads[0] != "pic.txt" {
		t.Errorf("uploads = %v, want [pic.txt]", fr.uploads)
	}
	if string(fr.bodies["pic.txt"]) != "hello ftp" {
		t.Errorf("uploaded body = %q", fr.bodies["pic.txt"])
	}
	if !fr.closed {
		t.Error("remote connection was not closed")
	}
}

func TestFTPUploadNoURLPrefixSkipsClipboard(t *testing.T) {
	src := filepath.Join(t.TempDir(), "pic.txt")
	if err := os.WriteFile(src, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	fr := &fakeRemote{}
	withFakeRemote(t, fr)

	svc := &recServices{}
	res, err := FTPUpload{}.Dropped(context.Background(), actions.Invocation{
		Target:   model.Target{Options: ftpBaseOptions()},
		Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{src}},
		Progress: nullProgress{},
		Services: svc,
	})
	if err != nil {
		t.Fatalf("Dropped: %v", err)
	}
	if res.URL != "" {
		t.Errorf("URL = %q, want empty when no url_prefix configured", res.URL)
	}
	if svc.Clipboard != "" {
		t.Errorf("clipboard = %q, want untouched", svc.Clipboard)
	}
}

// TestFTPUploadOptionModifierZipsFirst mirrors S3's Option-modifier behavior:
// holding Option while dropping multiple files zips them into a single
// archive and uploads exactly that one file, instead of each file
// individually. Verified through the same fakeRemote seam used above, so no
// live server is involved.
func TestFTPUploadOptionModifierZipsFirst(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("content-"+name), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	fr := &fakeRemote{}
	withFakeRemote(t, fr)

	_, err := FTPUpload{}.Dropped(context.Background(), actions.Invocation{
		Target: model.Target{Options: ftpBaseOptions()},
		Payload: model.Payload{
			Kind:      model.ItemFiles,
			Paths:     []string{filepath.Join(dir, "a.txt"), filepath.Join(dir, "b.txt")},
			Modifiers: []string{"Option"},
		},
		Progress: nullProgress{},
		Services: &recServices{},
	})
	if err != nil {
		t.Fatalf("Dropped: %v", err)
	}

	if len(fr.uploads) != 1 {
		t.Fatalf("upload count = %d, want 1 (single zip upload)", len(fr.uploads))
	}
	if filepath.Ext(fr.uploads[0]) != ".zip" {
		t.Errorf("uploaded path %q is not a .zip", fr.uploads[0])
	}

	body := fr.bodies[fr.uploads[0]]
	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		t.Fatalf("uploaded body is not a valid zip: %v", err)
	}
	if len(zr.File) != 2 {
		t.Errorf("zip entries = %d, want 2", len(zr.File))
	}
}

func TestFTPUploadOptionModifierAbsentUploadsEachFile(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	fr := &fakeRemote{}
	withFakeRemote(t, fr)

	_, err := FTPUpload{}.Dropped(context.Background(), actions.Invocation{
		Target: model.Target{Options: ftpBaseOptions()},
		Payload: model.Payload{
			Kind:  model.ItemFiles,
			Paths: []string{filepath.Join(dir, "a.txt"), filepath.Join(dir, "b.txt")},
		},
		Progress: nullProgress{},
		Services: &recServices{},
	})
	if err != nil {
		t.Fatalf("Dropped: %v", err)
	}
	if len(fr.uploads) != 2 {
		t.Fatalf("upload count = %d, want 2 (no zipping without Option)", len(fr.uploads))
	}
	sort.Strings(fr.uploads)
	if fr.uploads[0] != "a.txt" || fr.uploads[1] != "b.txt" {
		t.Errorf("uploads = %v, want [a.txt b.txt]", fr.uploads)
	}
}

// TestFTPConnectRemoteDialFailureWrapped exercises connectRemote's real
// (non-faked) dial path: connecting to a port nothing is listening on fails
// fast (connection refused) without requiring a live FTP/SFTP server, and
// Dropped must wrap that failure with "connecting to %s: %w".
func TestFTPConnectRemoteDialFailureWrapped(t *testing.T) {
	// Bind then immediately close a loopback listener to obtain a port that
	// is guaranteed nothing is listening on, so the dial fails fast with
	// "connection refused" instead of timing out.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatal(err)
	}

	src := filepath.Join(t.TempDir(), "a.txt")
	if err := os.WriteFile(src, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, protocol := range []string{"ftp", "sftp"} {
		opts := map[string]string{
			"host": host, "port": port, "username": "bob", "password": "secret", "protocol": protocol,
		}
		_, err := FTPUpload{}.Dropped(context.Background(), actions.Invocation{
			Target:   model.Target{Options: opts},
			Payload:  model.Payload{Kind: model.ItemFiles, Paths: []string{src}},
			Progress: nullProgress{},
			Services: &recServices{},
		})
		if err == nil {
			t.Fatalf("protocol %s: expected dial failure, got nil error", protocol)
		}
		wantPrefix := "connecting to " + host + ":"
		if !strings.HasPrefix(err.Error(), wantPrefix) {
			t.Errorf("protocol %s: error = %q, want prefix %q", protocol, err.Error(), wantPrefix)
		}
		if errors.Unwrap(err) == nil {
			t.Errorf("protocol %s: error not wrapped (%%w), Unwrap returned nil", protocol)
		}
	}
}

// TestFTPConnectRemoteDispatchesByProtocol proves connectRemote calls the FTP
// dialer for protocol "ftp" and the SSH/SFTP dialer for everything else
// (including the "sftp" default). dialFTPServer/dialSSHServer are swapped for
// recording fakes that return immediately with a sentinel error, so this is
// fully deterministic and involves no network I/O at all — a real dial (even
// one that fails fast) is unnecessary to prove which branch ran, and the SSH
// client's handshake has no read deadline, so pointing it at a wrong-protocol
// listener can hang indefinitely rather than failing fast.
func TestFTPConnectRemoteDispatchesByProtocol(t *testing.T) {
	ftpSentinel := errors.New("ftp dialer sentinel")
	sshSentinel := errors.New("ssh dialer sentinel")

	var ftpCalls, sshCalls int
	oldFTP, oldSSH := dialFTPServer, dialSSHServer
	dialFTPServer = func(ctx context.Context, addr string) (*ftp.ServerConn, error) {
		ftpCalls++
		return nil, ftpSentinel
	}
	dialSSHServer = func(addr, user, pass string) (*ssh.Client, error) {
		sshCalls++
		return nil, sshSentinel
	}
	t.Cleanup(func() { dialFTPServer, dialSSHServer = oldFTP, oldSSH })

	if _, err := connectRemote(context.Background(), "ftp", "h", "21", "bob", "secret"); !errors.Is(err, ftpSentinel) {
		t.Errorf("protocol=ftp: err = %v, want ftpSentinel", err)
	}
	if ftpCalls != 1 || sshCalls != 0 {
		t.Errorf("protocol=ftp: ftpCalls=%d sshCalls=%d, want 1,0", ftpCalls, sshCalls)
	}

	ftpCalls, sshCalls = 0, 0
	if _, err := connectRemote(context.Background(), "sftp", "h", "22", "bob", "secret"); !errors.Is(err, sshSentinel) {
		t.Errorf("protocol=sftp: err = %v, want sshSentinel", err)
	}
	if sshCalls != 1 || ftpCalls != 0 {
		t.Errorf("protocol=sftp: ftpCalls=%d sshCalls=%d, want 0,1", ftpCalls, sshCalls)
	}

	// The default (unrecognized/empty protocol) also routes to sftp.
	ftpCalls, sshCalls = 0, 0
	if _, err := connectRemote(context.Background(), "", "h", "22", "bob", "secret"); !errors.Is(err, sshSentinel) {
		t.Errorf("protocol=(default): err = %v, want sshSentinel", err)
	}
	if sshCalls != 1 || ftpCalls != 0 {
		t.Errorf("protocol=(default): ftpCalls=%d sshCalls=%d, want 0,1", ftpCalls, sshCalls)
	}
}
