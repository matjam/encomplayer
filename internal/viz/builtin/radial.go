package builtin

import (
	"math"

	"github.com/matjam/encomplayer/internal/viz"
)

func init() {
	viz.Register(viz.Info{Name: "radial", Description: "Spectrum radiating from a sun that swells with the bass"},
		func() viz.Visualizer { return &radial{} })
}

const radialBands = 48

type radial struct {
	levels []float64
	spin   float64
}

func (r *radial) Render(c *viz.Canvas, f *viz.Frame) error {
	w, h := float64(c.DotW()), float64(c.DotH())
	if w == 0 || h == 0 {
		return nil
	}
	dt := seconds(f)
	bands := f.Bands(radialBands)
	if len(r.levels) != len(bands) {
		r.levels = make([]float64, len(bands))
	}
	keep := fade(dt, 0.15)
	for i, v := range bands {
		r.levels[i] = math.Max(v, r.levels[i]*keep)
	}
	r.spin += dt * 0.15

	cx, cy := w/2, h/2
	limit := math.Min(cx, cy)
	core := limit * (0.18 + 0.12*f.Bass)
	p := c.Palette

	// Each band appears twice, mirrored, so the rays are symmetric.
	rays := 2 * len(r.levels)
	for i := range rays {
		band := i
		if band >= len(r.levels) {
			band = rays - 1 - i
		}
		v := r.levels[band]
		a := r.spin + 2*math.Pi*float64(i)/float64(rays)
		sin, cos := math.Sincos(a)
		outer := core + (limit-core)*v
		c.Line(cx+cos*core, cy+sin*core, cx+cos*outer, cy+sin*outer, p.Ramp(0.3+0.7*v))
	}
	for i := range 120 {
		sin, cos := math.Sincos(2 * math.Pi * float64(i) / 120)
		c.Dot(int(cx+cos*core), int(cy+sin*core), p.Accent)
	}
	return nil
}
