package builtin

import (
	"math"

	"github.com/matjam/encomplayer/internal/viz"
)

func init() {
	viz.Register(viz.Info{Name: "peaks", Description: "Winamp-style bars with peak caps that hang, then fall"},
		func() viz.Visualizer { return &peaks{} })
}

const (
	peakHold = 0.5 // seconds a cap hangs before falling
	peakFall = 0.9 // fraction of the height a cap falls per second
)

type peaks struct {
	levels, caps, hold []float64
}

func (p *peaks) Render(c *viz.Canvas, f *viz.Frame) error {
	// Bars are two cells wide with a one-cell gap.
	n := max(1, (c.W+1)/3)
	bands := f.Bands(n)
	if len(p.levels) != n {
		p.levels, p.caps, p.hold = make([]float64, n), make([]float64, n), make([]float64, n)
	}
	dt := seconds(f)
	keep := fade(dt, 0.12)
	pal := c.Palette
	steps := float64(c.H * (len(eighths) - 1))

	for i, v := range bands {
		p.levels[i] = math.Max(v, p.levels[i]*keep)
		if p.levels[i] >= p.caps[i] {
			p.caps[i], p.hold[i] = p.levels[i], peakHold
		} else if p.hold[i] -= dt; p.hold[i] < 0 {
			p.caps[i] = math.Max(p.levels[i], p.caps[i]-peakFall*dt)
		}

		lit := int(p.levels[i]*steps + 0.5)
		capRow := c.H - 1 - min(c.H-1, int(p.caps[i]*float64(c.H)))
		for y := range c.H {
			fromBottom := c.H - 1 - y
			col := pal.Ramp(0.35 + 0.65*float64(fromBottom+1)/float64(c.H))
			cellLit := lit - fromBottom*(len(eighths)-1)
			for dx := range 2 {
				x := i*3 + dx
				switch {
				case cellLit > 0:
					c.Set(x, y, eighths[min(cellLit, len(eighths)-1)], col)
				case y == capRow && p.caps[i] > 0.02:
					c.Set(x, y, '▔', pal.Accent)
				}
			}
		}
	}
	return nil
}
