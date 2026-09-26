package viz

import (
	"math"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

var testPalette = Palette{Text: Hex("#6fc3df"), Bright: Hex("#ffffff"), Accent: Hex("#ffb000")}

func plain(c *Canvas) []string {
	lines := c.Lines()
	for i, l := range lines {
		lines[i] = ansi.Strip(l)
	}
	return lines
}

// A pixel buffer goes beneath what was drawn directly: glyphs, dots and
// pixels already set all win, and negative entries stay empty.
func TestUnderlay(t *testing.T) {
	c := NewCanvas(3, 1, testPalette)
	c.Set(0, 0, 'x', testPalette.Text)
	c.Pixel(1, 0, testPalette.Accent)
	px := []int32{
		testPalette.Bright.Pack(), testPalette.Bright.Pack(), -1, // top row of pixels
		testPalette.Bright.Pack(), testPalette.Bright.Pack(), -1, // bottom row
	}
	c.Underlay(px)

	if got := plain(c)[0]; got != "x▀ " {
		t.Fatalf("row = %q, want %q", got, "x▀ ")
	}
	if cl := c.cells[1]; cl.fg != testPalette.Accent || cl.bg != testPalette.Bright || !cl.lower {
		t.Errorf("cell 1 = %+v: the drawn top pixel should keep its colour over the buffer's bottom one", cl)
	}
	c.Underlay(px[:2]) // a short buffer draws what it has
}

func TestPack(t *testing.T) {
	c := RGB{0x12, 0x34, 0x56}
	if c.Pack() != 0x123456 || Packed(c.Pack()) != c || Packed(int64(0x7f123456)) != c {
		t.Errorf("Pack/Packed round trip failed: %#x", c.Pack())
	}
}

func TestCanvasLayers(t *testing.T) {
	c := NewCanvas(4, 2, testPalette)
	c.Text(0, 0, "hi", testPalette.Text)
	c.Pixel(2, 0, testPalette.Bright) // top half of cell (2, 0)
	c.Pixel(3, 1, testPalette.Bright) // bottom half of cell (3, 0)
	c.Pixel(3, 0, testPalette.Accent) // and its top half
	c.Dot(0, 4, testPalette.Text)     // top-left dot of cell (0, 1)
	c.Dot(1, 7, testPalette.Text)     // bottom-right dot of the same cell
	c.Set(-1, 0, 'x', testPalette.Text)
	c.Pixel(99, 99, testPalette.Text)

	got := plain(c)
	want := []string{"hi▀▀", "⢁   "}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d = %q, want %q", i, got[i], want[i])
		}
	}
	// The cell with both halves carries a background colour for its
	// bottom pixel.
	if !strings.Contains(c.Lines()[0], "\x1b[48;2;255;255;255m") {
		t.Errorf("two-pixel cell has no background: %q", c.Lines()[0])
	}
}

func TestCanvasLinesAreExactWidth(t *testing.T) {
	c := NewCanvas(10, 3, testPalette)
	c.Line(-50, -50, 100, 100, testPalette.Text) // clipped, not dropped
	for i, l := range c.Lines() {
		if w := ansi.StringWidth(l); w != 10 {
			t.Errorf("row %d is %d cells", i, w)
		}
	}
	if strings.TrimSpace(strings.Join(plain(c), "")) == "" {
		t.Error("clipped line vanished")
	}
}

func TestClipLine(t *testing.T) {
	if _, _, _, _, ok := clipLine(-10, -10, -5, -5, 8, 8); ok {
		t.Error("segment outside the canvas was kept")
	}
	x0, y0, x1, y1, ok := clipLine(-100, 4, 100, 4, 8, 8)
	near := func(a, b float64) bool { return math.Abs(a-b) < 1e-9 }
	if !ok || !near(x0, -1) || !near(x1, 8) || y0 != 4 || y1 != 4 {
		t.Errorf("horizontal clip = %v %v %v %v %v", x0, y0, x1, y1, ok)
	}
}

func TestPaletteRamps(t *testing.T) {
	p := Palette{Grid: Hex("#000000"), Accent: Hex("#ffffff"), Dim: Hex("#404040"), Text: Hex("#808080"), Bright: Hex("#c0c0c0")}
	if got := p.Ramp(0); got != p.Grid {
		t.Errorf("Ramp(0) = %v", got)
	}
	if got := p.Ramp(1); got != p.Accent {
		t.Errorf("Ramp(1) = %v", got)
	}
	if got := p.Ramp(2); got != p.Accent {
		t.Errorf("Ramp clamps: got %v", got)
	}
	if got := Hex("#abc"); got != (RGB{0xaa, 0xbb, 0xcc}) {
		t.Errorf("short hex = %v", got)
	}
	if got := Hex("nonsense"); got != (RGB{128, 128, 128}) {
		t.Errorf("bad hex = %v", got)
	}
}

// BenchmarkLinesFullPixels is the worst case: every cell has two different
// pixel colours.
func BenchmarkLinesFullPixels(b *testing.B) {
	c := NewCanvas(200, 50, testPalette)
	for y := range c.PixelH() {
		for x := range c.PixelW() {
			c.Pixel(x, y, RGB{uint8(x), uint8(y), uint8(x + y)})
		}
	}
	b.ResetTimer()
	for range b.N {
		_ = c.Lines()
	}
}
