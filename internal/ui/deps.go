package ui

import (
	"context"
	"io/fs"
	"time"

	"github.com/matjam/encomplayer/internal/art"
	"github.com/matjam/encomplayer/internal/audio"
	"github.com/matjam/encomplayer/internal/config"
	"github.com/matjam/encomplayer/internal/domain"
	"github.com/matjam/encomplayer/internal/keymap"
	"github.com/matjam/encomplayer/internal/library"
)

// Player is the playback engine the UI drives.
type Player interface {
	Play(ctx context.Context, path string, start time.Duration) (uint64, error)
	Preload(path string)
	TogglePause()
	Stop()
	Seek(d time.Duration) error
	Progress() (pos, length time.Duration)
	SetVolume(percent int)
	Volume() int
	State() audio.State
	Spectrum(bands int) []float64
	Ended() <-chan uint64
}

// Scanner reads a music folder.
type Scanner interface {
	Scan(ctx context.Context, root string, prev *library.Snapshot, opts library.Options) (*library.Snapshot, library.Changes, error)
}

// TrackReader reads one file's tags, for files outside the scanned folder.
type TrackReader interface {
	ReadTrack(ctx context.Context, path string, info fs.FileInfo) domain.Track
}

// LibraryCache persists scan results between runs.
type LibraryCache interface {
	Load(root string) (*library.Snapshot, error)
	Save(s *library.Snapshot) error
}

// PlaylistStore persists playlists.
type PlaylistStore interface {
	List() ([]string, error)
	Load(name string) (domain.Playlist, error)
	Save(name string, tracks []domain.Track) error
	Delete(name string) error
	Rename(from, to string) error
}

// Deps are everything the UI needs from the rest of the program.
type Deps struct {
	Player    Player
	Scanner   Scanner
	Tags      TrackReader
	Cache     LibraryCache
	Playlists PlaylistStore
	Keymap    *keymap.Keymap

	// Art draws album art. Nil disables it.
	Art art.Renderer

	// ArtProtocol names the protocol Art implements, for display.
	ArtProtocol string

	// Formats lists the playable extensions, for display.
	Formats string

	Config    config.Config
	State     config.State
	StatePath string
	MusicDir  string
	Version   string
}
