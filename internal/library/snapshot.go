package library

import (
	"time"

	"github.com/matjam/encomplayer/internal/domain"
)

// DirRecord remembers one folder's listing. A folder's modification time
// changes when entries are added, removed or renamed, so a matching time lets
// a rescan reuse the listing without statting every file over the network.
type DirRecord struct {
	ModTime time.Time `json:"mod_time"`
	Files   []string  `json:"files,omitempty"`
	Dirs    []string  `json:"dirs,omitempty"`
}

// Snapshot is the result of a scan, and what the cache stores.
type Snapshot struct {
	Root   string               `json:"root"`
	Tracks []domain.Track       `json:"tracks"`
	Dirs   map[string]DirRecord `json:"dirs"`
}

// Changes summarises how a scan differs from the previous snapshot.
type Changes struct {
	Added, Updated, Removed int
}

// Any reports whether anything changed.
func (c Changes) Any() bool { return c.Added+c.Updated+c.Removed > 0 }

func (s *Snapshot) trackMap() map[string]domain.Track {
	out := make(map[string]domain.Track, len(s.Tracks))
	for _, t := range s.Tracks {
		out[t.Path] = t
	}
	return out
}

// emptySnapshot returns a snapshot with nothing cached.
func emptySnapshot(root string) *Snapshot {
	return &Snapshot{Root: root, Dirs: map[string]DirRecord{}}
}
