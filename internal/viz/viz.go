// Package viz defines EncomPlayer's visualisers: the audio each one
// receives per frame, the canvas it draws on, and the catalog that collects
// them.
//
// A visualiser is anything implementing Visualizer. The built-in ones live
// in package builtin, one file each, and register themselves from init:
//
//	func init() {
//		viz.Register(viz.Info{Name: "scope", Description: "..."}, func() viz.Visualizer { return &scope{} })
//	}
//
// Visualisers found at run time, such as script files, plug in through a
// Source instead. The catalog lists them beside the built-ins, and a source's
// visualiser replaces a built-in of the same name.
package viz

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"slices"
	"sync"

	"github.com/matjam/encomplayer/internal/registry"
)

// Visualizer draws audio onto a canvas, one frame at a time.
//
// A visualiser that holds resources, such as an interpreter, may also
// implement io.Closer. The player closes it when switching away.
type Visualizer interface {
	// Render draws one frame. The canvas is blank on entry and may change
	// size between calls. An error stops the visualiser; the player shows
	// the message and switches to the default.
	Render(c *Canvas, f *Frame) error
}

// Close releases v if it holds resources.
func Close(v Visualizer) error {
	if c, ok := v.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

// Info describes a visualiser for pickers and help.
type Info struct {
	Name        string
	Description string

	// Source says where the visualiser comes from, e.g. "builtin".
	Source string
}

// Factory builds a fresh visualiser. The catalog calls it on every
// selection, so a visualiser may keep state between frames and start clean
// next time.
type Factory func() Visualizer

// Source supplies visualisers discovered at run time.
type Source interface {
	// List describes the visualisers the source currently offers.
	List() []Info

	// New builds the named visualiser.
	New(name string) (Visualizer, error)

	// Reload looks for added, changed or removed visualisers.
	Reload() error
}

// Default is the name of the visualiser used when none is chosen, or when
// the chosen one fails.
const Default = "spectrum"

// ErrUnknown means no visualiser has the requested name.
var ErrUnknown = errors.New("unknown visualizer")

// Catalog collects visualisers from Register and from sources.
type Catalog struct {
	builtins *registry.Registry[builtin]

	mu      sync.RWMutex
	sources []Source
}

type builtin struct {
	info    Info
	factory Factory
}

// NewCatalog returns an empty catalog.
func NewCatalog() *Catalog {
	return &Catalog{builtins: registry.New[builtin]()}
}

// Builtins is the catalog that Register fills.
var Builtins = NewCatalog()

// Register adds a built-in visualiser to Builtins. Call it from init.
func Register(info Info, f Factory) { Builtins.Register(info, f) }

// Register adds a visualiser. It panics on a missing name or factory,
// because that is a programming error found at startup.
func (c *Catalog) Register(info Info, f Factory) {
	if info.Name == "" || f == nil {
		panic(fmt.Sprintf("viz: Register needs a name and a factory, got %q", info.Name))
	}
	if info.Source == "" {
		info.Source = "builtin"
	}
	c.builtins.Register(info.Name, builtin{info: info, factory: f})
}

// AddSource adds visualisers found at run time. Later sources take
// priority.
func (c *Catalog) AddSource(s Source) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sources = append(c.sources, s)
}

// List describes every visualiser, sorted by name.
func (c *Catalog) List() []Info {
	byName := map[string]Info{}
	for name, b := range c.builtins.All() {
		byName[name] = b.info
	}
	c.mu.RLock()
	for _, s := range c.sources {
		for _, info := range s.List() {
			byName[info.Name] = info
		}
	}
	c.mu.RUnlock()

	out := make([]Info, 0, len(byName))
	for _, info := range byName {
		out = append(out, info)
	}
	slices.SortFunc(out, func(a, b Info) int { return cmp.Compare(a.Name, b.Name) })
	return out
}

// Names lists every visualiser's name, sorted.
func (c *Catalog) Names() []string {
	infos := c.List()
	names := make([]string, len(infos))
	for i, info := range infos {
		names[i] = info.Name
	}
	return names
}

// New builds the named visualiser.
func (c *Catalog) New(name string) (Visualizer, Info, error) {
	c.mu.RLock()
	sources := slices.Clone(c.sources)
	c.mu.RUnlock()

	for _, s := range slices.Backward(sources) {
		for _, info := range s.List() {
			if info.Name == name {
				v, err := s.New(name)
				return v, info, err
			}
		}
	}
	if b, ok := c.builtins.Get(name); ok {
		return b.factory(), b.info, nil
	}
	return nil, Info{}, fmt.Errorf("%w: %q", ErrUnknown, name)
}

// Step returns the visualiser delta places after name in List order,
// wrapping around. An unknown name steps from the start.
func (c *Catalog) Step(name string, delta int) string {
	names := c.Names()
	if len(names) == 0 {
		return name
	}
	i := slices.Index(names, name)
	if i < 0 {
		i = 0
		if delta > 0 {
			delta--
		}
	}
	n := len(names)
	return names[((i+delta)%n+n)%n]
}

// Reload asks every source to look for changes.
func (c *Catalog) Reload() error {
	c.mu.RLock()
	sources := slices.Clone(c.sources)
	c.mu.RUnlock()

	var errs []error
	for _, s := range sources {
		errs = append(errs, s.Reload())
	}
	return errors.Join(errs...)
}
