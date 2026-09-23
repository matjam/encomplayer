package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/matjam/encomplayer/internal/collection"
	"github.com/matjam/encomplayer/internal/domain"
)

type column struct {
	title string
	width int
	value func(domain.Track) string
}

func trackColumns(w int) []column {
	const numW, timeW = 5, 6
	flex := max(10, w-numW-timeW-4)
	titleW := flex * 40 / 100
	artistW := flex * 30 / 100
	albumW := flex - titleW - artistW
	return []column{
		{title: "#", width: numW},
		{title: "TITLE", width: titleW, value: domain.Track.DisplayTitle},
		{title: "ARTIST", width: artistW, value: domain.Track.DisplayArtist},
		{title: "ALBUM", width: albumW, value: domain.Track.DisplayAlbum},
		{title: "TIME", width: timeW, value: func(t domain.Track) string { return duration(t.Duration) }},
	}
}

// trackTable renders a list of tracks with a column header. playing marks
// the row that is currently playing.
func (st *styles) trackTable(l *collection.List[domain.Track], w int, playing func(int, domain.Track) bool, empty string) []string {
	cols := trackColumns(w)
	header := make([]string, len(cols))
	for i, c := range cols {
		header[i] = fit(c.title, c.width)
	}
	out := []string{
		st.dim.Render(strings.Join(header, " ")),
		st.grid.Render(strings.Repeat("─", w)),
	}
	if l.Len() == 0 {
		return append(out, "", st.dim.Render("  "+empty))
	}

	for i, t := range l.Visible() {
		isPlaying := playing(i, t)
		marker := fmt.Sprintf("%4d", i+1)
		switch {
		case isPlaying:
			marker = "   ▶"
		case l.IsSelected(i):
			marker = "   ◆"
		}

		cells := make([]string, len(cols))
		cells[0] = fit(marker, cols[0].width)
		for c := 1; c < len(cols); c++ {
			cells[c] = fit(cols[c].value(t), cols[c].width)
		}
		row := strings.Join(cells, " ")

		switch {
		case i == l.Cursor():
			row = st.cursor.Render(fit(row, w))
		case isPlaying, l.IsSelected(i):
			row = st.accent.Render(ansi.Strip(row))
		default:
			row = st.text.Render(row)
		}
		out = append(out, row)
	}
	return out
}
