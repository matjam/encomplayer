package audio

import "sync"

// trackCache holds the files the player is using in memory: the playing
// track and the one preloaded to play next.
type trackCache struct {
	mu    sync.Mutex
	files map[string]*memFile
	load  func(path string) (*memFile, error)
}

func newTrackCache() *trackCache {
	return &trackCache{files: map[string]*memFile{}, load: loadFile}
}

// get returns the in-memory copy of path, starting a load if needed.
func (c *trackCache) get(path string) (*memFile, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if f, ok := c.files[path]; ok {
		return f, nil
	}
	f, err := c.load(path)
	if err != nil {
		return nil, err
	}
	c.files[path] = f
	return f, nil
}

// keep drops every file except those listed, cancelling their loads.
func (c *trackCache) keep(paths ...string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for p, f := range c.files {
		if !contains(paths, p) {
			f.cancel()
			delete(c.files, p)
		}
	}
}

// has reports whether path is cached.
func (c *trackCache) has(path string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.files[path]
	return ok
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s && s != "" {
			return true
		}
	}
	return false
}
