package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/matjam/encomplayer/internal/domain"
)

// duration formats d as m:ss or h:mm:ss, or dashes when unknown.
func duration(d time.Duration) string {
	if d <= 0 {
		return "--:--"
	}
	s := int(d.Round(time.Second).Seconds())
	if s >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", s/3600, s/60%60, s%60)
	}
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

func totalDuration(tracks []domain.Track) time.Duration {
	var d time.Duration
	for _, t := range tracks {
		d += t.Duration
	}
	return d
}

func stripStyles(s string) string { return ansi.Strip(s) }

// trackDetails renders a key/value summary of t.
func trackDetails(t domain.Track, w int) []string {
	rows := [][2]string{
		{"TITLE", t.DisplayTitle()},
		{"ARTIST", t.DisplayArtist()},
		{"ALBUM ARTIST", t.DisplayAlbumArtist()},
		{"ALBUM", t.DisplayAlbum()},
		{"GENRE", t.DisplayGenre()},
		{"YEAR", intOrDash(t.Year)},
		{"TRACK", intOrDash(t.TrackNo)},
		{"DISC", intOrDash(t.DiscNo)},
		{"LENGTH", duration(t.Duration)},
		{"FORMAT", strings.ToUpper(t.Ext())},
		{"SIZE", fmt.Sprintf("%.1f MB", float64(t.Size)/(1<<20))},
		{"FILE", filepath.Base(t.Path)},
		{"FOLDER", filepath.Dir(t.Path)},
	}
	const keyW = 13
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, stDim.Render(fit(r[0], keyW))+stText.Render(fit(r[1], max(1, w-keyW))))
	}
	return out
}

func intOrDash(n int) string {
	if n <= 0 {
		return "—"
	}
	return fmt.Sprint(n)
}

func (m *Model) playingPath() string {
	if m.playGen == 0 {
		return ""
	}
	if t, _, ok := m.queue.Current(); ok {
		return t.Path
	}
	return ""
}
