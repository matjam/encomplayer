package ui

import (
	"fmt"
	"maps"

	"github.com/matjam/encomplayer/internal/art"
	"github.com/matjam/encomplayer/internal/config"
	"github.com/matjam/encomplayer/internal/keymap"
	"github.com/matjam/encomplayer/internal/tea"
	"github.com/matjam/encomplayer/internal/theme"
)

// ReloadMsg asks the UI to reread the config file and the active theme. The
// program sends it on SIGUSR1; :reload does the same from inside.
type ReloadMsg struct{}

// applyTheme switches to the named theme. On failure the current styles
// stay, or the default theme is used if there are none yet.
func (m *Model) applyTheme(name string) error {
	var (
		t   theme.Theme
		err error
	)
	if m.deps.Themes != nil {
		t, err = m.deps.Themes.Load(name)
	} else {
		var ok bool
		if t, ok = theme.Builtin(name); !ok {
			err = fmt.Errorf("%w: %q", theme.ErrUnknown, name)
		}
	}
	if err != nil {
		if m.st == nil {
			fallback, _ := theme.Builtin(theme.Default)
			m.st = newStyles(fallback)
		}
		return fmt.Errorf("theme: %w", err)
	}
	m.st = newStyles(t)
	return nil
}

// applyConfig makes next the live config, updating whatever depends on the
// fields that changed.
func (m *Model) applyConfig(next config.Config) tea.Cmd {
	prev := m.deps.Config
	m.deps.Config = next

	var cmds []tea.Cmd
	if err := m.applyTheme(next.Theme); err != nil {
		m.status.errorf("%v", err)
	}
	if next.Visualizer != m.viz.info.Name {
		if err := m.selectViz(next.Visualizer); err != nil {
			m.status.errorf("%v", err)
		}
	}
	if next.AlbumArt != prev.AlbumArt {
		cmds = append(cmds, m.applyArt(next.AlbumArt))
	}
	if !keybindsEqual(prev.Keybinds, next.Keybinds) {
		if err := m.applyKeybinds(next.Keybinds); err != nil {
			m.status.errorf("%v", err)
		}
	}
	if next.MusicDir != prev.MusicDir && next.MusicDir != "" {
		m.snapshot = nil
		cmds = append(cmds, m.startScan(next.MusicDir, false))
	}
	m.layout()
	return tea.Batch(cmds...)
}

// applyArt swaps the album art renderer and redraws the current cover.
func (m *Model) applyArt(setting string) tea.Cmd {
	r, protocol, err := art.Choose(setting, m.terminal())
	if err != nil {
		m.status.errorf("%v", err)
		return nil
	}
	cleanup := m.art.release()
	m.deps.Art, m.deps.ArtProtocol, m.deps.ArtSetting = r, protocol, setting
	path := m.art.path
	m.art = artState{placeSeq: m.art.placeSeq}
	cmds := []tea.Cmd{tea.Raw(cleanup)}
	if t, ok := m.resolveTrack(path); ok && path != "" {
		cmds = append(cmds, m.showArt(&t))
	}
	return tea.Batch(cmds...)
}

// applyKeybinds rebuilds the keymap from the defaults plus overrides.
func (m *Model) applyKeybinds(binds map[string]map[string]string) error {
	km, err := keymap.WithOverrides(binds)
	if err != nil {
		return fmt.Errorf("keybinds: %w", err)
	}
	m.deps.Keymap = km
	m.resolver = keymap.NewResolver(km)
	return nil
}

// saveConfig writes the live config to disk.
func (m *Model) saveConfig() {
	if err := config.Save(m.deps.Paths.Config, m.deps.Config); err != nil {
		m.status.errorf("%v", err)
	}
}

// reload rereads config.json and the theme file, for edits made outside the
// player.
func (m *Model) reload() tea.Cmd {
	cfg, err := config.Load(m.deps.Paths.Config)
	if err != nil {
		m.status.errorf("RELOAD: %v", err)
		return nil
	}
	m.status = status{}
	if err := m.viz.catalog.Reload(); err != nil {
		m.status.errorf("RELOAD: %v", err)
	}
	cmd := m.applyConfig(cfg)
	if !m.status.isError {
		m.status.infof("config and theme %s reloaded", cfg.Theme)
	}
	return cmd
}

func keybindsEqual(a, b map[string]map[string]string) bool {
	return maps.EqualFunc(a, b, func(x, y map[string]string) bool { return maps.Equal(x, y) })
}
