package ui

import (
	"time"

	"github.com/matjam/encomplayer/internal/keymap"
	"github.com/matjam/encomplayer/internal/tea"
)

// doubleClick is the longest gap between clicks on one row that counts as a
// double click. Terminals report single clicks only.
const doubleClick = 400 * time.Millisecond

type divider int

const (
	noDivider divider = iota
	footerDivider
	artDivider
	parentDivider
	previewDivider
)

type mouseState struct {
	drag      divider
	lastRow   int
	lastClick time.Time
}

// mouseEnabled combines the config setting with a per-run --no-mouse.
func (m *Model) mouseEnabled() bool {
	return m.deps.Config.EnableMouse && !m.deps.Startup.NoMouse
}

func (m *Model) handleMouse(msg tea.MouseMsg) tea.Cmd {
	if !m.mouseEnabled() {
		return nil
	}
	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		if m.boot.active() {
			m.boot.skip()
			return nil
		}
		if msg.Button == tea.MouseLeft {
			return m.click(msg.X, msg.Y)
		}
	case tea.MouseMotionMsg:
		if m.mouse.drag != noDivider {
			m.dragTo(msg.X, msg.Y)
		}
	case tea.MouseReleaseMsg:
		if m.mouse.drag != noDivider {
			m.mouse.drag = noDivider
			m.saveState()
			return m.refreshArt()
		}
	case tea.MouseWheelMsg:
		return m.wheel(msg)
	}
	return nil
}

func (m *Model) wheel(msg tea.MouseWheelMsg) tea.Cmd {
	delta := max(1, m.deps.Config.ScrollAmount)
	switch msg.Button {
	case tea.MouseWheelUp:
		delta = -delta
	case tea.MouseWheelDown:
	default:
		return nil
	}
	if s, ok := m.modal.(*scrollModal); ok {
		s.scroll(delta)
		return nil
	}
	if m.modal != nil || msg.Y < m.bodyTop() || msg.Y >= m.footerTop() {
		return nil
	}
	if m.viz.full {
		// The wheel flips through visualisers, one per notch.
		return m.stepViz(max(-1, min(delta, 1)))
	}
	m.tabs[m.active].scroll(m, delta)
	return nil
}

func (m *Model) click(x, y int) tea.Cmd {
	if m.prompt != nil || m.modal != nil {
		return nil
	}
	now := time.Now()
	double := y == m.mouse.lastRow && now.Sub(m.mouse.lastClick) < doubleClick
	m.mouse.lastRow, m.mouse.lastClick = y, now
	if double {
		// A third click starts a new pair rather than another double.
		m.mouse.lastClick = time.Time{}
	}

	if d := m.dividerAt(x, y); d != noDivider {
		m.mouse.drag = d
		return nil
	}

	switch {
	case y == volumeRow:
		if v, ok := m.volumeAt(x); ok {
			m.deps.Player.SetVolume(v)
			m.saveState()
		}
	case y == modesRow:
		if i, ok := m.modeAt(x); ok {
			toggles := []string{keymap.ToggleRepeat, keymap.ToggleRandom, keymap.ToggleSingle, keymap.ToggleConsume}
			return m.dispatch(keymap.Action{Name: toggles[i]})
		}
	case m.viz.full && y >= m.bodyTop() && y < m.footerTop():
		// A double click closes the full-screen view.
		if double {
			return m.setVizFull(false)
		}
	case y == tabBarRow:
		if i, ok := m.tabAt(x); ok {
			m.active = i
		}
	case y >= tabBodyTop && y < m.footerTop():
		return m.tabs[m.active].click(m, x, y-tabBodyTop, double)
	case y == m.progressRow():
		m.seekToColumn(x)
	case double && y > m.footerTop() && y < m.progressRow():
		// A double click on the SIGNAL strip opens it full screen.
		return m.setVizFull(true)
	}
	return nil
}

// seekToColumn jumps to the point in the track under column x of the seek
// bar.
func (m *Model) seekToColumn(x int) {
	x0, w := m.progressBarSpan()
	if m.length <= 0 || x < x0 || x >= x0+w {
		return
	}
	frac := float64(x-x0) / float64(max(1, w-1))
	target := time.Duration(frac * float64(m.length))
	m.seek(target - m.position)
	m.position = target
}

// dividerAt reports which pane border, if any, is at (x, y). A divider is
// the pair of border columns where two panels meet.
func (m *Model) dividerAt(x, y int) divider {
	if m.viz.full {
		return noDivider
	}
	if y == m.footerTop() {
		return footerDivider
	}
	if y < tabBodyTop || y >= m.footerTop() {
		return noDivider
	}
	at := func(edge int) bool { return edge > 0 && (x == edge-1 || x == edge) }

	switch m.tabs[m.active].(type) {
	case *queueTab:
		if artW, _ := queueSplit(m, m.width); at(artW) {
			return artDivider
		}
	case *browserTab:
		parentW, currentW, _ := browserSplit(m, m.width)
		switch {
		case at(parentW):
			return parentDivider
		case at(parentW + currentW):
			return previewDivider
		}
	}
	return noDivider
}

// dragTo resizes the pane being dragged so its border follows the pointer.
func (m *Model) dragTo(x, y int) {
	w := max(1, m.width)
	switch m.mouse.drag {
	case footerDivider:
		m.sizes.FooterRows = m.height - statusRows - y
	case artDivider:
		m.sizes.ArtPercent = x * 100 / w
	case parentDivider:
		m.sizes.ParentPercent = x * 100 / w
	case previewDivider:
		m.sizes.PreviewPercent = (w - x) * 100 / w
	}
	m.layout()
}
