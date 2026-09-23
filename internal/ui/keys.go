package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/matjam/encomplayer/internal/keymap"
)

func (m *Model) handleKey(msg tea.KeyPressMsg) tea.Cmd {
	if m.boot.active() {
		m.boot.skip()
		return nil
	}
	if msg.String() == "ctrl+c" && m.prompt == nil && m.modal == nil && !m.tabs[m.active].captures() {
		return m.quit()
	}
	if m.prompt != nil {
		return m.prompt.update(m, msg)
	}
	if m.modal != nil {
		done, cmd := m.modal.update(m, msg)
		if done {
			m.modal = nil
		}
		return cmd
	}
	if t := m.tabs[m.active]; t.captures() {
		return t.key(m, msg)
	}

	res := m.resolver.Feed(msg.String(), m.tabs[m.active].contexts()...)
	switch {
	case res.Pending:
		gen := res.Generation
		return tea.Tick(chordTimeout, func(time.Time) tea.Msg { return keyTimeoutMsg(gen) })
	case res.Matched:
		return m.dispatch(res.Action)
	}
	return nil
}

// dispatch offers an action to the active tab, then handles it globally.
func (m *Model) dispatch(a keymap.Action) tea.Cmd {
	if handled, cmd := m.tabs[m.active].handle(m, a); handled {
		return cmd
	}

	p := m.deps.Player
	cfg := m.deps.Config
	switch a.Name {
	case keymap.Quit:
		return m.quit()
	case keymap.ShowHelp:
		m.modal = newHelpModal(m.deps.Keymap)
	case keymap.CommandMode:
		m.prompt = newPrompt(promptCommand, ":", "")
	case keymap.ShowCurrentSongInfo:
		if t, _, ok := m.queue.Current(); ok {
			m.modal = newInfoModal(t)
		}
	case keymap.ToggleRepeat:
		m.modes.Repeat = !m.modes.Repeat
		m.modesChanged()
	case keymap.ToggleRandom:
		m.modes.Random = !m.modes.Random
		m.modesChanged()
	case keymap.ToggleConsume:
		m.modes.Consume = !m.modes.Consume
		m.modesChanged()
	case keymap.ToggleSingle:
		m.modes.Single = !m.modes.Single
		m.modesChanged()
	case keymap.TogglePause:
		return m.togglePause()
	case keymap.Stop:
		p.Stop()
		m.playGen = 0
	case keymap.NextTrack:
		return m.advance(false)
	case keymap.PreviousTrack:
		return m.retreat()
	case keymap.SeekForward:
		m.seek(time.Duration(cfg.SeekSeconds) * time.Second)
	case keymap.SeekBack:
		m.seek(-time.Duration(cfg.SeekSeconds) * time.Second)
	case keymap.VolumeUp:
		p.SetVolume(p.Volume() + cfg.VolumeStep)
		m.saveState()
	case keymap.VolumeDown:
		p.SetVolume(p.Volume() - cfg.VolumeStep)
		m.saveState()
	case keymap.NextTab:
		m.active = (m.active + 1) % len(m.tabs)
	case keymap.PreviousTab:
		m.active = (m.active + len(m.tabs) - 1) % len(m.tabs)
	case keymap.SwitchToTab:
		m.switchTo(a.Arg)
	case keymap.Update:
		return m.rescan(false)
	case keymap.Rescan:
		return m.rescan(true)
	case keymap.AddRandom:
		m.addRandom(10)
	case keymap.ShuffleAll:
		return m.playShuffled(m.allTracks())
	case keymap.EnterSearch:
		m.prompt = newPrompt(promptFind, "/", "")
	case keymap.NextResult, keymap.PreviousResult:
		m.findAgain(a.Name == keymap.NextResult)
	case keymap.Close:
		m.resolver.Reset()
	}
	return nil
}

// modesChanged persists the modes and preloads whatever now plays next.
func (m *Model) modesChanged() {
	m.preloadNext()
	m.saveState()
}

func (m *Model) seek(d time.Duration) {
	if err := m.deps.Player.Seek(d); err != nil {
		m.status.errorf("%v", err)
	}
}

func (m *Model) findAgain(forward bool) {
	if m.lastFind == "" {
		return
	}
	if !m.tabs[m.active].find(m.lastFind, forward, false) {
		m.status.infof("pattern not found: %s", m.lastFind)
	}
}

func (m *Model) quit() tea.Cmd {
	m.saveState()
	m.deps.Player.Stop()
	cleanup := m.art.release()
	return tea.Sequence(tea.Raw(cleanup), tea.Quit)
}
