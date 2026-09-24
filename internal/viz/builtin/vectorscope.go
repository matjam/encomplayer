package builtin

import (
	"math"

	"github.com/matjam/encomplayer/internal/viz"
)

func init() {
	viz.Register(viz.Info{Name: "vectorscope", Description: "Stereo goniometer: mono is a vertical line, wide stereo a cloud"},
		func() viz.Visualizer { return &vectorscope{} })
}

// vectorscope plots each sample pair as a point, with phosphor that
// glows for a moment after the beam passes.
type vectorscope struct {
	w, h     int
	phosphor []float64
	gain     gain
}

func (v *vectorscope) Render(c *viz.Canvas, f *viz.Frame) error {
	w, h := c.DotW(), c.DotH()
	if w == 0 || h == 0 {
		return nil
	}
	if v.w != w || v.h != h {
		v.w, v.h, v.phosphor = w, h, make([]float64, w*h)
	}
	keep := fade(seconds(f), 0.08)
	for i := range v.phosphor {
		v.phosphor[i] *= keep
	}

	cx, cy := float64(w)/2, float64(h)/2
	radius := math.Min(cx, cy) * 0.95
	g := v.gain.update(f.Peak, seconds(f))
	for i := range f.Left {
		l, r := f.Left[i]*g, f.Right[i]*g
		// Rotate 45° so the mid signal points up and the side signal
		// spreads sideways.
		x := cx + (l-r)/math.Sqrt2*radius
		y := cy - (l+r)/math.Sqrt2*radius
		px, py := int(x), int(y)
		if px >= 0 && py >= 0 && px < w && py < h {
			v.phosphor[py*w+px] = math.Min(1, v.phosphor[py*w+px]+0.35)
		}
	}

	p := c.Palette
	// The L and R axes are the diagonals.
	c.Line(cx-radius*0.7, cy-radius*0.7, cx+radius*0.7, cy+radius*0.7, p.Grid)
	c.Line(cx-radius*0.7, cy+radius*0.7, cx+radius*0.7, cy-radius*0.7, p.Grid)
	for py := range h {
		for px := range w {
			if b := v.phosphor[py*w+px]; b > 0.05 {
				c.Dot(px, py, p.Ramp(0.3+0.7*b))
			}
		}
	}
	return nil
}
