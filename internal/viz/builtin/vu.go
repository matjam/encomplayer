package builtin

import (
	"math"

	"github.com/matjam/encomplayer/internal/viz"
)

func init() {
	viz.Register(viz.Info{Name: "vu", Description: "Pair of analogue VU meters with swinging needles"},
		func() viz.Visualizer { return &vu{} })
}

// Needles swing with VU ballistics, reaching their reading in about 300 ms.
const vuResponse = 0.3

// The scale sweeps from vuMin to vuMax radians, anticlockwise from the
// right. As on a real meter, the needle moves with voltage, 0 VU sits at
// -18 dBFS RMS, full scale is +3 VU, and the red zone starts at 0 VU.
const (
	vuMin, vuMax = math.Pi * 0.83, math.Pi * 0.17
	vuReference  = -18.0
	vuFullScale  = 3.0
)

// vuRed is where 0 VU falls on the scale.
var vuRed = math.Pow(10, -vuFullScale/20)

// needle converts a frame level, which spans 48 dB below full scale, to
// the needle's place on the scale.
func needle(level float64) float64 {
	if level <= 0 {
		return 0
	}
	dbfs := level*48 - 48
	vu := dbfs - vuReference
	return clamp(math.Pow(10, (vu-vuFullScale)/20), 0, 1)
}

type vu struct{ left, right float64 }

func (v *vu) Render(c *viz.Canvas, f *viz.Frame) error {
	dt := seconds(f)
	follow := 1 - math.Exp(-dt/vuResponse*3)
	v.left += (needle(f.LevelLeft) - v.left) * follow
	v.right += (needle(f.LevelRight) - v.right) * follow

	w := c.DotW() / 2
	drawMeter(c, 0, w, v.left, "L")
	drawMeter(c, w, c.DotW()-w, v.right, "R")
	return nil
}

// drawMeter draws one meter in the dot columns [x0, x0+w).
func drawMeter(c *viz.Canvas, x0, w int, level float64, label string) {
	h := c.DotH()
	if w < 8 || h < 8 {
		return
	}
	p := c.Palette
	cx, cy := float64(x0)+float64(w)/2, float64(h)-2
	radius := math.Min(float64(w)/2-2, float64(h)-4)

	// Scale arc and tick marks.
	const marks = 60
	for i := range marks + 1 {
		t := float64(i) / marks
		a := vuMin + (vuMax-vuMin)*t
		col := p.Dim
		if t > vuRed {
			col = p.Error
		}
		x, y := cx+math.Cos(a)*radius, cy-math.Sin(a)*radius
		c.Dot(int(x), int(y), col)
		if i%10 == 0 {
			c.Line(x, y, cx+math.Cos(a)*radius*0.9, cy-math.Sin(a)*radius*0.9, col)
		}
	}

	// The needle, from the pivot past the scale.
	a := vuMin + (vuMax-vuMin)*clamp(level, 0, 1)
	needle := p.Bright
	if level > vuRed {
		needle = p.Accent
	}
	c.Line(cx, cy, cx+math.Cos(a)*radius*1.02, cy-math.Sin(a)*radius*1.02, needle)

	c.Text(int(cx)/2, (int(cy)-int(radius*0.45))/4, label, p.Text)
	if w/2 >= 6 {
		c.Text(int(cx)/2-1, c.H-1, "VU", p.Dim)
	}
}

func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }
