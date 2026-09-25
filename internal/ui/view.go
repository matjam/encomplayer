package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/matjam/encomplayer/internal/audio"
	"github.com/matjam/encomplayer/internal/tea"
)

// View implements tea.Model.
func (m *Model) View() tea.View {
	var v tea.View
	v.AltScreen = true
	if m.mouseEnabled() {
		// Cell motion reports movement only while a button is held, which
		// is exactly what dragging a divider needs.
		v.MouseMode = tea.MouseModeCellMotion
	}
	v.BackgroundColor = m.st.background
	v.ForegroundColor = lipgloss.Color(m.st.theme.Text)
	v.WindowTitle = "ENCOMPLAYER"
	if t, _, ok := m.queue.Current(); ok && m.playGen != 0 {
		v.WindowTitle = "ENCOMPLAYER · " + t.DisplayTitle()
	}

	switch {
	case m.width == 0:
		return v
	case m.boot.active():
		v.SetContent(m.bootView())
		return v
	}

	lines := m.header()
	var body []string
	if m.viz.full {
		body = m.vizView()
	} else {
		lines = append(lines, m.tabBar())
		body = m.tabs[m.active].view(m, m.width, m.bodyHeight())
	}
	if m.modal != nil {
		body = overlay(body, m.modal.view(m.st, m.width, m.bodyHeight()), m.width)
	}
	lines = append(lines, body...)
	lines = append(lines, m.footer()...)
	lines = append(lines, m.statusLine())

	v.SetContent(strings.Join(lines, "\n"))
	return v
}

func (m *Model) header() []string {
	st := m.st
	inner := m.width - 2
	state := st.dim.Render("■ STOPPED")
	switch m.deps.Player.State() {
	case audio.Playing:
		state = st.bright.Render("▶ PLAYING")
	case audio.Paused:
		state = st.accent.Render("❚❚ PAUSED")
	}

	now, detail := st.dim.Render("NO PROGRAM LOADED"), ""
	if t, i, ok := m.queue.Current(); ok {
		now = st.bright.Render(t.DisplayTitle()) + st.dim.Render("  //  ") + st.text.Render(t.DisplayArtist())
		detail = st.text.Render(albumLine(t)) + st.dim.Render(fmt.Sprintf("  ·  %s  ·  %d/%d", strings.ToUpper(t.Ext()), i+1, m.queue.Len()))
	}

	vol := m.deps.Player.Volume()
	volBar := st.dim.Render(volLabel) + st.meter(vol, 100, volSegments) + st.text.Render(fmt.Sprintf(" %3d%%", vol))
	on := []bool{m.modes.Repeat, m.modes.Random, m.modes.Single, m.modes.Consume, m.modes.Careful}
	flags := make([]string, len(modeNames))
	for i, name := range modeNames {
		flags[i] = st.flag(name, on[i])
	}
	modes := strings.Join(flags, " ")

	rightW := max(volBarWidth, modesWidth) + 1
	leftW := max(1, inner-rightW)
	line1 := " " + fit(state+"  "+now, leftW-1) + fitRight(volBar, rightW)
	line2 := " " + fit(strings.Repeat(" ", 11)+detail, leftW-1) + fitRight(modes, rightW)

	title := "encom os-12 · encomplayer " + m.deps.Version
	return st.panel(title, []string{line1, line2}, m.width, headerRows, true)
}

func (st *styles) flag(name string, on bool) string {
	if on {
		return st.accent.Render(name)
	}
	return st.grid.Render(name)
}

// meter draws a segmented bar.
func (st *styles) meter(value, maximum, segments int) string {
	lit := 0
	if maximum > 0 {
		lit = (value*segments + maximum/2) / maximum
	}
	lit = max(0, min(lit, segments))
	return st.text.Render(strings.Repeat("▰", lit)) + st.grid.Render(strings.Repeat("▱", segments-lit))
}

func (m *Model) tabBar() string {
	st := m.st
	var parts []string
	for i, t := range m.tabs {
		label := tabLabel(i, t)
		if i == m.active {
			parts = append(parts, st.tabOn.Render(label))
		} else {
			parts = append(parts, st.tabOff.Render(label))
		}
	}
	bar := strings.Join(parts, st.grid.Render("│"))
	right := st.dim.Render("VISUAL " + strings.ToUpper(m.deps.ArtProtocol) + " ")
	return fit(bar, m.width-ansi.StringWidth(right)) + right
}

// footer is the SIGNAL strip: the visualiser over the seek bar, or just the
// seek bar while the visualiser is full screen.
func (m *Model) footer() []string {
	title := "signal · " + m.viz.info.Name
	body := make([]string, m.spectrumRows(), m.spectrumRows()+1)
	if m.viz.full {
		title = ""
	} else {
		copy(body, m.viz.lines)
	}
	body = append(body, m.progressLine())
	return m.st.panel(title, body, m.width, m.footerRows(), false)
}

func (m *Model) progressLine() string {
	st := m.st
	pos, length := m.position, m.length
	left := st.text.Render(" " + clock(pos) + " ")
	right := st.text.Render(" " + clock(length) + " ")
	_, barW := m.progressBarSpan()

	filled := 0
	if length > 0 {
		filled = int(float64(barW) * float64(pos) / float64(length))
	}
	filled = max(0, min(filled, barW-1))

	bar := st.bright.Render(strings.Repeat("━", filled)) + st.accent.Render("●") + st.grid.Render(strings.Repeat("─", barW-filled-1))
	if length == 0 {
		bar = st.grid.Render(strings.Repeat("─", barW))
	}
	return left + bar + right
}

func clock(d time.Duration) string {
	s := int(d.Seconds())
	return fmt.Sprintf("%02d:%02d", s/60, s%60)
}

func (m *Model) statusLine() string {
	st := m.st
	var right []string
	if p := m.resolver.Pending(); p != "" {
		right = append(right, st.accent.Render("["+p+"]"))
	}
	if m.scan.running {
		label := "INDEXING "
		if m.scan.background {
			label = "SYNCING "
		}
		right = append(right, st.dim.Render(label)+st.scanBar(m.scan.progress, 44))
	}
	right = append(right, st.dim.Render("? HELP  : CMD "))
	rightS := strings.Join(right, "  ")
	leftW := max(1, m.width-ansi.StringWidth(rightS))

	var left string
	switch {
	case m.prompt != nil:
		left = m.prompt.view(st, leftW)
	case m.status.text != "" && m.status.isError:
		left = st.err.Render(" ✖ " + m.status.text)
	case m.status.text != "":
		left = st.text.Render(" ◆ " + m.status.text)
	case m.lib != nil:
		left = st.dim.Render(fmt.Sprintf(" SECTOR %s · %d TRACKS", m.lib.Root(), len(m.lib.Tracks())))
	}
	return fit(left, leftW) + rightS
}
