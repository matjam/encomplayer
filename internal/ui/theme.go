package ui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/matjam/encomplayer/internal/tea/textinput"
	"github.com/matjam/encomplayer/internal/theme"
	"github.com/matjam/encomplayer/internal/viz"
)

// styles are the lipgloss styles for one theme. The model owns the current
// set and swaps it whole when the theme changes.
type styles struct {
	theme theme.Theme

	// background fills the screen, or is nil to keep the terminal's own.
	background color.Color

	// palette is the theme for visualisers.
	palette viz.Palette

	text, bright, dim, grid, accent, err lipgloss.Style
	cursor, dimCursor, tabOn, tabOff     lipgloss.Style
	border, borderOn                     lipgloss.Style
}

func newStyles(t theme.Theme) *styles {
	c := lipgloss.Color
	st := &styles{
		theme:     t,
		text:      lipgloss.NewStyle().Foreground(c(t.Text)),
		bright:    lipgloss.NewStyle().Foreground(c(t.Bright)).Bold(true),
		dim:       lipgloss.NewStyle().Foreground(c(t.Dim)),
		grid:      lipgloss.NewStyle().Foreground(c(t.Grid)),
		accent:    lipgloss.NewStyle().Foreground(c(t.Accent)).Bold(true),
		err:       lipgloss.NewStyle().Foreground(c(t.Error)).Bold(true),
		cursor:    lipgloss.NewStyle().Foreground(c(t.SelectionText)).Background(c(t.Selection)).Bold(true),
		dimCursor: lipgloss.NewStyle().Foreground(c(t.Bright)).Background(c(t.Grid)),
		tabOn:     lipgloss.NewStyle().Foreground(c(t.SelectionText)).Background(c(t.Selection)).Bold(true),
		tabOff:    lipgloss.NewStyle().Foreground(c(t.Dim)),
		border:    lipgloss.NewStyle().Foreground(c(t.Border)),
		borderOn:  lipgloss.NewStyle().Foreground(c(t.Text)),
	}
	st.palette = viz.Palette{
		Text: viz.Hex(t.Text), Bright: viz.Hex(t.Bright), Dim: viz.Hex(t.Dim),
		Grid: viz.Hex(t.Grid), Accent: viz.Hex(t.Accent), Error: viz.Hex(t.Error),
	}
	if t.Background != "" {
		st.background = c(t.Background)
		st.palette.Background = viz.Hex(t.Background)
	}
	return st
}

// styleInput colours a text input for the current theme. Inputs are styled
// at draw time so they follow theme changes.
func (st *styles) styleInput(in *textinput.Model) {
	s := in.Styles()
	s.Focused.Prompt = st.accent
	s.Focused.Text = st.bright
	s.Focused.Placeholder = st.dim
	s.Blurred.Prompt = st.dim
	s.Blurred.Text = st.text
	s.Blurred.Placeholder = st.dim
	in.SetStyles(s)
}

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
func (st *styles) panel(title string, body []string, w, h int, focused bool) []string {
	if w < 4 || h < 2 {
		return blankLines(w, h)
	}
	border := st.border
	if focused {
		border = st.borderOn
	}
	inner := w - 2

	label := ""
	if title != "" {
		label = st.accent.Render("◆") + " " + st.bright.Render(strings.ToUpper(title)) + " "
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
	lines = append(lines, border.Render("└")+st.text.Render(strings.Repeat("━", tick))+border.Render(strings.Repeat("─", inner-tick)+"┘"))
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
