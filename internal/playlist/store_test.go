package playlist

import (
	"errors"
	"path/filepath"
	"slices"
	"testing"

	"github.com/matjam/encomplayer/internal/domain"
)

func TestStoreRoundTrip(t *testing.T) {
	s := NewStore(t.TempDir())
	tracks := []domain.Track{{Path: "/music/a.flac", Title: "A"}, {Path: "/music/b.mp3"}}

	if err := s.Save("Grid Mix", tracks); err != nil {
		t.Fatal(err)
	}
	if err := s.Save("Arcade", nil); err != nil {
		t.Fatal(err)
	}

	names, err := s.List()
	if err != nil || !slices.Equal(names, []string{"Arcade", "Grid Mix"}) {
		t.Fatalf("List = %v, %v", names, err)
	}

	p, err := s.Load("Grid Mix")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"/music/a.flac", "/music/b.mp3"}; !slices.Equal(p.Paths, want) {
		t.Errorf("paths = %v, want %v", p.Paths, want)
	}

	if err := s.Rename("Arcade", "Grid Mix"); !errors.Is(err, ErrExists) {
		t.Errorf("rename onto existing = %v, want ErrExists", err)
	}
	if err := s.Rename("Arcade", "Flynn's"); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete("Flynn's"); err != nil {
		t.Fatal(err)
	}
	if names, _ := s.List(); !slices.Equal(names, []string{"Grid Mix"}) {
		t.Errorf("after delete List = %v", names)
	}
}

func TestStoreRejectsBadNames(t *testing.T) {
	s := NewStore(t.TempDir())
	for _, name := range []string{"", " ", "..", "a/b", `a\b`} {
		t.Run(name, func(t *testing.T) {
			if err := s.Save(name, nil); !errors.Is(err, ErrInvalidName) {
				t.Errorf("Save(%q) = %v, want ErrInvalidName", name, err)
			}
		})
	}
}

func TestLoadResolvesRelativePaths(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	if err := s.Save("rel", []domain.Track{{Path: "songs/x.ogg"}}); err != nil {
		t.Fatal(err)
	}
	p, err := s.Load("rel")
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "songs/x.ogg"); p.Paths[0] != want {
		t.Errorf("path = %q, want %q", p.Paths[0], want)
	}
}
