package ui

import (
	"fmt"
	"strings"
)

// Screen geometry shared by the view and mouse hit-testing, so a click always
// lands on what was drawn there.

const (
	volLabel    = "VOL "
	volSegments = 10

	// volBarWidth covers the label, the segments and " 100%".
	volBarWidth = len(volLabel) + volSegments + 5

	volumeRow  = 1
	modesRow   = 2
	tabBarRow  = headerRows
	tabBodyTop = headerRows + tabRows
)

// bodyTop is the first row under the header chrome. The full-screen
// visualiser hides the tab bar and starts right under the header.
func (m *Model) bodyTop() int {
	if m.viz.full {
		return headerRows
	}
	return tabBodyTop
}

var modeNames = []string{"REPEAT", "RANDOM", "SINGLE", "CONSUME"}

// modesWidth is the width of the mode flags joined by single spaces.
var modesWidth = len(strings.Join(modeNames, " "))

// headerInner is the header's width inside its border.
func (m *Model) headerInner() int { return m.width - 2 }

// volumeAt maps a click on the volume meter to a percentage.
func (m *Model) volumeAt(x int) (int, bool) {
	x0 := 1 + m.headerInner() - volBarWidth + len(volLabel)
	seg := x - x0
	if seg < 0 || seg >= volSegments {
		return 0, false
	}
	return (seg + 1) * 100 / volSegments, true
}

// modeAt returns the index into modeNames of the flag at column x.
func (m *Model) modeAt(x int) (int, bool) {
	pos := 1 + m.headerInner() - modesWidth
	for i, name := range modeNames {
		if x >= pos && x < pos+len(name) {
			return i, true
		}
		pos += len(name) + 1
	}
	return 0, false
}

func tabLabel(i int, t tab) string {
	return fmt.Sprintf(" %d %s ", i+1, strings.ToUpper(t.title()))
}

// tabAt returns the tab whose label covers column x. Labels are separated by
// a one-cell divider.
func (m *Model) tabAt(x int) (int, bool) {
	pos := 0
	for i, t := range m.tabs {
		w := len([]rune(tabLabel(i, t)))
		if x >= pos && x < pos+w {
			return i, true
		}
		pos += w + 1
	}
	return 0, false
}

// Limits that keep every pane usable while dividers are dragged.
const (
	minBodyRows   = 6
	minFooterRows = 4
	minArtPct     = 15
	maxArtPct     = 70
	minSidePct    = 10
	maxSidesPct   = 75
)

// clampLayout keeps the dragged sizes inside the window and above minimums.
func (m *Model) clampLayout() {
	l := &m.sizes
	maxFooter := max(minFooterRows, m.height-headerRows-tabRows-statusRows-minBodyRows)
	l.FooterRows = max(minFooterRows, min(l.FooterRows, maxFooter))
	l.ArtPercent = max(minArtPct, min(l.ArtPercent, maxArtPct))
	l.ParentPercent = max(minSidePct, min(l.ParentPercent, maxSidesPct-minSidePct))
	l.PreviewPercent = max(minSidePct, min(l.PreviewPercent, maxSidesPct-l.ParentPercent))
}

// footerRows is the signal strip's height, borders included. With the
// visualiser full screen the strip holds only the seek bar.
func (m *Model) footerRows() int {
	if m.viz.full {
		return fullFooterRows
	}
	return m.sizes.FooterRows
}

// spectrumRows is how many rows the visualiser fills in the strip.
func (m *Model) spectrumRows() int { return m.footerRows() - 3 }

// footerTop is the first row of the signal panel; dragging it resizes the
// strip.
func (m *Model) footerTop() int { return m.bodyTop() + m.bodyHeight() }

// progressRow is the row holding the seek bar, under the visualiser.
func (m *Model) progressRow() int { return m.footerTop() + 1 + m.spectrumRows() }

// progressBarSpan returns the column of the first seek-bar cell and the bar
// width. The clocks either side widen for tracks past 99 minutes.
func (m *Model) progressBarSpan() (int, int) {
	left := len(" " + clock(m.position) + " ")
	right := len(" " + clock(m.length) + " ")
	return 1 + left, max(1, m.width-2-left-right)
}

// queueSplit divides the queue tab between album art and the queue table.
func queueSplit(m *Model, w int) (artW, queueW int) {
	if m.deps.Art != nil && w >= 70 {
		artW = w * m.sizes.ArtPercent / 100
	}
	return artW, w - artW
}

// browserSplit divides a browser tab into parent, current and preview
// columns.
func browserSplit(m *Model, w int) (parentW, currentW, previewW int) {
	parentW = w * m.sizes.ParentPercent / 100
	previewW = w * m.sizes.PreviewPercent / 100
	return parentW, w - parentW - previewW, previewW
}
