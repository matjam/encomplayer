package builtin

import (
	"math"

	"github.com/matjam/encomplayer/internal/viz"
)

func init() {
	viz.Register(viz.Info{Name: "plasma", Description: "Demoscene plasma that churns faster as the music gets louder"},
		func() viz.Visualizer { return &plasma{} })
}

// plasmaShades is the size of the per-frame colour table.
const plasmaShades = 256

// plasma sums four sine fields. Three of them depend only on the column,
// the row, or their sum, so they are computed once per line rather than
// once per pixel.
type plasma struct {
	t, glow float64

	w, h       int
	dist       []float64 // distance of each pixel from the centre
	cols, rows []float64
	diag       []float64
	shades     [plasmaShades]viz.RGB
}

func (p *plasma) Render(c *viz.Canvas, f *viz.Frame) error {
	w, h := c.PixelW(), c.PixelH()
	if w == 0 || h == 0 {
		return nil
	}
	if p.w != w || p.h != h {
		p.resize(w, h)
	}
	dt := seconds(f)
	p.t += dt * (0.4 + 2.5*f.Level)
	p.glow = math.Max(f.Bass, p.glow*fade(dt, 0.3))
	if f.Beat {
		p.glow = 1
	}

	t := p.t
	scale := p.scale()
	for x := range w {
		p.cols[x] = math.Sin((float64(x)-float64(w)/2)*scale + t)
	}
	for y := range h {
		p.rows[y] = math.Sin((float64(y)-float64(h)/2)*scale*0.8 + t*1.3)
	}
	for d := range p.diag {
		p.diag[d] = math.Sin((float64(d)-float64(w+h)/2)*scale*0.6 + t*0.7)
	}
	brightness := 0.35 + 0.65*p.glow
	for i := range p.shades {
		v := float64(i)/(plasmaShades-1)*8 - 4
		p.shades[i] = c.Palette.Cycle(v/8 + t*0.03).Scale(brightness)
	}

	for y := range h {
		for x := range w {
			v := p.cols[x] + p.rows[y] + p.diag[x+y] + math.Sin(p.dist[y*w+x]*1.4-t*2)
			shade := int((v + 4) / 8 * (plasmaShades - 1))
			c.Pixel(x, y, p.shades[min(plasmaShades-1, max(0, shade))])
		}
	}
	return nil
}

func (p *plasma) scale() float64 { return 12 / math.Max(1, math.Min(float64(p.w), float64(p.h))) }

func (p *plasma) resize(w, h int) {
	p.w, p.h = w, h
	p.cols, p.rows, p.diag = make([]float64, w), make([]float64, h), make([]float64, w+h)
	p.dist = make([]float64, w*h)
	scale := p.scale()
	for y := range h {
		for x := range w {
			p.dist[y*w+x] = math.Hypot(float64(x)-float64(w)/2, float64(y)-float64(h)/2) * scale
		}
	}
}
