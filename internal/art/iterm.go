package art

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
)

// ITermRenderer uses iTerm2's inline image protocol (OSC 1337), which iTerm2
// and WezTerm support. The image draws over blank cells at an absolute
// position, so the frame's Lines are spaces and Place does the drawing.
type ITermRenderer struct{}

// Render implements Renderer.
func (ITermRenderer) Render(img image.Image, box Box) (Frame, error) {
	cols, rows := box.Cols, box.Rows
	var buf bytes.Buffer
	if err := png.Encode(&buf, Fit(img, 600, 600)); err != nil {
		return Frame{}, fmt.Errorf("encode art: %w", err)
	}
	payload := base64.StdEncoding.EncodeToString(buf.Bytes())
	seq := fmt.Sprintf("\x1b]1337;File=inline=1;size=%d;width=%d;height=%d;preserveAspectRatio=1:%s\a",
		buf.Len(), cols, rows, payload)

	erase := func(x, y int) string { return eraseCells(x, y, cols, rows) }
	return Frame{
		Lines: blankLines(cols, rows),
		Place: func(x, y int) string { return erase(x, y) + drawAt(x, y, seq) },
		Erase: erase,
	}, nil
}
