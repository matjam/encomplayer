package builtin

import (
	"math"

	"github.com/matjam/encomplayer/internal/viz"
)

func init() {
	viz.Register(viz.Info{Name: "milkdrop", Description: "Feedback trails: each frame zooms and swirls the last, MilkDrop-style"},
		func() viz.Visualizer { return &milkdrop{} })
}

type rgbf struct{ r, g, b float64 }

// milkdrop keeps the previous frame at pixel resolution, warps it, fades
// it, and draws the waveform on top.
type milkdrop struct {
	w, h      int
	cur, next []rgbf
	swirl     float64
}

func (m *milkdrop) Render(c *viz.Canvas, f *viz.Frame) error {
	w, h := c.PixelW(), c.PixelH()
	if w == 0 || h == 0 {
		return nil
	}
	if m.w != w || m.h != h {
		m.w, m.h = w, h
		m.cur, m.next = make([]rgbf, w*h), make([]rgbf, w*h)
	}
	dt := seconds(f)
	t := f.Time.Seconds()
	m.swirl = math.Sin(t*0.3) * (0.02 + 0.06*f.Mid)

	zoom := 1 - dt*(0.6+2*f.Bass)
	keep := fade(dt, 0.35)
	sin, cos := math.Sincos(m.swirl)
	cx, cy := float64(w)/2, float64(h)/2
	for py := range h {
		for px := range w {
			// Sample the old frame slightly closer to the centre, so the
			// picture flows outwards.
			x, y := float64(px)-cx, float64(py)-cy
			v := m.sample(cx+(x*cos-y*sin)*zoom, cy+(x*sin+y*cos)*zoom)
			m.next[py*w+px] = rgbf{v.r * keep, v.g * keep, v.b * keep}
		}
	}
	m.cur, m.next = m.next, m.cur

	col := c.Palette.Cycle(t * 0.07)
	bright := 0.6 + 0.4*f.Level
	g := 0.5 / math.Max(0.15, f.Peak)
	prevX, prevY := -1, 0
	for px := range w {
		s := f.Mono(px * len(f.Left) / w)
		py := int(cy - s*g*cy)
		if prevX >= 0 {
			m.stroke(prevX, prevY, px, py, col, bright)
		}
		prevX, prevY = px, py
	}
	if f.Beat {
		m.ring(cx, cy, math.Min(cx, cy)*0.6, c.Palette.Bright)
	}

	bg := c.Palette.Background
	for i, v := range m.cur {
		if v.r+v.g+v.b > 6 {
			c.Pixel(i%w, i/w, viz.RGB{R: channel(v.r, bg.R), G: channel(v.g, bg.G), B: channel(v.b, bg.B)})
		}
	}
	return nil
}

// channel clamps a faded value between the background and full brightness,
// so trails fade into the theme rather than to black.
func channel(v float64, floor uint8) uint8 {
	return uint8(math.Max(float64(floor), math.Min(255, v)))
}

// sample reads the previous frame at a fractional position, blending the
// four nearest pixels so repeated warping stays smooth instead of blocky.
func (m *milkdrop) sample(x, y float64) rgbf {
	x0, y0 := int(math.Floor(x)), int(math.Floor(y))
	fx, fy := x-float64(x0), y-float64(y0)
	at := func(px, py int) rgbf {
		if px < 0 || py < 0 || px >= m.w || py >= m.h {
			return rgbf{}
		}
		return m.cur[py*m.w+px]
	}
	a, b, c, d := at(x0, y0), at(x0+1, y0), at(x0, y0+1), at(x0+1, y0+1)
	mix := func(p, q, r, s float64) float64 {
		return (p*(1-fx)+q*fx)*(1-fy) + (r*(1-fx)+s*fx)*fy
	}
	return rgbf{mix(a.r, b.r, c.r, d.r), mix(a.g, b.g, c.g, d.g), mix(a.b, b.b, c.b, d.b)}
}

func (m *milkdrop) plot(x, y int, col viz.RGB, bright float64) {
	if x < 0 || y < 0 || x >= m.w || y >= m.h {
		return
	}
	m.cur[y*m.w+x] = rgbf{float64(col.R) * bright, float64(col.G) * bright, float64(col.B) * bright}
}

func (m *milkdrop) stroke(x0, y0, x1, y1 int, col viz.RGB, bright float64) {
	steps := max(abs(x1-x0), abs(y1-y0), 1)
	for i := range steps + 1 {
		m.plot(x0+(x1-x0)*i/steps, y0+(y1-y0)*i/steps, col, bright)
	}
}

func (m *milkdrop) ring(cx, cy, r float64, col viz.RGB) {
	for i := range 180 {
		sin, cos := math.Sincos(2 * math.Pi * float64(i) / 180)
		m.plot(int(cx+cos*r), int(cy+sin*r), col, 1)
	}
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
