// Package ipc provides the local control channel used by the `dz` command
// line tool: a unix socket in the app data directory carrying one JSON
// request/response pair per connection.
package ipc

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"dragzone/internal/storage"
)

const socketName = "dragzone.sock"

// Request is a single CLI command.
type Request struct {
	Cmd   string          `json:"cmd"`
	Args  []string        `json:"args,omitempty"`
	Flags map[string]bool `json:"flags,omitempty"`
}

// Response wraps a command result.
type Response struct {
	OK    bool            `json:"ok"`
	Error string          `json:"error,omitempty"`
	Data  json.RawMessage `json:"data,omitempty"`
}

// SocketPath returns the control socket location.
func SocketPath() (string, error) {
	dir, err := storage.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, socketName), nil
}

// Handler executes one request and returns the data to serialize.
type Handler func(req Request) (any, error)

// Server accepts CLI connections.
type Server struct {
	ln net.Listener
}

// Serve starts listening on the control socket, replacing any stale socket
// file from a previous run.
func Serve(handler Handler) (*Server, error) {
	path, err := SocketPath()
	if err != nil {
		return nil, err
	}
	os.Remove(path)
	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("listening on control socket: %w", err)
	}
	// The socket accepts commands that act on the user's files; keep it
	// owner-only.
	if err := os.Chmod(path, 0o600); err != nil {
		ln.Close()
		return nil, fmt.Errorf("securing control socket: %w", err)
	}
	s := &Server{ln: ln}
	go s.accept(handler)
	return s, nil
}

func (s *Server) accept(handler Handler) {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return // listener closed
		}
		go serveConn(conn, handler)
	}
}

func serveConn(conn net.Conn, handler Handler) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))

	var req Request
	if err := json.NewDecoder(conn).Decode(&req); err != nil {
		_ = json.NewEncoder(conn).Encode(Response{Error: "bad request: " + err.Error()})
		return
	}
	data, err := handler(req)
	if err != nil {
		_ = json.NewEncoder(conn).Encode(Response{Error: err.Error()})
		return
	}
	raw, err := json.Marshal(data)
	if err != nil {
		_ = json.NewEncoder(conn).Encode(Response{Error: err.Error()})
		return
	}
	_ = json.NewEncoder(conn).Encode(Response{OK: true, Data: raw})
}

// Close stops the server and removes the socket file.
func (s *Server) Close() {
	s.ln.Close()
	if path, err := SocketPath(); err == nil {
		os.Remove(path)
	}
}

// Call sends one request to the running app and returns the response data.
func Call(req Request) (json.RawMessage, error) {
	path, err := SocketPath()
	if err != nil {
		return nil, err
	}
	conn, err := net.DialTimeout("unix", path, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("DragZone is not running")
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return nil, err
	}
	var resp Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return nil, err
	}
	if !resp.OK {
		return nil, fmt.Errorf("%s", resp.Error)
	}
	return resp.Data, nil
}
