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
func scanBar(p library.Progress, w int) string {
	if p.Discovering {
		label := fmt.Sprintf(" DISCOVERING %d FILES", p.Found)
		return sweep(max(4, w-len(label))) + stText.Render(label)
	}

	label := fmt.Sprintf(" %3.0f%% %d/%d", p.Fraction()*100, p.Done, p.Found)
	barW := max(4, w-len(label))
	lit := int(p.Fraction() * float64(barW))
	return stBright.Render(strings.Repeat("▰", lit)) + stGrid.Render(strings.Repeat("▱", barW-lit)) + stText.Render(label)
}

// sweep draws a three-cell light bouncing across w cells, driven by the
// wall clock so it moves on every redraw.
func sweep(w int) string {
	span := max(1, w-3)
	step := int(time.Now().UnixMilli()/80) % (2 * span)
	pos := step
	if step >= span {
		pos = 2*span - step
	}
	return stGrid.Render(strings.Repeat("▱", pos)) + stAccent.Render("▰▰▰") + stGrid.Render(strings.Repeat("▱", max(0, w-pos-3)))
}
