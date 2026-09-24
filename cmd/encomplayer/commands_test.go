package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/matjam/encomplayer/internal/config"
	"github.com/matjam/encomplayer/internal/remote"
	"github.com/matjam/encomplayer/internal/theme"
	"github.com/matjam/encomplayer/internal/ui"
)

// The CLI validates arguments itself for clear errors, so its command list
// must match what the player accepts.
func TestCommandListsAgree(t *testing.T) {
	for _, c := range ui.RemoteCommands {
		if _, ok := commandArity[c]; !ok {
			t.Errorf("player accepts %q but the CLI does not", c)
		}
	}
	if len(commandArity) != len(ui.RemoteCommands) {
		t.Errorf("CLI knows %d commands, player %d", len(commandArity), len(ui.RemoteCommands))
	}
}

// fakePlayerSocket serves requests from handle on a short socket path, as a
// running player would. macOS limits socket paths to 104 bytes.
func fakePlayerSocket(t *testing.T, handle remote.Handler) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "ep")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	sock := filepath.Join(dir, "p.sock")
	s, err := remote.Listen(sock, handle)
	if err != nil {
		t.Fatal(err)
	}
	go s.Serve()
	t.Cleanup(func() { s.Close() })
	return sock
}

func TestRunControl(t *testing.T) {
	var got []remote.Request
	sock := fakePlayerSocket(t, func(req remote.Request) remote.Response {
		got = append(got, req)
		if req.Cmd == "seek" && req.Args[0] == "bad" {
			return remote.Response{Error: "seek: want +N"}
		}
		return remote.Response{OK: true, Status: &remote.Status{State: "playing", Title: "Derezzed", Artist: "Daft Punk", Position: 83, Duration: 104, Volume: 70, QueueLength: 4, Random: true}}
	})

	tests := []struct {
		name    string
		opts    options
		wantOut string
		wantErr string
	}{
		{name: "next is quiet", opts: options{command: "next"}},
		{name: "status line", opts: options{command: "status"}, wantOut: "▶ Derezzed — Daft Punk  1:23 / 1:44  1/4  vol 70%  [random]\n"},
		{name: "status json", opts: options{command: "status", json: true}, wantOut: `"title": "Derezzed"`},
		{name: "player error", opts: options{command: "seek", commandArgs: []string{"bad"}}, wantErr: "seek: want +N"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			err := runControl(&out, sock, tc.opts)
			switch {
			case tc.wantErr != "" && (err == nil || err.Error() != tc.wantErr):
				t.Fatalf("err = %v, want %q", err, tc.wantErr)
			case tc.wantErr == "" && err != nil:
				t.Fatal(err)
			}
			if !strings.Contains(out.String(), tc.wantOut) {
				t.Errorf("output = %q, want %q", out.String(), tc.wantOut)
			}
		})
	}

	// add sends an absolute path, since the player runs elsewhere.
	if err := runControl(io.Discard, sock, options{command: "add", commandArgs: []string{"music/x.flac"}}); err != nil {
		t.Fatal(err)
	}
	if last := got[len(got)-1]; !filepath.IsAbs(last.Args[0]) {
		t.Errorf("add sent %q, want an absolute path", last.Args[0])
	}
}

func TestRunControlNotRunning(t *testing.T) {
	err := runControl(io.Discard, filepath.Join(t.TempDir(), "none.sock"), options{command: "next"})
	if !errors.Is(err, remote.ErrNotRunning) {
		t.Errorf("err = %v, want ErrNotRunning", err)
	}
}

func TestFormatStatus(t *testing.T) {
	tests := []struct {
		name string
		s    remote.Status
		want string
	}{
		{name: "empty queue", s: remote.Status{State: "stopped"}, want: "■ stopped · queue empty"},
		{name: "paused long track", s: remote.Status{State: "paused", Title: "Echoes", Album: "Meddle", Position: 3700, Duration: 4200, QueueIndex: 2, QueueLength: 9, Volume: 40, Repeat: true, Single: true},
			want: "❚❚ Echoes · Meddle  1:01:40 / 1:10:00  3/9  vol 40%  [repeat single]"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatStatus(&tc.s); got != tc.want {
				t.Errorf("formatStatus = %q\n want %q", got, tc.want)
			}
		})
	}
}

func TestPrintPaths(t *testing.T) {
	var out bytes.Buffer
	printPaths(&out, config.Paths{Config: "/c/config.json", Themes: "/c/themes", Playlists: "/c/pl", Cache: "/k/lib.json", State: "/s/state.json"})
	for _, want := range []string{"config     /c/config.json", "themes     /c/themes", "socket     /s/encomplayer.sock"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("paths missing %q:\n%s", want, out.String())
		}
	}
}

func TestSocketPath(t *testing.T) {
	short := socketPath(config.Paths{State: "/s/state.json"})
	if short != "/s/encomplayer.sock" {
		t.Errorf("short state dir: socket = %q", short)
	}

	t.Setenv("XDG_RUNTIME_DIR", "/run/user/1000")
	deep := "/" + strings.Repeat("d", 120) + "/state.json"
	a, b := socketPath(config.Paths{State: deep}), socketPath(config.Paths{State: deep})
	if a != b || len(a) > maxSocketPath || filepath.Dir(a) != "/run/user/1000" {
		t.Errorf("deep state dir: socket = %q, then %q", a, b)
	}
	if other := socketPath(config.Paths{State: deep + "x/state.json"}); other == a {
		t.Error("different state dirs share a socket")
	}
}

func TestPrintThemes(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "mine.json"), []byte(`{"extends":"nord"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	store := theme.NewStore(dir)

	var plain bytes.Buffer
	printThemes(&plain, false, store, "dracula")
	lines := strings.Split(strings.TrimSpace(plain.String()), "\n")
	if len(lines) != len(theme.BuiltinNames())+1 || lines[len(lines)-1] != "mine" {
		t.Errorf("piped list has %d lines ending %q", len(lines), lines[len(lines)-1])
	}

	var fancy bytes.Buffer
	printThemes(&fancy, true, store, "dracula")
	for _, line := range strings.Split(ansi.Strip(fancy.String()), "\n") {
		if strings.Contains(line, "dracula") && !strings.HasPrefix(line, "▶ ") {
			t.Errorf("current theme not marked: %q", line)
		}
		if strings.Contains(line, "nord ") && !strings.Contains(line, "██") {
			t.Errorf("theme row without a swatch: %q", line)
		}
	}
}
