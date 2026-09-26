package script

import (
	"github.com/matjam/encomplayer/internal/viz"
)

// newFrameTable pushes the frame table, slotFrame, with its sample
// buffers. updateFrame fills it before every render.
func (p *plugin) newFrameTable() {
	l := p.l
	l.CreateTable(0, 24)
	for _, b := range []struct {
		name string
		buf  []float64
	}{{"left", p.left}, {"right", p.right}, {"spectrum", p.spectr}} {
		l.PushBuffer(b.buf)
		l.SetField(-2, b.name)
	}
	for _, name := range []string{"title", "artist", "album"} {
		l.PushString("")
		l.SetField(-2, name)
	}
}

// updateFrame copies f into the frame table.
func (p *plugin) updateFrame(f *viz.Frame) {
	l := p.l
	copy(p.left, f.Left)
	copy(p.right, f.Right)
	copy(p.spectr, f.Spectrum)

	for _, n := range []struct {
		name string
		v    float64
	}{
		{"time", f.Time.Seconds()},
		{"dt", f.Delta.Seconds()},
		{"bin_hz", f.BinHz},
		{"level", f.Level},
		{"level_left", f.LevelLeft},
		{"level_right", f.LevelRight},
		{"peak", f.Peak},
		{"bass", f.Bass},
		{"mid", f.Mid},
		{"treble", f.Treble},
		{"beat_strength", f.BeatStrength},
		{"position", f.Track.Position.Seconds()},
		{"duration", f.Track.Duration.Seconds()},
	} {
		l.PushNumber(n.v)
		l.SetField(slotFrame, n.name)
	}
	l.PushInteger(f.SampleRate)
	l.SetField(slotFrame, "rate")
	l.PushBoolean(f.Playing)
	l.SetField(slotFrame, "playing")
	l.PushBoolean(f.Beat)
	l.SetField(slotFrame, "beat")

	if t := f.Track; t.Title != p.track.Title || t.Artist != p.track.Artist || t.Album != p.track.Album {
		for _, s := range [][2]string{{"title", t.Title}, {"artist", t.Artist}, {"album", t.Album}} {
			l.PushString(s[1])
			l.SetField(slotFrame, s[0])
		}
	}
	p.track = f.Track
}

// updateCanvas describes the canvas's size to the script and blanks the
// pixel buffer, making a new one when the size changed.
func (p *plugin) updateCanvas(c *viz.Canvas) {
	l := p.l
	if pw, ph := c.PixelW(), c.PixelH(); pw != p.pw || ph != p.ph {
		p.pw, p.ph = pw, ph
		p.pixels = make([]int32, pw*ph)
		l.PushBuffer(p.pixels)
		l.SetField(slotCanvas, "pixels")
		for _, d := range []struct {
			name string
			v    int
		}{{"w", c.W}, {"h", c.H}, {"pw", pw}, {"ph", ph}, {"dw", c.DotW()}, {"dh", c.DotH()}} {
			l.PushInteger(d.v)
			l.SetField(slotCanvas, d.name)
		}
	}
	for i := range p.pixels {
		p.pixels[i] = -1
	}
}

// updatePalette copies the theme's colours into the palette table when
// they change.
func (p *plugin) updatePalette(pal viz.Palette) {
	if p.paletteSet && pal == p.palette {
		return
	}
	p.palette, p.paletteSet = pal, true
	l := p.l
	for _, c := range []struct {
		name string
		rgb  viz.RGB
	}{
		{"background", pal.Background},
		{"text", pal.Text},
		{"bright", pal.Bright},
		{"dim", pal.Dim},
		{"grid", pal.Grid},
		{"accent", pal.Accent},
		{"error", pal.Error},
	} {
		l.PushInteger(c.rgb.Pack())
		l.SetField(slotPalette, c.name)
	}
}
