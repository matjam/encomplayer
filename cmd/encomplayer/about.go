package main

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/matjam/encomplayer/internal/art"
	"github.com/matjam/encomplayer/internal/theme"
	"github.com/matjam/encomplayer/internal/ui"
)

// These match the LICENSE file.
const (
	author    = "Nathan Ollerenshaw"
	license   = "MIT License"
	copyright = "2026"
)

// terminalInfo is what the environment says about the terminal.
type terminalInfo struct {
	name   string
	colour string
	tmux   bool
}

// knownTerminals map an environment variable that only one terminal sets to
// that terminal's name, for terminals that do not set TERM_PROGRAM.
var knownTerminals = []struct{ env, name string }{
	{"KITTY_WINDOW_ID", "kitty"},
	{"GHOSTTY_RESOURCES_DIR", "Ghostty"},
	{"ALACRITTY_WINDOW_ID", "Alacritty"},
	{"WEZTERM_EXECUTABLE", "WezTerm"},
	{"KONSOLE_VERSION", "Konsole"},
	{"WT_SESSION", "Windows Terminal"},
}

// detectTerminal names the terminal from its environment variables.
func detectTerminal(getenv func(string) string) terminalInfo {
	info := terminalInfo{tmux: getenv("TMUX") != ""}

	switch program := getenv("TERM_PROGRAM"); program {
	case "", "tmux", "screen":
		for _, k := range knownTerminals {
			if getenv(k.env) != "" {
				info.name = k.name
				break
			}
		}
	case "Apple_Terminal":
		info.name = "Terminal.app"
	case "iTerm.app":
		info.name = "iTerm2"
	case "vscode":
		info.name = "VS Code"
	default:
		info.name = program
	}
	if info.name == "" {
		info.name = "unknown"
	}
	if v := getenv("TERM_PROGRAM_VERSION"); v != "" && !info.tmux {
		info.name += " " + v
	}
	if t := getenv("TERM"); t != "" {
		info.name += " (" + t + ")"
	}

	switch ct := getenv("COLORTERM"); {
	case ct == "truecolor" || ct == "24bit":
		info.colour = "24-BIT TRUECOLOR"
	case strings.Contains(getenv("TERM"), "256color"):
		info.colour = "256 COLOURS"
	default:
		info.colour = "16 COLOURS"
	}
	return info
}

// ffmpegVersion returns ffmpeg's version, or "" when it is not installed.
func ffmpegVersion() string {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "ffmpeg", "-version").Output()
	if err != nil {
		return ""
	}
	first, _, _ := strings.Cut(string(out), "\n")
	if f := strings.Fields(first); len(f) >= 3 {
		return f[2]
	}
	return ""
}

// banner is everything the --version screen reports.
type banner struct {
	version  string
	theme    theme.Theme
	terminal terminalInfo
	artProto string
	ffmpeg   string
	musicDir string
}

// printVersion writes --version. On a terminal it draws the ENCOM banner;
// otherwise it writes one plain line that scripts can parse.
func printVersion(w io.Writer, tty bool, b banner) {
	if !tty {
		fmt.Fprintf(w, "encomplayer %s\n", b.version)
		return
	}
	c := lipgloss.Color
	text := lipgloss.NewStyle().Foreground(c(b.theme.Text))
	bright := lipgloss.NewStyle().Foreground(c(b.theme.Bright)).Bold(true)
	dim := lipgloss.NewStyle().Foreground(c(b.theme.Dim))
	grid := lipgloss.NewStyle().Foreground(c(b.theme.Grid))
	accent := lipgloss.NewStyle().Foreground(c(b.theme.Accent)).Bold(true)

	var out []string
	for _, row := range ui.Logo() {
		out = append(out, text.Render(row))
	}
	out = append(out, dim.Render(strings.Repeat("━", 46)), "")

	decoders := "MP3 FLAC OGG WAV"
	if b.ffmpeg != "" {
		decoders += " + FFMPEG " + b.ffmpeg
	} else {
		decoders += " · FFMPEG NOT FOUND"
	}
	sector := b.musicDir
	if sector == "" {
		sector = "NONE MOUNTED"
	}
	multiplexer := "NONE"
	if b.terminal.tmux {
		multiplexer = "TMUX"
	}

	rows := [][2]string{
		{"ENCOMPLAYER", b.version},
		{"KERNEL", fmt.Sprintf("%s %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH)},
		{"TERMINAL", b.terminal.name},
		{"COLOUR DEPTH", b.terminal.colour},
		{"MULTIPLEXER", multiplexer},
		{"VISUAL INTERFACE", strings.ToUpper(b.artProto)},
		{"AUDIO DECODERS", decoders},
		{"THEME", b.theme.Name},
		{"SECTOR", sector},
		{"GRID LINK", "ESTABLISHED"},
		{"IDENTITY DISC", "SYNCHRONISED"},
	}
	// Values print as-is: paths and terminal names are case-sensitive.
	const labelW = 24
	for _, r := range rows {
		dots := grid.Render(strings.Repeat(".", labelW-len(r[0])))
		out = append(out, text.Render("> "+r[0]+" ")+dots+" "+bright.Render(r[1]))
	}

	out = append(out, "",
		dim.Render(fmt.Sprintf("(C) %s %s. %s.", copyright, strings.ToUpper(author), strings.ToUpper(license))),
		dim.Render("https://github.com/matjam/encomplayer"),
		"",
		accent.Render("END OF LINE."),
	)
	lipgloss.Fprintln(w, strings.Join(out, "\n"))
}

// artProtocolName reports the album art protocol a run would start with. A
// run can still switch to sixel once the terminal reports support for it.
func artProtocolName(setting string, getenv func(string) string) string {
	_, protocol, err := art.Choose(setting, art.Terminal{Getenv: getenv})
	if err != nil {
		return "UNKNOWN (" + setting + ")"
	}
	return protocol
}
