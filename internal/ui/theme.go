package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// ENCOM OS-12 palette: cyan grid lines on near-black, with orange reserved
// for the playing track, marks and warnings.
var (
	colBackground = lipgloss.Color("#02090C")
	colCyan       = lipgloss.Color("#6FC3DF")
	colBright     = lipgloss.Color("#E6FFFF")
	colDim        = lipgloss.Color("#2A6475")
	colGrid       = lipgloss.Color("#123C48")
	colOrange     = lipgloss.Color("#FF9A2E")
	colRed        = lipgloss.Color("#FF4A3D")
)

var (
	stText     = lipgloss.NewStyle().Foreground(colCyan)
	stBright   = lipgloss.NewStyle().Foreground(colBright).Bold(true)
	stDim      = lipgloss.NewStyle().Foreground(colDim)
	stGrid     = lipgloss.NewStyle().Foreground(colGrid)
	stAccent   = lipgloss.NewStyle().Foreground(colOrange).Bold(true)
	stError    = lipgloss.NewStyle().Foreground(colRed).Bold(true)
	stCursor   = lipgloss.NewStyle().Foreground(colBackground).Background(colCyan).Bold(true)
	stDimCur   = lipgloss.NewStyle().Foreground(colBright).Background(colGrid)
	stTabOn    = lipgloss.NewStyle().Foreground(colBackground).Background(colCyan).Bold(true)
	stTabOff   = lipgloss.NewStyle().Foreground(colDim)
	stBorder   = lipgloss.NewStyle().Foreground(colDim)
	stBorderOn = lipgloss.NewStyle().Foreground(colCyan)
)

// fit truncates or pads s, which may contain ANSI sequences, to exactly w
// cells.
func fit(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if ansi.StringWidth(s) > w {
		s = ansi.Truncate(s, w, "…")
	}
	return s + strings.Repeat(" ", w-ansi.StringWidth(s))
}

// fitRight right-aligns s in w cells.
func fitRight(s string, w int) string {
	if ansi.StringWidth(s) >= w {
		return fit(s, w)
	}
	return strings.Repeat(" ", w-ansi.StringWidth(s)) + s
}

// panel draws body inside an ENCOM-style frame of exactly w × h cells.
func panel(title string, body []string, w, h int, focused bool) []string {
	if w < 4 || h < 2 {
		return blankLines(w, h)
	}
	border := stBorder
	if focused {
		border = stBorderOn
	}
	inner := w - 2

	label := ""
	if title != "" {
		label = stAccent.Render("◆") + " " + stBright.Render(strings.ToUpper(title)) + " "
	}
	labelW := ansi.StringWidth(label)
	topFill := max(0, inner-labelW-1)

	lines := make([]string, 0, h)
	lines = append(lines, border.Render("┌─")+label+border.Render(strings.Repeat("─", topFill)+"┐"))

	for i := range h - 2 {
		row := ""
		if i < len(body) {
			row = body[i]
		}
		lines = append(lines, border.Render("│")+fit(row, inner)+border.Render("│"))
	}

	tick := min(3, inner)
	lines = append(lines, border.Render("└")+stText.Render(strings.Repeat("━", tick))+border.Render(strings.Repeat("─", inner-tick)+"┘"))
	return lines
}

func blankLines(w, h int) []string {
	lines := make([]string, max(0, h))
	for i := range lines {
		lines[i] = strings.Repeat(" ", max(0, w))
	}
	return lines
}

// hjoin places blocks of lines side by side. Every block must have the same
// number of lines and consistent widths.
func hjoin(blocks ...[]string) []string {
	if len(blocks) == 0 {
		return nil
	}
	out := make([]string, len(blocks[0]))
	for i := range out {
		var b strings.Builder
		for _, block := range blocks {
			if i < len(block) {
				b.WriteString(block[i])
			}
		}
		out[i] = b.String()
	}
	return out
}

// overlay draws box centred over base, replacing whole cells.
func overlay(base, box []string, width int) []string {
	if len(box) == 0 {
		return base
	}
	top := max(0, (len(base)-len(box))/2)
	boxW := ansi.StringWidth(box[0])
	left := max(0, (width-boxW)/2)

	out := make([]string, len(base))
	copy(out, base)
	for i, line := range box {
		row := top + i
		if row >= len(out) {
			break
		}
		prefix := ansi.Truncate(out[row], left, "")
		prefix += strings.Repeat(" ", left-ansi.StringWidth(prefix))
		suffix := ansi.TruncateLeft(out[row], left+boxW, "")
		out[row] = prefix + "\x1b[0m" + line + "\x1b[0m" + suffix
	}
	return out
}
