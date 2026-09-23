// Package library scans a music folder, reads tags, caches the result and
// exposes the tracks by folder and by tag.
package library

import (
	"cmp"
	"path/filepath"
	"slices"
	"strings"

	"github.com/matjam/encomplayer/internal/collection"
	"github.com/matjam/encomplayer/internal/domain"
)

// Dir is one folder in the library tree.
type Dir struct {
	Name   string
	Path   string
	Dirs   []*Dir
	Tracks []domain.Track
}

// AllTracks returns every track in d and its subfolders, folder by folder.
func (d *Dir) AllTracks() []domain.Track {
	out := slices.Clone(d.Tracks)
	for _, sub := range d.Dirs {
		out = append(out, sub.AllTracks()...)
	}
	return out
}

// AlbumKey identifies an album. Albums with the same name by different album
// artists stay separate.
type AlbumKey struct {
	Album  string
	Artist string
}

// CompareAlbumKeys orders albums by name, then artist.
func CompareAlbumKeys(a, b AlbumKey) int {
	return cmp.Or(domain.CompareFold(a.Album, b.Album), domain.CompareFold(a.Artist, b.Artist))
}

// AlbumOf returns the album key for t.
func AlbumOf(t domain.Track) AlbumKey {
	return AlbumKey{Album: t.DisplayAlbum(), Artist: t.DisplayAlbumArtist()}
}

// Library is an immutable snapshot of a scanned folder. A rescan builds a new
// one.
type Library struct {
	root   string
	tracks []domain.Track
	byPath map[string]int
	tree   *Dir

	artists      []collection.Group[string, domain.Track]
	albumArtists []collection.Group[string, domain.Track]
	albums       []collection.Group[AlbumKey, domain.Track]
	genres       []collection.Group[string, domain.Track]
}

// New builds a library from scanned tracks.
func New(root string, tracks []domain.Track) *Library {
	sorted := slices.Clone(tracks)
	slices.SortFunc(sorted, func(a, b domain.Track) int { return strings.Compare(a.Path, b.Path) })

	l := &Library{
		root:   root,
		tracks: sorted,
		byPath: make(map[string]int, len(sorted)),
	}
	for i, t := range sorted {
		l.byPath[t.Path] = i
	}

	l.tree = buildTree(root, sorted)
	l.artists = groupTracks(sorted, domain.Track.DisplayArtist, domain.CompareFold)
	l.albumArtists = groupTracks(sorted, domain.Track.DisplayAlbumArtist, domain.CompareFold)
	l.albums = groupTracks(sorted, AlbumOf, CompareAlbumKeys)
	l.genres = groupTracks(sorted, domain.Track.DisplayGenre, domain.CompareFold)
	return l
}

// Root returns the scanned folder.
func (l *Library) Root() string { return l.root }

// Tracks returns every track sorted by path. Callers must not modify it.
func (l *Library) Tracks() []domain.Track { return l.tracks }

// Tree returns the root folder.
func (l *Library) Tree() *Dir { return l.tree }

// Lookup finds a track by absolute path.
func (l *Library) Lookup(path string) (domain.Track, bool) {
	i, ok := l.byPath[path]
	if !ok {
		return domain.Track{}, false
	}
	return l.tracks[i], true
}

// Artists groups tracks by track artist.
func (l *Library) Artists() []collection.Group[string, domain.Track] { return l.artists }

// AlbumArtists groups tracks by album artist.
func (l *Library) AlbumArtists() []collection.Group[string, domain.Track] { return l.albumArtists }

// Albums groups tracks by album.
func (l *Library) Albums() []collection.Group[AlbumKey, domain.Track] { return l.albums }

// Genres groups tracks by genre.
func (l *Library) Genres() []collection.Group[string, domain.Track] { return l.genres }

// groupTracks groups and sorts each group's tracks in album order.
func groupTracks[K comparable](tracks []domain.Track, key func(domain.Track) K, compare func(a, b K) int) []collection.Group[K, domain.Track] {
	groups := collection.GroupBy(tracks, key, compare)
	for _, g := range groups {
		slices.SortFunc(g.Items, domain.CompareTracks)
	}
	return groups
}

func buildTree(root string, tracks []domain.Track) *Dir {
	top := &Dir{Name: filepath.Base(root), Path: root}
	dirs := map[string]*Dir{root: top}

	var ensure func(path string) *Dir
	ensure = func(path string) *Dir {
		if d, ok := dirs[path]; ok {
			return d
		}
		parent := ensure(filepath.Dir(path))
		d := &Dir{Name: filepath.Base(path), Path: path}
		parent.Dirs = append(parent.Dirs, d)
		dirs[path] = d
		return d
	}

	for _, t := range tracks {
		dir := filepath.Dir(t.Path)
		if dir != root && !strings.HasPrefix(dir, root+string(filepath.Separator)) {
			continue
		}
		d := ensure(dir)
		d.Tracks = append(d.Tracks, t)
	}

	for _, d := range dirs {
		slices.SortFunc(d.Dirs, func(a, b *Dir) int { return domain.CompareFold(a.Name, b.Name) })
		slices.SortFunc(d.Tracks, domain.CompareTracks)
	}
	return top
}
