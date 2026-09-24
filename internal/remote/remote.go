// Package remote lets other processes control a running player over a Unix
// domain socket. Go supports AF_UNIX on Windows 10 and later too, so one
// transport covers every platform.
//
// The protocol is one JSON Request per connection, answered by one JSON
// Response, each terminated by a newline:
//
//	{"cmd":"seek","args":["+10"]}
//	{"ok":true,"status":{"state":"playing",...}}
package remote

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"time"
)

// ErrNotRunning means no player is listening on the socket.
var ErrNotRunning = errors.New("no running encomplayer found")

// ErrInUse means another live player already owns the socket.
var ErrInUse = errors.New("another encomplayer is already listening")

// maxRequest bounds a request line; real requests are tiny.
const maxRequest = 64 << 10

// Request is one command for the player.
type Request struct {
	Cmd  string   `json:"cmd"`
	Args []string `json:"args,omitempty"`
}

// Response answers a Request. Status reflects the player after the command
// ran.
type Response struct {
	OK     bool    `json:"ok"`
	Error  string  `json:"error,omitempty"`
	Status *Status `json:"status,omitempty"`
}

// Status describes the player.
type Status struct {
	State       string  `json:"state"`
	Title       string  `json:"title,omitempty"`
	Artist      string  `json:"artist,omitempty"`
	Album       string  `json:"album,omitempty"`
	Path        string  `json:"path,omitempty"`
	Position    float64 `json:"position_seconds"`
	Duration    float64 `json:"duration_seconds"`
	Volume      int     `json:"volume"`
	Repeat      bool    `json:"repeat"`
	Random      bool    `json:"random"`
	Single      bool    `json:"single"`
	Consume     bool    `json:"consume"`
	QueueIndex  int     `json:"queue_index"`
	QueueLength int     `json:"queue_length"`

	Visualizer     string `json:"visualizer"`
	VisualizerFull bool   `json:"visualizer_full"`
}

// Handler runs a request inside the player and returns its answer.
type Handler func(Request) Response

// Server accepts control connections.
type Server struct {
	path   string
	ln     net.Listener
	handle Handler
}

// Listen claims the socket at path. A socket left behind by a player that
// crashed is replaced; one that still answers belongs to a live player and
// yields ErrInUse.
func Listen(path string, handle Handler) (*Server, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("remote control: %w", err)
	}
	if _, err := os.Stat(path); err == nil {
		if conn, err := net.DialTimeout("unix", path, 300*time.Millisecond); err == nil {
			conn.Close()
			return nil, ErrInUse
		}
		if err := os.Remove(path); err != nil {
			return nil, fmt.Errorf("remote control: remove stale socket: %w", err)
		}
	}

	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, fmt.Errorf("remote control: %w", err)
	}
	// Only this user may control the player. On Windows the call is a
	// no-op and the socket inherits the directory's access control.
	if err := os.Chmod(path, 0o600); err != nil {
		ln.Close()
		return nil, fmt.Errorf("remote control: %w", err)
	}
	return &Server{path: path, ln: ln, handle: handle}, nil
}

// Serve answers connections until Close is called.
func (s *Server) Serve() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.serveConn(conn)
	}
}

func (s *Server) serveConn(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(10 * time.Second))

	var resp Response
	line, err := bufio.NewReaderSize(conn, maxRequest).ReadSlice('\n')
	switch {
	case errors.Is(err, bufio.ErrBufferFull):
		resp = Response{Error: "request too large"}
	case err != nil:
		resp = Response{Error: "no request received"}
	default:
		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			resp = Response{Error: "malformed request: " + err.Error()}
		} else {
			resp = s.handle(req)
		}
	}
	// The client may have gone; there is nobody left to report to.
	_ = json.NewEncoder(conn).Encode(resp)
}

// Close stops listening and removes the socket file.
func (s *Server) Close() error {
	err := s.ln.Close()
	if rmErr := os.Remove(s.path); rmErr != nil && !errors.Is(rmErr, fs.ErrNotExist) {
		err = errors.Join(err, rmErr)
	}
	return err
}

// Send delivers req to the player listening at path.
func Send(ctx context.Context, path string, req Request) (Response, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) || isRefused(err) {
			return Response{}, ErrNotRunning
		}
		return Response{}, fmt.Errorf("connect to player: %w", err)
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return Response{}, fmt.Errorf("send to player: %w", err)
	}
	var resp Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return Response{}, fmt.Errorf("read player reply: %w", err)
	}
	return resp, nil
}

// isRefused reports a socket file with no listener behind it, which is what
// a crashed player leaves.
func isRefused(err error) bool {
	var op *net.OpError
	if errors.As(err, &op) {
		var sys *os.SyscallError
		if errors.As(op.Err, &sys) {
			return sys.Syscall == "connect"
		}
	}
	return false
}
