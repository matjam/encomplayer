package library

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

// countingReader counts tag reads so tests can see what a rescan touched.
type countingReader struct{ reads atomic.Int64 }

func (c *countingReader) ReadTags(_ context.Context, path string) (Tags, error) {
	c.reads.Add(1)
	return Tags{Title: filepath.Base(path)}, nil
}

func writeFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("audio"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// bumpDir moves a folder's mtime forward, since filesystems with coarse
// timestamps may not change it within one test.
func bumpDir(t *testing.T, dir string, by time.Duration) {
	t.Helper()
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	mt := info.ModTime().Add(by)
	if err := os.Chtimes(dir, mt, mt); err != nil {
		t.Fatal(err)
	}
}

func TestIncrementalScan(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a/one.mp3"))
	writeFile(t, filepath.Join(root, "a/two.flac"))
	writeFile(t, filepath.Join(root, "b/three.ogg"))
	writeFile(t, filepath.Join(root, "b/cover.jpg"))
	writeFile(t, filepath.Join(root, ".hidden/four.mp3"))

	reader := &countingReader{}
	s := NewScanner(func(ext string) bool { return ext == "mp3" || ext == "flac" || ext == "ogg" }, NewTagger(reader), nil)
	ctx := context.Background()

	snap, changes, err := s.Scan(ctx, root, nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Tracks) != 3 || changes.Added != 3 || reader.reads.Load() != 3 {
		t.Fatalf("first scan: %d tracks, %+v, %d reads", len(snap.Tracks), changes, reader.reads.Load())
	}

	tests := []struct {
		name        string
		mutate      func()
		opts        Options
		wantTracks  int
		wantChanges Changes
		wantReads   int64
	}{
		{
			name:       "unchanged reads nothing",
			mutate:     func() {},
			wantTracks: 3,
		},
		{
			name: "new file in existing folder",
			mutate: func() {
				writeFile(t, filepath.Join(root, "a/five.mp3"))
				bumpDir(t, filepath.Join(root, "a"), time.Second)
			},
			wantTracks:  4,
			wantChanges: Changes{Added: 1},
			wantReads:   1,
		},
		{
			name: "new folder",
			mutate: func() {
				writeFile(t, filepath.Join(root, "c/six.flac"))
				bumpDir(t, root, time.Second)
			},
			wantTracks:  5,
			wantChanges: Changes{Added: 1},
			wantReads:   1,
		},
		{
			name: "deleted file",
			mutate: func() {
				if err := os.Remove(filepath.Join(root, "b/three.ogg")); err != nil {
					t.Fatal(err)
				}
				bumpDir(t, filepath.Join(root, "b"), time.Second)
			},
			wantTracks:  4,
			wantChanges: Changes{Removed: 1},
		},
		{
			name:        "full rescan rereads everything",
			mutate:      func() {},
			opts:        Options{Full: true},
			wantTracks:  4,
			wantChanges: Changes{Added: 4},
			wantReads:   4,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.mutate()
			before := reader.reads.Load()
			next, changes, err := s.Scan(ctx, root, snap, tc.opts)
			if err != nil {
				t.Fatal(err)
			}
			if len(next.Tracks) != tc.wantTracks {
				t.Errorf("tracks = %d, want %d", len(next.Tracks), tc.wantTracks)
			}
			if changes != tc.wantChanges {
				t.Errorf("changes = %+v, want %+v", changes, tc.wantChanges)
			}
			if reads := reader.reads.Load() - before; reads != tc.wantReads {
				t.Errorf("tag reads = %d, want %d", reads, tc.wantReads)
			}
			snap = next
		})
	}
}

func TestCacheRoundTrip(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "x/song.mp3"))
	s := NewScanner(func(string) bool { return true }, NewTagger(&countingReader{}), nil)
	snap, _, err := s.Scan(context.Background(), root, nil, Options{})
	if err != nil {
		t.Fatal(err)
	}

	c := NewCache(filepath.Join(t.TempDir(), "library.json"))
	if err := c.Save(snap); err != nil {
		t.Fatal(err)
	}
	got, err := c.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Tracks) != 1 || len(got.Dirs) != 2 {
		t.Errorf("loaded %d tracks, %d dirs; want 1, 2", len(got.Tracks), len(got.Dirs))
	}
	if other, _ := c.Load("/elsewhere"); len(other.Tracks) != 0 {
		t.Error("cache for another root should be ignored")
	}
}

func TestCheckpointSurvivesInterruption(t *testing.T) {
	root := t.TempDir()
	for i := range checkpointEvery + 10 {
		writeFile(t, filepath.Join(root, "d", fmt.Sprintf("t%04d.mp3", i)))
	}
	var last *Snapshot
	s := NewScanner(func(string) bool { return true }, NewTagger(&countingReader{}), nil)
	if _, _, err := s.Scan(context.Background(), root, nil, Options{Checkpoint: func(sn *Snapshot) { last = sn }}); err != nil {
		t.Fatal(err)
	}
	if last == nil || len(last.Tracks) < checkpointEvery {
		t.Fatalf("checkpoint missing or short: %v", last)
	}
}
