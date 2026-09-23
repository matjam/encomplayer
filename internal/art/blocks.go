package art

import (
	"fmt"
	"image"
	"image/color"
	"strings"
)

// BlockRenderer draws images with the upper half block, giving two pixels per
// cell in any truecolor terminal.
type BlockRenderer struct{}

// Render implements Renderer.
func (BlockRenderer) Render(img image.Image, cols, rows int) (Frame, error) {
	// A cell is about twice as tall as wide, so each half is square.
	fit := Fit(img, cols, rows*2)
	fw, fh := fit.Bounds().Dx(), fit.Bounds().Dy()
	padX := (cols - fw) / 2
	padY := (rows*2 - fh) / 2

	at := func(x, y int) (color.RGBA, bool) {
		x, y = x-padX, y-padY
		if x < 0 || y < 0 || x >= fw || y >= fh {
			return color.RGBA{}, false
		}
		return fit.RGBAAt(x, y), true
	}

	lines := make([]string, rows)
	var b strings.Builder
	for row := range rows {
		b.Reset()
		for x := range cols {
			top, okTop := at(x, row*2)
			bot, okBot := at(x, row*2+1)
			switch {
			case okTop && okBot:
				fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀", top.R, top.G, top.B, bot.R, bot.G, bot.B)
			case okTop:
				fmt.Fprintf(&b, "\x1b[49m\x1b[38;2;%d;%d;%dm▀", top.R, top.G, top.B)
			case okBot:
				fmt.Fprintf(&b, "\x1b[49m\x1b[38;2;%d;%d;%dm▄", bot.R, bot.G, bot.B)
			default:
				b.WriteString("\x1b[0m ")
			}
		}
		b.WriteString("\x1b[0m")
		lines[row] = b.String()
	}
	return Frame{Lines: lines}, nil
}
