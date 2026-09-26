// Package script runs visualisers written in Lua, on the apogee VM.
//
// A visualiser is one file, name.lua, returning a table with a description
// and a render function that the player calls once a frame:
//
//	local t = 0
//	return {
//		description = "A dot that circles with the bass",
//		render = function(f, c)
//			t = t + f.dt * (1 + f.bass)
//			pixel(c.pw / 2 + math.cos(t) * 10, c.ph / 2 + math.sin(t) * 10, palette.accent)
//		end,
//	}
//
// The built-in visualisers are such files, embedded in the binary; the user
// adds or replaces them with files in a folder. The README's "Writing a
// visualizer" lists everything a script can use: the functions in api.go,
// the tables frame.go fills, and the helpers in prelude.lua.
package script

import (
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"slices"
	"strings"
	"sync"

	"github.com/matjam/encomplayer/internal/viz"
)

//go:embed builtin/*.lua
var builtinFiles embed.FS

// Builtin returns the source of the built-in visualisers.
func Builtin() *Source {
	sub, err := fs.Sub(builtinFiles, "builtin")
	if err != nil {
		panic(err) // the embedded folder always exists
	}
	return NewSource(sub, "builtin")
}

// Catalog returns a catalog of the built-in visualisers and those in dir,
// which replace built-ins of the same name. dir need not exist. Call
// Reload on the catalog to load them.
func Catalog(dir string) *viz.Catalog {
	c := viz.NewCatalog()
	c.AddSource(Builtin())
	if dir != "" {
		c.AddSource(NewSource(os.DirFS(dir), "user"))
	}
	return c
}

// Source is a folder of visualiser scripts.
type Source struct {
	fsys   fs.FS
	origin string

	mu    sync.RWMutex
	infos []viz.Info
}

// NewSource returns the visualisers in the top level of fsys. origin fills
// each one's Info.Source. It lists nothing until Reload.
func NewSource(fsys fs.FS, origin string) *Source {
	return &Source{fsys: fsys, origin: origin}
}

// List describes the scripts that loaded at the last Reload.
func (s *Source) List() []viz.Info {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return slices.Clone(s.infos)
}

// New loads the named script afresh, so edits show without a reload.
func (s *Source) New(name string) (viz.Visualizer, error) {
	src, err := fs.ReadFile(s.fsys, name+".lua")
	if err != nil {
		return nil, fmt.Errorf("visualizer %s: %w", name, err)
	}
	p, _, err := load(name, src)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// Reload loads every script to read its description. A script that fails
// is left out, and its error returned with the others'.
func (s *Source) Reload() error {
	files, err := fs.Glob(s.fsys, "*.lua")
	if err != nil {
		return fmt.Errorf("list %s visualizers: %w", s.origin, err)
	}
	var infos []viz.Info
	var errs []error
	for _, file := range files {
		name := strings.TrimSuffix(path.Base(file), ".lua")
		src, err := fs.ReadFile(s.fsys, file)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		p, desc, err := load(name, src)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		p.Close()
		infos = append(infos, viz.Info{Name: name, Description: desc, Source: s.origin})
	}
	s.mu.Lock()
	s.infos = infos
	s.mu.Unlock()
	return errors.Join(errs...)
}
