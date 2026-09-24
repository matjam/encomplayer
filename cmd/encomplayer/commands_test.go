package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/matjam/encomplayer/internal/config"
	"github.com/matjam/encomplayer/internal/theme"
)

func TestPIDFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "encomplayer.pid")

	if _, err := readPID(path); !errors.Is(err, errNotRunning) {
		t.Fatalf("readPID without a file = %v, want errNotRunning", err)
	}

	remove, err := writePID(path)
	if err != nil {
		t.Fatal(err)
	}
	if pid, err := readPID(path); err != nil || pid != os.Getpid() {
		t.Fatalf("readPID = %d, %v; want %d", pid, err, os.Getpid())
	}

	// A newer player took the file over; this one must not delete it.
	if err := os.WriteFile(path, []byte("999999\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	remove()
	if _, err := os.Stat(path); err != nil {
		t.Error("removed another player's pid file")
	}

	if err := os.WriteFile(path, []byte("garbage"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := readPID(path); err == nil || errors.Is(err, errNotRunning) {
		t.Errorf("readPID on garbage = %v, want a parse error", err)
	}
}

func TestPrintPaths(t *testing.T) {
	var out bytes.Buffer
	printPaths(&out, config.Paths{Config: "/c/config.json", Themes: "/c/themes", Playlists: "/c/pl", Cache: "/k/lib.json", State: "/s/state.json"})
	for _, want := range []string{"config     /c/config.json", "themes     /c/themes", "pid        /s/encomplayer.pid"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("paths missing %q:\n%s", want, out.String())
		}
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
