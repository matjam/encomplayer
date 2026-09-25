package ui

import (
	"image"
	"testing"

	uv "github.com/charmbracelet/ultraviolet"

	"github.com/matjam/encomplayer/internal/art"
	"github.com/matjam/encomplayer/internal/tea"
)

// recordingRenderer remembers the boxes it was asked to fill, and draws as an
// overlay protocol does.
type recordingRenderer struct{ boxes []art.Box }

func (r *recordingRenderer) Render(_ image.Image, box art.Box) (art.Frame, error) {
	r.boxes = append(r.boxes, box)
	return art.Frame{
		Lines: make([]string, box.Rows),
		Place: func(x, y int) string { return "place" },
		Erase: func(x, y int) string { return "erase" },
	}, nil
}

// messages runs cmd and returns the messages it produces, expanding batches.
func messages(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	batch, ok := msg.(tea.BatchMsg)
	if !ok {
		return []tea.Msg{msg}
	}
	var out []tea.Msg
	for _, c := range batch {
		out = append(out, messages(c)...)
	}
	return out
}

func TestOverlayArtTurnsOffScrollOptimization(t *testing.T) {
	m, _ := newTestModel(t)
	overlay := &art.Frame{Place: func(x, y int) string { return "" }, Erase: func(x, y int) string { return "" }}
	inline := &art.Frame{}

	steps := []struct {
		name  string
		frame *art.Frame
		want  tea.Msg // nil means no change is sent
	}{
		{name: "overlay art loads", frame: overlay, want: tea.SetScrollOptimization(false)()},
		{name: "still overlay art", frame: overlay},
		{name: "inline art loads", frame: inline, want: tea.SetScrollOptimization(true)()},
		{name: "art cleared", frame: nil},
		{name: "overlay art again", frame: overlay, want: tea.SetScrollOptimization(false)()},
		{name: "art cleared again", frame: nil, want: tea.SetScrollOptimization(true)()},
	}
	for _, s := range steps {
		m.art.frame = s.frame
		var got tea.Msg
		for _, msg := range messages(m.scheduleArtPlacement()) {
			if msg == tea.SetScrollOptimization(true)() || msg == tea.SetScrollOptimization(false)() {
				got = msg
			}
		}
		if got != s.want {
			t.Errorf("%s: sent %#v, want %#v", s.name, got, s.want)
		}
	}
}

func TestStaleArtIsNotPlaced(t *testing.T) {
	m, _ := newTestModel(t)
	m.deps.Art = &recordingRenderer{}
	m.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	m.art.path, m.art.img = "cover.flac", image.NewRGBA(image.Rect(0, 0, 4, 4))
	m.receiveArt(m.renderArt(m.art.img)().(artMsg))

	steps := []struct {
		name      string
		footer    int
		wantPlace bool
	}{
		{name: "frame fits the box", footer: m.sizes.FooterRows, wantPlace: true},
		{name: "divider dragged, frame is stale", footer: m.sizes.FooterRows + 10, wantPlace: false},
		{name: "dragged back", footer: m.sizes.FooterRows, wantPlace: true},
	}
	for _, s := range steps {
		m.sizes.FooterRows = s.footer
		m.layout()
		m.art.placeSeq++
		if got := m.placeArt(artPlaceMsg(m.art.placeSeq)) != nil; got != s.wantPlace {
			t.Errorf("%s: placed = %t, want %t", s.name, got, s.wantPlace)
		}
	}
}

func TestCellSizeReportRerendersArt(t *testing.T) {
	m, _ := newTestModel(t)
	rec := &recordingRenderer{}
	m.deps.Art = rec
	m.Update(tea.WindowSizeMsg{Width: 160, Height: 50})

	box, _, _ := m.artBox()
	if box.Cols == 0 {
		t.Fatal("no room for art at 160x50")
	}
	m.art.path, m.art.img, m.art.box = "cover.flac", image.NewRGBA(image.Rect(0, 0, 4, 4)), box

	steps := []struct {
		name     string
		report   uv.CellSizeEvent
		wantCell image.Point // zero means no re-render
	}{
		{name: "empty report ignored", report: uv.CellSizeEvent{}},
		{name: "first report", report: uv.CellSizeEvent{Width: 9, Height: 19}, wantCell: image.Pt(9, 19)},
		{name: "same size again", report: uv.CellSizeEvent{Width: 9, Height: 19}},
		{name: "font zoom", report: uv.CellSizeEvent{Width: 12, Height: 25}, wantCell: image.Pt(12, 25)},
	}
	for _, s := range steps {
		cmd := m.update(s.report)
		if s.wantCell == (image.Point{}) {
			if cmd != nil {
				t.Errorf("%s: re-rendered", s.name)
			}
			continue
		}
		if cmd == nil {
			t.Fatalf("%s: did not re-render", s.name)
		}
		msg, ok := cmd().(artMsg)
		if !ok || msg.box.Cell != s.wantCell {
			t.Fatalf("%s: rendered %+v, want cell %v", s.name, msg.box, s.wantCell)
		}
		m.update(msg)
		if m.art.box != msg.box {
			t.Errorf("%s: art box = %+v, want %+v", s.name, m.art.box, msg.box)
		}
	}
	if len(rec.boxes) != 2 {
		t.Errorf("rendered %d times, want 2", len(rec.boxes))
	}
}
