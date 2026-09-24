// Package builtin holds EncomPlayer's built-in visualisers. Each file
// defines one and registers it from init, so adding a visualiser means
// adding a file here.
package builtin

import (
	"math"
	"math/rand/v2"

	"github.com/matjam/encomplayer/internal/viz"
)

// seconds is the frame's time step, capped so a long pause does not make
// an animation jump.
func seconds(f *viz.Frame) float64 { return math.Min(f.Delta.Seconds(), 0.25) }

// fade returns the factor that halves a value every halfLife seconds over
// dt seconds.
func fade(dt, halfLife float64) float64 { return math.Pow(0.5, dt/halfLife) }

// newRand returns a generator with a fixed seed, so visualisers are
// repeatable in tests.
func newRand() *rand.Rand { return rand.New(rand.NewPCG(0x454E434F4D, 0x4F532D3132)) }

// resample reads n evenly spaced values from s.
func resample(s []float64, n int) []float64 {
	out := make([]float64, n)
	if len(s) == 0 {
		return out
	}
	for i := range n {
		out[i] = s[i*len(s)/max(1, n)]
	}
	return out
}

// tilted lifts higher bands by up to boost at the top of the range. Music
// carries far less energy in the treble than the bass, so effects that map
// the spectrum across the screen would otherwise go cold on the right.
func tilted(bands []float64, boost float64) []float64 {
	out := make([]float64, len(bands))
	for i, v := range bands {
		t := float64(i) / math.Max(1, float64(len(bands)-1))
		if v > 0.05 {
			v = math.Min(1, v+boost*t)
		}
		out[i] = v
	}
	return out
}

// vec3 is a point or direction in 3D.
type vec3 struct{ x, y, z float64 }

func (v vec3) add(o vec3) vec3      { return vec3{v.x + o.x, v.y + o.y, v.z + o.z} }
func (v vec3) scale(f float64) vec3 { return vec3{v.x * f, v.y * f, v.z * f} }
func (v vec3) length() float64      { return math.Sqrt(v.x*v.x + v.y*v.y + v.z*v.z) }

// rotate turns v by angles around the x, y and z axes, in that order.
func (v vec3) rotate(ax, ay, az float64) vec3 {
	s, c := math.Sincos(ax)
	v = vec3{v.x, v.y*c - v.z*s, v.y*s + v.z*c}
	s, c = math.Sincos(ay)
	v = vec3{v.x*c + v.z*s, v.y, -v.x*s + v.z*c}
	s, c = math.Sincos(az)
	return vec3{v.x*c - v.y*s, v.x*s + v.y*c, v.z}
}
