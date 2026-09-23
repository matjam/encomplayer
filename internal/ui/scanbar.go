package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/matjam/encomplayer/internal/library"
)

// scanBar renders scan progress in w cells. While the walk is still
// counting files, the total is unknown, so a light sweeps back and forth
// beside the running count.
func (st *styles) scanBar(p library.Progress, w int) string {
	if p.Discovering {
		label := fmt.Sprintf(" DISCOVERING %d FILES", p.Found)
		return st.sweep(max(4, w-len(label))) + st.text.Render(label)
	}

	label := fmt.Sprintf(" %3.0f%% %d/%d", p.Fraction()*100, p.Done, p.Found)
	barW := max(4, w-len(label))
	lit := int(p.Fraction() * float64(barW))
	return st.bright.Render(strings.Repeat("▰", lit)) + st.grid.Render(strings.Repeat("▱", barW-lit)) + st.text.Render(label)
}

// sweep draws a three-cell light bouncing across w cells, driven by the
// wall clock so it moves on every redraw.
func (st *styles) sweep(w int) string {
	span := max(1, w-3)
	step := int(time.Now().UnixMilli()/80) % (2 * span)
	pos := step
	if step >= span {
		pos = 2*span - step
	}
	return st.grid.Render(strings.Repeat("▱", pos)) + st.accent.Render("▰▰▰") + st.grid.Render(strings.Repeat("▱", max(0, w-pos-3)))
}
