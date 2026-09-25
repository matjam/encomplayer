package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/matjam/encomplayer/internal/tea"
)

func click(m *Model, x, y int) {
	m.Update(tea.MouseClickMsg{X: x, Y: y, Button: tea.MouseLeft})
	m.Update(tea.MouseReleaseMsg{X: x, Y: y, Button: tea.MouseLeft})
}

func drag(m *Model, x0, y0, x1, y1 int) {
	m.Update(tea.MouseClickMsg{X: x0, Y: y0, Button: tea.MouseLeft})
	m.Update(tea.MouseMotionMsg{X: x1, Y: y1, Button: tea.MouseLeft})
	m.Update(tea.MouseReleaseMsg{X: x1, Y: y1, Button: tea.MouseLeft})
}

func wheel(m *Model, y int, button tea.MouseButton) {
	m.Update(tea.MouseWheelMsg{X: 60, Y: y, Button: button})
}

// tabX returns a column inside tab i's label.
func tabX(m *Model, i int) int {
	x := 0
	for j := range i {
		x += len([]rune(tabLabel(j, m.tabs[j]))) + 1
	}
	return x + 2
}

func TestMouseTabsAndQueue(t *testing.T) {
	m, fp := newTestModel(t)
	m.enqueue(testTracks("/music"))

	click(m, tabX(m, 0), tabBarRow)
	if m.active != 0 {
		t.Fatalf("tab click selected %d, want Queue", m.active)
	}

	row := tabBodyTop + queueFirstRow + 2
	click(m, 60, row)
	if m.queueList.Cursor() != 2 {
		t.Fatalf("click set cursor %d, want 2", m.queueList.Cursor())
	}
	if len(fp.played) != 0 {
		t.Fatal("single click started playback")
	}
	click(m, 60, row)
	if len(fp.played) != 1 || !strings.HasSuffix(fp.played[0], "03.ogg") {
		t.Fatalf("double click played %v, want 03.ogg", fp.played)
	}

	wheel(m, row, tea.MouseWheelUp)
	if m.queueList.Cursor() != 1 {
		t.Errorf("wheel up left cursor at %d, want 1", m.queueList.Cursor())
	}
}

func TestMouseHeaderAndSeek(t *testing.T) {
	m, fp := newTestModel(t)

	x, ok := 0, false
	for x = range m.width {
		if v, hit := m.volumeAt(x); hit && v == 30 {
			ok = true
			break
		}
	}
	if !ok {
		t.Fatal("no volume segment maps to 30%")
	}
	click(m, x, volumeRow)
	if fp.volume != 30 {
		t.Errorf("volume click set %d, want 30", fp.volume)
	}

	for x = range m.width {
		if i, hit := m.modeAt(x); hit && modeNames[i] == "RANDOM" {
			break
		}
	}
	click(m, x, modesRow)
	if !m.modes.Random {
		t.Error("clicking RANDOM did not toggle it")
	}

	m.position, m.length = 0, 100*time.Second
	x0, w := m.progressBarSpan()
	click(m, x0+w/2, m.progressRow())
	if len(fp.seeks) != 1 || fp.seeks[0] < 45*time.Second || fp.seeks[0] > 55*time.Second {
		t.Errorf("seek bar click sought %v, want about 50s", fp.seeks)
	}
}

func TestMouseBrowserColumns(t *testing.T) {
	m, fp := newTestModel(t)
	press(m, "3") // Artists: Daft Punk, Wendy Carlos
	parentW, currentW, _ := browserSplit(m, m.width)
	firstRow := tabBodyTop + 1

	click(m, parentW+currentW+3, firstRow) // preview column: open the album
	b := m.tabs[m.active].(*browserTab)
	if b.browser.Depth() != 1 {
		t.Fatalf("preview click depth = %d, want 1", b.browser.Depth())
	}

	click(m, parentW+3, firstRow) // into the album's tracks
	click(m, parentW+3, firstRow)
	if b.browser.Depth() != 2 {
		t.Fatalf("double click did not open the album, depth %d", b.browser.Depth())
	}

	click(m, parentW+3, firstRow+1)
	click(m, parentW+3, firstRow+1)
	if len(fp.played) != 1 || !strings.HasSuffix(fp.played[0], "02.mp3") {
		t.Fatalf("double click on a track played %v, want 02.mp3", fp.played)
	}

	click(m, 3, firstRow) // parent column goes back up
	if b.browser.Depth() != 1 {
		t.Errorf("parent click depth = %d, want 1", b.browser.Depth())
	}
}

func TestDragDividers(t *testing.T) {
	m, _ := newTestModel(t)

	top := m.footerTop()
	drag(m, 40, top, 40, top-6)
	if m.footerRows() != 10 || m.spectrumRows() != 7 {
		t.Fatalf("footer rows %d, spectrum rows %d; want 10 and 7", m.footerRows(), m.spectrumRows())
	}
	screen(t, m)

	drag(m, 40, m.footerTop(), 40, 0)
	if m.bodyHeight() < minBodyRows {
		t.Errorf("footer drag squeezed the body to %d rows", m.bodyHeight())
	}

	press(m, "2")
	parentW, _, _ := browserSplit(m, m.width)
	drag(m, parentW, tabBodyTop+3, m.width*40/100, tabBodyTop+3)
	if m.sizes.ParentPercent != 40 {
		t.Errorf("parent column is %d%%, want 40%%", m.sizes.ParentPercent)
	}
	drag(m, parentW, tabBodyTop+3, m.width, tabBodyTop+3)
	if _, currentW, _ := browserSplit(m, m.width); currentW < m.width/4 {
		t.Errorf("current column squeezed to %d cells", currentW)
	}
	screen(t, m)
}
