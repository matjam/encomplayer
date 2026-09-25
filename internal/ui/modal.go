package ui

import (
	"strings"

	"github.com/matjam/encomplayer/internal/domain"
	"github.com/matjam/encomplayer/internal/keymap"
	"github.com/matjam/encomplayer/internal/tea"
	"github.com/matjam/encomplayer/internal/tea/textinput"
)

// modal is a dialog drawn over the body. It receives every keystroke until
// it reports done. It is drawn with the current styles each frame, so a
// theme change recolours it immediately.
type modal interface {
	update(m *Model, msg tea.KeyPressMsg) (done bool, cmd tea.Cmd)
	view(st *styles, w, h int) []string
}

// scrollModal shows read-only lines, such as help or track info.
type scrollModal struct {
	title  string
	lines  func(st *styles) []string
	count  int
	offset int
	height int
}

func (s *scrollModal) update(_ *Model, msg tea.KeyPressMsg) (bool, tea.Cmd) {
	page := max(1, s.height-2)
	switch msg.String() {
	case "j", "down":
		s.offset++
	case "k", "up":
		s.offset--
	case "ctrl+d", "ctrl+f", "pgdown", "space":
		s.offset += page
	case "ctrl+u", "ctrl+b", "pgup":
		s.offset -= page
	case "g":
		s.offset = 0
	case "G":
		s.offset = s.count
	case "esc", "q", "?", "enter", "ctrl+c":
		return true, nil
	}
	s.offset = max(0, min(s.offset, s.count-page))
	return false, nil
}

// scroll moves the view by delta lines, for the mouse wheel.
func (s *scrollModal) scroll(delta int) {
	s.offset = max(0, min(s.offset+delta, s.count-max(1, s.height-2)))
}

func (s *scrollModal) view(st *styles, w, h int) []string {
	lines := s.lines(st)
	s.count = len(lines)
	boxW := min(w-4, 96)
	boxH := min(h-2, len(lines)+2)
	s.height = boxH
	s.offset = max(0, min(s.offset, len(lines)-max(1, boxH-2)))
	end := min(len(lines), s.offset+boxH-2)
	return st.panel(s.title, lines[s.offset:end], boxW, boxH, true)
}

func newHelpModal(km *keymap.Keymap) modal {
	return &scrollModal{title: "help · j/k scroll · esc close", lines: func(st *styles) []string {
		var lines []string
		section := func(name string, rows [][2]string) {
			lines = append(lines, "", st.accent.Render("  ◆ "+strings.ToUpper(name)))
			for _, r := range rows {
				lines = append(lines, "    "+st.bright.Render(fit(r[0], 18))+st.text.Render(r[1]))
			}
		}
		section("global", km.Help(keymap.Global))
		section("navigation", km.Help(keymap.Navigation))
		section("queue", km.Help(keymap.Queue))
		section("command mode", commandHelp)
		return lines[1:]
	}}
}

func newInfoModal(t domain.Track) modal {
	return &scrollModal{title: "track info", lines: func(st *styles) []string { return st.trackDetails(t, 90) }}
}

// confirmModal asks a yes/no question.
type confirmModal struct {
	question string
	onYes    func()
}

func newConfirmModal(question string, onYes func()) modal {
	return &confirmModal{question: question, onYes: onYes}
}

func (c *confirmModal) update(_ *Model, msg tea.KeyPressMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "y", "Y", "enter":
		c.onYes()
		return true, nil
	case "n", "N", "esc", "q", "ctrl+c":
		return true, nil
	}
	return false, nil
}

func (c *confirmModal) view(st *styles, w, _ int) []string {
	boxW := min(w-4, max(40, len(c.question)+6))
	body := []string{
		"",
		"  " + st.bright.Render(c.question),
		"",
		"  " + st.accent.Render("[Y]") + st.text.Render(" CONFIRM    ") + st.accent.Render("[N]") + st.text.Render(" CANCEL"),
	}
	return st.panel("confirm", body, boxW, len(body)+2, true)
}

// inputModal asks for one line of text.
type inputModal struct {
	title    string
	input    textinput.Model
	onSubmit func(string)
}

func newInputModal(title, value string, onSubmit func(string)) modal {
	in := textinput.New()
	in.Prompt = "▸ "
	in.SetValue(value)
	in.CursorEnd()
	in.Focus()
	return &inputModal{title: title, input: in, onSubmit: onSubmit}
}

func (i *inputModal) update(_ *Model, msg tea.KeyPressMsg) (bool, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		return true, nil
	case "enter":
		if v := strings.TrimSpace(i.input.Value()); v != "" {
			i.onSubmit(v)
		}
		return true, nil
	}
	var cmd tea.Cmd
	i.input, cmd = i.input.Update(msg)
	return false, cmd
}

func (i *inputModal) view(st *styles, w, _ int) []string {
	boxW := min(w-4, 60)
	i.input.SetWidth(boxW - 8)
	st.styleInput(&i.input)
	body := []string{"", "  " + i.input.View(), "", st.dim.Render("  ENTER CONFIRM · ESC CANCEL")}
	return st.panel(i.title, body, boxW, len(body)+2, true)
}
