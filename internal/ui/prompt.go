package ui

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type promptKind int

const (
	promptCommand promptKind = iota
	promptFind
)

// prompt is the single-line input on the status row, used for command mode
// (":") and incremental find ("/").
type prompt struct {
	kind  promptKind
	input textinput.Model
}

func newPrompt(kind promptKind, symbol, value string) *prompt {
	in := textinput.New()
	in.Prompt = symbol
	in.SetValue(value)
	in.Focus()
	return &prompt{kind: kind, input: in}
}

func (p *prompt) update(m *Model, msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.prompt = nil
		return nil
	case "enter":
		m.prompt = nil
		if p.kind == promptCommand {
			return m.runCommand(p.input.Value())
		}
		m.lastFind = p.input.Value()
		return nil
	}

	var cmd tea.Cmd
	p.input, cmd = p.input.Update(msg)
	if p.kind == promptFind && p.input.Value() != "" {
		m.tabs[m.active].find(p.input.Value(), true, true)
	}
	return cmd
}

func (p *prompt) view(st *styles, w int) string {
	p.input.SetWidth(max(1, w-2))
	st.styleInput(&p.input)
	return p.input.View()
}
