package builtin

import (
	"math"

	"github.com/matjam/encomplayer/internal/viz"
)

func init() {
	viz.Register(viz.Info{Name: "tunnel", Description: "Flying down a chequered tunnel that twists with the music"},
		func() viz.Visualizer { return &tunnel{} })
}

type tunnel struct {
	w, h         int
	angle, depth []float64 // per pixel, precomputed for the size
	travel, turn float64
	flash        float64
}

func (t *tunnel) Render(c *viz.Canvas, f *viz.Frame) error {
	w, h := c.PixelW(), c.PixelH()
	if w == 0 || h == 0 {
		return nil
	}
	if t.w != w || t.h != h {
		t.resize(w, h)
	}
	dt := seconds(f)
	t.travel += dt * (0.3 + 1.8*f.Level)
	t.turn += dt * (0.1 + 0.6*f.Mid)
	t.flash = math.Max(f.BeatStrength, t.flash*fade(dt, 0.15))

	p := c.Palette
	for i, d := range t.depth {
		u := t.angle[i]/math.Pi*4 + t.turn
		v := d + t.travel
		check := (int(math.Floor(u)) + int(math.Floor(v*4))) & 1
		// Pixels near the centre are far away, so they fade into the dark.
		shade := clamp(1.6/d, 0, 1)
		level := shade * (0.35 + 0.5*float64(check))
		col := p.Ramp(level + t.flash*0.3)
		c.Pixel(i%w, i/w, p.Background.Lerp(col, shade+t.flash*0.2))
	}
	return nil
}

func (t *tunnel) resize(w, h int) {
	t.w, t.h = w, h
	t.angle, t.depth = make([]float64, w*h), make([]float64, w*h)
	cx, cy := float64(w)/2, float64(h)/2
	scale := 2 / math.Min(float64(w), float64(h))
	for py := range h {
		for px := range w {
			x, y := (float64(px)-cx)*scale, (float64(py)-cy)*scale
			i := py*w + px
			t.angle[i] = math.Atan2(y, x)
			t.depth[i] = 1 / math.Max(0.02, math.Hypot(x, y))
		}
	}
}
