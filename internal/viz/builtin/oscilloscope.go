package builtin

import (
	"math"

	"github.com/matjam/encomplayer/internal/viz"
)

func init() {
	viz.Register(viz.Info{Name: "oscilloscope", Description: "Triggered waveform trace, like a bench scope"},
		func() viz.Visualizer { return &oscilloscope{} })
	viz.Register(viz.Info{Name: "stereo-scope", Description: "Left and right channels traced one above the other"},
		func() viz.Visualizer { return &stereoScope{} })
}

// scopeSpan is how many samples one trace shows, about 23 ms at 44.1 kHz.
const scopeSpan = 1024

// gain tracks the signal's peak so quiet passages still fill the screen,
// without jumping from frame to frame.
type gain struct{ peak float64 }

func (g *gain) update(peak, dt float64) float64 {
	g.peak = math.Max(peak, g.peak*fade(dt, 1))
	return 0.9 / math.Max(0.2, g.peak)
}

type oscilloscope struct{ gain gain }

func (o *oscilloscope) Render(c *viz.Canvas, f *viz.Frame) error {
	mono := make([]float64, len(f.Left))
	for i := range mono {
		mono[i] = f.Mono(i)
	}
	g := o.gain.update(f.Peak, seconds(f))
	drawTrace(c, mono, g, 0, c.DotH(), c.Palette.Bright)
	return nil
}

type stereoScope struct{ gain gain }

func (s *stereoScope) Render(c *viz.Canvas, f *viz.Frame) error {
	g := s.gain.update(f.Peak, seconds(f))
	half := c.DotH() / 2
	drawTrace(c, f.Left, g, 0, half, c.Palette.Bright)
	drawTrace(c, f.Right, g, half, c.DotH()-half, c.Palette.Accent)
	return nil
}

// drawTrace plots samples in the band of dot rows [top, top+height),
// starting at a rising zero crossing so a steady tone stands still.
func drawTrace(c *viz.Canvas, samples []float64, gain float64, top, height int, col viz.RGB) {
	w := c.DotW()
	if w == 0 || height <= 0 {
		return
	}
	mid := float64(top) + float64(height-1)/2
	for x := 0; x < w; x += 4 {
		c.Dot(x, int(mid), c.Palette.Grid)
	}

	start := trigger(samples, len(samples)-scopeSpan)
	span := min(scopeSpan, len(samples)-start)
	prevX, prevY := -1.0, 0.0
	for x := range w {
		s := samples[start+x*span/w]
		y := mid - s*gain*float64(height-1)/2
		if prevX >= 0 {
			c.Line(prevX, prevY, float64(x), y, col)
		}
		prevX, prevY = float64(x), y
	}
}

// trigger finds a rising zero crossing in the first part of samples,
// leaving at least scopeSpan samples after it.
func trigger(samples []float64, limit int) int {
	for i := 1; i < limit; i++ {
		if samples[i-1] < 0 && samples[i] >= 0 {
			return i
		}
	}
	return max(0, limit)
}
