package ui

import (
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/matjam/encomplayer/internal/audio"
	"github.com/matjam/encomplayer/internal/domain"
)

// playIndex starts the queue item at i.
func (m *Model) playIndex(i int) tea.Cmd {
	if !m.queue.SetCurrent(i) {
		return nil
	}
	m.saveState()
	return m.startCurrent()
}

// startCurrent plays the current queue item, or stops when there is none.
func (m *Model) startCurrent() tea.Cmd {
	t, _, ok := m.queue.Current()
	if !ok {
		m.deps.Player.Stop()
		m.playGen = 0
		return m.showArt(nil)
	}
	gen, err := m.deps.Player.Play(m.ctx, t.Path)
	if err != nil {
		m.playGen = 0
		m.status.errorf("CANNOT PLAY %s: %v", t.DisplayTitle(), err)
		return m.showArt(nil)
	}
	m.playGen = gen
	return m.showArt(&t)
}

// advance moves to the next track. auto is true when the previous track
// finished by itself.
func (m *Model) advance(auto bool) tea.Cmd {
	_, ok := m.queue.Advance(m.modes, auto, m.rng)
	m.syncQueue()
	if !ok {
		m.deps.Player.Stop()
		m.playGen = 0
		return m.showArt(nil)
	}
	return m.startCurrent()
}

func (m *Model) retreat() tea.Cmd {
	if _, ok := m.queue.Retreat(m.modes); !ok {
		return nil
	}
	m.saveState()
	return m.startCurrent()
}

// togglePause pauses or resumes. From a stop it plays the current track, or
// the one under the queue cursor.
func (m *Model) togglePause() tea.Cmd {
	if m.deps.Player.State() != audio.Stopped {
		m.deps.Player.TogglePause()
		return nil
	}
	if _, i, ok := m.queue.Current(); ok {
		return m.playIndex(i)
	}
	if m.queue.Len() > 0 {
		return m.playIndex(m.queueList.Cursor())
	}
	return nil
}

// enqueue appends tracks and reports how many were added.
func (m *Model) enqueue(tracks []domain.Track) {
	if len(tracks) == 0 {
		return
	}
	m.queue.Append(tracks...)
	m.syncQueue()
	m.status.infof("%d track(s) added to queue", len(tracks))
}

// enqueueAndPlay appends tracks and starts the first of them.
func (m *Model) enqueueAndPlay(tracks []domain.Track) tea.Cmd {
	if len(tracks) == 0 {
		return nil
	}
	first := m.queue.Len()
	m.enqueue(tracks)
	return m.playIndex(first)
}

// playShuffled replaces the queue with tracks in random order and starts
// playing.
func (m *Model) playShuffled(tracks []domain.Track) tea.Cmd {
	if len(tracks) == 0 {
		return nil
	}
	m.queue.Clear()
	m.queue.Append(tracks...)
	m.queue.Shuffle(m.rng)
	m.syncQueue()
	m.queueList.Top()
	m.status.infof("shuffling %d tracks", len(tracks))
	return m.playIndex(0)
}

// allTracks returns the whole library, or nothing before the first scan.
func (m *Model) allTracks() []domain.Track {
	if m.lib == nil {
		return nil
	}
	return m.lib.Tracks()
}

func (m *Model) addRandom(n int) {
	if m.lib == nil || len(m.lib.Tracks()) == 0 {
		return
	}
	all := m.lib.Tracks()
	picks := make([]domain.Track, 0, n)
	for range min(n, len(all)) {
		picks = append(picks, all[m.rng.IntN(len(all))])
	}
	m.enqueue(picks)
}

// resolveTrack returns library tags for path, reading the file directly when
// it lies outside the scanned folder.
func (m *Model) resolveTrack(path string) (domain.Track, bool) {
	if m.lib != nil {
		if t, ok := m.lib.Lookup(path); ok {
			return t, true
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		return domain.Track{}, false
	}
	return m.deps.Tags.ReadTrack(m.ctx, path, info), true
}

func (m *Model) resolveTracks(paths []string) []domain.Track {
	out := make([]domain.Track, 0, len(paths))
	for _, p := range paths {
		if t, ok := m.resolveTrack(p); ok {
			out = append(out, t)
		}
	}
	return out
}

// restoreQueue loads the queue saved by the previous session, once.
func (m *Model) restoreQueue() {
	if m.restored {
		return
	}
	m.restored = true

	// Files deleted since the last session drop out, so the saved current
	// index shifts down past each one.
	current := m.deps.State.Current
	var tracks []domain.Track
	for i, p := range m.deps.State.Queue {
		t, ok := m.resolveTrack(p)
		switch {
		case ok:
			tracks = append(tracks, t)
		case i == current:
			current = -1
		case i < current:
			current--
		}
	}
	m.queue = domain.NewQueue(tracks...)
	m.queue.SetCurrent(current)
	m.queueList.SetItems(m.queue.Items())
	if _, i, ok := m.queue.Current(); ok {
		m.queueList.SetCursor(i)
	}
}
