package viz

import (
	"math"
	"strconv"
	"strings"
)

// RGB is a 24-bit colour. The terminal program downsamples it when the
// terminal cannot show true colour.
type RGB struct{ R, G, B uint8 }

// Hex parses "#rrggbb" or "#rgb". Anything else is mid grey, so a bad theme
// colour still draws something.
func Hex(s string) RGB {
	s = strings.TrimPrefix(s, "#")
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if len(s) != 6 || err != nil {
		return RGB{128, 128, 128}
	}
	return RGB{uint8(v >> 16), uint8(v >> 8), uint8(v)}
}

// Lerp blends from c towards d; t is clamped to [0, 1].
func (c RGB) Lerp(d RGB, t float64) RGB {
	t = clamp01(t)
	mix := func(a, b uint8) uint8 { return uint8(math.Round(float64(a) + (float64(b)-float64(a))*t)) }
	return RGB{mix(c.R, d.R), mix(c.G, d.G), mix(c.B, d.B)}
}

// Scale multiplies each channel by f, saturating at white.
func (c RGB) Scale(f float64) RGB {
	s := func(a uint8) uint8 { return uint8(math.Min(255, math.Max(0, float64(a)*f))) }
	return RGB{s(c.R), s(c.G), s(c.B)}
}

// Palette is the active theme's colours, so every visualiser follows the
// theme.
type Palette struct {
	// Background is the theme's background, or black when the theme keeps
	// the terminal's own. Fading towards it makes trails disappear.
	Background RGB

	Text, Bright, Dim, Grid, Accent, Error RGB
}

// Ramp runs from faint to hot through the theme: grid, dim, text, bright,
// accent.
func (p Palette) Ramp(t float64) RGB {
	return gradient(t, p.Grid, p.Dim, p.Text, p.Bright, p.Accent)
}

// Heat runs from the background through error and accent to bright, for
// fire and heat maps.
func (p Palette) Heat(t float64) RGB {
	return gradient(t, p.Background, p.Error, p.Accent, p.Bright)
}

// Cycle picks a hue-like colour by walking the theme's vivid colours
// around a loop, for effects that sweep through colours over time.
func (p Palette) Cycle(t float64) RGB {
	t -= math.Floor(t)
	return gradient(t, p.Text, p.Bright, p.Accent, p.Error, p.Text)
}

// gradient interpolates evenly spaced stops.
func gradient(t float64, stops ...RGB) RGB {
	t = clamp01(t)
	if len(stops) == 1 {
		return stops[0]
	}
	pos := t * float64(len(stops)-1)
	i := min(int(pos), len(stops)-2)
	return stops[i].Lerp(stops[i+1], pos-float64(i))
}

func clamp01(t float64) float64 {
	if math.IsNaN(t) {
		return 0
	}
	return math.Max(0, math.Min(1, t))
}
