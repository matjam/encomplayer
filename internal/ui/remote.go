package ui

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/matjam/encomplayer/internal/audio"
	"github.com/matjam/encomplayer/internal/keymap"
	"github.com/matjam/encomplayer/internal/remote"
)

// RemoteMsg carries a command from the remote control socket into the event
// loop, so it runs exactly like the matching key. Reply receives the answer
// and must be buffered.
type RemoteMsg struct {
	Req   remote.Request
	Reply chan<- remote.Response
}

// NoticeMsg shows a message from outside the event loop in the status line.
type NoticeMsg struct{ Text string }

// RemoteCommands lists the commands the player accepts, for the CLI.
var RemoteCommands = []string{
	"status", "play", "pause", "toggle", "stop", "next", "prev",
	"seek", "volume", "repeat", "random", "single", "consume",
	"shuffle", "shuffle-all", "add", "reload", "viz",
}

func (m *Model) handleRemote(msg RemoteMsg) tea.Cmd {
	cmd, err := m.runRemote(msg.Req)
	resp := remote.Response{OK: err == nil, Status: m.remoteStatus()}
	if err != nil {
		resp.Error = err.Error()
	}
	msg.Reply <- resp
	return cmd
}

func (m *Model) runRemote(req remote.Request) (tea.Cmd, error) {
	arg := ""
	if len(req.Args) > 0 {
		arg = req.Args[0]
	}
	act := func(name string) (tea.Cmd, error) { return m.dispatch(keymap.Action{Name: name}), nil }
	state := m.deps.Player.State()

	switch req.Cmd {
	case "status":
		return nil, nil
	case "play":
		if state == audio.Playing {
			return nil, nil
		}
		return m.togglePause(), nil
	case "pause":
		if state == audio.Playing {
			m.deps.Player.TogglePause()
		}
		return nil, nil
	case "toggle":
		return act(keymap.TogglePause)
	case "stop":
		return act(keymap.Stop)
	case "next":
		return act(keymap.NextTrack)
	case "prev":
		return act(keymap.PreviousTrack)
	case "seek":
		return nil, m.remoteSeek(arg)
	case "volume":
		return nil, m.remoteVolume(arg)
	case "repeat", "random", "single", "consume":
		return nil, m.remoteMode(req.Cmd, arg)
	case "shuffle":
		return m.dispatchQueue(keymap.Shuffle), nil
	case "shuffle-all":
		if m.lib == nil || len(m.lib.Tracks()) == 0 {
			return nil, fmt.Errorf("the library is still loading")
		}
		return m.playShuffled(m.allTracks()), nil
	case "add":
		if arg == "" {
			return nil, fmt.Errorf("add needs a file or folder")
		}
		return m.addPath(arg), nil
	case "reload":
		return m.reload(), nil
	case "viz":
		if arg != "" && arg != "next" && arg != "prev" && !slices.Contains(m.viz.catalog.Names(), arg) {
			return nil, fmt.Errorf("viz: no visualizer named %q (see encomplayer --list-visualizers)", arg)
		}
		return m.runVizCommand(arg), nil
	default:
		return nil, fmt.Errorf("unknown command %q", req.Cmd)
	}
}

// remoteSeek accepts "+10" or "-30" to move relative to the position, and
// "83" or "1:23" to jump to an absolute time.
func (m *Model) remoteSeek(arg string) error {
	if m.deps.Player.State() == audio.Stopped {
		return fmt.Errorf("nothing is playing")
	}
	pos, length := m.deps.Player.Progress()
	switch {
	case strings.HasPrefix(arg, "+"), strings.HasPrefix(arg, "-"):
		secs, err := strconv.ParseFloat(arg, 64)
		if err != nil {
			return fmt.Errorf("seek: %q is not a number of seconds", arg)
		}
		return m.deps.Player.Seek(time.Duration(secs * float64(time.Second)))
	default:
		target, err := parseClock(arg)
		if err != nil {
			return err
		}
		if length > 0 && target >= length {
			return fmt.Errorf("seek: %s is past the end of the track (%s)", arg, clock(length))
		}
		return m.deps.Player.Seek(target - pos)
	}
}

// parseClock reads "83", "1:23" or "1:02:03".
func parseClock(s string) (time.Duration, error) {
	parts := strings.Split(s, ":")
	if s == "" || len(parts) > 3 {
		return 0, fmt.Errorf("seek: want +N, -N, seconds or m:ss, got %q", s)
	}
	var total float64
	for _, p := range parts {
		v, err := strconv.ParseFloat(p, 64)
		if err != nil || v < 0 {
			return 0, fmt.Errorf("seek: want +N, -N, seconds or m:ss, got %q", s)
		}
		total = total*60 + v
	}
	return time.Duration(total * float64(time.Second)), nil
}

// remoteVolume accepts "60" to set the volume, or "+5" / "-5" to step it.
func (m *Model) remoteVolume(arg string) error {
	v, err := strconv.Atoi(arg)
	if err != nil {
		return fmt.Errorf("volume: want 0-100, +N or -N, got %q", arg)
	}
	if strings.HasPrefix(arg, "+") || strings.HasPrefix(arg, "-") {
		v += m.deps.Player.Volume()
	}
	m.deps.Player.SetVolume(v)
	m.saveState()
	return nil
}

// remoteMode sets a playback mode on or off, or toggles it with no argument.
func (m *Model) remoteMode(name, arg string) error {
	modes := map[string]*bool{
		"repeat": &m.modes.Repeat, "random": &m.modes.Random,
		"single": &m.modes.Single, "consume": &m.modes.Consume,
	}
	flag := modes[name]
	switch arg {
	case "", "toggle":
		*flag = !*flag
	case "on":
		*flag = true
	case "off":
		*flag = false
	default:
		return fmt.Errorf("%s: want on, off or toggle, got %q", name, arg)
	}
	m.modesChanged()
	return nil
}

func (m *Model) remoteStatus() *remote.Status {
	p := m.deps.Player
	pos, length := p.Progress()
	s := &remote.Status{
		State:          stateName(p.State()),
		Position:       pos.Seconds(),
		Duration:       length.Seconds(),
		Volume:         p.Volume(),
		Repeat:         m.modes.Repeat,
		Random:         m.modes.Random,
		Single:         m.modes.Single,
		Consume:        m.modes.Consume,
		QueueIndex:     m.queue.CurrentIndex(),
		QueueLength:    m.queue.Len(),
		Visualizer:     m.viz.info.Name,
		VisualizerFull: m.viz.full,
	}
	if t, _, ok := m.queue.Current(); ok {
		s.Title, s.Artist, s.Album, s.Path = t.DisplayTitle(), t.DisplayArtist(), t.DisplayAlbum(), t.Path
		if s.Duration == 0 {
			s.Duration = t.Duration.Seconds()
		}
		if p.State() == audio.Stopped && m.resumeAt > 0 {
			s.Position = m.resumeAt.Seconds()
		}
	}
	return s
}

func stateName(s audio.State) string {
	switch s {
	case audio.Playing:
		return "playing"
	case audio.Paused:
		return "paused"
	default:
		return "stopped"
	}
}
