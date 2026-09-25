package ui

import (
	"os"
	"time"

	"github.com/matjam/encomplayer/internal/audio"
	"github.com/matjam/encomplayer/internal/domain"
	"github.com/matjam/encomplayer/internal/tea"
)

// playIndex starts the queue item at i.
func (m *Model) playIndex(i int) tea.Cmd {
	if !m.queue.SetCurrent(i) {
		return nil
	}
	m.saveState()
	return m.startCurrent()
}

// startCurrent plays the current queue item from the beginning, or stops
// when there is none.
func (m *Model) startCurrent() tea.Cmd { return m.startCurrentAt(0) }

// startCurrentAt plays the current queue item from offset start. Any saved
// resume position is used up either way.
func (m *Model) startCurrentAt(start time.Duration) tea.Cmd {
	m.resumeAt = 0
	t, _, ok := m.queue.Current()
	if !ok {
		m.deps.Player.Stop()
		m.playGen = 0
		return m.showArt(nil)
	}
	gen, err := m.deps.Player.Play(m.ctx, t.Path, start)
	if err != nil {
		m.playGen = 0
		m.status.errorf("CANNOT PLAY %s: %v", t.DisplayTitle(), err)
		return m.showArt(nil)
	}
	m.playGen = gen
	m.preloadNext()

	// Save now so the stored position belongs to the new track, not the
	// one the player was on a moment ago.
	m.saveState()
	return m.showArt(&t)
}

// preloadNext asks the player to load the track that will play when the
// current one ends, so the change is instant and never waits on the disk.
func (m *Model) preloadNext() {
	if m.playGen == 0 {
		return
	}
	if next, ok := m.queue.PeekNext(m.modes, m.rng); ok {
		m.deps.Player.Preload(next.Path)
	}
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

// currentDeleted continues after the current track was deleted from the
// queue. The queue has already made the following track current. Playback
// moves on to it, as next-track would; a paused player stops there so p
// starts it; a stopped one just keeps the new current track.
func (m *Model) currentDeleted() tea.Cmd {
	m.resumeAt = 0
	switch m.deps.Player.State() {
	case audio.Paused:
		m.deps.Player.Stop()
		m.playGen = 0
		m.saveState()
		return m.showArt(nil)
	case audio.Stopped:
		return nil
	}

	_, _, hasNext := m.queue.Current()
	switch {
	case m.modes.Random && m.queue.Len() > 0:
		return m.advance(false)
	case hasNext:
		return m.startCurrent()
	case m.modes.Repeat && m.queue.Len() > 0:
		return m.playIndex(0)
	default:
		m.deps.Player.Stop()
		m.playGen = 0
		m.saveState()
		return m.showArt(nil)
	}
}

func (m *Model) retreat() tea.Cmd {
	if _, ok := m.queue.Retreat(m.modes); !ok {
		return nil
	}
	m.saveState()
	return m.startCurrent()
}

// togglePause pauses or resumes. From a stop it plays the current track,
// resuming where the last session left off, or the one under the queue
// cursor.
func (m *Model) togglePause() tea.Cmd {
	if m.deps.Player.State() != audio.Stopped {
		m.deps.Player.TogglePause()
		return nil
	}
	if _, _, ok := m.queue.Current(); ok {
		return m.startCurrentAt(m.resumeAt)
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
	if m.queue.SetCurrent(current) {
		m.resumeAt = time.Duration(m.deps.State.PositionSeconds * float64(time.Second))
	}
	m.queueList.SetItems(m.queue.Items())
	if _, i, ok := m.queue.Current(); ok {
		m.queueList.SetCursor(i)
	}
}
