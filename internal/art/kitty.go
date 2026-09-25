package art

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"strings"
	"sync/atomic"
)

// placeholder is kitty's Unicode placeholder for image cells.
const placeholder = '\U0010EEEE'

// idBase keeps our image IDs away from low IDs other programs tend to use.
const idBase = 0x4E_0000

// KittyRenderer uses kitty's graphics protocol with Unicode placeholders. The
// image is uploaded once with a virtual placement, and the layout carries
// placeholder characters whose foreground colour encodes the image ID. The
// terminal draws the image wherever those characters land, so the TUI
// framework positions it like ordinary text.
type KittyRenderer struct {
	next atomic.Uint32
}

// NewKitty returns a kitty renderer.
func NewKitty() *KittyRenderer { return &KittyRenderer{} }

// Render implements Renderer.
func (k *KittyRenderer) Render(img image.Image, box Box) (Frame, error) {
	rows := min(box.Rows, len(diacritics))
	cols := min(box.Cols, len(diacritics))
	id := idBase + k.next.Add(1)%0xFFFF

	var buf bytes.Buffer
	if err := png.Encode(&buf, Fit(img, 600, 600)); err != nil {
		return Frame{}, fmt.Errorf("encode art: %w", err)
	}

	return Frame{
		Setup:   transmit(id, buf.Bytes(), cols, rows),
		Lines:   placeholders(id, cols, rows),
		Cleanup: fmt.Sprintf("\x1b_Ga=d,d=I,i=%d,q=2\x1b\\", id),
	}, nil
}

// transmit uploads PNG data in 4 KiB chunks and creates a virtual placement.
func transmit(id uint32, data []byte, cols, rows int) string {
	payload := base64.StdEncoding.EncodeToString(data)
	var b strings.Builder
	const chunk = 4096
	for i := 0; i < len(payload); i += chunk {
		end := min(i+chunk, len(payload))
		more := 0
		if end < len(payload) {
			more = 1
		}
		if i == 0 {
			fmt.Fprintf(&b, "\x1b_Ga=T,U=1,f=100,t=d,i=%d,c=%d,r=%d,q=2,m=%d;%s\x1b\\", id, cols, rows, more, payload[i:end])
		} else {
			fmt.Fprintf(&b, "\x1b_Gm=%d;%s\x1b\\", more, payload[i:end])
		}
	}
	return b.String()
}

// placeholders writes every cell with explicit row and column diacritics, so
// the image survives partial redraws that start mid-row.
func placeholders(id uint32, cols, rows int) []string {
	color := fmt.Sprintf("\x1b[38;2;%d;%d;%dm", (id>>16)&0xFF, (id>>8)&0xFF, id&0xFF)
	lines := make([]string, rows)
	var b strings.Builder
	for r := range rows {
		b.Reset()
		b.WriteString(color)
		for c := range cols {
			b.WriteRune(placeholder)
			b.WriteRune(diacritics[r])
			b.WriteRune(diacritics[c])
		}
		b.WriteString("\x1b[39m")
		lines[r] = b.String()
	}
	return lines
}
