package art

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/sixel"
)

func TestDetect(t *testing.T) {
	tests := []struct {
		name  string
		env   map[string]string
		sixel bool // the terminal reported sixel in its device attributes
		want  Protocol
	}{
		{name: "kitty", env: map[string]string{"TERM": "xterm-kitty", "KITTY_WINDOW_ID": "1"}, want: Kitty},
		{name: "ghostty", env: map[string]string{"TERM_PROGRAM": "ghostty"}, want: Kitty},
		{name: "iterm2", env: map[string]string{"TERM_PROGRAM": "iTerm.app"}, want: ITerm},
		{name: "wezterm", env: map[string]string{"TERM_PROGRAM": "WezTerm"}, want: ITerm},
		{name: "foot", env: map[string]string{"TERM": "foot"}, want: Sixel},
		{name: "foot direct colour", env: map[string]string{"TERM": "foot-direct"}, want: Sixel},
		{name: "footlike name", env: map[string]string{"TERM": "football"}, want: Blocks},
		{name: "tmux inside kitty", env: map[string]string{"TERM": "tmux-256color", "TMUX": "/tmp/x", "KITTY_WINDOW_ID": "1"}, want: Blocks},
		{name: "unknown", env: map[string]string{"TERM": "xterm-256color"}, want: Blocks},
		{name: "foot as xterm reporting sixel", env: map[string]string{"TERM": "xterm-256color"}, sixel: true, want: Sixel},
		{name: "wezterm reporting sixel keeps iterm", env: map[string]string{"TERM_PROGRAM": "WezTerm"}, sixel: true, want: ITerm},
		{name: "kitty reporting sixel keeps kitty", env: map[string]string{"KITTY_WINDOW_ID": "1"}, sixel: true, want: Kitty},
		{name: "tmux reporting sixel", env: map[string]string{"TERM": "tmux-256color", "TMUX": "/tmp/x"}, sixel: true, want: Sixel},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			term := Terminal{Getenv: func(k string) string { return tc.env[k] }, Sixel: tc.sixel}
			if got := Detect(term); got != tc.want {
				t.Errorf("Detect = %q, want %q", got, tc.want)
			}
		})
	}
}

func testImage() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	for y := range 64 {
		for x := range 64 {
			img.Set(x, y, color.RGBA{R: uint8(x * 4), G: uint8(y * 4), B: 200, A: 255})
		}
	}
	return img
}

func TestRenderersFillBox(t *testing.T) {
	const cols, rows = 20, 10
	for name, r := range DefaultRenderers().All() {
		t.Run(name, func(t *testing.T) {
			f, err := r.Render(testImage(), Box{Cols: cols, Rows: rows, Cell: DefaultCell})
			if err != nil {
				t.Fatal(err)
			}
			if len(f.Lines) != rows {
				t.Fatalf("got %d lines, want %d", len(f.Lines), rows)
			}
			for i, line := range f.Lines {
				if w := ansi.StringWidth(line); w != cols {
					t.Errorf("line %d width = %d, want %d", i, w, cols)
				}
			}
		})
	}
}

func TestSixelFitsAndCentresInBox(t *testing.T) {
	tests := []struct {
		name       string
		w, h       int
		cell       image.Point
		wantRaster string
		wantAt     string // cursor position the sixel is drawn from
	}{
		{name: "square in square box", w: 64, h: 64, cell: image.Pt(10, 20), wantRaster: `"1;1;200;200`, wantAt: "\x1b[4;6H\x1bP"},
		{name: "tall centres across", w: 10, h: 100, cell: image.Pt(10, 20), wantRaster: `"1;1;20;200`, wantAt: "\x1b[4;15H\x1bP"},
		{name: "wide centres down", w: 100, h: 10, cell: image.Pt(10, 20), wantRaster: `"1;1;200;20`, wantAt: "\x1b[8;6H\x1bP"},
		{name: "unknown cell uses default", w: 64, h: 64, wantRaster: `"1;1;200;200`, wantAt: "\x1b[4;6H\x1bP"},
		{name: "reported cell size", w: 64, h: 64, cell: image.Pt(8, 17), wantRaster: `"1;1;160;160`, wantAt: "\x1b[4;6H\x1bP"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			img := image.NewRGBA(image.Rect(0, 0, tc.w, tc.h))
			for i := range img.Pix {
				img.Pix[i] = 0xFF
			}
			f, err := SixelRenderer{}.Render(img, Box{Cols: 20, Rows: 10, Cell: tc.cell})
			if err != nil {
				t.Fatal(err)
			}
			placed := f.Place(5, 3)
			if !strings.HasPrefix(placed, f.Erase(5, 3)) {
				t.Error("Place does not erase the box first")
			}
			if !strings.Contains(placed, tc.wantAt) {
				t.Errorf("Place = %q, want drawing from %q", placed[:min(len(placed), 200)], tc.wantAt)
			}
			if !strings.Contains(placed, "\x1bP0;1q"+tc.wantRaster+"#") {
				t.Errorf("sixel header missing %s: %q", tc.wantRaster, placed[:min(len(placed), 200)])
			}
			if !strings.HasSuffix(placed, "\x1b\\\x1b8") {
				t.Error("Place does not end the sixel and restore the cursor")
			}
		})
	}
}

func TestSixelDecodesToImage(t *testing.T) {
	want := color.RGBA{R: 200, G: 40, B: 90, A: 255}
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := range 32 {
		for x := range 32 {
			img.SetRGBA(x, y, want)
		}
	}
	f, err := SixelRenderer{}.Render(img, Box{Cols: 8, Rows: 4, Cell: image.Pt(10, 20)})
	if err != nil {
		t.Fatal(err)
	}
	placed := f.Place(0, 0)
	start := strings.Index(placed, "\x1bP0;1q")
	end := strings.LastIndex(placed, "\x1b\\")
	if start < 0 || end < start {
		t.Fatalf("no sixel sequence in %q", placed)
	}
	got, err := new(sixel.Decoder).Decode(strings.NewReader(placed[start+len("\x1bP0;1q") : end]))
	if err != nil {
		t.Fatal(err)
	}
	if b := got.Bounds(); b.Dx() != 80 || b.Dy() != 80 {
		t.Fatalf("decoded size = %v, want 80x80", b.Size())
	}
	r, g, b, _ := got.At(40, 40).RGBA()
	// Sixel colour registers are percentages, so allow a step of rounding.
	for i, c := range []struct{ got, want uint8 }{{uint8(r >> 8), want.R}, {uint8(g >> 8), want.G}, {uint8(b >> 8), want.B}} {
		if d := int(c.got) - int(c.want); d < -3 || d > 3 {
			t.Errorf("channel %d = %d, want %d", i, c.got, c.want)
		}
	}
}

func TestKittyPlaceholderEncoding(t *testing.T) {
	lines := placeholders(0x4E0102, 2, 2)
	if !strings.HasPrefix(lines[0], "\x1b[38;2;78;1;2m") {
		t.Errorf("colour prefix = %q", lines[0][:20])
	}
	want := string([]rune{placeholder, diacritics[1], diacritics[0], placeholder, diacritics[1], diacritics[1]})
	if !strings.Contains(lines[1], want) {
		t.Errorf("row 1 = %q, want cells %q", lines[1], want)
	}
}
