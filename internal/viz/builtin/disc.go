package builtin

import (
	"math"
	"strings"

	"github.com/matjam/encomplayer/internal/viz"
)

func init() {
	viz.Register(viz.Info{Name: "disc", Description: "ENCOM identity disc: its segments light with the spectrum"},
		func() viz.Visualizer { return &disc{} })
}

const discSegments = 32

type disc struct {
	levels []float64
	spin   float64
	pulse  float64
}

func (d *disc) Render(c *viz.Canvas, f *viz.Frame) error {
	w, h := float64(c.DotW()), float64(c.DotH())
	if w == 0 || h == 0 {
		return nil
	}
	dt := seconds(f)
	bands := f.Bands(discSegments)
	if len(d.levels) != len(bands) {
		d.levels = make([]float64, len(bands))
	}
	keep := fade(dt, 0.2)
	for i, v := range bands {
		d.levels[i] = math.Max(v, d.levels[i]*keep)
	}
	d.spin += dt * (0.3 + 2*f.Level)
	d.pulse = math.Max(f.BeatStrength, d.pulse*fade(dt, 0.12))

	pal := c.Palette
	cx, cy := w/2, h/2
	outer := math.Min(cx, cy) * 0.95
	arc := func(r, from, to float64, col viz.RGB) {
		steps := max(2, int((to-from)*r))
		for i := range steps + 1 {
			a := from + (to-from)*float64(i)/float64(steps)
			sin, cos := math.Sincos(a)
			c.Dot(int(cx+cos*r), int(cy+sin*r), col)
		}
	}

	// Outer rim: one segment per band, lit by its level.
	seg := 2 * math.Pi / discSegments
	for i, v := range d.levels {
		from := float64(i)*seg + seg*0.12
		col := pal.Grid
		if v > 0.25 {
			col = pal.Ramp(0.4 + 0.6*v)
		}
		for r := outer * 0.84; r <= outer; r += 1 {
			arc(r, from, from+seg*0.76, col)
		}
	}

	// Middle ring: dashes turning with the music.
	for i := range 12 {
		from := d.spin + float64(i)*math.Pi/6
		arc(outer*0.68, from, from+math.Pi/12, pal.Text)
	}
	// Inner ring turns the other way.
	for i := range 6 {
		from := -d.spin*1.6 + float64(i)*math.Pi/3
		arc(outer*0.5, from, from+math.Pi/5, pal.Bright)
	}

	// The core glows and swells on each beat.
	core := outer * (0.22 + 0.12*d.pulse)
	for r := 1.0; r <= core; r++ {
		arc(r, 0, 2*math.Pi, pal.Accent.Lerp(pal.Bright, d.pulse*(1-r/core)))
	}

	if label := strings.ToUpper(f.Track.Title); label != "" && c.H >= 6 {
		label = string([]rune(label)[:min(len([]rune(label)), c.W)])
		c.Text((c.W-len([]rune(label)))/2, c.H-1, label, pal.Dim)
	}
	return nil
}
