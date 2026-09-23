// Package ui is EncomPlayer's terminal interface, built on bubbletea and
// styled after ENCOM OS-12.
package ui

import (
	"context"
	"math/rand/v2"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/matjam/encomplayer/internal/collection"
	"github.com/matjam/encomplayer/internal/config"
	"github.com/matjam/encomplayer/internal/domain"
	"github.com/matjam/encomplayer/internal/keymap"
	"github.com/matjam/encomplayer/internal/library"
)

const (
	tickInterval = 100 * time.Millisecond
	chordTimeout = time.Second
	spectrumBars = 48
)

// Layout rows outside the tab body.
const (
	headerRows = 4
	tabRows    = 1
	footerRows = 4
	statusRows = 1
)

// Model is the root bubbletea model.
type Model struct {
	ctx  context.Context
	deps Deps
	rng  *rand.Rand

	width, height int

	lib       *library.Library
	snapshot  *library.Snapshot
	queue     *domain.Queue[domain.Track]
	queueList *collection.List[domain.Track]
	modes     domain.Modes
	playGen   uint64
	restored  bool

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

	spectrum []float64
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
		spectrum:  make([]float64, spectrumBars),
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
	cmds := []tea.Cmd{tick(), bootTick(), m.waitEnded(), tea.RequestWindowSize}
	if m.deps.MusicDir != "" {
		cmds = append(cmds, m.loadCache(m.deps.MusicDir))
	} else {
		m.boot.scanDone = true
	}
	return tea.Batch(cmds...)
}

type tickMsg struct{}

func tick() tea.Cmd {
	return tea.Tick(tickInterval, func(time.Time) tea.Msg { return tickMsg{} })
}

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
		return m.refreshArt()

	case tickMsg:
		m.position, m.length = m.deps.Player.Progress()
		m.spectrum = smooth(m.spectrum, m.deps.Player.Spectrum(spectrumBars))
		m.status.expire()
		return tick()

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
		res := m.resolver.Flush(uint64(msg), m.tabs[m.active].contexts()...)
		if res.Matched {
			return m.dispatch(res.Action)
		}
		return nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return nil
}

// layout pushes the current body size into every tab.
func (m *Model) layout() {
	h := m.bodyHeight()
	for _, t := range m.tabs {
		t.resize(m, m.width, h)
	}
}

func (m *Model) bodyHeight() int {
	return max(3, m.height-headerRows-tabRows-footerRows-statusRows)
}

// syncQueue refreshes the queue list after the queue changes.
func (m *Model) syncQueue() {
	m.queueList.SetItems(m.queue.Items())
	m.saveState()
}

func (m *Model) saveState() {
	paths := collection.Map(m.queue.Items(), func(t domain.Track) string { return t.Path })
	st := config.State{
		Queue:   paths,
		Current: m.queue.CurrentIndex(),
		Modes:   m.modes,
		Volume:  m.deps.Player.Volume(),
		Tab:     m.tabs[m.active].title(),
	}
	if err := config.SaveState(m.deps.StatePath, st); err != nil {
		m.status.errorf("STATE NOT SAVED: %v", err)
	}
}

func (m *Model) switchTo(name string) {
	for i, t := range m.tabs {
		if t.title() == name {
			m.active = i
			return
		}
	}
}

// smooth lets bars fall gradually instead of flickering.
func smooth(prev, next []float64) []float64 {
	out := make([]float64, len(next))
	for i := range next {
		p := 0.0
		if i < len(prev) {
			p = prev[i]
		}
		out[i] = max(next[i], p*0.75)
	}
	return out
}
