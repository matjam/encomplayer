package script

import (
	"math"
	"unicode/utf8"
	"unsafe"

	"github.com/matjam/apogee/lua"

	"github.com/matjam/encomplayer/internal/viz"
)

// maxBuffer is the most elements f64 or i32 make, 128 MB of floats.
const maxBuffer = 1 << 24

// register sets the host functions as globals. Functions of numbers are
// number functions, which scripts call without a Go call frame; they draw
// nothing outside render.
func (p *plugin) register() {
	l := p.l

	l.RegisterNumberFunction("pixel", func(x, y, col float64) {
		if c, ok := unpack(col); ok && p.canvas != nil {
			p.canvas.Pixel(cellOf(x), cellOf(y), c)
		}
	})
	l.RegisterNumberFunction("dot", func(x, y, col float64) {
		if c, ok := unpack(col); ok && p.canvas != nil {
			p.canvas.Dot(cellOf(x), cellOf(y), c)
		}
	})
	l.RegisterNumberFunction("set", func(x, y, r, col float64) {
		if c, ok := unpack(col); ok && p.canvas != nil {
			p.canvas.Set(cellOf(x), cellOf(y), glyph(r), c)
		}
	})

	l.RegisterNumberFunction("rgb", func(r, g, b float64) float64 {
		return packed(viz.RGB{R: channel(r), G: channel(g), B: channel(b)})
	})
	l.RegisterNumberFunction("lerp", func(a, b, t float64) float64 {
		return packed(viz.Packed(int64(a)).Lerp(viz.Packed(int64(b)), t))
	})
	l.RegisterNumberFunction("scale", func(c, f float64) float64 {
		return packed(viz.Packed(int64(c)).Scale(f))
	})
	l.RegisterNumberFunction("ramp", func(t float64) float64 { return packed(p.palette.Ramp(t)) })
	l.RegisterNumberFunction("heat", func(t float64) float64 { return packed(p.palette.Heat(t)) })
	l.RegisterNumberFunction("cycle", func(t float64) float64 { return packed(p.palette.Cycle(t)) })

	l.Register("line", p.line)
	l.Register("text", p.text)
	l.Register("cell", p.cell)
	l.Register("hex", func(l *lua.State) int {
		l.PushInteger(viz.Hex(l.Arg[string](1)).Pack())
		return 1
	})
	l.Register("f64", newBuffer[float64])
	l.Register("i32", newBuffer[int32])
	l.Register("bands", p.bandsBuffer)
}

// line(x0, y0, x1, y1, colour) draws a braille line in dot coordinates.
func (p *plugin) line(l *lua.State) int {
	x0, y0 := l.Arg[float64](1), l.Arg[float64](2)
	x1, y1 := l.Arg[float64](3), l.Arg[float64](4)
	if c, ok := unpack(l.Arg[float64](5)); ok && p.canvas != nil {
		p.canvas.Line(x0, y0, x1, y1, c)
	}
	return 0
}

// text(x, y, s, colour) writes s from cell (x, y) rightwards.
func (p *plugin) text(l *lua.State) int {
	x, y, s := l.Arg[float64](1), l.Arg[float64](2), l.Arg[string](3)
	if c, ok := unpack(l.Arg[float64](4)); ok && p.canvas != nil {
		p.canvas.Text(cellOf(x), cellOf(y), s, c)
	}
	return 0
}

// cell(x, y, glyph, fg[, bg]) draws a glyph, given as a string or a code
// point, with an optional background.
func (p *plugin) cell(l *lua.State) int {
	x, y := l.Arg[float64](1), l.Arg[float64](2)
	var r rune
	if l.TypeOf(3) == lua.TypeString {
		r, _ = utf8.DecodeRuneInString(l.Arg[string](3))
	} else {
		r = glyph(l.Arg[float64](3))
	}
	fg, ok := unpack(l.Arg[float64](4))
	if !ok || p.canvas == nil {
		return 0
	}
	if l.IsNoneOrNil(5) {
		p.canvas.Set(cellOf(x), cellOf(y), r, fg)
	} else if bg, ok := unpack(l.Arg[float64](5)); ok {
		p.canvas.SetColors(cellOf(x), cellOf(y), r, fg, bg)
	}
	return 0
}

// bandsBuffer is bands(n): the spectrum in n bands, as Frame.Bands gives
// it, in a buffer the script may keep. The same n returns the same buffer,
// refilled each frame.
func (p *plugin) bandsBuffer(l *lua.State) int {
	n := l.Arg[int](1)
	l.ArgumentCheck(n >= 1 && n <= 1<<16, 1, "band count out of range")
	l.Field(lua.RegistryIndex, bandsKey)
	buf, ok := p.bands[n]
	if ok {
		l.RawGetInt(-1, n)
	} else {
		buf = make([]float64, n)
		p.bands[n] = buf
		l.PushBuffer(buf)
		l.PushValue(-1)
		l.RawSetInt(-3, n)
	}
	if p.frame != nil {
		copy(buf, p.frame.Bands(n))
	}
	return 1
}

// newBuffer is f64(n) or i32(n): a new buffer of n zeros.
func newBuffer[T float64 | int32](l *lua.State) int {
	n := l.Arg[int](1)
	l.ArgumentCheck(n >= 0 && n <= maxBuffer, 1, "buffer size out of range")
	var zero T
	l.Charge(n * int(unsafe.Sizeof(zero)))
	l.PushBuffer(make([]T, n))
	return 1
}

// cellOf rounds a coordinate down to its cell, pixel or dot.
func cellOf(v float64) int {
	if math.IsNaN(v) || math.Abs(v) > 1<<30 {
		return -1
	}
	return int(math.Floor(v))
}

// unpack reads a packed colour; a negative one means none.
func unpack(v float64) (viz.RGB, bool) {
	if !(v >= 0) {
		return viz.RGB{}, false
	}
	return viz.Packed(int64(v)), true
}

func packed(c viz.RGB) float64 { return float64(c.Pack()) }

func channel(v float64) uint8 { return uint8(math.Round(math.Max(0, math.Min(255, v)))) }

func glyph(v float64) rune {
	if !(v > 0 && v <= utf8.MaxRune) {
		return ' '
	}
	return rune(v)
}
