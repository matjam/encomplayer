package ui

import (
	"maps"
	"strings"

	"github.com/matjam/encomplayer/internal/collection"
	"github.com/matjam/encomplayer/internal/config"
	"github.com/matjam/encomplayer/internal/tea"
	"github.com/matjam/encomplayer/internal/tea/textinput"
)

type configTab struct {
	name string
	rows func(m *Model) []configRow
}

// configModal edits the config file in place. Every change applies at once
// and is saved, so closing the screen never loses anything.
type configModal struct {
	m      *Model
	tabs   []configTab
	active int
	rows   *collection.List[configRow]

	input      *textinput.Model
	inputLabel string
	onSubmit   func(string) (tea.Cmd, error)

	choice *choiceState
}

// choiceState is an open picker, such as the theme list.
type choiceState struct {
	title   string
	items   *collection.List[string]
	preview func(int)
	commit  func(int) tea.Cmd
	cancel  func()
}

func newConfigModal(m *Model) *configModal {
	c := &configModal{m: m, tabs: configTabs(), rows: collection.NewList[configRow](nil)}
	c.refresh(m)
	return c
}

// refresh rebuilds the rows for the active tab, keeping the cursor.
func (c *configModal) refresh(m *Model) {
	cursor := c.rows.Cursor()
	c.rows.SetItems(c.tabs[c.active].rows(m))
	c.rows.SetCursor(cursor)
}

func (c *configModal) openInput(label, value string, submit func(string) (tea.Cmd, error)) {
	in := textinput.New()
	in.Prompt = "▸ "
	in.SetValue(value)
	in.CursorEnd()
	in.Focus()
	c.input, c.inputLabel, c.onSubmit = &in, label, submit
}

func (c *configModal) openChoice(title string, items []string, current int, preview func(int), commit func(int) tea.Cmd, cancel func()) {
	l := collection.NewList(items)
	l.SetCursor(current)
	c.choice = &choiceState{title: title, items: l, preview: preview, commit: commit, cancel: cancel}
}

func (c *configModal) update(m *Model, msg tea.KeyPressMsg) (bool, tea.Cmd) {
	key := msg.String()
	switch {
	case c.input != nil:
		return false, c.updateInput(m, msg)
	case c.choice != nil:
		return false, c.updateChoice(m, key)
	}

	switch key {
	case "esc", "q", "ctrl+c":
		return true, nil
	case "h", "left", "shift+tab":
		c.switchTab(m, len(c.tabs)-1)
	case "l", "right", "tab":
		c.switchTab(m, 1)
	case "j", "down":
		c.rows.Move(1)
	case "k", "up":
		c.rows.Move(-1)
	case "ctrl+d", "pgdown":
		c.rows.Move(c.rows.Height() / 2)
	case "ctrl+u", "pgup":
		c.rows.Move(-c.rows.Height() / 2)
	case "g":
		c.rows.Top()
	case "G":
		c.rows.Bottom()
	case "enter", "space":
		if row, ok := c.rows.Current(); ok {
			cmd := row.activate(m, c)
			c.refresh(m)
			return false, cmd
		}
	case "d", "D", "delete":
		if row, ok := c.rows.Current(); ok {
			if k, isKey := row.(keyRow); isKey {
				cmd, err := k.removeOverrides(m)
				if err != nil {
					m.status.errorf("%v", err)
				}
				c.refresh(m)
				return false, cmd
			}
		}
	}
	return false, nil
}

func (c *configModal) switchTab(m *Model, step int) {
	c.active = (c.active + step) % len(c.tabs)
	c.rows.SetItems(c.tabs[c.active].rows(m))
	c.rows.Top()
}

func (c *configModal) updateInput(m *Model, msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "ctrl+c":
		c.input = nil
		return nil
	case "enter":
		cmd, err := c.onSubmit(c.input.Value())
		if err != nil {
			m.status.errorf("%v", err)
			return nil
		}
		c.input = nil
		c.refresh(m)
		return cmd
	}
	in, cmd := c.input.Update(msg)
	c.input = &in
	return cmd
}

func (c *configModal) updateChoice(m *Model, key string) tea.Cmd {
	ch := c.choice
	moved := true
	switch key {
	case "j", "down":
		ch.items.Move(1)
	case "k", "up":
		ch.items.Move(-1)
	case "ctrl+d", "pgdown":
		ch.items.Move(ch.items.Height() / 2)
	case "ctrl+u", "pgup":
		ch.items.Move(-ch.items.Height() / 2)
	case "g":
		ch.items.Top()
	case "G":
		ch.items.Bottom()
	case "enter":
		c.choice = nil
		cmd := ch.commit(ch.items.Cursor())
		c.refresh(m)
		return cmd
	case "esc", "q", "ctrl+c":
		c.choice = nil
		ch.cancel()
		return nil
	default:
		moved = false
	}
	if moved {
		ch.preview(ch.items.Cursor())
	}
	return nil
}

func (c *configModal) view(st *styles, w, h int) []string {
	boxW := min(w-4, 100)
	boxH := max(10, min(h-2, 28))
	inner := boxW - 2
	listH := boxH - 2 - 4

	var tabs []string
	for i, t := range c.tabs {
		label := " " + strings.ToUpper(t.name) + " "
		if i == c.active {
			tabs = append(tabs, st.tabOn.Render(label))
		} else {
			tabs = append(tabs, st.tabOff.Render(label))
		}
	}
	body := []string{strings.Join(tabs, st.grid.Render("│")), st.grid.Render(strings.Repeat("─", inner))}

	hint := "h/l TAB · j/k MOVE · ENTER EDIT · ESC CLOSE"
	switch {
	case c.choice != nil:
		body = append(body, c.choiceLines(st, inner, listH)...)
		hint = "j/k PREVIEW · ENTER APPLY · ESC CANCEL"
	default:
		body = append(body, c.rowLines(st, inner, listH)...)
		if c.tabs[c.active].name == "Keys" {
			hint = "ENTER ADD KEY · d REMOVE CONFIG KEYS · ESC CLOSE"
		}
	}
	for len(body) < boxH-3 {
		body = append(body, "")
	}
	if c.input != nil {
		c.input.SetWidth(inner - 4)
		st.styleInput(c.input)
		body[len(body)-1] = " " + st.accent.Render(c.inputLabel)
		body = append(body, " "+c.input.View())
	} else {
		body = append(body, st.dim.Render(" "+hint))
	}
	return st.panel("config", body, boxW, boxH, true)
}

// rowLines draws the active tab's settings as label / value columns.
func (c *configModal) rowLines(st *styles, w, h int) []string {
	c.rows.SetHeight(h)
	labelW := min(34, w/2)
	var out []string
	for i, row := range c.rows.Visible() {
		label := fit(" "+row.label(), labelW)
		value := fit(row.display(c.m), w-labelW)
		switch {
		case i == c.rows.Cursor():
			out = append(out, st.cursor.Render(label+value))
		default:
			out = append(out, st.dim.Render(label)+st.text.Render(value))
		}
	}
	return out
}

func (c *configModal) choiceLines(st *styles, w, h int) []string {
	ch := c.choice
	ch.items.SetHeight(h - 1)
	out := []string{st.accent.Render(" ◆ " + strings.ToUpper(ch.title))}
	for i, item := range ch.items.Visible() {
		line := fit("   "+item, w)
		if i == ch.items.Cursor() {
			line = st.cursor.Render(line)
		} else {
			line = st.text.Render(line)
		}
		out = append(out, line)
	}
	return out
}

// updateConfig applies an edit to a copy of the config, makes it live and
// saves it.
func (m *Model) updateConfig(edit func(*config.Config)) tea.Cmd {
	next := cloneConfig(m.deps.Config)
	edit(&next)
	cmd := m.applyConfig(next)
	m.saveConfig()
	return cmd
}

// cloneConfig copies cfg deeply enough that editing the copy leaves the
// original untouched.
func cloneConfig(cfg config.Config) config.Config {
	if cfg.Keybinds != nil {
		binds := make(map[string]map[string]string, len(cfg.Keybinds))
		for ctx, b := range cfg.Keybinds {
			binds[ctx] = maps.Clone(b)
		}
		cfg.Keybinds = binds
	}
	return cfg
}
