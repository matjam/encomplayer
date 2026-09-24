package builtin

import (
	"math/rand/v2"

	"github.com/matjam/encomplayer/internal/viz"
)

func init() {
	viz.Register(viz.Info{Name: "life", Description: "Conway's Game of Life, seeded by beats and the loudest bands"},
		func() viz.Visualizer { return &life{rng: newRand()} })
}

// lifeStep is the time between generations at silence; louder music runs
// the simulation faster.
const lifeStep = 0.12

// rPentomino is a tiny seed that grows for a long time.
var rPentomino = [][2]int{{1, 0}, {2, 0}, {0, 1}, {1, 1}, {1, 2}}

type life struct {
	rng        *rand.Rand
	w, h       int
	age, fresh []uint16 // 0 is dead; otherwise generations alive
	glow       []float64
	since      float64
}

func (l *life) Render(c *viz.Canvas, f *viz.Frame) error {
	w, h := c.PixelW(), c.PixelH()
	if w == 0 || h == 0 {
		return nil
	}
	if l.w != w || l.h != h {
		l.w, l.h = w, h
		l.age, l.fresh, l.glow = make([]uint16, w*h), make([]uint16, w*h), make([]float64, w*h)
		for range w * h / 6 {
			l.age[l.rng.IntN(w*h)] = 1
		}
	}
	dt := seconds(f)

	if f.Beat {
		// Drop a seed under the loudest band.
		bands := f.Bands(w)
		loudest := 0
		for x, v := range bands {
			if v > bands[loudest] {
				loudest = x
			}
		}
		l.seed(loudest, l.rng.IntN(h))
	}
	l.since += dt * (1 + 3*f.Level)
	for ; l.since >= lifeStep; l.since -= lifeStep {
		l.step()
	}

	keep := fade(dt, 0.4)
	p := c.Palette
	for i, a := range l.age {
		if a > 0 {
			l.glow[i] = 1
			// Newborn cells flare; old ones settle into the text colour.
			c.Pixel(i%w, i/w, p.Accent.Lerp(p.Text, float64(min(a, 12))/12))
			continue
		}
		l.glow[i] *= keep
		if l.glow[i] > 0.1 {
			c.Pixel(i%w, i/w, p.Background.Lerp(p.Dim, l.glow[i]))
		}
	}
	return nil
}

func (l *life) seed(x, y int) {
	for _, d := range rPentomino {
		l.age[l.wrap(x+d[0], y+d[1])] = 1
	}
}

func (l *life) wrap(x, y int) int {
	x, y = (x%l.w+l.w)%l.w, (y%l.h+l.h)%l.h
	return y*l.w + x
}

// step runs one generation on a torus.
func (l *life) step() {
	for y := range l.h {
		for x := range l.w {
			n := 0
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					if (dx != 0 || dy != 0) && l.age[l.wrap(x+dx, y+dy)] > 0 {
						n++
					}
				}
			}
			i := y*l.w + x
			switch a := l.age[i]; {
			case a > 0 && (n == 2 || n == 3):
				l.fresh[i] = min(a+1, 1000)
			case a == 0 && n == 3:
				l.fresh[i] = 1
			default:
				l.fresh[i] = 0
			}
		}
	}
	l.age, l.fresh = l.fresh, l.age
}
