package ui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

const (
	bootInterval = 70 * time.Millisecond
	bootLinger   = 8
)

// bootState drives the ENCOM OS-12 start-up screen. It stays up until every
// line has printed and the first scan is done, or until a key is pressed.
type bootState struct {
	step     int
	linger   int
	scanDone bool
	finished bool
}

type bootTickMsg struct{}

func bootTick() tea.Cmd {
	return tea.Tick(bootInterval, func(time.Time) tea.Msg { return bootTickMsg{} })
}

func (b *bootState) active() bool { return !b.finished }

func (b *bootState) skip() { b.finished = true }

func (b *bootState) advance() tea.Cmd {
	if b.finished {
		return nil
	}
	b.step++
	if b.step >= len(bootScript) && b.scanDone {
		b.linger++
		if b.linger > bootLinger {
			b.finished = true
			return nil
		}
	}
	return bootTick()
}

// logo is "ENCOM" in the ANSI Shadow figlet font, one letter per entry.
var logo = [][]string{
	{"███████╗", "██╔════╝", "█████╗  ", "██╔══╝  ", "███████╗", "╚══════╝"},
	{"███╗   ██╗", "████╗  ██║", "██╔██╗ ██║", "██║╚██╗██║", "██║ ╚████║", "╚═╝  ╚═══╝"},
	{" ██████╗", "██╔════╝", "██║     ", "██║     ", "╚██████╗", " ╚═════╝"},
	{" ██████╗ ", "██╔═══██╗", "██║   ██║", "██║   ██║", "╚██████╔╝", " ╚═════╝ "},
	{"███╗   ███╗", "████╗ ████║", "██╔████╔██║", "██║╚██╔╝██║", "██║ ╚═╝ ██║", "╚═╝     ╚═╝"},
}

// bootScript lines print one per tick. Values are filled in from the model.
var bootScript = []struct{ label, value string }{
	{"ENCOM OS-12 · KERNEL 12.0.4-GRID", ""},
	{"(C) 1982-2026 ENCOM INTERNATIONAL. ALL RIGHTS RESERVED.", ""},
	{"", ""},
	{"INITIALISING AUDIO SUBSYSTEM", "44.1 KHZ STEREO"},
	{"LOADING DECODERS", "{formats}"},
	{"KEYMAP", "RMPC COMPATIBLE"},
	{"VISUAL INTERFACE", "{art}"},
	{"MOUNTING SECTOR", "{root}"},
	{"INDEXING", "{scan}"},
	{"", ""},
	{"GREETINGS, PROGRAM.", ""},
}

func (m *Model) bootView() string {
	var out []string
	for row := range len(logo[0]) {
		var b strings.Builder
		for _, letter := range logo {
			b.WriteString(letter[row])
		}
		out = append(out, stText.Render(b.String()))
	}
	out = append(out, "", stDim.Render(strings.Repeat("━", 46)), "")

	const labelW, lineW = 34, 72
	for i, line := range bootScript[:min(m.boot.step, len(bootScript))] {
		switch {
		case line.label == "":
			out = append(out, "")
		case line.value == "":
			style := stDim
			if i == len(bootScript)-1 {
				style = stAccent
			}
			out = append(out, style.Render(line.label))
		case line.value == "{scan}" && m.scan.running && !m.scan.background:
			dots := stGrid.Render(strings.Repeat(".", labelW-len(line.label)))
			out = append(out, stText.Render("> "+line.label+" ")+dots+" "+scanBar(m.scan.progress, lineW-labelW-4))
		default:
			dots := stGrid.Render(strings.Repeat(".", labelW-len(line.label)))
			out = append(out, stText.Render("> "+line.label+" ")+dots+" "+stBright.Render(ansi.Truncate(m.bootValue(line.value), lineW-labelW-4, "…")))
		}
	}

	for i := range out {
		out[i] = fit(out[i], lineW)
	}
	block := strings.Join(out, "\n")
	top := max(0, (m.height-len(out))/2)
	left := max(0, (m.width-lineW)/2)
	indent := strings.Repeat(" ", left)
	return strings.Repeat("\n", top) + indent + strings.ReplaceAll(block, "\n", "\n"+indent)
}

func (m *Model) bootValue(v string) string {
	switch v {
	case "{formats}":
		return m.deps.Formats
	case "{art}":
		return strings.ToUpper(m.deps.ArtProtocol)
	case "{root}":
		if m.scan.root == "" {
			return "NONE · USE :scan <dir>"
		}
		return ansi.Truncate(m.scan.root, 34, "…")
	case "{scan}":
		switch {
		case m.scan.root == "":
			return "SKIPPED"
		case m.scan.running && m.lib != nil:
			return fmt.Sprintf("%d CACHED · SYNCING", len(m.lib.Tracks()))
		case m.boot.scanDone && m.lib != nil:
			return fmt.Sprintf("%d TRACKS", len(m.lib.Tracks()))
		default:
			return "FAILED"
		}
	}
	return v
}
