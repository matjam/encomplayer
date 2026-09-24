package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/matjam/encomplayer/internal/theme"
)

func TestDetectTerminal(t *testing.T) {
	tests := []struct {
		name       string
		env        map[string]string
		wantName   string
		wantColour string
		wantTmux   bool
	}{
		{
			name:     "kitty",
			env:      map[string]string{"KITTY_WINDOW_ID": "1", "TERM": "xterm-kitty", "COLORTERM": "truecolor"},
			wantName: "kitty (xterm-kitty)", wantColour: "24-BIT TRUECOLOR",
		},
		{
			name:     "iTerm2 with version",
			env:      map[string]string{"TERM_PROGRAM": "iTerm.app", "TERM_PROGRAM_VERSION": "3.6.2", "TERM": "xterm-256color"},
			wantName: "iTerm2 3.6.2 (xterm-256color)", wantColour: "256 COLOURS",
		},
		{
			name:     "Ghostty",
			env:      map[string]string{"TERM_PROGRAM": "ghostty", "TERM_PROGRAM_VERSION": "1.2.0", "TERM": "xterm-ghostty", "COLORTERM": "truecolor"},
			wantName: "ghostty 1.2.0 (xterm-ghostty)", wantColour: "24-BIT TRUECOLOR",
		},
		{
			name:     "tmux inside kitty reports the outer terminal",
			env:      map[string]string{"TERM_PROGRAM": "tmux", "TERM_PROGRAM_VERSION": "3.5", "TMUX": "/tmp/tmux", "KITTY_WINDOW_ID": "1", "TERM": "tmux-256color"},
			wantName: "kitty (tmux-256color)", wantColour: "256 COLOURS", wantTmux: true,
		},
		{
			name:     "bare",
			env:      map[string]string{"TERM": "xterm"},
			wantName: "unknown (xterm)", wantColour: "16 COLOURS",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := detectTerminal(func(k string) string { return tc.env[k] })
			if got.name != tc.wantName || got.colour != tc.wantColour || got.tmux != tc.wantTmux {
				t.Errorf("detectTerminal = %+v, want name %q colour %q tmux %v", got, tc.wantName, tc.wantColour, tc.wantTmux)
			}
		})
	}
}

func TestPrintVersion(t *testing.T) {
	encom, _ := theme.Builtin(theme.Default)
	b := banner{
		version:  "v1.2.0",
		theme:    encom,
		terminal: terminalInfo{name: "kitty (xterm-kitty)", colour: "24-BIT TRUECOLOR"},
		artProto: "kitty",
		ffmpeg:   "8.0",
		musicDir: "/music",
	}

	t.Run("pipe prints one parseable line", func(t *testing.T) {
		var out bytes.Buffer
		printVersion(&out, false, b)
		if out.String() != "encomplayer v1.2.0\n" {
			t.Errorf("non-terminal output = %q", out.String())
		}
	})

	t.Run("terminal prints the banner", func(t *testing.T) {
		var out bytes.Buffer
		printVersion(&out, true, b)
		got := ansi.Strip(out.String())
		for _, want := range []string{"███████╗", "v1.2.0", "kitty (xterm-kitty)", "FFMPEG 8.0", "/music", "NATHAN OLLERENSHAW", "MIT LICENSE", "END OF LINE."} {
			if !strings.Contains(got, want) {
				t.Errorf("banner missing %q:\n%s", want, got)
			}
		}
	})
}
