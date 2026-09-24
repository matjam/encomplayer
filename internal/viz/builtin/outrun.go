package builtin

import (
	"math"

	"github.com/matjam/encomplayer/internal/viz"
)

func init() {
	viz.Register(viz.Info{Name: "outrun", Description: "Synthwave sunset over mountains raised by the spectrum"},
		func() viz.Visualizer { return &outrun{} })
}

const (
	ridgeCount   = 18
	ridgePoints  = 32
	ridgeSpacing = 1.0
	ridgeDepth   = ridgeCount * ridgeSpacing
)

// ridge is one line of terrain across the view. Heights come from the
// spectrum when the ridge appeared on the horizon.
type ridge struct {
	z       float64
	heights []float64
}

type outrun struct {
	ridges []ridge
	travel float64
}

func (o *outrun) Render(c *viz.Canvas, f *viz.Frame) error {
	w, h := float64(c.DotW()), float64(c.DotH())
	if w == 0 || h == 0 {
		return nil
	}
	dt := seconds(f)
	o.travel = dt * (1.5 + 4*f.Level)
	o.advance(f)

	horizon := h * 0.45
	o.drawSun(c, w/2, horizon, math.Min(w, h)*0.3, f)

	focal := w * 0.45
	project := func(x, y, z float64) (float64, float64) {
		return w/2 + x/z*focal, horizon + (1.2-y)/z*focal
	}
	pal := c.Palette
	var prev [ridgePoints][2]float64
	for i, r := range o.ridges {
		near := 1 - r.z/ridgeDepth
		col := pal.Accent.Lerp(pal.Grid, 1-near)
		var line [ridgePoints][2]float64
		for j, y := range r.heights {
			x := (float64(j)/(ridgePoints-1) - 0.5) * 14
			line[j][0], line[j][1] = project(x, y, r.z)
			if j > 0 {
				c.Line(line[j-1][0], line[j-1][1], line[j][0], line[j][1], col)
			}
			if i > 0 {
				c.Line(prev[j][0], prev[j][1], line[j][0], line[j][1], col)
			}
		}
		prev = line
	}
	return nil
}

// advance moves every ridge towards the viewer and raises new ones on the
// horizon. Ridges are kept far to near.
func (o *outrun) advance(f *viz.Frame) {
	if o.ridges == nil {
		for i := range ridgeCount {
			o.ridges = append(o.ridges, ridge{z: ridgeDepth - float64(i)*ridgeSpacing + 0.5, heights: make([]float64, ridgePoints)})
		}
	}
	for i := range o.ridges {
		o.ridges[i].z -= o.travel
	}
	for len(o.ridges) > 0 && o.ridges[len(o.ridges)-1].z < 0.5 {
		o.ridges = o.ridges[:len(o.ridges)-1]
	}
	for len(o.ridges) < ridgeCount {
		far := ridgeDepth + 0.5
		if len(o.ridges) > 0 {
			far = o.ridges[0].z + ridgeSpacing
		}
		o.ridges = append([]ridge{{z: far, heights: mountains(f)}}, o.ridges...)
	}
}

// mountains shapes a ridge: a flat road in the middle and peaks rising
// towards the sides, where the bass lifts the tallest ones.
func mountains(f *viz.Frame) []float64 {
	half := ridgePoints / 2
	bands := tilted(f.Bands(half), 0.2)
	out := make([]float64, ridgePoints)
	for j := range out {
		d := j - half
		if d < 0 {
			d = -d - 1
		}
		side := clamp(float64(d-2)/float64(half-2), 0, 1)
		v := bands[half-1-d]
		out[j] = v * v * side * 5
	}
	return out
}

// drawSun draws a striped sun sinking into the horizon.
func (o *outrun) drawSun(c *viz.Canvas, cx, horizon, radius float64, f *viz.Frame) {
	pal := c.Palette
	r := radius * (0.9 + 0.1*f.Bass)
	for y := horizon - r; y < horizon; y++ {
		t := (y - (horizon - r)) / r
		// Stripes widen towards the horizon, as in every synthwave poster.
		if t > 0.45 && math.Mod(y, 4+t*6) < 1+t*4 {
			continue
		}
		half := math.Sqrt(math.Max(0, r*r-(y-horizon)*(y-horizon)))
		col := pal.Accent.Lerp(pal.Error, t)
		c.Line(cx-half, y, cx+half, y, col)
	}
}
