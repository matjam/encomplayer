package builtin

import (
	"math/rand/v2"

	"github.com/matjam/encomplayer/internal/viz"
)

func init() {
	viz.Register(viz.Info{Name: "matrix", Description: "Digital rain that pours harder with the music"},
		func() viz.Visualizer { return &matrix{rng: newRand()} })
}

// rainGlyphs are single-width: half-width katakana, digits and symbols.
var rainGlyphs = []rune("ｱｲｳｴｵｶｷｸｹｺｻｼｽｾｿﾀﾁﾂﾃﾄﾅﾆﾇﾈﾉﾊﾋﾌﾍﾎﾏﾐﾑﾒﾓﾔﾕﾖﾗﾘﾙﾚﾛﾜﾝ0123456789:.=*+<>")

type drop struct {
	y, speed float64
	length   int
	active   bool
}

type matrix struct {
	rng    *rand.Rand
	w, h   int
	glyphs []rune
	drops  []drop
}

func (m *matrix) Render(c *viz.Canvas, f *viz.Frame) error {
	if c.W == 0 || c.H == 0 {
		return nil
	}
	if m.w != c.W || m.h != c.H {
		m.w, m.h = c.W, c.H
		m.glyphs = make([]rune, c.W*c.H)
		for i := range m.glyphs {
			m.glyphs[i] = m.glyph()
		}
		m.drops = make([]drop, c.W)
	}
	dt := seconds(f)
	pace := 0.4 + 1.8*f.Level
	spawn := dt * (0.3 + 3*f.Level + 6*f.BeatStrength)

	p := c.Palette
	for x := range m.drops {
		d := &m.drops[x]
		if !d.active {
			if m.rng.Float64() < spawn/4 {
				*d = drop{y: -1, speed: 6 + m.rng.Float64()*14, length: 4 + m.rng.IntN(max(1, c.H)), active: true}
			}
			continue
		}
		d.y += d.speed * pace * dt
		head := int(d.y)
		if head-d.length > c.H {
			d.active = false
			continue
		}
		for i := range d.length {
			y := head - i
			if y < 0 || y >= c.H {
				continue
			}
			if m.rng.Float64() < 0.02 {
				m.glyphs[y*c.W+x] = m.glyph()
			}
			col := p.Bright
			if i > 0 {
				col = p.Ramp(0.85 - 0.7*float64(i)/float64(d.length))
			}
			c.Set(x, y, m.glyphs[y*c.W+x], col)
		}
	}
	return nil
}

func (m *matrix) glyph() rune { return rainGlyphs[m.rng.IntN(len(rainGlyphs))] }
