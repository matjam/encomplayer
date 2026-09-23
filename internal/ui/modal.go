package ui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/matjam/encomplayer/internal/domain"
	"github.com/matjam/encomplayer/internal/keymap"
)

// modal is a dialog drawn over the body. It receives every keystroke until
// it reports done.
type modal interface {
	update(m *Model, msg tea.KeyPressMsg) (done bool, cmd tea.Cmd)
	view(w, h int) []string
}

// scrollModal shows read-only lines, such as help or track info.
type scrollModal struct {
	title  string
	lines  []string
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
		s.offset = len(s.lines)
	case "esc", "q", "?", "enter", "ctrl+c":
		return true, nil
	}
	s.offset = max(0, min(s.offset, len(s.lines)-page))
	return false, nil
}

func (s *scrollModal) view(w, h int) []string {
	boxW := min(w-4, 96)
	boxH := min(h-2, len(s.lines)+2)
	s.height = boxH
	end := min(len(s.lines), s.offset+boxH-2)
	return panel(s.title, s.lines[s.offset:end], boxW, boxH, true)
}

func newHelpModal(km *keymap.Keymap) modal {
	var lines []string
	section := func(name string, rows [][2]string) {
		lines = append(lines, "", stAccent.Render("  ◆ "+strings.ToUpper(name)))
		for _, r := range rows {
			lines = append(lines, "    "+stBright.Render(fit(r[0], 18))+stText.Render(r[1]))
		}
	}
	section("global", km.Help(keymap.Global))
	section("navigation", km.Help(keymap.Navigation))
	section("queue", km.Help(keymap.Queue))
	section("command mode", commandHelp)
	return &scrollModal{title: "help · j/k scroll · esc close", lines: lines[1:]}
}

func newInfoModal(t domain.Track) modal {
	return &scrollModal{title: "track info", lines: trackDetails(t, 90)}
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

func (c *confirmModal) view(w, _ int) []string {
	boxW := min(w-4, max(40, len(c.question)+6))
	body := []string{
		"",
		"  " + stBright.Render(c.question),
		"",
		"  " + stAccent.Render("[Y]") + stText.Render(" CONFIRM    ") + stAccent.Render("[N]") + stText.Render(" CANCEL"),
	}
	return panel("confirm", body, boxW, len(body)+2, true)
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
	styles := in.Styles()
	styles.Focused.Prompt = stAccent
	styles.Focused.Text = stBright
	in.SetStyles(styles)
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

func (i *inputModal) view(w, _ int) []string {
	boxW := min(w-4, 60)
	i.input.SetWidth(boxW - 8)
	body := []string{"", "  " + i.input.View(), "", stDim.Render("  ENTER CONFIRM · ESC CANCEL")}
	return panel(i.title, body, boxW, len(body)+2, true)
}
