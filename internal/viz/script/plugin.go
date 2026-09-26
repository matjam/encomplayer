package script

import (
	_ "embed"
	"fmt"
	"time"

	"github.com/matjam/apogee/lua"
	"github.com/matjam/apogee/stdlib"

	"github.com/matjam/encomplayer/internal/viz"
)

// Limits on a script, so a broken one cannot hang or exhaust the player.
// Loading gets longer, for tables a script builds up front.
const (
	loadTime   = time.Second
	frameTime  = 250 * time.Millisecond
	loadMemory = 256 << 20
	frameBytes = 64 << 20
)

// prelude defines helpers every script can use.
//
//go:embed prelude.lua
var prelude string

// The plugin keeps these on its Lua stack for its lifetime. Go functions a
// script calls see their own stack, so they reach shared tables through
// the registry instead.
const (
	slotRender  = 1
	slotFrame   = 2
	slotCanvas  = 3
	slotPalette = 4

	bandsKey = "encomplayer.bands" // registry: band count to buffer
)

// plugin is one running script: a visualiser with its own Lua state.
type plugin struct {
	name string
	l    *lua.State

	// canvas and frame are set during Render, for the host functions.
	canvas *viz.Canvas
	frame  *viz.Frame

	// palette is the theme the palette table holds, once paletteSet.
	palette    viz.Palette
	paletteSet bool

	pixels              []int32
	pw, ph              int
	left, right, spectr []float64
	bands               map[int][]float64
	track               viz.Track
}

// load runs a script and returns it ready to render, with its description.
func load(name string, src []byte) (*plugin, string, error) {
	p := &plugin{
		name:   name,
		l:      lua.NewState(),
		left:   make([]float64, viz.Window),
		right:  make([]float64, viz.Window),
		spectr: make([]float64, viz.Window/2),
		bands:  map[int][]float64{},
		pw:     -1,
	}
	desc, err := p.start(src)
	if err != nil {
		p.Close()
		return nil, "", fmt.Errorf("visualizer %s: %w", name, err)
	}
	return p, desc, nil
}

func (p *plugin) start(src []byte) (string, error) {
	l := p.l
	openLibraries(l)
	p.register()

	l.PushNil() // slotRender, once the script has run
	p.newFrameTable()
	l.CreateTable(0, 8) // slotCanvas, filled by every Render
	l.CreateTable(0, 8) // slotPalette
	l.PushValue(slotPalette)
	l.SetGlobal("palette")
	l.NewTable()
	l.SetField(lua.RegistryIndex, bandsKey)

	if err := p.run(prelude, "prelude.lua", 0); err != nil {
		return "", err
	}
	if err := p.run(string(src), p.name+".lua", 1); err != nil {
		return "", err
	}
	if !l.IsTable(-1) {
		return "", fmt.Errorf("%s.lua must return a table, not %s", p.name, l.TypeName(-1))
	}
	l.Field(-1, "description")
	desc, _ := l.ToString(-1)
	l.Pop(1)
	if l.Field(-1, "render") != lua.TypeFunction {
		return "", fmt.Errorf("%s.lua: render must be a function", p.name)
	}
	l.Replace(slotRender)
	l.Pop(1) // the module table
	return desc, nil
}

// openLibraries opens the standard libraries a visualiser needs, leaving
// out files, processes and loading code.
func openLibraries(l *lua.State) {
	for _, lib := range []lua.RegistryFunction{
		{Name: "_G", Function: stdlib.OpenBase},
		{Name: "string", Function: stdlib.OpenString},
		{Name: "math", Function: stdlib.OpenMath},
		{Name: "table", Function: stdlib.OpenTable},
		{Name: "utf8", Function: stdlib.OpenUTF8},
	} {
		l.Require(lib.Name, lib.Function, true)
		l.Pop(1)
	}
	for _, name := range []string{"dofile", "loadfile", "load"} {
		l.PushNil()
		l.SetGlobal(name)
	}
	// The terminal belongs to the player, so print goes nowhere.
	l.Register("print", func(*lua.State) int { return 0 })
}

// run compiles and runs a chunk, leaving nresults results.
func (p *plugin) run(src, file string, nresults int) error {
	if err := p.l.LoadBuffer(src, "@"+file, "t"); err != nil {
		msg, _ := p.l.ToString(-1) // the parser leaves the message
		p.l.Pop(1)
		return fmt.Errorf("%w: %s", err, msg)
	}
	return p.call(0, nresults, loadTime, loadMemory)
}

// call runs the function below nargs arguments with a time and memory
// limit, leaving nresults results.
func (p *plugin) call(nargs, nresults int, limit time.Duration, memory int) error {
	timer := time.AfterFunc(limit, p.l.Interrupt)
	defer timer.Stop()
	p.l.SetAllocationLimit(memory)
	if err := p.l.ProtectedCall(nargs, nresults, 0); err != nil {
		p.l.Pop(1)
		return err
	}
	return nil
}

// Render draws one frame by calling the script's render(frame, canvas).
func (p *plugin) Render(c *viz.Canvas, f *viz.Frame) error {
	p.canvas, p.frame = c, f
	defer func() { p.canvas, p.frame = nil, nil }()

	p.updateCanvas(c)
	p.updatePalette(c.Palette)
	p.updateFrame(f)

	l := p.l
	l.PushValue(slotRender)
	l.PushValue(slotFrame)
	l.PushValue(slotCanvas)
	if err := p.call(2, 0, frameTime, frameBytes); err != nil {
		return err
	}
	c.Underlay(p.pixels)
	return nil
}

// Close releases the Lua state.
func (p *plugin) Close() error {
	p.l.Close()
	return nil
}
