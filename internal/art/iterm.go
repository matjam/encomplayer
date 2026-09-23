package art

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"strings"
)

// ITermRenderer uses iTerm2's inline image protocol (OSC 1337), which iTerm2
// and WezTerm support. The image draws over blank cells at an absolute
// position, so the frame's Lines are spaces and Place does the drawing.
type ITermRenderer struct{}

// Render implements Renderer.
func (ITermRenderer) Render(img image.Image, cols, rows int) (Frame, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, Fit(img, 600, 600)); err != nil {
		return Frame{}, fmt.Errorf("encode art: %w", err)
	}
	payload := base64.StdEncoding.EncodeToString(buf.Bytes())
	seq := fmt.Sprintf("\x1b]1337;File=inline=1;size=%d;width=%d;height=%d;preserveAspectRatio=1:%s\a",
		buf.Len(), cols, rows, payload)

	blank := strings.Repeat(" ", cols)
	lines := make([]string, rows)
	for i := range lines {
		lines[i] = blank
	}

	erase := func(x, y int) string {
		var b strings.Builder
		b.WriteString("\x1b7")
		for r := range rows {
			fmt.Fprintf(&b, "\x1b[%d;%dH\x1b[%dX", y+r+1, x+1, cols)
		}
		b.WriteString("\x1b8")
		return b.String()
	}

	return Frame{
		Lines: lines,
		Place: func(x, y int) string {
			return erase(x, y) + fmt.Sprintf("\x1b7\x1b[%d;%dH%s\x1b8", y+1, x+1, seq)
		},
		Erase: erase,
	}, nil
}
