// Package config loads EncomPlayer's settings and persists session state.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Config is the user's settings file, config.json in the config directory.
type Config struct {
	// MusicDir is scanned at startup when no directory is given.
	MusicDir string `json:"music_dir"`

	// AlbumArt selects the image protocol: auto, kitty, iterm, blocks or off.
	AlbumArt string `json:"album_art"`

	// VolumeStep is the percentage VolumeUp and VolumeDown change.
	VolumeStep int `json:"volume_step"`

	// SeekSeconds is how far SeekForward and SeekBack move.
	SeekSeconds int `json:"seek_seconds"`

	// RescanSeconds is how often the library is checked for changes. Zero
	// picks 60 s for local disks and 600 s for network mounts; a negative
	// value turns periodic rescans off.
	RescanSeconds int `json:"rescan_seconds"`

	// Keybinds overrides bindings per context (global, navigation, queue)
	// using rmpc notation, e.g. {"global": {"<C-p>": "TogglePause"}}.
	Keybinds map[string]map[string]string `json:"keybinds"`
}

// Default returns the built-in settings.
func Default() Config {
	return Config{
		AlbumArt:    "auto",
		VolumeStep:  5,
		SeekSeconds: 5,
	}
}

// Paths are the files EncomPlayer reads and writes.
type Paths struct {
	Config    string
	State     string
	Cache     string
	Playlists string
}

// DefaultPaths follows the XDG base directory spec, falling back to
// ~/.config and ~/.cache as rmpc does on every platform.
func DefaultPaths() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, fmt.Errorf("locate home directory: %w", err)
	}
	configDir := envOr("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	cacheDir := envOr("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	stateDir := envOr("XDG_STATE_HOME", filepath.Join(home, ".local", "state"))

	return Paths{
		Config:    filepath.Join(configDir, "encomplayer", "config.json"),
		Playlists: filepath.Join(configDir, "encomplayer", "playlists"),
		State:     filepath.Join(stateDir, "encomplayer", "state.json"),
		Cache:     filepath.Join(cacheDir, "encomplayer", "library.json"),
	}, nil
}

// Load reads the config file over the defaults. A missing file is not an
// error.
func Load(path string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
