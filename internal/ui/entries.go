package ui

import (
	"fmt"
	"slices"
	"sync"

	"github.com/matjam/encomplayer/internal/collection"
	"github.com/matjam/encomplayer/internal/domain"
	"github.com/matjam/encomplayer/internal/library"
)

type entryKind int

const (
	kindDir entryKind = iota
	kindGroup
	kindPlaylist
	kindTrack
)

// entry is one row in a browser tab: a folder, a tag group, a playlist or a
// track. Non-leaf entries build their children lazily.
type entry struct {
	kind     entryKind
	key      string
	label    string
	detail   string
	track    domain.Track
	children func() []entry
	tracks   func() []domain.Track
}

func (e entry) isLeaf() bool { return e.kind == kindTrack }

// allTracks returns the tracks an Add on this entry queues.
func (e entry) allTracks() []domain.Track {
	if e.isLeaf() {
		return []domain.Track{e.track}
	}
	if e.tracks != nil {
		return e.tracks()
	}
	return collection.FlatMap(e.children(), entry.allTracks)
}

// entryProvider adapts a root builder to collection.Provider.
type entryProvider struct {
	root func() []entry
}

func (p entryProvider) Root() []entry { return p.root() }

func (p entryProvider) Children(e entry) []entry {
	if e.children == nil {
		return nil
	}
	return e.children()
}

func (p entryProvider) IsLeaf(e entry) bool { return e.isLeaf() }
func (p entryProvider) Key(e entry) string  { return e.key }

func trackEntry(t domain.Track) entry {
	return entry{kind: kindTrack, key: t.Path, label: trackLabel(t), detail: duration(t.Duration), track: t}
}

func trackLabel(t domain.Track) string {
	if t.TrackNo > 0 {
		return fmt.Sprintf("%02d  %s", t.TrackNo, t.DisplayTitle())
	}
	return t.DisplayTitle()
}

func trackEntries(tracks []domain.Track) []entry {
	return collection.Map(tracks, trackEntry)
}

func dirEntries(d *library.Dir) []entry {
	out := make([]entry, 0, len(d.Dirs)+len(d.Tracks))
	for _, sub := range d.Dirs {
		out = append(out, entry{
			kind:     kindDir,
			key:      sub.Path,
			label:    sub.Name + "/",
			children: func() []entry { return dirEntries(sub) },
			tracks:   sub.AllTracks,
		})
	}
	return append(out, trackEntries(d.Tracks)...)
}

// groupEntries turns tag groups into entries whose children come from next.
func groupEntries[K comparable](groups []collection.Group[K, domain.Track], label func(K) string, next func([]domain.Track) []entry) []entry {
	out := make([]entry, len(groups))
	for i, g := range groups {
		out[i] = entry{
			kind:     kindGroup,
			key:      label(g.Key),
			label:    label(g.Key),
			detail:   fmt.Sprintf("%d", len(g.Items)),
			children: func() []entry { return next(g.Items) },
			tracks:   func() []domain.Track { return g.Items },
		}
	}
	return out
}

// albumsOf groups tracks into albums, ordered by year then name.
func albumsOf(tracks []domain.Track) []entry {
	groups := collection.GroupBy(tracks, library.AlbumOf, library.CompareAlbumKeys)
	year := func(g collection.Group[library.AlbumKey, domain.Track]) int {
		for _, t := range g.Items {
			if t.Year > 0 {
				return t.Year
			}
		}
		return 0
	}
	slices.SortStableFunc(groups, func(a, b collection.Group[library.AlbumKey, domain.Track]) int {
		return year(a) - year(b)
	})

	out := groupEntries(groups, func(k library.AlbumKey) string { return k.Album }, trackEntries)
	for i, g := range groups {
		if y := year(g); y > 0 {
			out[i].label = fmt.Sprintf("%s (%d)", g.Key.Album, y)
		}
		out[i].key = g.Key.Album + "\x00" + g.Key.Artist
	}
	return out
}

func identity(s string) string { return s }

func directoriesProvider(lib *library.Library) entryProvider {
	return entryProvider{root: func() []entry { return dirEntries(lib.Tree()) }}
}

func artistsProvider(lib *library.Library) entryProvider {
	return entryProvider{root: func() []entry { return groupEntries(lib.Artists(), identity, albumsOf) }}
}

func albumArtistsProvider(lib *library.Library) entryProvider {
	return entryProvider{root: func() []entry { return groupEntries(lib.AlbumArtists(), identity, albumsOf) }}
}

func albumsProvider(lib *library.Library) entryProvider {
	label := func(k library.AlbumKey) string { return k.Album + " — " + k.Artist }
	return entryProvider{root: func() []entry {
		out := groupEntries(lib.Albums(), label, trackEntries)
		for i, g := range lib.Albums() {
			out[i].key = g.Key.Album + "\x00" + g.Key.Artist
		}
		return out
	}}
}

func genresProvider(lib *library.Library) entryProvider {
	byArtist := func(tracks []domain.Track) []entry {
		groups := collection.GroupBy(tracks, domain.Track.DisplayAlbumArtist, domain.CompareFold)
		return groupEntries(groups, identity, albumsOf)
	}
	return entryProvider{root: func() []entry { return groupEntries(lib.Genres(), identity, byArtist) }}
}

// allMusicKey identifies the pinned whole-library entry. The NUL byte keeps
// it from colliding with a playlist file name.
const allMusicKey = "\x00all-music"

// playlistsProvider lists ALL MUSIC followed by the saved playlists. load
// turns a playlist name into tracks.
func playlistsProvider(all func() []domain.Track, names func() []string, load func(string) []domain.Track) entryProvider {
	return entryProvider{root: func() []entry {
		tracks := all()
		children := sync.OnceValue(func() []entry { return trackEntries(tracks) })
		out := []entry{{
			kind:     kindPlaylist,
			key:      allMusicKey,
			label:    "ALL MUSIC",
			detail:   fmt.Sprintf("%d", len(tracks)),
			children: children,
			tracks:   func() []domain.Track { return tracks },
		}}
		for _, name := range names() {
			out = append(out, entry{
				kind:     kindPlaylist,
				key:      name,
				label:    name,
				children: func() []entry { return trackEntries(load(name)) },
				tracks:   func() []domain.Track { return load(name) },
			})
		}
		return out
	}}
}
