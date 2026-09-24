package builtin

import "github.com/matjam/encomplayer/internal/viz"

func init() {
	viz.Register(viz.Info{Name: "waterfall", Description: "Scrolling spectrogram: time runs down, pitch runs across"},
		func() viz.Visualizer { return &waterfall{} })
}

// waterfallRow is how often a new row of history scrolls in.
const waterfallRow = 0.04

type waterfall struct {
	rows  [][]float64 // newest first
	since float64
}

func (w *waterfall) Render(c *viz.Canvas, f *viz.Frame) error {
	h := c.PixelH()
	w.since += seconds(f)
	if w.since >= waterfallRow || len(w.rows) == 0 {
		w.since = 0
		row := tilted(f.Bands(max(1, c.W)), 0.25)
		w.rows = append([][]float64{row}, w.rows...)
	}
	if len(w.rows) > h {
		w.rows = w.rows[:h]
	}

	for py, row := range w.rows {
		for x := range c.W {
			v := row[x*len(row)/max(1, c.W)]
			if v > 0.05 {
				c.Pixel(x, py, c.Palette.Heat(v))
			}
		}
	}
	return nil
}
