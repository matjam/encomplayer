package ui

import (
	"slices"

	"github.com/matjam/encomplayer/internal/domain"
	"github.com/matjam/encomplayer/internal/tea"
)

// askSavePlaylist prompts for a name and appends tracks to that playlist,
// creating it when new. Tracks already in the playlist are skipped.
func (m *Model) askSavePlaylist(tracks []domain.Track) {
	if len(tracks) == 0 {
		return
	}
	m.modal = newInputModal("SAVE TO PLAYLIST", "", func(name string) {
		m.savePlaylist(name, tracks)
	})
}

func (m *Model) savePlaylist(name string, tracks []domain.Track) {
	store := m.deps.Playlists
	var existing []domain.Track
	if names, err := store.List(); err == nil && slices.Contains(names, name) {
		p, err := store.Load(name)
		if err != nil {
			m.status.errorf("%v", err)
			return
		}
		existing = m.resolveTracks(p.Paths)
	}

	seen := make(map[string]bool, len(existing))
	for _, t := range existing {
		seen[t.Path] = true
	}
	added := 0
	for _, t := range tracks {
		if !seen[t.Path] {
			existing = append(existing, t)
			seen[t.Path] = true
			added++
		}
	}

	if err := store.Save(name, existing); err != nil {
		m.status.errorf("%v", err)
		return
	}
	m.status.infof("%d track(s) saved to %s", added, name)
	m.reloadPlaylists()
}

// loadPlaylist replaces the queue with a playlist and starts playing it.
func (m *Model) loadPlaylist(name string) tea.Cmd {
	p, err := m.deps.Playlists.Load(name)
	if err != nil {
		m.status.errorf("%v", err)
		return nil
	}
	m.queue.Clear()
	m.syncQueue()
	return m.enqueueAndPlay(m.resolveTracks(p.Paths))
}

func (m *Model) reloadPlaylists() {
	for _, t := range m.tabs {
		if b, ok := t.(*browserTab); ok && b.playlists != nil {
			b.reload()
		}
	}
}
