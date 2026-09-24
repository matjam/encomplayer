package builtin

import (
	"math"
	"sort"

	"github.com/matjam/encomplayer/internal/viz"
)

func init() {
	viz.Register(viz.Info{Name: "wireframe", Description: "Rotating icosahedron whose vertices push out with the music"},
		func() viz.Visualizer { return newWireframe() })
}

type wireframe struct {
	verts      []vec3
	edges      [][2]int
	ax, ay, az float64
	pulse      float64
}

func newWireframe() *wireframe {
	phi := (1 + math.Sqrt(5)) / 2
	w := &wireframe{}
	for _, v := range [][3]float64{
		{-1, phi, 0}, {1, phi, 0}, {-1, -phi, 0}, {1, -phi, 0},
		{0, -1, phi}, {0, 1, phi}, {0, -1, -phi}, {0, 1, -phi},
		{phi, 0, -1}, {phi, 0, 1}, {-phi, 0, -1}, {-phi, 0, 1},
	} {
		p := vec3{v[0], v[1], v[2]}
		w.verts = append(w.verts, p.scale(1/p.length()))
	}
	// Edges join vertices at the shortest distance.
	for i := range w.verts {
		for j := i + 1; j < len(w.verts); j++ {
			d := w.verts[i].add(w.verts[j].scale(-1)).length()
			if d < 1.1 {
				w.edges = append(w.edges, [2]int{i, j})
			}
		}
	}
	return w
}

func (wf *wireframe) Render(c *viz.Canvas, f *viz.Frame) error {
	w, h := float64(c.DotW()), float64(c.DotH())
	if w == 0 || h == 0 {
		return nil
	}
	dt := seconds(f)
	speed := 0.3 + 1.5*f.Level
	wf.ax += dt * speed * 0.7
	wf.ay += dt * speed
	wf.az += dt * speed * 0.3
	wf.pulse = math.Max(f.BeatStrength, wf.pulse*fade(dt, 0.15))

	bands := f.Bands(len(wf.verts))
	size := math.Min(w, h) * 0.32 * (1 + 0.25*wf.pulse)
	cx, cy := w/2, h/2

	type projected struct{ x, y, z float64 }
	pts := make([]projected, len(wf.verts))
	for i, v := range wf.verts {
		v = v.scale(1+0.5*bands[i]).rotate(wf.ax, wf.ay, wf.az)
		persp := 3 / (3 + v.z)
		pts[i] = projected{cx + v.x*size*persp, cy + v.y*size*persp, v.z}
	}

	// Draw far edges first, so near ones stay on top.
	order := make([]int, len(wf.edges))
	for i := range order {
		order[i] = i
	}
	depth := func(e [2]int) float64 { return pts[e[0]].z + pts[e[1]].z }
	sort.Slice(order, func(a, b int) bool { return depth(wf.edges[order[a]]) > depth(wf.edges[order[b]]) })

	p := c.Palette
	for _, i := range order {
		e := wf.edges[i]
		a, b := pts[e[0]], pts[e[1]]
		near := clamp(0.5-depth(e)/4, 0, 1)
		c.Line(a.x, a.y, b.x, b.y, p.Dim.Lerp(p.Bright, near))
	}
	for _, pt := range pts {
		c.Line(pt.x-1, pt.y, pt.x+1, pt.y, p.Accent)
	}
	return nil
}
