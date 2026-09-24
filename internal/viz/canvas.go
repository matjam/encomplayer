package viz

import (
	"math"
	"strconv"
	"unicode/utf8"
)

// Canvas is the grid of terminal cells a visualiser draws on. It offers
// three resolutions:
//
//   - cells, holding any single-width glyph (Set, Text)
//   - half-block pixels, two stacked per cell (Pixel)
//   - braille dots, a 2×4 grid per cell (Dot, Line)
//
// A cell shows whichever of these last drew into it. The canvas is cleared
// before every frame.
type Canvas struct {
	W, H    int
	Palette Palette

	cells []cell
	buf   []byte
}

type layer uint8

const (
	layerEmpty layer = iota
	layerGlyph
	layerPixel
	layerDot
)

// cell holds one terminal cell. In the pixel layer fg is the top pixel and
// bg the bottom one.
type cell struct {
	layer      layer
	r          rune
	fg, bg     RGB
	hasFG      bool
	hasBG      bool
	dots       uint8
	top, lower bool
}

// NewCanvas returns a blank canvas of w × h cells.
func NewCanvas(w, h int, p Palette) *Canvas {
	c := &Canvas{Palette: p}
	c.Resize(w, h)
	return c
}

// Resize changes the size and clears the canvas.
func (c *Canvas) Resize(w, h int) {
	c.W, c.H = max(0, w), max(0, h)
	if n := c.W * c.H; cap(c.cells) >= n {
		c.cells = c.cells[:n]
	} else {
		c.cells = make([]cell, n)
	}
	c.Clear()
}

// Clear blanks every cell.
func (c *Canvas) Clear() { clear(c.cells) }

func (c *Canvas) at(x, y int) *cell {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return nil
	}
	return &c.cells[y*c.W+x]
}

// Set draws glyph r at cell (x, y) over the background.
func (c *Canvas) Set(x, y int, r rune, fg RGB) {
	if cl := c.at(x, y); cl != nil {
		*cl = cell{layer: layerGlyph, r: r, fg: fg, hasFG: true}
	}
}

// SetColors draws glyph r at cell (x, y) with a background colour.
func (c *Canvas) SetColors(x, y int, r rune, fg, bg RGB) {
	if cl := c.at(x, y); cl != nil {
		*cl = cell{layer: layerGlyph, r: r, fg: fg, hasFG: true, bg: bg, hasBG: true}
	}
}

// Text writes s from cell (x, y) rightwards. Every rune must be single
// width.
func (c *Canvas) Text(x, y int, s string, fg RGB) {
	for _, r := range s {
		c.Set(x, y, r, fg)
		x++
	}
}

// PixelW and PixelH are the half-block pixel resolution.
func (c *Canvas) PixelW() int { return c.W }
func (c *Canvas) PixelH() int { return c.H * 2 }

// Pixel colours half-block pixel (px, py).
func (c *Canvas) Pixel(px, py int, col RGB) {
	cl := c.at(px, py/2)
	if cl == nil || py < 0 {
		return
	}
	if cl.layer != layerPixel {
		*cl = cell{layer: layerPixel}
	}
	if py%2 == 0 {
		cl.fg, cl.top = col, true
	} else {
		cl.bg, cl.lower = col, true
	}
}

// DotW and DotH are the braille dot resolution.
func (c *Canvas) DotW() int { return c.W * 2 }
func (c *Canvas) DotH() int { return c.H * 4 }

// brailleBits maps a dot's column and row within its cell to its bit.
var brailleBits = [2][4]uint8{
	{0x01, 0x02, 0x04, 0x40},
	{0x08, 0x10, 0x20, 0x80},
}

// Dot lights braille dot (dx, dy). The cell takes the colour of the last
// dot drawn into it.
func (c *Canvas) Dot(dx, dy int, col RGB) {
	if dx < 0 || dy < 0 {
		return
	}
	cl := c.at(dx/2, dy/4)
	if cl == nil {
		return
	}
	if cl.layer != layerDot {
		*cl = cell{layer: layerDot}
	}
	cl.dots |= brailleBits[dx%2][dy%4]
	cl.fg, cl.hasFG = col, true
}

// Line draws a braille line between two points in dot coordinates.
func (c *Canvas) Line(x0, y0, x1, y1 float64, col RGB) {
	var ok bool
	if x0, y0, x1, y1, ok = clipLine(x0, y0, x1, y1, float64(c.DotW()), float64(c.DotH())); !ok {
		return
	}
	steps := int(math.Max(math.Abs(x1-x0), math.Abs(y1-y0))) + 1
	for i := range steps + 1 {
		t := float64(i) / float64(steps)
		c.Dot(int(math.Round(x0+(x1-x0)*t)), int(math.Round(y0+(y1-y0)*t)), col)
	}
}

// clipLine trims a segment to the rectangle [-1, w] × [-1, h] with the
// Liang–Barsky method, so a line with a far-off endpoint still draws its
// visible part in bounded time.
func clipLine(x0, y0, x1, y1, w, h float64) (float64, float64, float64, float64, bool) {
	if math.IsNaN(x0+y0+x1+y1) || math.IsInf(x0+y0+x1+y1, 0) {
		return 0, 0, 0, 0, false
	}
	dx, dy := x1-x0, y1-y0
	t0, t1 := 0.0, 1.0
	for _, edge := range [4][2]float64{{-dx, x0 + 1}, {dx, w - x0}, {-dy, y0 + 1}, {dy, h - y0}} {
		p, q := edge[0], edge[1]
		if p == 0 {
			if q < 0 {
				return 0, 0, 0, 0, false
			}
			continue
		}
		r := q / p
		if p < 0 {
			if r > t1 {
				return 0, 0, 0, 0, false
			}
			t0 = math.Max(t0, r)
		} else {
			if r < t0 {
				return 0, 0, 0, 0, false
			}
			t1 = math.Min(t1, r)
		}
	}
	return x0 + t0*dx, y0 + t0*dy, x0 + t1*dx, y0 + t1*dy, true
}

// Lines renders the canvas as one string per row, each exactly W cells
// wide, with ANSI colour sequences.
func (c *Canvas) Lines() []string {
	out := make([]string, c.H)
	for y := range c.H {
		c.buf = c.buf[:0]
		var fg, bg RGB
		fgOn, bgOn := false, false
		for x := range c.W {
			r, cfg, cbg, hasFG, hasBG := c.cells[y*c.W+x].glyph()
			if hasFG && (!fgOn || cfg != fg) {
				c.buf = appendSGR(c.buf, "\x1b[38;2;", cfg)
				fg, fgOn = cfg, true
			}
			switch {
			case hasBG && (!bgOn || cbg != bg):
				c.buf = appendSGR(c.buf, "\x1b[48;2;", cbg)
				bg, bgOn = cbg, true
			case !hasBG && bgOn:
				c.buf = append(c.buf, "\x1b[49m"...)
				bgOn = false
			}
			c.buf = utf8.AppendRune(c.buf, r)
		}
		c.buf = append(c.buf, "\x1b[m"...)
		out[y] = string(c.buf)
	}
	return out
}

func (cl cell) glyph() (r rune, fg, bg RGB, hasFG, hasBG bool) {
	switch cl.layer {
	case layerGlyph:
		return cl.r, cl.fg, cl.bg, cl.hasFG, cl.hasBG
	case layerDot:
		return rune(0x2800 + int(cl.dots)), cl.fg, RGB{}, true, false
	case layerPixel:
		switch {
		case cl.top && cl.lower:
			return '▀', cl.fg, cl.bg, true, true
		case cl.top:
			return '▀', cl.fg, RGB{}, true, false
		case cl.lower:
			return '▄', cl.bg, RGB{}, true, false
		}
	}
	return ' ', RGB{}, RGB{}, false, false
}

func appendSGR(b []byte, prefix string, c RGB) []byte {
	b = append(b, prefix...)
	b = strconv.AppendUint(b, uint64(c.R), 10)
	b = append(b, ';')
	b = strconv.AppendUint(b, uint64(c.G), 10)
	b = append(b, ';')
	b = strconv.AppendUint(b, uint64(c.B), 10)
	return append(b, 'm')
}
