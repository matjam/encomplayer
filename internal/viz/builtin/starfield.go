package builtin

import (
	"math"
	"math/rand/v2"

	"github.com/matjam/encomplayer/internal/viz"
)

func init() {
	viz.Register(viz.Info{Name: "starfield", Description: "Warp-speed starfield; louder music, faster travel"},
		func() viz.Visualizer { return &starfield{rng: newRand()} })
}

const starCount = 400

type star struct{ x, y, z float64 }

type starfield struct {
	rng   *rand.Rand
	stars []star
	boost float64
}

func (s *starfield) Render(c *viz.Canvas, f *viz.Frame) error {
	w, h := float64(c.DotW()), float64(c.DotH())
	if w == 0 || h == 0 {
		return nil
	}
	if s.stars == nil {
		s.stars = make([]star, starCount)
		for i := range s.stars {
			s.stars[i] = s.spawn(s.rng.Float64())
		}
	}
	dt := seconds(f)
	s.boost = math.Max(f.BeatStrength*2, s.boost*fade(dt, 0.25))
	speed := 0.15 + 1.2*f.Level + s.boost

	cx, cy := w/2, h/2
	focal := math.Min(cx, cy)
	for i := range s.stars {
		st := &s.stars[i]
		oldZ := st.z
		st.z -= speed * dt
		if st.z <= 0.02 {
			*st = s.spawn(1)
			continue
		}
		x0, y0 := cx+st.x/oldZ*focal, cy+st.y/oldZ*focal
		x1, y1 := cx+st.x/st.z*focal, cy+st.y/st.z*focal
		if x1 < 0 || y1 < 0 || x1 >= w || y1 >= h {
			*st = s.spawn(1)
			continue
		}
		col := c.Palette.Ramp(1 - st.z)
		// Streak from where the star was, so speed reads as length.
		c.Line(x0, y0, x1, y1, col)
	}
	return nil
}

func (s *starfield) spawn(maxZ float64) star {
	return star{
		x: s.rng.Float64()*4 - 2,
		y: s.rng.Float64()*4 - 2,
		z: 0.1 + s.rng.Float64()*(maxZ-0.1+0.01),
	}
}
