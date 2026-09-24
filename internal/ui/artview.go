package ui

import (
	"errors"
	"fmt"
	"image"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/matjam/encomplayer/internal/art"
	"github.com/matjam/encomplayer/internal/domain"
)

// placeDelay lets the renderer flush a frame with a blank art box before an
// overlay protocol such as iTerm2's draws into it.
const placeDelay = 80 * time.Millisecond

// artMetaRows is the space under the image for title, artist and album.
const artMetaRows = 5

type artState struct {
	path       string
	img        image.Image
	frame      *art.Frame
	cols, rows int
	loading    bool
	missing    bool

	placeSeq uint64
	erase    string
}

type artMsg struct {
	path       string
	img        image.Image
	frame      art.Frame
	err        error
	cols, rows int
}

type artPlaceMsg uint64

// artBox returns the image size in cells and its top-left screen cell.
func (m *Model) artBox() (cols, rows, x, y int) {
	artW, _ := queueSplit(m, m.width)
	if artW == 0 {
		return 0, 0, 0, 0
	}
	innerW := artW - 2
	innerH := m.tabBodyHeight() - 2
	rows = min(innerH-artMetaRows-1, innerW/2)
	if rows < 4 {
		return 0, 0, 0, 0
	}
	cols = min(innerW, rows*2)
	x = 1 + (innerW-cols)/2
	y = tabBodyTop + 1 + 1
	return cols, rows, x, y
}

// showArt switches the art to t, or clears it when t is nil.
func (m *Model) showArt(t *domain.Track) tea.Cmd {
	if t != nil && t.Path == m.art.path {
		return nil
	}
	cleanup := m.art.release()
	m.art = artState{placeSeq: m.art.placeSeq}
	if t == nil || m.deps.Art == nil {
		return tea.Raw(cleanup)
	}
	m.art.path = t.Path
	m.art.loading = true
	return tea.Batch(tea.Raw(cleanup), m.renderArt(nil))
}

// refreshArt re-renders for a new layout.
func (m *Model) refreshArt() tea.Cmd {
	cols, rows, _, _ := m.artBox()
	if m.art.path == "" || m.art.missing || (cols == m.art.cols && rows == m.art.rows) {
		return nil
	}
	return m.renderArt(m.art.img)
}

func (m *Model) renderArt(img image.Image) tea.Cmd {
	cols, rows, _, _ := m.artBox()
	if cols == 0 {
		return nil
	}
	renderer, path := m.deps.Art, m.art.path
	return func() tea.Msg {
		if img == nil {
			var err error
			if img, err = art.Load(path); err != nil {
				return artMsg{path: path, err: err}
			}
		}
		frame, err := renderer.Render(img, cols, rows)
		return artMsg{path: path, img: img, frame: frame, err: err, cols: cols, rows: rows}
	}
}

func (m *Model) receiveArt(msg artMsg) tea.Cmd {
	if msg.path != m.art.path {
		return nil
	}
	m.art.loading = false
	if msg.err != nil {
		m.art.missing = true
		if !errors.Is(msg.err, art.ErrNoArt) {
			m.status.errorf("ALBUM ART: %v", msg.err)
		}
		return nil
	}
	if cols, rows, _, _ := m.artBox(); cols != msg.cols || rows != msg.rows {
		m.art.img = msg.img
		return m.renderArt(msg.img)
	}

	cleanup := ""
	if m.art.frame != nil {
		cleanup = m.art.frame.Cleanup
	}
	m.art.img, m.art.frame = msg.img, &msg.frame
	m.art.cols, m.art.rows = msg.cols, msg.rows
	return tea.Sequence(tea.Raw(msg.frame.Setup), tea.Raw(cleanup))
}

// artSignature changes whenever something may cover or move the art.
func (m *Model) artSignature() string {
	return fmt.Sprintf("%d|%t|%t|%t|%d|%d|%p|%v", m.active, m.modal != nil, m.boot.active(), m.viz.full, m.width, m.height, m.art.frame, m.sizes)
}

// scheduleArtPlacement erases any overlay image now and redraws it after the
// next frame, for protocols that draw over the text grid.
func (m *Model) scheduleArtPlacement() tea.Cmd {
	erase := m.art.erase
	m.art.erase = ""
	m.art.placeSeq++
	seq := m.art.placeSeq

	var cmds []tea.Cmd
	if erase != "" {
		cmds = append(cmds, tea.Raw(erase))
	}
	if m.art.frame != nil && m.art.frame.Place != nil {
		cmds = append(cmds, tea.Tick(placeDelay, func(time.Time) tea.Msg { return artPlaceMsg(seq) }))
	}
	return tea.Batch(cmds...)
}

func (m *Model) placeArt(seq artPlaceMsg) tea.Cmd {
	f := m.art.frame
	if uint64(seq) != m.art.placeSeq || f == nil || f.Place == nil {
		return nil
	}
	if _, isQueue := m.tabs[m.active].(*queueTab); !isQueue || m.modal != nil || m.boot.active() || m.viz.full {
		return nil
	}
	_, _, x, y := m.artBox()
	m.art.erase = f.Erase(x, y)
	return tea.Raw(f.Place(x, y))
}

// release returns the sequences that free the current image.
func (a *artState) release() string {
	s := a.erase
	if a.frame != nil {
		s += a.frame.Cleanup
	}
	a.erase = ""
	return s
}

// artBody fills the queue tab's visual panel.
func (m *Model) artBody(w, h int) []string {
	st := m.st
	cols, rows, _, _ := m.artBox()
	pad := strings.Repeat(" ", max(0, (w-cols)/2))
	out := []string{""}

	switch {
	case m.art.frame != nil && m.art.cols == cols && m.art.rows == rows:
		for _, line := range m.art.frame.Lines {
			out = append(out, pad+line)
		}
	case cols > 0:
		label := "NO VISUAL DATA"
		if m.art.loading {
			label = "DECODING…"
		}
		for _, line := range st.gridPlaceholder(cols, rows, label) {
			out = append(out, pad+line)
		}
	}

	out = append(out, "")
	if t, _, ok := m.queue.Current(); ok {
		out = append(out,
			" "+st.bright.Render(fit(t.DisplayTitle(), w-2)),
			" "+st.text.Render(fit(t.DisplayArtist(), w-2)),
			" "+st.dim.Render(fit(albumLine(t), w-2)),
		)
	} else {
		out = append(out, " "+st.dim.Render("STANDING BY"))
	}
	return out[:min(len(out), h)]
}

func albumLine(t domain.Track) string {
	if t.Year > 0 {
		return fmt.Sprintf("%s · %d", t.DisplayAlbum(), t.Year)
	}
	return t.DisplayAlbum()
}

// gridPlaceholder draws a TRON grid with a centred label.
func (st *styles) gridPlaceholder(cols, rows int, label string) []string {
	lines := make([]string, rows)
	for r := range rows {
		var b strings.Builder
		for c := range cols {
			switch {
			case r%3 == 0 && c%6 == 0:
				b.WriteString("┼")
			case r%3 == 0:
				b.WriteString("─")
			case c%6 == 0:
				b.WriteString("│")
			default:
				b.WriteString(" ")
			}
		}
		lines[r] = st.grid.Render(b.String())
	}
	if mid := rows / 2; mid < rows && len(label)+4 <= cols {
		tag := " " + label + " "
		left := (cols - len([]rune(tag))) / 2
		row := []rune(stripStyles(lines[mid]))
		lines[mid] = st.grid.Render(string(row[:left])) + st.accent.Render(tag) + st.grid.Render(string(row[left+len([]rune(tag)):]))
	}
	return lines
}
