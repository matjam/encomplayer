package remote

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// socketPath returns a short path: macOS limits socket paths to 104 bytes,
// which a test's TempDir can exceed.
func socketPath(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "ep")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return filepath.Join(dir, "p.sock")
}

func ctx(t *testing.T) context.Context {
	c, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return c
}

func echo(req Request) Response {
	if req.Cmd == "fail" {
		return Response{Error: "nope"}
	}
	return Response{OK: true, Status: &Status{State: req.Cmd, QueueLength: len(req.Args)}}
}

func TestRoundTrip(t *testing.T) {
	path := socketPath(t)
	s, err := Listen(path, echo)
	if err != nil {
		t.Fatal(err)
	}
	go s.Serve()
	defer s.Close()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("socket mode = %v, want 0600", perm)
	}

	tests := []struct {
		name    string
		req     Request
		wantOK  bool
		wantArg int
	}{
		{name: "command with args", req: Request{Cmd: "seek", Args: []string{"+10"}}, wantOK: true, wantArg: 1},
		{name: "handler error", req: Request{Cmd: "fail"}, wantOK: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := Send(ctx(t), path, tc.req)
			if err != nil {
				t.Fatal(err)
			}
			if resp.OK != tc.wantOK {
				t.Fatalf("OK = %v (%s)", resp.OK, resp.Error)
			}
			if tc.wantOK && (resp.Status.State != tc.req.Cmd || resp.Status.QueueLength != tc.wantArg) {
				t.Errorf("status = %+v", resp.Status)
			}
		})
	}
}

func TestMalformedRequest(t *testing.T) {
	path := socketPath(t)
	s, err := Listen(path, echo)
	if err != nil {
		t.Fatal(err)
	}
	go s.Serve()
	defer s.Close()

	conn, err := net.Dial("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := conn.Write([]byte("not json\n")); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 256)
	n, _ := conn.Read(buf)
	if got := string(buf[:n]); got == "" || got[0] != '{' {
		t.Errorf("reply = %q, want a JSON error", got)
	}
}

func TestNotRunningAndStaleSocket(t *testing.T) {
	path := socketPath(t)
	if _, err := Send(ctx(t), path, Request{Cmd: "status"}); !errors.Is(err, ErrNotRunning) {
		t.Fatalf("no socket: err = %v, want ErrNotRunning", err)
	}

	// A crashed player leaves the socket file with nobody listening.
	ln, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	ln.(*net.UnixListener).SetUnlinkOnClose(false)
	ln.Close()
	if _, err := Send(ctx(t), path, Request{Cmd: "status"}); !errors.Is(err, ErrNotRunning) {
		t.Fatalf("stale socket: err = %v, want ErrNotRunning", err)
	}

	s, err := Listen(path, echo)
	if err != nil {
		t.Fatalf("Listen over a stale socket: %v", err)
	}
	go s.Serve()

	if _, err := Listen(path, echo); !errors.Is(err, ErrInUse) {
		t.Errorf("second Listen = %v, want ErrInUse", err)
	}

	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Error("Close left the socket file behind")
	}
}
