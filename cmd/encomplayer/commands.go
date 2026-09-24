package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/matjam/encomplayer/internal/config"
	"github.com/matjam/encomplayer/internal/theme"
)

// errNotRunning means --reload found no player to signal.
var errNotRunning = errors.New("no running encomplayer found")

// pidPath is where a running player records its process ID for --reload.
func pidPath(paths config.Paths) string {
	return filepath.Join(filepath.Dir(paths.State), "encomplayer.pid")
}

// writePID records this process for --reload and returns a function that
// removes the record if it is still ours.
func writePID(path string) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return func() {}, fmt.Errorf("record pid: %w", err)
	}
	pid := strconv.Itoa(os.Getpid())
	if err := os.WriteFile(path, []byte(pid+"\n"), 0o644); err != nil {
		return func() {}, fmt.Errorf("record pid: %w", err)
	}
	return func() {
		// A newer player may have taken the file over; leave its record.
		if data, err := os.ReadFile(path); err == nil && strings.TrimSpace(string(data)) == pid {
			_ = os.Remove(path)
		}
	}, nil
}

// readPID returns the process ID a running player recorded.
func readPID(path string) (int, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, errNotRunning
	}
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", path, err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0, fmt.Errorf("%s does not hold a process ID", path)
	}
	return pid, nil
}

// printPaths lists the files EncomPlayer reads and writes.
func printPaths(w io.Writer, paths config.Paths) {
	rows := [][2]string{
		{"config", paths.Config},
		{"themes", paths.Themes},
		{"playlists", paths.Playlists},
		{"cache", paths.Cache},
		{"state", paths.State},
		{"pid", pidPath(paths)},
	}
	for _, r := range rows {
		fmt.Fprintf(w, "%-10s %s\n", r[0], r[1])
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
