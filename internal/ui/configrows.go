package ui

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/matjam/encomplayer/internal/art"
	"github.com/matjam/encomplayer/internal/config"
	"github.com/matjam/encomplayer/internal/keymap"
)

// configRow is one line on a config screen tab.
type configRow interface {
	label() string
	display(m *Model) string

	// activate runs when the row is chosen with Enter. Read-only rows
	// return nil.
	activate(m *Model, c *configModal) tea.Cmd
}

// field edits one typed config value. Rows with choices open a picker;
// rows with flip toggle on Enter; the rest take typed input.
type field[T comparable] struct {
	name    string
	get     func(*config.Config) T
	set     func(*config.Config, T)
	format  func(T) string
	parse   func(string) (T, error)
	choices func(m *Model) []T
	flip    func(T) T

	// preview applies a choice before it is committed, as the theme
	// picker does; nil means no preview.
	preview func(m *Model, v T)
}

func (f field[T]) label() string { return f.name }

func (f field[T]) display(m *Model) string { return f.format(f.get(&m.deps.Config)) }

func (f field[T]) activate(m *Model, c *configModal) tea.Cmd {
	current := f.get(&m.deps.Config)
	commit := func(v T) tea.Cmd {
		return m.updateConfig(func(cfg *config.Config) { f.set(cfg, v) })
	}

	switch {
	case f.flip != nil:
		return commit(f.flip(current))
	case f.choices != nil:
		values := f.choices(m)
		labels := make([]string, len(values))
		for i, v := range values {
			labels[i] = f.format(v)
		}
		c.openChoice(f.name, labels, slices.Index(values, current),
			func(i int) {
				if f.preview != nil {
					f.preview(m, values[i])
				}
			},
			func(i int) tea.Cmd { return commit(values[i]) },
			func() {
				if f.preview != nil {
					f.preview(m, current)
				}
			})
	default:
		c.openInput(f.name, f.format(current), func(s string) (tea.Cmd, error) {
			v, err := f.parse(s)
			if err != nil {
				return nil, err
			}
			return commit(v), nil
		})
	}
	return nil
}

// intField edits an integer bounded to [lo, hi].
func intField(name string, lo, hi int, get func(*config.Config) *int) field[int] {
	return field[int]{
		name:   name,
		get:    func(c *config.Config) int { return *get(c) },
		set:    func(c *config.Config, v int) { *get(c) = v },
		format: strconv.Itoa,
		parse: func(s string) (int, error) {
			v, err := strconv.Atoi(strings.TrimSpace(s))
			if err != nil || v < lo || v > hi {
				return 0, fmt.Errorf("%s must be a whole number from %d to %d", name, lo, hi)
			}
			return v, nil
		},
	}
}

func stringField(name string, get func(*config.Config) *string) field[string] {
	return field[string]{
		name:   name,
		get:    func(c *config.Config) string { return *get(c) },
		set:    func(c *config.Config, v string) { *get(c) = v },
		format: func(s string) string { return s },
		parse:  func(s string) (string, error) { return strings.TrimSpace(s), nil },
	}
}

func boolField(name string, get func(*config.Config) *bool) field[bool] {
	return field[bool]{
		name: name,
		get:  func(c *config.Config) bool { return *get(c) },
		set:  func(c *config.Config, v bool) { *get(c) = v },
		format: func(b bool) string {
			if b {
				return "ON"
			}
			return "OFF"
		},
		flip: func(b bool) bool { return !b },
	}
}

func themeField() field[string] {
	f := stringField("Theme", func(c *config.Config) *string { return &c.Theme })
	f.choices = func(m *Model) []string {
		if m.deps.Themes == nil {
			return []string{m.deps.Config.Theme}
		}
		return m.deps.Themes.Names()
	}
	f.preview = func(m *Model, name string) {
		if err := m.applyTheme(name); err != nil {
			m.status.errorf("%v", err)
		}
	}
	return f
}

func artField() field[string] {
	f := stringField("Album art", func(c *config.Config) *string { return &c.AlbumArt })
	f.choices = func(*Model) []string { return art.Settings }
	return f
}

// keyRow shows the keys bound to one action. Enter adds another key; the
// config screen's Delete removes keys that the config file added.
type keyRow struct {
	ctx    keymap.Context
	action string
	keys   []string
}

func (k keyRow) label() string { return string(k.ctx) + " · " + k.action }

func (k keyRow) display(*Model) string { return strings.Join(k.keys, "  ") }

func (k keyRow) activate(m *Model, c *configModal) tea.Cmd {
	c.openInput("New key for "+k.action+" (rmpc notation, e.g. <C-p>)", "", func(s string) (tea.Cmd, error) {
		notation := strings.TrimSpace(s)
		if _, err := keymap.Parse(notation); err != nil {
			return nil, err
		}
		return m.updateConfig(func(cfg *config.Config) {
			if cfg.Keybinds == nil {
				cfg.Keybinds = map[string]map[string]string{}
			}
			if cfg.Keybinds[string(k.ctx)] == nil {
				cfg.Keybinds[string(k.ctx)] = map[string]string{}
			}
			cfg.Keybinds[string(k.ctx)][notation] = k.action
		}), nil
	})
	return nil
}

// errNoOverride means the row has no config-file keys to remove.
var errNoOverride = errors.New("only keys added in the config file can be removed")

// removeOverrides deletes the config-file keys bound to this action.
func (k keyRow) removeOverrides(m *Model) (tea.Cmd, error) {
	binds := m.deps.Config.Keybinds[string(k.ctx)]
	found := false
	for _, action := range binds {
		found = found || action == k.action
	}
	if !found {
		return nil, errNoOverride
	}
	return m.updateConfig(func(cfg *config.Config) {
		maps.DeleteFunc(cfg.Keybinds[string(k.ctx)], func(_, action string) bool { return action == k.action })
	}), nil
}

// keyRows lists every bound action, grouped by context.
func keyRows(m *Model) []configRow {
	var rows []configRow
	for _, ctx := range keymap.Contexts {
		byAction := map[string][]string{}
		var order []string
		for _, b := range m.deps.Keymap.Help(ctx) {
			if _, seen := byAction[b[1]]; !seen {
				order = append(order, b[1])
			}
			byAction[b[1]] = append(byAction[b[1]], b[0])
		}
		for _, action := range order {
			rows = append(rows, keyRow{ctx: ctx, action: action, keys: byAction[action]})
		}
	}
	return rows
}

// infoRow shows a value that cannot be edited here.
type infoRow struct{ name, value string }

func (i infoRow) label() string                         { return i.name }
func (i infoRow) display(*Model) string                 { return i.value }
func (i infoRow) activate(*Model, *configModal) tea.Cmd { return nil }

// configTabs defines the config screen.
func configTabs() []configTab {
	return []configTab{
		{name: "General", rows: func(*Model) []configRow {
			return []configRow{
				stringField("Music folder", func(c *config.Config) *string { return &c.MusicDir }),
				intField("Rescan seconds (0 auto, -1 off)", -1, 86400, func(c *config.Config) *int { return &c.RescanSeconds }),
				intField("Volume step %", 1, 50, func(c *config.Config) *int { return &c.VolumeStep }),
				intField("Seek step seconds", 1, 600, func(c *config.Config) *int { return &c.SeekSeconds }),
			}
		}},
		{name: "Appearance", rows: func(*Model) []configRow {
			return []configRow{themeField(), artField()}
		}},
		{name: "Mouse", rows: func(*Model) []configRow {
			return []configRow{
				boolField("Enable mouse", func(c *config.Config) *bool { return &c.EnableMouse }),
				intField("Scroll amount", 1, 50, func(c *config.Config) *int { return &c.ScrollAmount }),
			}
		}},
		{name: "Keys", rows: keyRows},
		{name: "Paths", rows: func(m *Model) []configRow {
			p := m.deps.Paths
			return []configRow{
				infoRow{"Config file", p.Config},
				infoRow{"Themes", p.Themes},
				infoRow{"Playlists", p.Playlists},
				infoRow{"Library cache", p.Cache},
				infoRow{"Session state", p.State},
			}
		}},
	}
}
