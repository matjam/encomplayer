package ui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/matjam/encomplayer/internal/audio"
)

// View implements tea.Model.
func (m *Model) View() tea.View {
	var v tea.View
	v.AltScreen = true
	v.BackgroundColor = colBackground
	v.ForegroundColor = colCyan
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
	lines = append(lines, m.tabBar())

	body := m.tabs[m.active].view(m, m.width, m.bodyHeight())
	if m.modal != nil {
		body = overlay(body, m.modal.view(m.width, m.bodyHeight()), m.width)
	}
	lines = append(lines, body...)
	lines = append(lines, m.footer()...)
	lines = append(lines, m.statusLine())

	v.SetContent(strings.Join(lines, "\n"))
	return v
}

func (m *Model) header() []string {
	inner := m.width - 2
	state := stDim.Render("■ STOPPED")
	switch m.deps.Player.State() {
	case audio.Playing:
		state = stBright.Render("▶ PLAYING")
	case audio.Paused:
		state = stAccent.Render("❚❚ PAUSED")
	}

	now, detail := stDim.Render("NO PROGRAM LOADED"), ""
	if t, i, ok := m.queue.Current(); ok {
		now = stBright.Render(t.DisplayTitle()) + stDim.Render("  //  ") + stText.Render(t.DisplayArtist())
		detail = stText.Render(albumLine(t)) + stDim.Render(fmt.Sprintf("  ·  %s  ·  %d/%d", strings.ToUpper(t.Ext()), i+1, m.queue.Len()))
	}

	vol := m.deps.Player.Volume()
	volBar := stDim.Render("VOL ") + meter(vol, 100, 10) + stText.Render(fmt.Sprintf(" %3d%%", vol))
	modes := strings.Join([]string{
		flag("REPEAT", m.modes.Repeat),
		flag("RANDOM", m.modes.Random),
		flag("SINGLE", m.modes.Single),
		flag("CONSUME", m.modes.Consume),
	}, " ")

	rightW := max(ansi.StringWidth(volBar), ansi.StringWidth(modes)) + 1
	leftW := max(1, inner-rightW)
	line1 := " " + fit(state+"  "+now, leftW-1) + fitRight(volBar, rightW)
	line2 := " " + fit(strings.Repeat(" ", 11)+detail, leftW-1) + fitRight(modes, rightW)

	title := "encom os-12 · encomplayer " + m.deps.Version
	return panel(title, []string{line1, line2}, m.width, headerRows, true)
}

func flag(name string, on bool) string {
	if on {
		return stAccent.Render(name)
	}
	return stGrid.Render(name)
}

// meter draws a segmented bar.
func meter(value, maximum, segments int) string {
	lit := 0
	if maximum > 0 {
		lit = (value*segments + maximum/2) / maximum
	}
	lit = max(0, min(lit, segments))
	return stText.Render(strings.Repeat("▰", lit)) + stGrid.Render(strings.Repeat("▱", segments-lit))
}

func (m *Model) tabBar() string {
	var parts []string
	for i, t := range m.tabs {
		label := fmt.Sprintf(" %d %s ", i+1, strings.ToUpper(t.title()))
		if i == m.active {
			parts = append(parts, stTabOn.Render(label))
		} else {
			parts = append(parts, stTabOff.Render(label))
		}
	}
	bar := strings.Join(parts, stGrid.Render("│"))
	right := stDim.Render("VISUAL " + strings.ToUpper(m.deps.ArtProtocol) + " ")
	return fit(bar, m.width-ansi.StringWidth(right)) + right
}

func (m *Model) footer() []string {
	inner := m.width - 2
	return panel("signal", []string{m.spectrumLine(inner), m.progressLine(inner)}, m.width, footerRows, false)
}

var levels = []rune(" ▁▂▃▄▅▆▇█")

func (m *Model) spectrumLine(w int) string {
	var b strings.Builder
	for x := range w {
		v := m.spectrum[x*len(m.spectrum)/max(1, w)]
		ch := string(levels[int(v*float64(len(levels)-1)+0.5)])
		switch {
		case v > 0.85:
			b.WriteString(stAccent.Render(ch))
		case v > 0.5:
			b.WriteString(stBright.Render(ch))
		default:
			b.WriteString(stText.Render(ch))
		}
	}
	return b.String()
}

func (m *Model) progressLine(w int) string {
	pos, length := m.position, m.length
	left := stText.Render(" " + clock(pos) + " ")
	right := stText.Render(" " + clock(length) + " ")
	barW := max(1, w-ansi.StringWidth(left)-ansi.StringWidth(right))

	filled := 0
	if length > 0 {
		filled = int(float64(barW) * float64(pos) / float64(length))
	}
	filled = max(0, min(filled, barW-1))

	bar := stBright.Render(strings.Repeat("━", filled)) + stAccent.Render("●") + stGrid.Render(strings.Repeat("─", barW-filled-1))
	if length == 0 {
		bar = stGrid.Render(strings.Repeat("─", barW))
	}
	return left + bar + right
}

func clock(d time.Duration) string {
	s := int(d.Seconds())
	return fmt.Sprintf("%02d:%02d", s/60, s%60)
}

func (m *Model) statusLine() string {
	var right []string
	if p := m.resolver.Pending(); p != "" {
		right = append(right, stAccent.Render("["+p+"]"))
	}
	if m.scan.running {
		label := "INDEXING "
		if m.scan.background {
			label = "SYNCING "
		}
		right = append(right, stDim.Render(label)+scanBar(m.scan.progress, 44))
	}
	right = append(right, stDim.Render("? HELP  : CMD "))
	rightS := strings.Join(right, "  ")
	leftW := max(1, m.width-ansi.StringWidth(rightS))

	var left string
	switch {
	case m.prompt != nil:
		left = m.prompt.view(leftW)
	case m.status.text != "" && m.status.isError:
		left = stError.Render(" ✖ " + m.status.text)
	case m.status.text != "":
		left = stText.Render(" ◆ " + m.status.text)
	case m.lib != nil:
		left = stDim.Render(fmt.Sprintf(" SECTOR %s · %d TRACKS", m.lib.Root(), len(m.lib.Tracks())))
	}
	return fit(left, leftW) + rightS
}
