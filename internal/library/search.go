package library

import (
	"path/filepath"
	"strings"

	"github.com/matjam/encomplayer/internal/domain"
)

// Field is a tag that search can match against.
type Field string

// Search fields, matching rmpc's default search tags.
const (
	AnyField    Field = "Any Tag"
	ArtistField Field = "Artist"
	AlbumField  Field = "Album"
	AlbumArtist Field = "Album Artist"
	TitleField  Field = "Title"
	FileField   Field = "Filename"
	GenreField  Field = "Genre"
)

// Fields lists the search fields in display order.
var Fields = []Field{AnyField, ArtistField, AlbumField, AlbumArtist, TitleField, FileField, GenreField}

// Search returns tracks whose field contains every word of query,
// case-insensitively.
func (l *Library) Search(field Field, query string) []domain.Track {
	words := strings.Fields(strings.ToLower(query))
	if len(words) == 0 {
		return nil
	}

	var out []domain.Track
	for _, t := range l.tracks {
		hay := strings.ToLower(fieldValue(t, field))
		if containsAll(hay, words) {
			out = append(out, t)
		}
	}
	return out
}

func fieldValue(t domain.Track, f Field) string {
	switch f {
	case ArtistField:
		return t.Artist
	case AlbumField:
		return t.Album
	case AlbumArtist:
		return t.AlbumArtist
	case TitleField:
		return t.DisplayTitle()
	case FileField:
		return filepath.Base(t.Path)
	case GenreField:
		return t.Genre
	default:
		return strings.Join([]string{t.DisplayTitle(), t.Artist, t.AlbumArtist, t.Album, t.Genre, filepath.Base(t.Path)}, "\x00")
	}
}

func containsAll(hay string, words []string) bool {
	for _, w := range words {
		if !strings.Contains(hay, w) {
			return false
		}
	}
	return true
}
