package ui

import (
	"fmt"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/matjam/encomplayer/internal/collection"
	"github.com/matjam/encomplayer/internal/domain"
	"github.com/matjam/encomplayer/internal/keymap"
	"github.com/matjam/encomplayer/internal/library"
)

// searchTab filters the library by one tag field as the query is typed.
type searchTab struct {
	baseTab
	lib     *library.Library
	input   textinput.Model
	editing bool
	field   int
	results *collection.List[domain.Track]
}

func newSearchTab() *searchTab {
	in := textinput.New()
	in.Prompt = ""
	in.Placeholder = "press i to enter a query"
	styles := in.Styles()
	styles.Focused.Text = stBright
	styles.Blurred.Text = stText
	styles.Focused.Placeholder = stDim
	styles.Blurred.Placeholder = stDim
	in.SetStyles(styles)
	return &searchTab{input: in, results: collection.NewList[domain.Track](nil)}
}

func (s *searchTab) title() string  { return "Search" }
func (s *searchTab) captures() bool { return s.editing }

func (s *searchTab) resize(_ *Model, w, h int) {
	s.input.SetWidth(max(10, w-40))
	s.results.SetHeight(h - 3 - 4)
}

func (s *searchTab) run() {
	if s.lib == nil {
		s.results.SetItems(nil)
		return
	}
	s.results.SetItems(s.lib.Search(library.Fields[s.field], s.input.Value()))
}

func (s *searchTab) find(string, bool, bool) bool { return false }

func (s *searchTab) key(m *Model, msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "enter", "esc", "ctrl+c":
		s.editing = false
		s.input.Blur()
		return nil
	case "tab":
		s.field = (s.field + 1) % len(library.Fields)
		s.run()
		return nil
	}
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	s.run()
	return cmd
}

// Search layout: a three-row form panel, then the results panel with its
// border, column header and rule.
const (
	searchFormRows   = 3
	searchResultsRow = searchFormRows + queueFirstRow
)

func (s *searchTab) click(m *Model, _, y int, double bool) tea.Cmd {
	if y < searchFormRows {
		s.editing = true
		return s.input.Focus()
	}
	if listClick(s.results, y-searchResultsRow) && double {
		if t, ok := s.results.Current(); ok {
			return m.enqueueAndPlay([]domain.Track{t})
		}
	}
	return nil
}

func (s *searchTab) scroll(_ *Model, delta int) { s.results.Move(delta) }

func (s *searchTab) handle(m *Model, a keymap.Action) (bool, tea.Cmd) {
	l := s.results
	if navigate(l, a) {
		return true, nil
	}
	switch a.Name {
	case keymap.FocusInput, keymap.EnterSearch:
		s.editing = true
		return true, s.input.Focus()
	case keymap.Left, keymap.Right:
		step := 1
		if a.Name == keymap.Left {
			step = len(library.Fields) - 1
		}
		s.field = (s.field + step) % len(library.Fields)
		s.run()
	case keymap.Confirm:
		if t, ok := l.Current(); ok {
			return true, m.enqueueAndPlay([]domain.Track{t})
		}
		s.editing = true
		return true, s.input.Focus()
	case keymap.Add:
		m.enqueue(targets(l))
		l.ClearSelection()
	case keymap.AddAll:
		m.enqueue(l.Items())
	case keymap.ShufflePlay:
		return true, m.playShuffled(l.Items())
	case keymap.Save:
		m.askSavePlaylist(targets(l))
	case keymap.SaveAll:
		m.askSavePlaylist(l.Items())
	case keymap.ShowInfo:
		if t, ok := l.Current(); ok {
			m.modal = newInfoModal(t)
		}
	default:
		return false, nil
	}
	return true, nil
}

func (s *searchTab) view(m *Model, w, h int) []string {
	field := stAccent.Render("‹ " + string(library.Fields[s.field]) + " ›")
	query := stDim.Render("QUERY ▸ ") + s.input.View()
	form := []string{fit(query, w-2-30) + fitRight(stDim.Render("FIELD ")+field+stDim.Render(" h/l"), 30)}

	playing := m.playingPath()
	isPlaying := func(_ int, t domain.Track) bool { return t.Path == playing && playing != "" }
	results := trackTable(s.results, w-2, isPlaying, "NO MATCHES")

	title := fmt.Sprintf("results · %d", s.results.Len())
	return append(panel("search", form, w, 3, s.editing), panel(title, results, w, h-3, !s.editing)...)
}
