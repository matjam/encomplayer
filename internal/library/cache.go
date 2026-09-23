package library

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/matjam/encomplayer/internal/fsutil"
)

const cacheVersion = 2

// Cache stores the last scan on disk, so startup can show the library
// immediately and a rescan only touches what changed.
type Cache struct {
	path string
}

// NewCache returns a cache stored at path.
func NewCache(path string) *Cache { return &Cache{path: path} }

type cacheFile struct {
	Version int `json:"version"`
	Snapshot
}

// Load returns the cached snapshot for root. A missing, outdated or foreign
// cache yields an empty snapshot rather than an error.
func (c *Cache) Load(root string) (*Snapshot, error) {
	data, err := os.ReadFile(c.path)
	if errors.Is(err, fs.ErrNotExist) {
		return emptySnapshot(root), nil
	}
	if err != nil {
		return nil, fmt.Errorf("read library cache: %w", err)
	}

	var f cacheFile
	if err := json.Unmarshal(data, &f); err != nil || f.Version != cacheVersion || f.Root != root {
		return emptySnapshot(root), nil
	}
	if f.Dirs == nil {
		f.Dirs = map[string]DirRecord{}
	}
	return &f.Snapshot, nil
}

// Save writes s, replacing the file atomically.
func (c *Cache) Save(s *Snapshot) error {
	data, err := json.Marshal(cacheFile{Version: cacheVersion, Snapshot: *s})
	if err != nil {
		return fmt.Errorf("encode library cache: %w", err)
	}
	return fsutil.WriteFileAtomic(c.path, data)
}
