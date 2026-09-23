package art

import (
	"image"
	"image/color"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestDetect(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		want Protocol
	}{
		{name: "kitty", env: map[string]string{"TERM": "xterm-kitty", "KITTY_WINDOW_ID": "1"}, want: Kitty},
		{name: "ghostty", env: map[string]string{"TERM_PROGRAM": "ghostty"}, want: Kitty},
		{name: "iterm2", env: map[string]string{"TERM_PROGRAM": "iTerm.app"}, want: ITerm},
		{name: "wezterm", env: map[string]string{"TERM_PROGRAM": "WezTerm"}, want: ITerm},
		{name: "tmux inside kitty", env: map[string]string{"TERM": "tmux-256color", "TMUX": "/tmp/x", "KITTY_WINDOW_ID": "1"}, want: Blocks},
		{name: "unknown", env: map[string]string{"TERM": "xterm-256color"}, want: Blocks},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Detect(func(k string) string { return tc.env[k] }); got != tc.want {
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
			f, err := r.Render(testImage(), cols, rows)
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
