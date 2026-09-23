package ui

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/matjam/encomplayer/internal/library"
)

const (
	localRescan   = time.Minute
	networkRescan = 10 * time.Minute
)

type scanState struct {
	running bool

	// background is true when a library is already on screen, so the scan
	// is a sync rather than the first load.
	background bool

	root     string
	progress library.Progress
	timerGen uint64
}

type cacheLoadedMsg struct {
	root string
	snap *library.Snapshot
	err  error
}

type scanProgressMsg struct {
	progress library.Progress
	ch       <-chan tea.Msg
}

type scanDoneMsg struct {
	root    string
	snap    *library.Snapshot
	changes library.Changes
	err     error
	saveErr error
}

type rescanTickMsg uint64

// loadCache reads the cached snapshot for root off the UI goroutine.
func (m *Model) loadCache(root string) tea.Cmd {
	root = expandHome(root)
	m.scan.root = root
	cache := m.deps.Cache
	return func() tea.Msg {
		snap, err := cache.Load(root)
		return cacheLoadedMsg{root: root, snap: snap, err: err}
	}
}

// receiveCache shows the cached library at once, then syncs it with disk.
func (m *Model) receiveCache(msg cacheLoadedMsg) tea.Cmd {
	if msg.err != nil {
		m.status.errorf("LIBRARY CACHE UNREADABLE: %v", msg.err)
	}
	if msg.snap != nil && len(msg.snap.Tracks) > 0 {
		m.snapshot = msg.snap
		m.applySnapshot(msg.snap)
		m.boot.scanDone = true
	}
	return m.startScan(msg.root, false)
}

// startScan syncs root with disk in the background. full rereads every
// file's tags.
func (m *Model) startScan(root string, full bool) tea.Cmd {
	if m.scan.running {
		m.status.infof("scan already running")
		return nil
	}
	root = expandHome(root)
	m.scan = scanState{
		running:    true,
		background: m.lib != nil && m.lib.Root() == root,
		root:       root,
		timerGen:   m.scan.timerGen,
	}

	scanner, cache, ctx, prev := m.deps.Scanner, m.deps.Cache, m.ctx, m.snapshot
	ch := make(chan tea.Msg, 8)
	go func() {
		defer close(ch)
		snap, changes, err := scanner.Scan(ctx, root, prev, library.Options{
			Full: full,
			Progress: func(p library.Progress) {
				select {
				case ch <- scanProgressMsg{progress: p, ch: ch}:
				default:
				}
			},

			// A failed checkpoint only costs resumability; the final
			// save below reports persistent write errors.
			Checkpoint: func(s *library.Snapshot) { _ = cache.Save(s) },
		})
		if err != nil {
			ch <- scanDoneMsg{root: root, err: err}
			return
		}
		ch <- scanDoneMsg{root: root, snap: snap, changes: changes, saveErr: cache.Save(snap)}
	}()
	return waitScan(ch)
}

func waitScan(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

func (m *Model) rescan(full bool) tea.Cmd {
	root := m.scan.root
	if m.lib != nil {
		root = m.lib.Root()
	}
	if root == "" {
		m.status.errorf("NO MUSIC FOLDER. USE :scan <dir>")
		return nil
	}
	return m.startScan(root, full)
}

func (m *Model) finishScan(msg scanDoneMsg) tea.Cmd {
	background := m.scan.background
	m.scan.running = false
	m.boot.scanDone = true

	if msg.err != nil {
		m.status.errorf("SCAN FAILED: %v", msg.err)
		m.restoreQueue()
		return m.scheduleRescan(msg.root)
	}

	m.snapshot = msg.snap
	if !background || msg.changes.Any() {
		m.applySnapshot(msg.snap)
	}

	c := msg.changes
	switch {
	case msg.saveErr != nil:
		m.status.errorf("LIBRARY CACHE NOT SAVED: %v", msg.saveErr)
	case background && c.Any():
		m.status.infof("sector sync: +%d new · %d updated · -%d removed", c.Added, c.Updated, c.Removed)
	case !background:
		m.status.infof("sector indexed: %d tracks", len(msg.snap.Tracks))
	}
	return m.scheduleRescan(msg.root)
}

// applySnapshot replaces the library and refreshes every view of it.
func (m *Model) applySnapshot(s *library.Snapshot) {
	m.lib = library.New(s.Root, s.Tracks)
	m.restoreQueue()
	m.setLibrary(m.lib)
}

// scheduleRescan arms the periodic sync. Network mounts get a longer
// interval because each check costs a round trip per folder.
func (m *Model) scheduleRescan(root string) tea.Cmd {
	every := time.Duration(m.deps.Config.RescanSeconds) * time.Second
	switch {
	case every < 0:
		return nil
	case every == 0 && library.IsNetworkMount(root):
		every = networkRescan
	case every == 0:
		every = localRescan
	}
	m.scan.timerGen++
	gen := m.scan.timerGen
	return tea.Tick(every, func(time.Time) tea.Msg { return rescanTickMsg(gen) })
}

func (m *Model) onRescanTick(gen rescanTickMsg) tea.Cmd {
	if uint64(gen) != m.scan.timerGen || m.scan.running {
		return nil
	}
	return m.rescan(false)
}

// setLibrary points every browser tab at a new library snapshot. Browsers
// re-walk their previous path, so a sync leaves the cursor in place.
func (m *Model) setLibrary(lib *library.Library) {
	providers := map[string]entryProvider{
		"Directories":   directoriesProvider(lib),
		"Artists":       artistsProvider(lib),
		"Album Artists": albumArtistsProvider(lib),
		"Albums":        albumsProvider(lib),
		"Genres":        genresProvider(lib),
	}
	for _, t := range m.tabs {
		switch t := t.(type) {
		case *browserTab:
			if p, ok := providers[t.name]; ok {
				t.browser.SetProvider(p)
			} else {
				t.reload()
			}
		case *searchTab:
			t.lib = lib
			t.run()
		}
	}
	m.layout()
}

func emptyProvider() entryProvider {
	return entryProvider{root: func() []entry { return nil }}
}

func expandHome(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(home, strings.TrimPrefix(path, "~"))
		}
	}
	if abs, err := filepath.Abs(path); err == nil {
		return abs
	}
	return path
}
