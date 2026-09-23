package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/matjam/encomplayer/internal/collection"
	"github.com/matjam/encomplayer/internal/keymap"
)

// tab is one screen reachable from the tab bar.
type tab interface {
	title() string

	// contexts lists binding contexts in priority order.
	contexts() []keymap.Context

	// handle runs an action. It reports false for actions it leaves to the
	// global handler.
	handle(m *Model, a keymap.Action) (bool, tea.Cmd)

	// captures reports whether a text field wants raw keystrokes.
	captures() bool

	// key receives raw keystrokes while captures is true.
	key(m *Model, msg tea.KeyPressMsg) tea.Cmd

	// find moves the cursor to the next row matching query.
	find(query string, forward, includeCursor bool) bool

	// click handles a left click at (x, y) relative to the tab body.
	click(m *Model, x, y int, double bool) tea.Cmd

	// scroll moves the focused list by delta rows for the mouse wheel.
	scroll(m *Model, delta int)

	resize(m *Model, w, h int)
	view(m *Model, w, h int) []string
}

// baseTab supplies defaults for tabs without a text field.
type baseTab struct{}

func (baseTab) captures() bool                               { return false }
func (baseTab) key(*Model, tea.KeyPressMsg) tea.Cmd          { return nil }
func (baseTab) contexts() []keymap.Context                   { return []keymap.Context{keymap.Navigation, keymap.Global} }
func (baseTab) find(string, bool, bool) bool                 { return false }
func (baseTab) handle(*Model, keymap.Action) (bool, tea.Cmd) { return false, nil }
func (baseTab) click(*Model, int, int, bool) tea.Cmd         { return nil }
func (baseTab) scroll(*Model, int)                           {}

// listClick moves l's cursor to the item on viewport row r and reports
// whether a row was hit.
func listClick[T any](l *collection.List[T], r int) bool {
	i, ok := l.IndexAtRow(r)
	if ok {
		l.SetCursor(i)
	}
	return ok
}

// navigate applies the cursor-movement actions shared by every list.
func navigate[T any](l *collection.List[T], a keymap.Action) bool {
	half := max(1, l.Height()/2)
	switch a.Name {
	case keymap.Up:
		l.Move(-1)
	case keymap.Down:
		l.Move(1)
	case keymap.UpHalf:
		l.Move(-half)
	case keymap.DownHalf:
		l.Move(half)
	case keymap.PageUp:
		l.Move(-l.Height())
	case keymap.PageDown:
		l.Move(l.Height())
	case keymap.Top:
		l.Top()
	case keymap.Bottom:
		l.Bottom()
	case keymap.Select:
		l.ToggleSelect()
	case keymap.InvertSelection:
		l.InvertSelection()
	default:
		return false
	}
	return true
}

// targets returns the selected items, or the item under the cursor.
func targets[T any](l *collection.List[T]) []T {
	return collection.Map(l.Targets(), func(i int) T { return l.Items()[i] })
}
