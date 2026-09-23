// Package playlist stores playlists as extended M3U files, one per playlist,
// so other players can read them.
package playlist

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/matjam/encomplayer/internal/domain"
	"github.com/matjam/encomplayer/internal/fsutil"
)

const ext = ".m3u8"

// ErrInvalidName means a playlist name is empty or contains a path separator.
var ErrInvalidName = errors.New("invalid playlist name")

// ErrExists means the target playlist name is already taken.
var ErrExists = errors.New("playlist already exists")

// Store reads and writes playlists in one directory.
type Store struct {
	dir string
}

// NewStore returns a store rooted at dir. The directory is created on first
// save.
func NewStore(dir string) *Store { return &Store{dir: dir} }

// List returns playlist names, sorted.
func (s *Store) List() ([]string, error) {
	entries, err := os.ReadDir(s.dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list playlists: %w", err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ext) {
			names = append(names, strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())))
		}
	}
	slices.SortFunc(names, domain.CompareFold)
	return names, nil
}

// Load reads a playlist. Relative entries resolve against the store directory.
func (s *Store) Load(name string) (domain.Playlist, error) {
	path, err := s.path(name)
	if err != nil {
		return domain.Playlist{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return domain.Playlist{}, fmt.Errorf("load playlist %q: %w", name, err)
	}

	p := domain.Playlist{Name: name}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if !filepath.IsAbs(line) {
			line = filepath.Join(s.dir, line)
		}
		p.Paths = append(p.Paths, line)
	}
	if err := sc.Err(); err != nil {
		return domain.Playlist{}, fmt.Errorf("load playlist %q: %w", name, err)
	}
	return p, nil
}

// Save writes tracks as playlist name, replacing it if it exists.
func (s *Store) Save(name string, tracks []domain.Track) error {
	path, err := s.path(name)
	if err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString("#EXTM3U\n")
	for _, t := range tracks {
		fmt.Fprintf(&b, "#EXTINF:%d,%s - %s\n", int(t.Duration.Seconds()), t.DisplayArtist(), t.DisplayTitle())
		b.WriteString(t.Path)
		b.WriteByte('\n')
	}
	if err := fsutil.WriteFileAtomic(path, []byte(b.String())); err != nil {
		return fmt.Errorf("save playlist %q: %w", name, err)
	}
	return nil
}

// Delete removes a playlist.
func (s *Store) Delete(name string) error {
	path, err := s.path(name)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("delete playlist %q: %w", name, err)
	}
	return nil
}

// Rename changes a playlist's name. It refuses to overwrite another playlist.
func (s *Store) Rename(from, to string) error {
	src, err := s.path(from)
	if err != nil {
		return err
	}
	dst, err := s.path(to)
	if err != nil {
		return err
	}
	if _, err := os.Stat(dst); err == nil {
		return fmt.Errorf("rename playlist %q to %q: %w", from, to, ErrExists)
	}
	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("rename playlist %q: %w", from, err)
	}
	return nil
}

func (s *Store) path(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("%w: %q", ErrInvalidName, name)
	}
	return filepath.Join(s.dir, name+ext), nil
}
