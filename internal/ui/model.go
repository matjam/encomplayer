// Package ui is EncomPlayer's terminal interface, built on bubbletea and
// styled after ENCOM OS-12.
package ui

import (
	"context"
	"image"
	"math/rand/v2"
	"time"

	uv "github.com/charmbracelet/ultraviolet"

	"github.com/matjam/encomplayer/internal/audio"
	"github.com/matjam/encomplayer/internal/collection"
	"github.com/matjam/encomplayer/internal/config"
	"github.com/matjam/encomplayer/internal/domain"
	"github.com/matjam/encomplayer/internal/keymap"
	"github.com/matjam/encomplayer/internal/library"
	"github.com/matjam/encomplayer/internal/tea"
	"github.com/matjam/encomplayer/internal/viz"
)

const (
	chordTimeout = time.Second

	// positionSaveEvery bounds how much listening a crash can lose from
	// the saved resume position.
	positionSaveEvery = 5 * time.Second
)

// Fixed layout rows outside the tab body. The signal strip's height is
// user-sized; see Model.footerRows.
const (
	headerRows = 4
	tabRows    = 1
	statusRows = 1
)

// Model is the root bubbletea model.
type Model struct {
	ctx  context.Context
	deps Deps
	rng  *rand.Rand

	width, height int
	cell          image.Point // pixels per cell; zero until the terminal reports it

	lib       *library.Library
	snapshot  *library.Snapshot
	queue     *domain.Queue[domain.Track]
	queueList *collection.List[domain.Track]
	modes     domain.Modes
	playGen   uint64
	restored  bool

	// resumeAt is where the current track stopped in the last session.
	// Resuming from a stop starts there; playing any track clears it.
	resumeAt  time.Duration
	lastSaved time.Time

	// pendingShuffle is set by --shuffle and consumed once the library is
	// first available.
	pendingShuffle bool

	tabs   []tab
	active int

	resolver *keymap.Resolver
	prompt   *prompt
	lastFind string
	modal    modal
	status   status
	boot     bootState
	scan     scanState
	art      artState
	mouse    mouseState
	sizes    config.Layout
	st       *styles
	viz      vizState

	position time.Duration
	length   time.Duration
}

// New builds the model. ctx bounds background work such as scans.
func New(ctx context.Context, deps Deps) *Model {
	m := &Model{
		ctx:       ctx,
		deps:      deps,
		rng:       rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0x454E434F4D)),
		queue:     domain.NewQueue[domain.Track](),
		queueList: collection.NewList[domain.Track](nil),
		modes:     deps.State.Modes,
		resolver:  keymap.NewResolver(deps.Keymap),
		viz:       newVizState(deps.State.VisualizerFull),
		sizes:     deps.State.Layout,
	}
	if m.sizes == (config.Layout{}) {
		m.sizes = config.DefaultLayout()
	}
	startTheme := deps.Config.Theme
	if deps.Startup.Theme != "" {
		startTheme = deps.Startup.Theme
	}
	m.pendingShuffle = deps.Startup.Shuffle
	if err := m.applyTheme(startTheme); err != nil {
		m.status.errorf("%v", err)
	}
	if err := m.selectViz(deps.Config.Visualizer); err != nil {
		m.status.errorf("%v", err)
		_ = m.selectViz(viz.Default)
	}
	deps.Player.SetVolume(deps.State.Volume)

	m.tabs = []tab{
		&queueTab{list: m.queueList},
		newBrowserTab("Directories", "folder tree", emptyProvider()),
		newBrowserTab("Artists", "artist › album › track", emptyProvider()),
		newBrowserTab("Album Artists", "album artist › album › track", emptyProvider()),
		newBrowserTab("Albums", "album › track", emptyProvider()),
		newPlaylistsTab(m),
		newSearchTab(),
		newBrowserTab("Genres", "genre › artist › album › track", emptyProvider()),
	}
	m.switchTo(deps.State.Tab)
	return m
}

// Init starts the boot sequence, the first scan and the background loops.
func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.tick(), bootTick(), m.waitEnded(), tea.RequestWindowSize, requestCellSize}
	if m.deps.MusicDir != "" {
		cmds = append(cmds, m.loadCache(m.deps.MusicDir))
	} else {
		m.boot.scanDone = true
	}
	return tea.Batch(cmds...)
}

type tickMsg struct{}

type endedMsg uint64

func (m *Model) waitEnded() tea.Cmd {
	ch := m.deps.Player.Ended()
	return func() tea.Msg { return endedMsg(<-ch) }
}

type keyTimeoutMsg uint64

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	before := m.artSignature()
	cmd := m.update(msg)
	if m.artSignature() != before {
		cmd = tea.Batch(cmd, m.scheduleArtPlacement())
	}
	return m, cmd
}

func (m *Model) update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		m.renderViz(time.Now())
		return tea.Batch(m.refreshArt(), requestCellSize)

	case uv.CellSizeEvent:
		return m.setCellSize(image.Pt(msg.Width, msg.Height))

	case tickMsg:
		m.position, m.length = m.deps.Player.Progress()
		if t, _, ok := m.queue.Current(); ok && m.resumeAt > 0 && m.deps.Player.State() == audio.Stopped {
			m.position, m.length = m.resumeAt, t.Duration
		}
		if m.deps.Player.State() == audio.Playing && time.Since(m.lastSaved) >= positionSaveEvery {
			m.saveState()
		}
		m.renderViz(time.Now())
		m.status.expire()
		return m.tick()

	case bootTickMsg:
		return m.boot.advance()

	case endedMsg:
		var next tea.Cmd
		if uint64(msg) == m.playGen {
			next = m.advance(true)
		}
		return tea.Batch(next, m.waitEnded())

	case scanProgressMsg:
		m.scan.progress = msg.progress
		return waitScan(msg.ch)

	case cacheLoadedMsg:
		return m.receiveCache(msg)

	case scanDoneMsg:
		return m.finishScan(msg)

	case rescanTickMsg:
		return m.onRescanTick(msg)

	case enqueueMsg:
		if msg.err != nil {
			m.status.errorf("%v", msg.err)
		}
		m.enqueue(msg.tracks)
		return nil

	case artMsg:
		return m.receiveArt(msg)

	case artPlaceMsg:
		return m.placeArt(msg)

	case keyTimeoutMsg:
		res := m.resolver.Flush(uint64(msg), m.contexts()...)
		if res.Matched {
			return m.dispatch(res.Action)
		}
		return nil

	case ReloadMsg:
		return m.reload()

	case RemoteMsg:
		return m.handleRemote(msg)

	case NoticeMsg:
		m.status.errorf("%s", msg.Text)
		return nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case tea.MouseMsg:
		return m.handleMouse(msg)
	}
	return nil
}

// layout pushes the current body size into every tab.
func (m *Model) layout() {
	m.clampLayout()
	h := m.tabBodyHeight()
	for _, t := range m.tabs {
		t.resize(m, m.width, h)
	}
}

// bodyHeight is the height of whatever fills the middle of the screen: the
// active tab, or the visualiser when it is full screen.
func (m *Model) bodyHeight() int {
	return max(3, m.height-m.bodyTop()-m.footerRows()-statusRows)
}

// tabBodyHeight is the tab body's height in the normal layout. Tabs and
// album art keep this size while the visualiser fills the screen, so they
// are ready when it closes.
func (m *Model) tabBodyHeight() int {
	return max(3, m.height-tabBodyTop-m.sizes.FooterRows-statusRows)
}

// syncQueue refreshes the queue list after the queue changes.
func (m *Model) syncQueue() {
	m.queueList.SetItems(m.queue.Items())
	m.preloadNext()
	m.saveState()
}

func (m *Model) saveState() {
	paths := collection.Map(m.queue.Items(), func(t domain.Track) string { return t.Path })
	st := config.State{
		Queue:           paths,
		Current:         m.queue.CurrentIndex(),
		Modes:           m.modes,
		Volume:          m.deps.Player.Volume(),
		Tab:             m.tabs[m.active].title(),
		Layout:          m.sizes,
		PositionSeconds: m.resumePosition().Seconds(),
		VisualizerFull:  m.viz.full,
	}
	m.lastSaved = time.Now()
	if err := config.SaveState(m.deps.Paths.State, st); err != nil {
		m.status.errorf("STATE NOT SAVED: %v", err)
	}
}

// resumePosition is where the next launch should resume the current track:
// the live position while a track is loaded, otherwise the position carried
// over from the last session.
func (m *Model) resumePosition() time.Duration {
	if m.deps.Player.State() != audio.Stopped {
		pos, _ := m.deps.Player.Progress()
		return pos
	}
	return m.resumeAt
}

func (m *Model) switchTo(name string) {
	for i, t := range m.tabs {
		if t.title() == name {
			m.active = i
			return
		}
	}
}
