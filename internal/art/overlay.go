package art

import (
	"fmt"
	"strings"
)

// Overlay protocols draw over the text grid at an absolute position, so their
// frames reserve the box with blank lines and draw in Place.

// blankLines returns rows lines of cols spaces.
func blankLines(cols, rows int) []string {
	blank := strings.Repeat(" ", cols)
	lines := make([]string, rows)
	for i := range lines {
		lines[i] = blank
	}
	return lines
}

// eraseCells clears the cols × rows box at cell (x, y) and restores the
// cursor.
func eraseCells(x, y, cols, rows int) string {
	var b strings.Builder
	b.WriteString("\x1b7")
	for r := range rows {
		fmt.Fprintf(&b, "\x1b[%d;%dH\x1b[%dX", y+r+1, x+1, cols)
	}
	b.WriteString("\x1b8")
	return b.String()
}

// drawAt writes seq with the cursor at cell (x, y) and restores the cursor.
func drawAt(x, y int, seq string) string {
	return fmt.Sprintf("\x1b7\x1b[%d;%dH%s\x1b8", y+1, x+1, seq)
}
