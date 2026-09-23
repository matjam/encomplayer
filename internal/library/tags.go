package library

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/dhowden/tag"

	"github.com/matjam/encomplayer/internal/domain"
)

// Tags are the fields EncomPlayer reads from a file.
type Tags struct {
	Title, Artist, AlbumArtist, Album, Genre string
	Year, Track, Disc                        int
}

func (t Tags) empty() bool { return t == Tags{} }

// TagReader reads tags from one file.
type TagReader interface {
	ReadTags(ctx context.Context, path string) (Tags, error)
}

// Tagger builds tracks using a chain of tag readers. The first reader that
// returns any tag wins, so a pure-Go reader can go first and a slower,
// broader one can catch the formats it misses.
type Tagger struct {
	readers []TagReader
}

// NewTagger returns a tagger that tries readers in order.
func NewTagger(readers ...TagReader) *Tagger {
	return &Tagger{readers: readers}
}

// ReadTrack builds a Track from path's tags and file info. Files no reader
// understands still produce a track titled from the file name, so the
// library never hides a playable file.
func (tg *Tagger) ReadTrack(ctx context.Context, path string, info fs.FileInfo) domain.Track {
	t := domain.Track{Path: path, Size: info.Size(), ModTime: info.ModTime()}
	for _, r := range tg.readers {
		tags, err := r.ReadTags(ctx, path)
		if err != nil || tags.empty() {
			continue
		}
		t.Title = clean(tags.Title)
		t.Artist = clean(tags.Artist)
		t.AlbumArtist = clean(tags.AlbumArtist)
		t.Album = clean(tags.Album)
		t.Genre = clean(tags.Genre)
		t.Year, t.TrackNo, t.DiscNo = tags.Year, tags.Track, tags.Disc
		break
	}
	return t
}

// NativeReader reads ID3, MP4, FLAC and Ogg tags in pure Go.
type NativeReader struct{}

// ReadTags implements TagReader.
func (NativeReader) ReadTags(_ context.Context, path string) (Tags, error) {
	f, err := os.Open(path)
	if err != nil {
		return Tags{}, fmt.Errorf("read tags: %w", err)
	}
	defer f.Close()

	m, err := tag.ReadFrom(f)
	if err != nil {
		return Tags{}, fmt.Errorf("read tags %s: %w", path, err)
	}
	track, _ := m.Track()
	disc, _ := m.Disc()
	return Tags{
		Title:       m.Title(),
		Artist:      m.Artist(),
		AlbumArtist: m.AlbumArtist(),
		Album:       m.Album(),
		Genre:       m.Genre(),
		Year:        m.Year(),
		Track:       track,
		Disc:        disc,
	}, nil
}

func clean(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\x00", ""))
}
