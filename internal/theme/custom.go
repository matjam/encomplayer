package theme

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// customFile is the JSON form of a custom theme. Extends names a built-in
// theme to start from, so a file only lists the colours it changes.
//
//	{"extends": "catppuccin-mocha", "colors": {"accent": "#ff00ff"}}
type customFile struct {
	Extends string            `json:"extends"`
	Colors  map[string]string `json:"colors"`
}

// Store finds themes: built-ins, plus custom themes in a directory where
// each <name>.json file defines the theme called name.
type Store struct {
	dir string
}

// NewStore returns a store that reads custom themes from dir.
func NewStore(dir string) *Store { return &Store{dir: dir} }

// Dir returns the custom theme directory.
func (s *Store) Dir() string { return s.dir }

// Names lists every available theme: built-ins first, then custom themes.
// A custom theme with a built-in's name replaces it and is listed once.
func (s *Store) Names() []string {
	names := BuiltinNames()
	for _, n := range s.customNames() {
		if !slices.Contains(names, n) {
			names = append(names, n)
		}
	}
	return names
}

// Load returns the named theme. A custom file wins over a built-in of the
// same name, so a built-in can be adjusted in place.
func (s *Store) Load(name string) (Theme, error) {
	data, err := os.ReadFile(filepath.Join(s.dir, name+".json"))
	switch {
	case err == nil:
		return parseCustom(name, data)
	case !errors.Is(err, fs.ErrNotExist):
		return Theme{}, fmt.Errorf("read theme %q: %w", name, err)
	}
	if t, ok := Builtin(name); ok {
		return t, nil
	}
	return Theme{}, fmt.Errorf("%w: %q", ErrUnknown, name)
}

func parseCustom(name string, data []byte) (Theme, error) {
	var f customFile
	if err := json.Unmarshal(data, &f); err != nil {
		return Theme{}, fmt.Errorf("parse theme %q: %w", name, err)
	}
	base := f.Extends
	if base == "" {
		base = Default
	}
	t, ok := Builtin(base)
	if !ok {
		return Theme{}, fmt.Errorf("theme %q extends %w: %q", name, ErrUnknown, base)
	}
	t = t.overlay(f.Colors)
	t.Name = name
	if err := t.Validate(); err != nil {
		return Theme{}, err
	}
	return t, nil
}

func (s *Store) customNames() []string {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		// No theme directory simply means no custom themes.
		return nil
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".json") {
			names = append(names, strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())))
		}
	}
	slices.Sort(names)
	return names
}
