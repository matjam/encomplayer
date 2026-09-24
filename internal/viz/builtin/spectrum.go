package builtin

import (
	"math"

	"github.com/matjam/encomplayer/internal/viz"
)

func init() {
	viz.Register(viz.Info{Name: "spectrum", Description: "Spectrum analyser bars that glow hotter towards the top"},
		func() viz.Visualizer { return &spectrum{} })
}

// spectrumBands is how many bands spread across the strip, whatever its
// width.
const spectrumBands = 64

// eighths are the block glyphs that fill a cell from the bottom in eighths.
var eighths = []rune(" ▁▂▃▄▅▆▇█")

type spectrum struct{ levels []float64 }

func (s *spectrum) Render(c *viz.Canvas, f *viz.Frame) error {
	bands := f.Bands(spectrumBands)
	if len(s.levels) != len(bands) {
		s.levels = make([]float64, len(bands))
	}
	// Bars jump up at once and fall back gradually, so they do not flicker.
	keep := fade(seconds(f), 0.25)
	for i, v := range bands {
		s.levels[i] = math.Max(v, s.levels[i]*keep)
	}

	p := c.Palette
	steps := c.H * (len(eighths) - 1)
	for y := range c.H {
		fromBottom := c.H - 1 - y
		col := p.Text
		switch height := float64(fromBottom+1) / float64(c.H); {
		case height > 0.85 && c.H > 1:
			col = p.Accent
		case height > 0.5:
			col = p.Bright
		}
		for x := range c.W {
			v := s.levels[x*len(s.levels)/max(1, c.W)]
			lit := int(v*float64(steps)+0.5) - fromBottom*(len(eighths)-1)
			if lit > 0 {
				c.Set(x, y, eighths[min(lit, len(eighths)-1)], col)
			}
		}
	}
	return nil
}
