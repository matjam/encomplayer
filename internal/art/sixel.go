package art

import (
	"bytes"
	"fmt"
	"image"

	"github.com/charmbracelet/x/ansi"
	"github.com/charmbracelet/x/ansi/sixel"
)

// SixelRenderer draws images as DEC sixel graphics, which foot, xterm in
// VT340 mode, mlterm and Windows Terminal support. Sixels are pixels drawn at
// the cursor over the text grid, so like iTerm2 the frame's Lines are blank
// and Place does the drawing.
//
// Sixel images start at a cell corner, so the image is centred to the nearest
// cell rather than padded: transparent padding would enter the encoder's
// colour quantisation alongside the image's own colours.
type SixelRenderer struct{}

// Render implements Renderer.
func (SixelRenderer) Render(img image.Image, box Box) (Frame, error) {
	cols, rows, cell := box.Cols, box.Rows, box.Cell
	if cell.X <= 0 || cell.Y <= 0 {
		cell = DefaultCell
	}
	fit := Fit(img, cols*cell.X, rows*cell.Y)
	offX := (cols*cell.X - fit.Bounds().Dx()) / 2 / cell.X
	offY := (rows*cell.Y - fit.Bounds().Dy()) / 2 / cell.Y

	var buf bytes.Buffer
	if err := new(sixel.Encoder).Encode(&buf, fit); err != nil {
		return Frame{}, fmt.Errorf("encode art: %w", err)
	}
	// P2=1 leaves pixels the image does not set alone rather than filling
	// them with colour register 0.
	seq := ansi.SixelGraphics(0, 1, 0, buf.Bytes())

	erase := func(x, y int) string { return eraseCells(x, y, cols, rows) }
	return Frame{
		Lines: blankLines(cols, rows),
		Place: func(x, y int) string { return erase(x, y) + drawAt(x+offX, y+offY, seq) },
		Erase: erase,
	}, nil
}
