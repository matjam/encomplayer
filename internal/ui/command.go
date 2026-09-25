package ui

import (
	"os"
	"strconv"
	"strings"

	"github.com/matjam/encomplayer/internal/config"
	"github.com/matjam/encomplayer/internal/domain"
	"github.com/matjam/encomplayer/internal/keymap"
	"github.com/matjam/encomplayer/internal/library"
	"github.com/matjam/encomplayer/internal/tea"
)

// commandHelp documents command mode for the help screen.
var commandHelp = [][2]string{
	{":scan [dir]", "scan a music folder (default: current)"},
	{":rescan", "reread every file's tags"},
	{":add <path>", "queue a file or folder"},
	{":save <name>", "save the queue as a playlist"},
	{":saveall <name>", "save the whole library as a playlist"},
	{":shuffleall", "play the whole library shuffled (S)"},
	{":load <name>", "replace the queue with a playlist"},
	{":clear", "empty the queue"},
	{":shuffle", "shuffle the queue"},
	{":volume <0-100>", "set the volume"},
	{":repeat :random :single :consume", "toggle a mode"},
	{":config", "open the config screen (oc)"},
	{":theme <name>", "switch theme and save it"},
	{":viz", "show the visualizer full screen, or close it (ov)"},
	{":viz <name|next|prev>", "switch visualizer ([ and ])"},
	{":reload", "reread config.json and the theme (also SIGUSR1)"},
	{":help", "show this screen"},
	{":q", "quit"},
}

func (m *Model) runCommand(line string) tea.Cmd {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return nil
	}
	cmd, arg := fields[0], strings.TrimSpace(strings.TrimPrefix(line, fields[0]))
	arg = strings.TrimSpace(arg)

	switch cmd {
	case "q", "quit", "exit":
		return m.quit()
	case "scan", "update":
		if arg != "" {
			return m.startScan(arg, false)
		}
		return m.rescan(false)
	case "rescan":
		return m.rescan(true)
	case "add":
		return m.addPath(arg)
	case "save", "w":
		m.savePlaylist(arg, m.queue.Items())
	case "saveall":
		m.savePlaylist(arg, m.allTracks())
	case "shuffleall":
		return m.playShuffled(m.allTracks())
	case "load":
		return m.loadPlaylist(arg)
	case "clear":
		return m.dispatchQueue(keymap.DeleteAll)
	case "shuffle":
		return m.dispatchQueue(keymap.Shuffle)
	case "volume", "vol":
		v, err := strconv.Atoi(arg)
		if err != nil {
			m.status.errorf("VOLUME MUST BE A NUMBER")
			return nil
		}
		m.deps.Player.SetVolume(v)
		m.saveState()
	case "repeat":
		return m.dispatch(keymap.Action{Name: keymap.ToggleRepeat})
	case "random":
		return m.dispatch(keymap.Action{Name: keymap.ToggleRandom})
	case "single":
		return m.dispatch(keymap.Action{Name: keymap.ToggleSingle})
	case "consume":
		return m.dispatch(keymap.Action{Name: keymap.ToggleConsume})
	case "config", "settings":
		m.modal = newConfigModal(m)
	case "reload":
		return m.reload()
	case "theme":
		if err := m.applyTheme(arg); err != nil {
			m.status.errorf("%v", err)
			return nil
		}
		return m.updateConfig(func(c *config.Config) { c.Theme = arg })
	case "viz", "visualizer":
		return m.runVizCommand(arg)
	case "help":
		m.modal = newHelpModal(m.deps.Keymap)
	default:
		m.status.errorf("UNKNOWN COMMAND: %s", cmd)
	}
	return nil
}

// dispatchQueue runs a queue action whatever tab is showing.
func (m *Model) dispatchQueue(name string) tea.Cmd {
	for _, t := range m.tabs {
		if q, ok := t.(*queueTab); ok {
			_, cmd := q.handle(m, keymap.Action{Name: name})
			return cmd
		}
	}
	return nil
}

// addPath queues a file, or scans a folder in the background and queues it.
func (m *Model) addPath(arg string) tea.Cmd {
	if arg == "" {
		m.status.errorf("USAGE: :add <path>")
		return nil
	}
	path := expandHome(arg)
	info, err := os.Stat(path)
	if err != nil {
		m.status.errorf("%v", err)
		return nil
	}
	if !info.IsDir() {
		if t, ok := m.resolveTrack(path); ok {
			m.enqueue([]domain.Track{t})
		}
		return nil
	}

	scanner, ctx := m.deps.Scanner, m.ctx
	return func() tea.Msg {
		snap, _, err := scanner.Scan(ctx, path, nil, library.Options{})
		if err != nil {
			return enqueueMsg{err: err}
		}
		return enqueueMsg{tracks: library.New(path, snap.Tracks).Tree().AllTracks()}
	}
}

// enqueueMsg carries tracks found by a background :add scan.
type enqueueMsg struct {
	tracks []domain.Track
	err    error
}
