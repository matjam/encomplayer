package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/matjam/encomplayer/internal/config"
	"github.com/matjam/encomplayer/internal/remote"
	"github.com/matjam/encomplayer/internal/tea"
	"github.com/matjam/encomplayer/internal/theme"
	"github.com/matjam/encomplayer/internal/ui"
	"github.com/matjam/encomplayer/internal/viz"
)

// maxSocketPath is the longest socket path every platform accepts. macOS
// allows 104 bytes including the terminating NUL; Linux and Windows 108.
const maxSocketPath = 103

// socketPath is where a running player listens for control commands:
// beside the state file, or, when that path is too long for a socket, in
// the per-user runtime or temp directory under a name derived from the state
// directory, so that player and client still agree on it.
func socketPath(paths config.Paths) string {
	dir := filepath.Dir(paths.State)
	if p := filepath.Join(dir, "encomplayer.sock"); len(p) <= maxSocketPath {
		return p
	}
	base := os.Getenv("XDG_RUNTIME_DIR")
	if base == "" {
		base = os.TempDir()
	}
	sum := sha256.Sum256([]byte(dir))
	return filepath.Join(base, fmt.Sprintf("encomplayer-%d-%x.sock", os.Getuid(), sum[:6]))
}

// serveRemote lets `encomplayer <command>` reach this player. Each request
// becomes a ui.RemoteMsg handled in the event loop, so it runs exactly like
// the matching key. Remote control is optional: if the socket cannot be
// claimed the player still runs, and says why in its status line.
func serveRemote(sock string, program *tea.Program) func() {
	server, err := remote.Listen(sock, func(req remote.Request) remote.Response {
		reply := make(chan remote.Response, 1)
		program.Send(ui.RemoteMsg{Req: req, Reply: reply})
		select {
		case resp := <-reply:
			return resp
		case <-time.After(5 * time.Second):
			return remote.Response{Error: "the player did not respond"}
		}
	})
	if err != nil {
		go program.Send(ui.NoticeMsg{Text: "remote control unavailable: " + err.Error()})
		return func() {}
	}
	go server.Serve()
	return func() {
		// Nothing to recover if the socket file is already gone.
		_ = server.Close()
	}
}

// runControl sends a command to the running player and prints the result.
func runControl(w io.Writer, sock string, opts options) error {
	args := opts.commandArgs
	if opts.command == "add" {
		// The player may run in another directory, so send an absolute
		// path.
		abs, err := filepath.Abs(args[0])
		if err != nil {
			return fmt.Errorf("add: %w", err)
		}
		args = []string{abs}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	resp, err := remote.Send(ctx, sock, remote.Request{Cmd: opts.command, Args: args})
	switch {
	case errors.Is(err, remote.ErrNotRunning):
		return fmt.Errorf("%w (start one with: encomplayer)", err)
	case err != nil:
		return err
	case !resp.OK:
		return errors.New(resp.Error)
	}

	if opts.command != "status" {
		return nil
	}
	if opts.json {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(resp.Status)
	}
	fmt.Fprintln(w, formatStatus(resp.Status))
	return nil
}

// formatStatus renders status as one line, short enough for a tmux or
// shell prompt.
func formatStatus(s *remote.Status) string {
	icon := map[string]string{"playing": "▶", "paused": "❚❚", "stopped": "■"}[s.State]
	if s.Title == "" {
		return icon + " stopped · queue empty"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s %s", icon, s.Title)
	if s.Artist != "" {
		fmt.Fprintf(&b, " — %s", s.Artist)
	}
	if s.Album != "" {
		fmt.Fprintf(&b, " · %s", s.Album)
	}
	fmt.Fprintf(&b, "  %s / %s", clockSecs(s.Position), clockSecs(s.Duration))
	fmt.Fprintf(&b, "  %d/%d  vol %d%%", s.QueueIndex+1, s.QueueLength, s.Volume)

	var modes []string
	for _, m := range []struct {
		on   bool
		name string
	}{{s.Repeat, "repeat"}, {s.Random, "random"}, {s.Single, "single"}, {s.Consume, "consume"}} {
		if m.on {
			modes = append(modes, m.name)
		}
	}
	if len(modes) > 0 {
		fmt.Fprintf(&b, "  [%s]", strings.Join(modes, " "))
	}
	return b.String()
}

func clockSecs(secs float64) string {
	s := int(secs)
	if s >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", s/3600, s/60%60, s%60)
	}
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

// printPaths lists the files EncomPlayer reads and writes.
func printPaths(w io.Writer, paths config.Paths) {
	rows := [][2]string{
		{"config", paths.Config},
		{"themes", paths.Themes},
		{"playlists", paths.Playlists},
		{"cache", paths.Cache},
		{"state", paths.State},
		{"socket", socketPath(paths)},
	}
	for _, r := range rows {
		fmt.Fprintf(w, "%-10s %s\n", r[0], r[1])
	}
}

// printVisualizers lists visualisers. On a terminal each gets its
// description and the configured one is marked; piped, it prints bare
// names.
func printVisualizers(w io.Writer, tty bool, infos []viz.Info, current string) {
	width := 0
	for _, info := range infos {
		width = max(width, len(info.Name))
	}
	for _, info := range infos {
		switch {
		case !tty:
			fmt.Fprintln(w, info.Name)
		case info.Name == current:
			fmt.Fprintf(w, "▶ %-*s  %s\n", width, info.Name, info.Description)
		default:
			fmt.Fprintf(w, "  %-*s  %s\n", width, info.Name, info.Description)
		}
	}
}

// printThemes lists theme names. On a terminal each name gets a swatch of
// its colours and the active theme is marked; piped, it prints bare names.
func printThemes(w io.Writer, tty bool, store *theme.Store, current string) {
	names := store.Names()
	if !tty {
		for _, n := range names {
			fmt.Fprintln(w, n)
		}
		return
	}

	width := 0
	for _, n := range names {
		width = max(width, len(n))
	}
	var out []string
	for _, n := range names {
		t, err := store.Load(n)
		if err != nil {
			out = append(out, fmt.Sprintf("  %-*s  %v", width, n, err))
			continue
		}
		mark := "  "
		if n == current {
			mark = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Accent)).Bold(true).Render("▶ ")
		}
		out = append(out, mark+fmt.Sprintf("%-*s", width, n)+"  "+swatch(t))
	}
	lipgloss.Fprintln(w, strings.Join(out, "\n"))
}

// swatch draws a theme's main colours as blocks on its own background.
func swatch(t theme.Theme) string {
	bg := lipgloss.NewStyle()
	if t.Background != "" {
		bg = bg.Background(lipgloss.Color(t.Background))
	}
	var b strings.Builder
	for _, c := range []string{t.Text, t.Bright, t.Dim, t.Accent, t.Selection, t.Error} {
		b.WriteString(bg.Foreground(lipgloss.Color(c)).Render("██"))
	}
	return bg.Render(" ") + b.String() + bg.Render(" ")
}
