package ui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/matjam/encomplayer/internal/audio"
	"github.com/matjam/encomplayer/internal/config"
	"github.com/matjam/encomplayer/internal/viz"
	// Built-in visualisers register themselves.
	_ "github.com/matjam/encomplayer/internal/viz/builtin"
)

const (
	defaultVizFPS        = 30
	minVizFPS, maxVizFPS = 5, 60

	// idleTick is the redraw rate while nothing plays.
	idleTick = 100 * time.Millisecond

	// fullFooterRows is the footer in full-screen mode: just the seek bar.
	fullFooterRows = 3
)

// vizState drives the visualiser in the SIGNAL strip, or full screen.
type vizState struct {
	catalog  *viz.Catalog
	current  viz.Visualizer
	info     viz.Info
	analyzer *viz.Analyzer
	canvas   *viz.Canvas
	lines    []string
	full     bool

	started, last time.Time
}

func newVizState(full bool) vizState {
	return vizState{catalog: viz.Builtins, analyzer: viz.NewAnalyzer(), full: full}
}

// selectViz switches to the named visualiser without saving the choice.
func (m *Model) selectViz(name string) error {
	v, info, err := m.viz.catalog.New(name)
	if err != nil {
		return fmt.Errorf("visualizer: %w", err)
	}
	m.closeViz()
	m.viz.current, m.viz.info = v, info
	m.viz.started = time.Time{}
	m.renderViz(time.Now())
	return nil
}

func (m *Model) closeViz() {
	if m.viz.current == nil {
		return
	}
	if err := viz.Close(m.viz.current); err != nil {
		m.status.errorf("VISUALIZER %s: %v", m.viz.info.Name, err)
	}
	m.viz.current = nil
}

// switchViz selects a visualiser and saves it as the configured one.
func (m *Model) switchViz(name string) tea.Cmd {
	if _, _, err := m.viz.catalog.New(name); err != nil {
		m.status.errorf("visualizer: %v", err)
		return nil
	}
	cmd := m.updateConfig(func(c *config.Config) { c.Visualizer = name })
	m.status.infof("VISUALIZER %s · %s", strings.ToUpper(m.viz.info.Name), m.viz.info.Description)
	return cmd
}

// stepViz moves delta places through the visualisers.
func (m *Model) stepViz(delta int) tea.Cmd {
	return m.switchViz(m.viz.catalog.Step(m.viz.info.Name, delta))
}

// setVizFull shows the visualiser full screen, or returns to the tabs.
func (m *Model) setVizFull(full bool) tea.Cmd {
	if m.viz.full == full {
		return nil
	}
	m.viz.full = full
	m.resolver.Reset()
	m.layout()
	m.renderViz(time.Now())
	m.saveState()
	return m.refreshArt()
}

// vizSize is the canvas size where the visualiser shows now.
func (m *Model) vizSize() (w, h int) {
	if m.viz.full {
		return m.width - 2, m.bodyHeight() - 2
	}
	return m.width - 2, m.spectrumRows()
}

// renderViz draws the next frame into m.viz.lines. A visualiser that fails
// or panics is replaced by the default, so it can never stop playback.
func (m *Model) renderViz(now time.Time) {
	v := &m.viz
	w, h := m.vizSize()
	if v.current == nil || w <= 0 || h <= 0 {
		v.lines = nil
		return
	}
	if v.canvas == nil {
		v.canvas = viz.NewCanvas(w, h, m.st.palette)
	}
	v.canvas.Palette = m.st.palette
	v.canvas.Resize(w, h)

	var dt time.Duration
	if !v.last.IsZero() {
		dt = now.Sub(v.last)
	}
	v.last = now
	if v.started.IsZero() {
		v.started = now
	}

	left, right, rate := m.deps.Player.Samples()
	f := v.analyzer.Analyze(left, right, rate, dt)
	f.Time, f.Delta = now.Sub(v.started), dt
	f.Playing = m.deps.Player.State() == audio.Playing
	if t, _, ok := m.queue.Current(); ok {
		f.Track = viz.Track{Title: t.DisplayTitle(), Artist: t.DisplayArtist(), Album: t.DisplayAlbum(), Position: m.position, Duration: m.length}
	}

	if err := safeRender(v.current, v.canvas, f); err != nil {
		m.status.errorf("VISUALIZER %s: %v", strings.ToUpper(v.info.Name), err)
		failed := v.info.Name
		m.closeViz()
		v.lines = nil
		// Falling back only when the default was not the one that failed
		// keeps this from looping.
		if failed != viz.Default {
			_ = m.selectViz(viz.Default)
		}
		return
	}
	v.lines = v.canvas.Lines()
}

func safeRender(v viz.Visualizer, c *viz.Canvas, f *viz.Frame) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("crashed: %v", r)
		}
	}()
	return v.Render(c, f)
}

// tickEvery is the redraw interval: the visualiser frame rate while music
// plays, and slower otherwise.
func (m *Model) tickEvery() time.Duration {
	if m.deps.Player.State() != audio.Playing {
		return idleTick
	}
	fps := m.deps.Config.VisualizerFPS
	if fps == 0 {
		fps = defaultVizFPS
	}
	return time.Second / time.Duration(max(minVizFPS, min(fps, maxVizFPS)))
}

func (m *Model) tick() tea.Cmd {
	return tea.Tick(m.tickEvery(), func(time.Time) tea.Msg { return tickMsg{} })
}

// vizView is the full-screen visualiser panel.
func (m *Model) vizView() []string {
	title := "visual · " + m.viz.info.Name
	return m.st.panel(title, m.viz.lines, m.width, m.bodyHeight(), true)
}

// runVizCommand handles ":viz" and the remote "viz": no argument toggles
// full screen, otherwise it names a visualiser or steps with next / prev.
func (m *Model) runVizCommand(arg string) tea.Cmd {
	switch arg {
	case "":
		return m.setVizFull(!m.viz.full)
	case "next":
		return m.stepViz(1)
	case "prev", "previous":
		return m.stepViz(-1)
	default:
		return m.switchViz(arg)
	}
}
