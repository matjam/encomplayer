package builtin

import (
	"math/rand/v2"

	"github.com/matjam/encomplayer/internal/viz"
)

func init() {
	viz.Register(viz.Info{Name: "fire", Description: "Doom-style fire fed by the spectrum along its base"},
		func() viz.Visualizer { return &fire{rng: newRand()} })
}

// fireStep is how often the flames move, so they burn at the same speed
// whatever the frame rate.
const fireStep = 1.0 / 40

type fire struct {
	rng   *rand.Rand
	w, h  int
	heat  []float64
	since float64
}

func (fi *fire) Render(c *viz.Canvas, f *viz.Frame) error {
	w, h := c.PixelW(), c.PixelH()
	if w == 0 || h == 0 {
		return nil
	}
	if fi.w != w || fi.h != h {
		fi.w, fi.h, fi.heat = w, h, make([]float64, w*h)
	}

	// The bottom row burns as hot as the band beneath it.
	bands := tilted(f.Bands(w), 0.3)
	for x := range w {
		fi.heat[(h-1)*w+x] = clamp(bands[x]*1.3+f.BeatStrength*0.3, 0, 1)
	}

	fi.since += seconds(f)
	for ; fi.since >= fireStep; fi.since -= fireStep {
		fi.spread()
	}

	for py := range h {
		for px := range w {
			if v := fi.heat[py*w+px]; v > 0.04 {
				c.Pixel(px, py, c.Palette.Heat(v))
			}
		}
	}
	return nil
}

// spread moves heat up a row, cooling it and letting it drift sideways.
func (fi *fire) spread() {
	w, h := fi.w, fi.h
	cooling := 3.0 / float64(h)
	for y := 0; y < h-1; y++ {
		for x := range w {
			src := fi.heat[(y+1)*w+x]
			drift := min(w-1, max(0, x-fi.rng.IntN(3)+1))
			fi.heat[y*w+drift] = max(0, src-fi.rng.Float64()*cooling)
		}
	}
}
