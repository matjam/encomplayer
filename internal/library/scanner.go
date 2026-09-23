package library

import (
	"context"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/matjam/encomplayer/internal/domain"
)

// checkpointEvery is how many freshly read tracks trigger a checkpoint.
const checkpointEvery = 250

// Progress reports how far a scan has got.
type Progress struct {
	// Discovering is true while folders are still being listed, so Found
	// is a running total rather than the final count.
	Discovering bool

	Found int
	Done  int
}

// Fraction returns completion in [0, 1], or 0 while discovering.
func (p Progress) Fraction() float64 {
	if p.Discovering || p.Found == 0 {
		return 0
	}
	return float64(p.Done) / float64(p.Found)
}

// Prober reports a track's length.
type Prober func(ctx context.Context, path string) (time.Duration, error)

// Options tune one scan.
type Options struct {
	// Full ignores the previous snapshot and rereads every file.
	Full bool

	// Progress receives updates. It must not block.
	Progress func(Progress)

	// Checkpoint receives partial snapshots during long scans, so an
	// interrupted scan can resume from the cache.
	Checkpoint func(*Snapshot)
}

// Scanner finds audio files under a directory and reads their tags.
type Scanner struct {
	supported func(ext string) bool
	tagger    *Tagger
	probe     Prober
	workers   int
}

// NewScanner returns a scanner that keeps files whose lower-case extension,
// without the dot, satisfies supported. probe fills in track lengths, which
// tags rarely carry; nil leaves them unknown.
func NewScanner(supported func(ext string) bool, tagger *Tagger, probe Prober) *Scanner {
	// The work is I/O bound, and libraries often sit on network mounts,
	// so run well past the CPU count.
	return &Scanner{supported: supported, tagger: tagger, probe: probe, workers: max(16, runtime.NumCPU()*2)}
}

// pending is a file whose tags must be read.
type pending struct {
	path string
	info fs.FileInfo
}

// walkState collects the results of the parallel folder walk.
type walkState struct {
	mu     sync.Mutex
	dirs   map[string]DirRecord
	reused []string
	stat   []pending
	found  atomic.Int64
}

// Scan brings prev up to date with the folder at root and returns the new
// snapshot and what changed. prev may be nil.
//
// Folders whose modification time matches prev are not relisted, and files
// whose size and modification time match are not reread. A file edited in
// place without touching its folder is only noticed by a Full scan.
func (s *Scanner) Scan(ctx context.Context, root string, prev *Snapshot, opts Options) (*Snapshot, Changes, error) {
	if prev == nil || prev.Root != root || opts.Full {
		prev = emptySnapshot(root)
	}
	report := func(p Progress) {
		if opts.Progress != nil {
			opts.Progress(p)
		}
	}
	report(Progress{Discovering: true})

	ws := &walkState{dirs: map[string]DirRecord{}}
	prevTracks := prev.trackMap()
	if err := s.walk(ctx, root, prev, prevTracks, ws, report); err != nil {
		return nil, Changes{}, fmt.Errorf("scan %s: %w", root, err)
	}

	total := len(ws.reused) + len(ws.stat)
	next := make(map[string]domain.Track, total)
	for _, p := range ws.reused {
		next[p] = prevTracks[p]
	}

	var (
		mu      sync.Mutex
		done    atomic.Int64
		fresh   int
		changes Changes
	)
	done.Store(int64(len(ws.reused)))
	reportDone := func() { report(Progress{Found: total, Done: int(done.Load())}) }
	reportDone()

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(s.workers)
	for _, f := range ws.stat {
		g.Go(func() error {
			if err := gctx.Err(); err != nil {
				return err
			}
			old, known := prevTracks[f.path]
			t := old
			reread := !known || old.Size != f.info.Size() || !old.ModTime.Equal(f.info.ModTime())
			if reread {
				t = s.readTrack(gctx, f)
			}

			mu.Lock()
			next[f.path] = t
			switch {
			case !known:
				changes.Added++
			case reread:
				changes.Updated++
			}
			if reread {
				fresh++
				if fresh%checkpointEvery == 0 && opts.Checkpoint != nil {
					opts.Checkpoint(checkpoint(prev, next))
				}
			}
			mu.Unlock()

			if done.Add(1)%32 == 0 {
				reportDone()
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, Changes{}, fmt.Errorf("scan %s: %w", root, err)
	}
	reportDone()

	for p := range prevTracks {
		if _, ok := next[p]; !ok {
			changes.Removed++
		}
	}
	return &Snapshot{Root: root, Tracks: slices.Collect(maps.Values(next)), Dirs: ws.dirs}, changes, nil
}

func (s *Scanner) readTrack(ctx context.Context, f pending) domain.Track {
	t := s.tagger.ReadTrack(ctx, f.path, f.info)
	if s.probe != nil {
		// An unprobeable file stays in the library with an unknown
		// length; playback reports real errors.
		t.Duration, _ = s.probe(ctx, f.path)
	}
	return t
}

// checkpoint merges the tracks read so far over prev. It keeps prev's folder
// records: a folder that has changed since has a newer modification time, so
// the next scan relists it anyway.
func checkpoint(prev *Snapshot, next map[string]domain.Track) *Snapshot {
	merged := prev.trackMap()
	maps.Copy(merged, next)
	return &Snapshot{Root: prev.Root, Tracks: slices.Collect(maps.Values(merged)), Dirs: maps.Clone(prev.Dirs)}
}

// walk lists folders in parallel. A semaphore bounds concurrent I/O while
// goroutines per folder keep the tree walk itself unbounded.
func (s *Scanner) walk(ctx context.Context, root string, prev *Snapshot, prevTracks map[string]domain.Track, ws *walkState, report func(Progress)) error {
	g, ctx := errgroup.WithContext(ctx)
	sem := make(chan struct{}, s.workers)

	var visit func(dir string) error
	visit = func(dir string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		sem <- struct{}{}
		rec, reused, files, err := s.list(dir, prev, prevTracks)
		<-sem
		if err != nil {
			// Only the root is fatal; an unreadable subfolder is
			// skipped so one bad directory does not stop the scan.
			if dir == root {
				return err
			}
			return nil
		}

		ws.mu.Lock()
		ws.dirs[dir] = rec
		if reused {
			for _, name := range rec.Files {
				ws.reused = append(ws.reused, filepath.Join(dir, name))
			}
		} else {
			ws.stat = append(ws.stat, files...)
		}
		ws.mu.Unlock()

		if n := ws.found.Add(int64(len(rec.Files))); n > 0 {
			report(Progress{Discovering: true, Found: int(n)})
		}
		for _, sub := range rec.Dirs {
			g.Go(func() error { return visit(filepath.Join(dir, sub)) })
		}
		return nil
	}

	g.Go(func() error { return visit(root) })
	return g.Wait()
}

// list returns dir's record. It reuses the previous record when the folder
// is unchanged and every file in it is cached; otherwise it reads the folder
// and stats each audio file.
func (s *Scanner) list(dir string, prev *Snapshot, prevTracks map[string]domain.Track) (DirRecord, bool, []pending, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return DirRecord{}, false, nil, err
	}
	if old, ok := prev.Dirs[dir]; ok && old.ModTime.Equal(info.ModTime()) && allCached(dir, old.Files, prevTracks) {
		return old, true, nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return DirRecord{}, false, nil, err
	}
	rec := DirRecord{ModTime: info.ModTime()}
	var files []pending
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		path := filepath.Join(dir, name)
		switch {
		case e.IsDir():
			rec.Dirs = append(rec.Dirs, name)
		case e.Type().IsRegular() || e.Type()&fs.ModeSymlink != 0:
			if !s.supported(extOf(name)) {
				continue
			}
			fi, err := os.Stat(path)
			if err != nil || !fi.Mode().IsRegular() {
				continue
			}
			rec.Files = append(rec.Files, name)
			files = append(files, pending{path: path, info: fi})
		}
	}
	return rec, false, files, nil
}

func allCached(dir string, names []string, tracks map[string]domain.Track) bool {
	for _, n := range names {
		if _, ok := tracks[filepath.Join(dir, n)]; !ok {
			return false
		}
	}
	return true
}

func extOf(path string) string {
	return strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
}
