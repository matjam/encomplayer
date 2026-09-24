package builtin

import (
	"math"
	"math/rand/v2"

	"github.com/matjam/encomplayer/internal/viz"
)

func init() {
	viz.Register(viz.Info{Name: "lightcycles", Description: "Light cycles racing the Grid, turning hard on the beat"},
		func() viz.Visualizer { return &lightcycles{rng: newRand()} })
}

// The Grid is a floor plane in world units. The camera rides above it,
// moving forward along +z.
const (
	gridSpacing = 2.0
	camHeight   = 2.6
	gridFar     = 60.0
	trailLen    = 90
)

type point2 struct{ x, z float64 }

type cycle struct {
	pos, dir point2
	trail    []point2 // corners and the current position, oldest first
	color    int
}

type lightcycles struct {
	rng    *rand.Rand
	camZ   float64
	cycles []*cycle
}

func (l *lightcycles) Render(c *viz.Canvas, f *viz.Frame) error {
	w, h := float64(c.DotW()), float64(c.DotH())
	if w == 0 || h == 0 {
		return nil
	}
	if l.cycles == nil {
		for i, x := range []float64{-3, 0, 3} {
			start := point2{x, 5 + float64(i)*2}
			// The trail's last point follows the cycle; the first stays
			// where it started.
			l.cycles = append(l.cycles, &cycle{pos: start, dir: point2{0, 1}, trail: []point2{start, start}, color: i})
		}
	}
	dt := seconds(f)
	speed := 5 + 14*f.Level
	l.camZ += speed * dt

	if f.Beat && len(l.cycles) > 0 {
		l.turn(l.cycles[l.rng.IntN(len(l.cycles))], l.rng.IntN(2) == 0)
	}
	for _, cy := range l.cycles {
		l.steer(cy)
		cy.pos.x += cy.dir.x * speed * 1.15 * dt
		cy.pos.z += cy.dir.z * speed * 1.15 * dt
		cy.trail[len(cy.trail)-1] = cy.pos
		if len(cy.trail) > trailLen {
			cy.trail = cy.trail[len(cy.trail)-trailLen:]
		}
	}

	horizon := h * 0.25
	focal := w * 0.35
	project := func(p point2) (float64, float64, bool) {
		z := p.z - l.camZ
		if z < 0.3 {
			return 0, 0, false
		}
		return w/2 + p.x/z*focal, horizon + camHeight/z*focal, true
	}

	pal := c.Palette
	// Cross lines scroll towards the camera; rails converge on the horizon.
	first := math.Ceil(l.camZ/gridSpacing) * gridSpacing
	for z := first; z < l.camZ+gridFar; z += gridSpacing {
		x0, y, ok0 := project(point2{-40, z})
		x1, _, ok1 := project(point2{40, z})
		if ok0 && ok1 {
			col := pal.Grid.Lerp(pal.Dim, 1-(z-l.camZ)/gridFar)
			c.Line(x0, y, x1, y, col)
		}
	}
	for x := -40.0; x <= 40; x += gridSpacing {
		x0, y0, _ := project(point2{x, l.camZ + 0.5})
		x1, y1, _ := project(point2{x, l.camZ + gridFar})
		c.Line(x0, y0, x1, y1, pal.Grid)
	}
	c.Line(0, horizon, w, horizon, pal.Text)

	colors := []viz.RGB{pal.Accent, pal.Bright, pal.Error}
	for _, cy := range l.cycles {
		col := colors[cy.color%len(colors)]
		for i := 1; i < len(cy.trail); i++ {
			drawGridSegment(c, project, cy.trail[i-1], cy.trail[i], col)
		}
		if x, y, ok := project(cy.pos); ok {
			c.Line(x-1, y-1, x+1, y-1, pal.Bright)
		}
	}
	return nil
}

// drawGridSegment draws a trail segment in short pieces, so the parts
// behind the camera drop out and the rest keeps its perspective.
func drawGridSegment(c *viz.Canvas, project func(point2) (float64, float64, bool), a, b point2, col viz.RGB) {
	const pieces = 8
	prevX, prevY, prevOK := project(a)
	for i := 1; i <= pieces; i++ {
		t := float64(i) / pieces
		x, y, ok := project(point2{a.x + (b.x-a.x)*t, a.z + (b.z-a.z)*t})
		if ok && prevOK {
			c.Line(prevX, prevY, x, y, col)
		}
		prevX, prevY, prevOK = x, y, ok
	}
}

// steer keeps a cycle in view. It rides forward when it falls behind,
// stops heading for the edge when it strays, and cuts back towards the
// middle once it has room ahead.
func (l *lightcycles) steer(cy *cycle) {
	ahead := cy.pos.z - l.camZ
	forward := cy.dir.z > 0
	outward := cy.dir.x*cy.pos.x > 0
	switch {
	case ahead < 4 && !forward:
		l.face(cy, point2{0, 1})
	case math.Abs(cy.pos.x) > 7 && outward:
		l.face(cy, point2{0, 1})
	case forward && ahead > 16:
		l.turn(cy, cy.pos.x > 0)
	case forward && ahead > 9 && math.Abs(cy.pos.x) > 4:
		l.turn(cy, cy.pos.x > 0)
	}
}

// turn swings the cycle 90° left or right.
func (l *lightcycles) turn(cy *cycle, left bool) {
	d := point2{cy.dir.z, -cy.dir.x}
	if left {
		d = point2{-cy.dir.z, cy.dir.x}
	}
	l.face(cy, d)
}

func (l *lightcycles) face(cy *cycle, d point2) {
	if d == cy.dir {
		return
	}
	cy.dir = d
	cy.trail = append(cy.trail, cy.pos)
}
