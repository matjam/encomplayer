package ui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/matjam/encomplayer/internal/config"
	"github.com/matjam/encomplayer/internal/tea"
	"github.com/matjam/encomplayer/internal/viz"
	"github.com/matjam/encomplayer/internal/viz/script"
)

func TestVisualizerStripAndSwitching(t *testing.T) {
	m, _ := newTestModel(t)
	m.enqueue(testTracks("/music"))
	m.playIndex(0)
	m.renderViz(time.Now().Add(time.Second))

	if got := screen(t, m); !strings.Contains(got, "SIGNAL · SPECTRUM") {
		t.Fatalf("strip title missing:\n%s", got)
	}
	if len(m.viz.lines) != m.spectrumRows() {
		t.Errorf("strip has %d visualizer rows, want %d", len(m.viz.lines), m.spectrumRows())
	}

	names := m.viz.catalog.Names()
	next := m.viz.catalog.Step("spectrum", 1)
	press(m, "]")
	if m.viz.info.Name != next || m.deps.Config.Visualizer != next {
		t.Fatalf("] selected %q, config %q; want %q", m.viz.info.Name, m.deps.Config.Visualizer, next)
	}
	saved, err := config.Load(m.deps.Paths.Config)
	if err != nil || saved.Visualizer != next {
		t.Errorf("config file has %q, %v", saved.Visualizer, err)
	}
	press(m, "[")
	if m.viz.info.Name != "spectrum" {
		t.Errorf("[ went to %q, want spectrum", m.viz.info.Name)
	}
	if len(names) < 20 {
		t.Errorf("only %d visualizers available", len(names))
	}
}

func TestVisualizerFullScreen(t *testing.T) {
	m, fp := newTestModel(t)
	m.enqueue(testTracks("/music"))
	m.playIndex(0)
	cursor := m.queueList.Cursor()

	press(m, "ov")
	got := screen(t, m)
	if !m.viz.full || !strings.Contains(got, "VISUAL · SPECTRUM") || strings.Contains(got, "1 QUEUE") {
		t.Fatalf("ov did not show the visualizer full screen:\n%s", got)
	}
	if w, h := m.vizSize(); w != m.width-2 || h != m.height-headerRows-fullFooterRows-statusRows-2 {
		t.Errorf("full-screen canvas %dx%d", w, h)
	}

	// Tab keys do nothing to the hidden tab; global keys still work.
	press(m, "j")
	if m.queueList.Cursor() != cursor {
		t.Error("j moved the hidden queue cursor")
	}
	press(m, ">")
	if len(fp.played) != 2 {
		t.Errorf("> in full screen played %d tracks, want 2", len(fp.played))
	}

	// The seek bar still answers clicks in its new place.
	m.position, m.length = 0, 100*time.Second
	x0, w := m.progressBarSpan()
	click(m, x0+w/2, m.progressRow())
	if len(fp.seeks) != 1 {
		t.Errorf("seek bar click in full screen sought %v", fp.seeks)
	}

	press(m, "<Esc>")
	if m.viz.full {
		t.Fatal("Esc did not close the full-screen visualizer")
	}
	screen(t, m)

	press(m, "ov")
	press(m, "3")
	if m.viz.full || m.tabs[m.active].title() != "Artists" {
		t.Errorf("3 in full screen: full %v, tab %q", m.viz.full, m.tabs[m.active].title())
	}
}

func TestVisualizerFullScreenIsRestored(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	m, _ := newTestModelAt(t, statePath)
	press(m, "ov")
	m.saveState()

	again, _ := newTestModelAt(t, statePath)
	if !again.viz.full {
		t.Error("full-screen visualizer not restored")
	}
	screen(t, again)
}

func TestVisualizerMouse(t *testing.T) {
	m, _ := newTestModel(t)
	stripRow := m.footerTop() + 1
	click(m, 10, stripRow)
	click(m, 10, stripRow)
	if !m.viz.full {
		t.Fatal("double click on the strip did not open full screen")
	}

	wheel(m, m.bodyTop()+3, tea.MouseWheelDown)
	if m.viz.info.Name != m.viz.catalog.Step("spectrum", 1) {
		t.Errorf("wheel selected %q", m.viz.info.Name)
	}
	click(m, 10, m.bodyTop()+3)
	click(m, 10, m.bodyTop()+3)
	if m.viz.full {
		t.Error("double click did not close full screen")
	}
}

// Every visualizer draws at exactly the space it is given, in the strip and
// full screen.
func TestEveryVisualizerFitsTheScreen(t *testing.T) {
	m, _ := newTestModel(t)
	m.enqueue(testTracks("/music"))
	m.playIndex(0)
	for _, name := range m.viz.catalog.Names() {
		press(m, ":")
		typeText(m, "viz "+name)
		press(m, "<CR>")
		if m.viz.info.Name != name {
			t.Fatalf(":viz %s selected %q", name, m.viz.info.Name)
		}
		for range 3 {
			m.Update(tickMsg{})
		}
		screen(t, m)
		press(m, "ov")
		m.Update(tickMsg{})
		if got := screen(t, m); !strings.Contains(got, "VISUAL · "+strings.ToUpper(name)) {
			t.Errorf("%s: full screen title missing", name)
		}
		press(m, "ov")
	}
}

// A user's script shows beside the built-ins, and reload restarts it with
// the edits made since.
func TestUserVisualizerReload(t *testing.T) {
	dir := t.TempDir()
	write := func(desc string) {
		src := `return { description = "` + desc + `", render = function(f, c) text(0, 0, "` + desc + `", palette.text) end }`
		if err := os.WriteFile(filepath.Join(dir, "mine.lua"), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("first")
	deps := testDeps(filepath.Join(t.TempDir(), "state.json"), Startup{})
	deps.Visualizers = script.Catalog(dir)
	deps.Config.Visualizer = "mine"
	if err := config.Save(deps.Paths.Config, deps.Config); err != nil {
		t.Fatal(err)
	}
	m, _ := startTestModel(deps)
	if m.viz.info.Name != "mine" || m.viz.info.Description != "first" || m.status.isError {
		t.Fatalf("showing %+v, status %q", m.viz.info, m.status.text)
	}

	write("second")
	m.Update(ReloadMsg{})
	m.renderViz(time.Now())
	if m.viz.info.Description != "second" || !strings.Contains(strings.Join(m.viz.lines, ""), "second") {
		t.Errorf("after reload: %+v, lines %q", m.viz.info, m.viz.lines)
	}

	// A script broken by an edit falls back to the default, with its error.
	if err := os.WriteFile(filepath.Join(dir, "mine.lua"), []byte("return {"), 0o644); err != nil {
		t.Fatal(err)
	}
	m.Update(ReloadMsg{})
	if m.viz.info.Name != viz.Default || !m.status.isError || !strings.Contains(m.status.text, "mine") {
		t.Errorf("broken script: showing %q, status %q", m.viz.info.Name, m.status.text)
	}
}

// Switching builds the new visualiser once and closes the old one, so a
// script's setup runs once and no Lua state is left open.
func TestSwitchBuildsVisualizerOnce(t *testing.T) {
	m, _ := newTestModel(t)
	built, closed := 0, 0
	c := script.Catalog("")
	if err := c.Reload(); err != nil {
		t.Fatal(err)
	}
	c.Register(viz.Info{Name: "counted"}, func() viz.Visualizer {
		built++
		return counted{&closed}
	})
	m.viz.catalog = c

	m.runVizCommand("counted")
	if built != 1 || m.viz.info.Name != "counted" {
		t.Fatalf("built %d times, showing %q", built, m.viz.info.Name)
	}
	m.runVizCommand("spectrum")
	if closed != 1 {
		t.Errorf("closed %d times after switching away", closed)
	}
}

type counted struct{ closed *int }

func (counted) Render(*viz.Canvas, *viz.Frame) error { return nil }
func (c counted) Close() error                       { *c.closed++; return nil }

type failing struct{ panics bool }

func (f failing) Render(*viz.Canvas, *viz.Frame) error {
	if f.panics {
		panic("boom")
	}
	return errors.New("out of cheese")
}

func TestBrokenVisualizerFallsBack(t *testing.T) {
	for _, panics := range []bool{false, true} {
		m, _ := newTestModel(t)
		c := script.Catalog("")
		if err := c.Reload(); err != nil {
			t.Fatal(err)
		}
		c.Register(viz.Info{Name: "broken"}, func() viz.Visualizer { return failing{panics} })
		m.viz.catalog = c

		if err := m.selectViz("broken"); err != nil {
			t.Fatal(err)
		}
		m.Update(tickMsg{})
		if m.viz.info.Name != viz.Default || !m.status.isError || !strings.Contains(m.status.text, "BROKEN") {
			t.Errorf("panics=%v: now showing %q, status %q", panics, m.viz.info.Name, m.status.text)
		}
		screen(t, m)
	}
}

func TestRemoteViz(t *testing.T) {
	m, _ := newTestModel(t)
	r := send(m, "viz", "fire")
	if !r.OK || r.Status.Visualizer != "fire" || m.deps.Config.Visualizer != "fire" {
		t.Fatalf("viz fire = %+v", r)
	}
	if r := send(m, "viz"); !r.OK || !r.Status.VisualizerFull {
		t.Errorf("viz alone = %+v, want full screen", r)
	}
	if r := send(m, "viz", "next"); !r.OK || r.Status.Visualizer == "fire" {
		t.Errorf("viz next = %+v", r)
	}
	if bad := send(m, "viz", "lava-lamp"); bad.OK || !strings.Contains(bad.Error, "lava-lamp") {
		t.Errorf("viz lava-lamp = %+v", bad)
	}
}

func TestTickRate(t *testing.T) {
	m, fp := newTestModel(t)
	if got := m.tickEvery(); got != idleTick {
		t.Errorf("stopped tick = %v", got)
	}
	m.enqueue(testTracks("/music"))
	m.playIndex(0)
	if got := m.tickEvery(); got != time.Second/defaultVizFPS {
		t.Errorf("playing tick = %v", got)
	}
	m.deps.Config.VisualizerFPS = 500
	if got := m.tickEvery(); got != time.Second/maxVizFPS {
		t.Errorf("clamped tick = %v", got)
	}
	fp.TogglePause()
	if got := m.tickEvery(); got != idleTick {
		t.Errorf("paused tick = %v", got)
	}
}
