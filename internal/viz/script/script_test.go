package script_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/matjam/apogee/lua"

	"github.com/matjam/encomplayer/internal/viz"
	"github.com/matjam/encomplayer/internal/viz/script"
)

// source loads scripts given as file name to text.
func source(t *testing.T, files map[string]string) *script.Source {
	t.Helper()
	fsys := fstest.MapFS{}
	for name, text := range files {
		fsys[name] = &fstest.MapFile{Data: []byte(text)}
	}
	return script.NewSource(fsys, "user")
}

// render draws one frame of the named script on a w × h canvas.
func render(t *testing.T, s *script.Source, name string, w, h int) ([]string, error) {
	t.Helper()
	v, err := s.New(name)
	if err != nil {
		t.Fatal(err)
	}
	defer viz.Close(v)
	c := viz.NewCanvas(w, h, testPalette)
	f := viz.NewAnalyzer().Analyze(nil, nil, 44100, 33*time.Millisecond)
	f.Delta = 33 * time.Millisecond
	err = v.Render(c, f)
	lines := c.Lines()
	for i, l := range lines {
		lines[i] = ansi.Strip(l)
	}
	return lines, err
}

func TestReload(t *testing.T) {
	s := source(t, map[string]string{
		"good.lua":     `return { description = "fine", render = function() end }`,
		"syntax.lua":   `return {`,
		"notable.lua":  `return 42`,
		"norender.lua": `return { description = "no render" }`,
		"readme.txt":   `not a script`,
	})
	err := s.Reload()
	for _, want := range []string{"syntax.lua", "must return a table", "render must be a function"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Reload error %v, want it to mention %q", err, want)
		}
	}
	want := []viz.Info{{Name: "good", Description: "fine", Source: "user"}}
	if got := s.List(); !slices.Equal(got, want) {
		t.Errorf("List = %+v, want %+v", got, want)
	}
}

// A user's script replaces the built-in of the same name.
func TestUserScriptReplacesBuiltin(t *testing.T) {
	c := viz.NewCatalog()
	c.AddSource(script.Builtin())
	c.AddSource(source(t, map[string]string{
		"spectrum.lua": `return { description = "mine", render = function() end }`,
	}))
	if err := c.Reload(); err != nil {
		t.Fatal(err)
	}
	if _, info, err := c.New("spectrum"); err != nil || info.Description != "mine" || info.Source != "user" {
		t.Errorf("New(spectrum) = %+v, %v", info, err)
	}
}

func TestDrawing(t *testing.T) {
	s := source(t, map[string]string{"draw.lua": `
		return {
			description = "draws",
			render = function(f, c)
				text(0, 0, "hi", palette.text)
				set(2, 0, utf8.codepoint("#"), palette.accent)
				cell(3, 0, "@", palette.text, palette.dim)
				-- Pixel (4, 0) and (4, 1) share cell (4, 0).
				c.pixels[4] = rgb(255, 0, 0)
				c.pixels[c.pw + 4] = hex("#00ff00")
				pixel(5, 1, palette.bright)
				dot(12, 4, palette.text)
				line(0, 4, 3, 4, lerp(palette.text, palette.bright, 0.5))
				assert(c.w == 8 and c.h == 2 and c.pw == 8 and c.ph == 4 and c.dw == 16 and c.dh == 8)
			end,
		}`})
	lines, err := render(t, s, "draw", 8, 2)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"hi#@▀▄  ", "⠉⠉    ⠁ "}
	if !slices.Equal(lines, want) {
		t.Errorf("drew %q, want %q", lines, want)
	}
}

// Scripts get no files, processes or code loading, and print is silent.
func TestSandbox(t *testing.T) {
	s := source(t, map[string]string{"probe.lua": `
		assert(io == nil and os == nil and require == nil and package == nil and debug == nil)
		assert(dofile == nil and loadfile == nil and load == nil)
		print("nothing")
		return { description = "sandboxed", render = function() end }`})
	if err := s.Reload(); err != nil {
		t.Fatal(err)
	}
}

func TestLimits(t *testing.T) {
	tests := []struct {
		name, render string
		want         error
		wantText     string
	}{
		{"runaway", `while true do end`, nil, "interrupted"},
		// One allocation past the limit, so the time limit cannot win the
		// race on a slow machine.
		{"greedy", `local s = ("x"):rep(128 << 20)`, lua.ErrMemory, ""},
		{"huge buffer", `f64(1e9)`, nil, "buffer size out of range"},
		{"error", `error("out of cheese")`, nil, "err.lua:1: out of cheese"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := source(t, map[string]string{"err.lua": "return { render = function() " + tc.render + " end }"})
			start := time.Now()
			_, err := render(t, s, "err", 10, 4)
			if err == nil {
				t.Fatal("no error")
			}
			if tc.want != nil && !errors.Is(err, tc.want) {
				t.Errorf("error %v, want %v", err, tc.want)
			}
			if !strings.Contains(err.Error(), tc.wantText) {
				t.Errorf("error %q, want it to mention %q", err, tc.wantText)
			}
			if d := time.Since(start); d > 2*time.Second {
				t.Errorf("took %v to stop", d)
			}
		})
	}
}

// A script can keep bands(n)'s buffer: the same n is the same buffer,
// refilled each frame.
func TestBandsBuffer(t *testing.T) {
	s := source(t, map[string]string{"bands.lua": `
		local kept
		return { render = function()
			local b = bands(16)
			assert(#b == 16)
			if kept then assert(rawequal(kept, b)) end
			kept = b
		end }`})
	v, err := s.New("bands")
	if err != nil {
		t.Fatal(err)
	}
	defer viz.Close(v)
	c := viz.NewCanvas(10, 4, testPalette)
	f := viz.NewAnalyzer().Analyze(nil, nil, 44100, 0)
	for range 3 {
		if err := v.Render(c, f); err != nil {
			t.Fatal(err)
		}
	}
}
