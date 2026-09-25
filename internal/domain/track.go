// Package domain holds EncomPlayer's core types: tracks, playlists, the play
// queue and playback modes. It has no infrastructure dependencies.
package domain

import (
	"cmp"
	"path/filepath"
	"strings"
	"time"
)

// Unknown labels a missing tag value.
const Unknown = "Unknown"

// Track is one audio file and its tags.
type Track struct {
	Path        string        `json:"path"`
	Title       string        `json:"title,omitempty"`
	Artist      string        `json:"artist,omitempty"`
	AlbumArtist string        `json:"album_artist,omitempty"`
	Album       string        `json:"album,omitempty"`
	Genre       string        `json:"genre,omitempty"`
	Year        int           `json:"year,omitempty"`
	TrackNo     int           `json:"track,omitempty"`
	DiscNo      int           `json:"disc,omitempty"`
	Duration    time.Duration `json:"duration,omitempty"`
	Size        int64         `json:"size"`
	ModTime     time.Time     `json:"mod_time"`
}

// DisplayTitle returns the title tag, or the file name without extension.
func (t Track) DisplayTitle() string {
	if t.Title != "" {
		return t.Title
	}
	base := filepath.Base(t.Path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

// DisplayArtist returns the track artist, falling back to the album artist.
func (t Track) DisplayArtist() string {
	return firstNonEmpty(t.Artist, t.AlbumArtist, Unknown)
}

// DisplayAlbumArtist returns the album artist, falling back to the track
// artist.
func (t Track) DisplayAlbumArtist() string {
	return firstNonEmpty(t.AlbumArtist, t.Artist, Unknown)
}

// DisplayAlbum returns the album tag or Unknown.
func (t Track) DisplayAlbum() string {
	return firstNonEmpty(t.Album, Unknown)
}

// DisplayGenre returns the genre tag or Unknown.
func (t Track) DisplayGenre() string {
	return firstNonEmpty(t.Genre, Unknown)
}

// Ext returns the lower-case file extension without the dot.
func (t Track) Ext() string {
	return strings.ToLower(strings.TrimPrefix(filepath.Ext(t.Path), "."))
}

// CompareTracks orders tracks by disc, track number, artist, then title,
// matching rmpc's default browser_song_sort.
func CompareTracks(a, b Track) int {
	return cmp.Or(
		cmp.Compare(a.DiscNo, b.DiscNo),
		cmp.Compare(a.TrackNo, b.TrackNo),
		CompareFold(a.DisplayArtist(), b.DisplayArtist()),
		CompareFold(a.DisplayTitle(), b.DisplayTitle()),
		cmp.Compare(a.Path, b.Path),
	)
}

// CompareFold compares strings case-insensitively, breaking ties by exact
// comparison so the order is total.
func CompareFold(a, b string) int {
	return cmp.Or(
		strings.Compare(strings.ToLower(a), strings.ToLower(b)),
		strings.Compare(a, b),
	)
}

// ArtistKey groups tracks by artist for Careful mode: the track artist,
// falling back to the album artist, ignoring case. It is empty when neither
// is tagged, so untagged tracks never count as the same artist.
func (t Track) ArtistKey() string {
	return strings.ToLower(firstNonEmpty(t.Artist, t.AlbumArtist))
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
