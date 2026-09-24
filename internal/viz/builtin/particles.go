package builtin

import (
	"math"
	"math/rand/v2"

	"github.com/matjam/encomplayer/internal/viz"
)

func init() {
	viz.Register(viz.Info{Name: "particles", Description: "Fountains of sparks thrown up on every beat"},
		func() viz.Visualizer { return &particles{rng: newRand()} })
}

const (
	maxParticles = 1500
	gravity      = 1.6 // canvas heights per second squared
)

// particle positions are fractions of the canvas, so resizing keeps them.
type particle struct {
	x, y, vx, vy, life float64
	hue                float64
}

type particles struct {
	rng  *rand.Rand
	list []particle
}

func (p *particles) Render(c *viz.Canvas, f *viz.Frame) error {
	w, h := float64(c.DotW()), float64(c.DotH())
	if w == 0 || h == 0 {
		return nil
	}
	dt := seconds(f)

	if f.Beat {
		p.burst(p.rng.Float64()*0.8+0.1, 40+int(160*f.BeatStrength), 0.9+f.BeatStrength)
	}
	// A steady trickle from the middle follows the level.
	if trickle := int(f.Level * 300 * dt); trickle > 0 {
		p.burst(0.5, trickle, 0.5+f.Level*0.6)
	}

	aspect := h / w
	live := p.list[:0]
	for _, pt := range p.list {
		pt.vy += gravity * dt
		oldX, oldY := pt.x, pt.y
		pt.x += pt.vx * dt * aspect
		pt.y += pt.vy * dt
		pt.life -= dt * 0.6
		if pt.life <= 0 || pt.y > 1 || pt.x < 0 || pt.x > 1 {
			continue
		}
		col := c.Palette.Cycle(pt.hue).Lerp(c.Palette.Background, 1-pt.life)
		c.Line(oldX*w, oldY*h, pt.x*w, pt.y*h, col)
		live = append(live, pt)
	}
	p.list = live
	return nil
}

// burst throws n sparks upwards from the bottom at x, with speed in canvas
// heights per second.
func (p *particles) burst(x float64, n int, speed float64) {
	hue := p.rng.Float64()
	for range n {
		if len(p.list) >= maxParticles {
			return
		}
		a := -math.Pi/2 + (p.rng.Float64()-0.5)*0.9
		v := speed * (0.6 + 0.6*p.rng.Float64())
		p.list = append(p.list, particle{
			x: x, y: 1, vx: math.Cos(a) * v, vy: math.Sin(a) * v * 1.4,
			life: 0.7 + 0.3*p.rng.Float64(), hue: hue + p.rng.Float64()*0.15,
		})
	}
}
