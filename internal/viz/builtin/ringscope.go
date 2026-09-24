package builtin

import (
	"math"

	"github.com/matjam/encomplayer/internal/viz"
)

func init() {
	viz.Register(viz.Info{Name: "ring-scope", Description: "Waveform wrapped around a slowly turning circle"},
		func() viz.Visualizer { return &ringScope{} })
}

type ringScope struct {
	gain  gain
	angle float64
}

func (r *ringScope) Render(c *viz.Canvas, f *viz.Frame) error {
	w, h := float64(c.DotW()), float64(c.DotH())
	if w == 0 || h == 0 {
		return nil
	}
	dt := seconds(f)
	r.angle += dt * (0.2 + f.Level)
	g := r.gain.update(f.Peak, dt)

	cx, cy := w/2, h/2
	base := math.Min(cx, cy) * 0.55
	amp := math.Min(cx, cy) * 0.4

	const points = 360
	col := c.Palette.Cycle(f.Time.Seconds() * 0.05)
	var firstX, firstY, prevX, prevY float64
	for i := range points {
		s := f.Mono(i * len(f.Left) / points)
		rad := base + s*g*amp
		a := r.angle + 2*math.Pi*float64(i)/points
		x, y := cx+math.Cos(a)*rad, cy+math.Sin(a)*rad
		if i == 0 {
			firstX, firstY = x, y
		} else {
			c.Line(prevX, prevY, x, y, col)
		}
		prevX, prevY = x, y
	}
	c.Line(prevX, prevY, firstX, firstY, col)

	// A faint inner ring marks silence.
	for i := range 90 {
		a := 2 * math.Pi * float64(i) / 90
		c.Dot(int(cx+math.Cos(a)*base*0.5), int(cy+math.Sin(a)*base*0.5), c.Palette.Grid)
	}
	return nil
}
