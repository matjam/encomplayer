package builtin

import (
	"math"

	"github.com/matjam/encomplayer/internal/viz"
)

func init() {
	viz.Register(viz.Info{Name: "mirror", Description: "Spectrum mirrored about the centre line, bass in the middle"},
		func() viz.Visualizer { return &mirror{} })
}

type mirror struct{ levels []float64 }

func (m *mirror) Render(c *viz.Canvas, f *viz.Frame) error {
	half := (c.W + 1) / 2
	bands := f.Bands(max(1, half))
	if len(m.levels) != len(bands) {
		m.levels = make([]float64, len(bands))
	}
	keep := fade(seconds(f), 0.15)
	for i, v := range bands {
		m.levels[i] = math.Max(v, m.levels[i]*keep)
	}

	h := c.PixelH()
	mid := float64(h) / 2
	for x := range c.W {
		// The bass sits in the centre and treble spreads to both edges.
		d := x - c.W/2
		if d < 0 {
			d = -d - 1
		}
		v := m.levels[min(len(m.levels)-1, d)]
		reach := v * mid
		for py := range h {
			dist := math.Abs(float64(py) + 0.5 - mid)
			if dist <= reach {
				c.Pixel(x, py, c.Palette.Ramp(0.3+0.7*dist/math.Max(1, mid)))
			}
		}
	}
	return nil
}
