package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/matjam/encomplayer/internal/collection"
	"github.com/matjam/encomplayer/internal/domain"
	"github.com/matjam/encomplayer/internal/keymap"
)

// queueTab shows album art beside the play queue, like rmpc's Queue tab.
type queueTab struct {
	baseTab
	list *collection.List[domain.Track]
}

func (*queueTab) title() string { return "Queue" }

func (*queueTab) contexts() []keymap.Context {
	return []keymap.Context{keymap.Queue, keymap.Navigation, keymap.Global}
}

func (*queueTab) resize(m *Model, _, h int) {
	// Border, column header and rule take four rows.
	m.queueList.SetHeight(h - 4)
}

func (q *queueTab) find(query string, forward, include bool) bool {
	query = strings.ToLower(query)
	return q.list.Find(func(t domain.Track) bool {
		hay := strings.ToLower(t.DisplayTitle() + "\x00" + t.DisplayArtist() + "\x00" + t.DisplayAlbum())
		return strings.Contains(hay, query)
	}, forward, include)
}

// Queue rows start below the border, column header and rule.
const queueFirstRow = 3

func (q *queueTab) click(m *Model, x, y int, double bool) tea.Cmd {
	if artW, _ := queueSplit(m, m.width); x < artW {
		return nil
	}
	if listClick(m.queueList, y-queueFirstRow) && double {
		return m.playIndex(m.queueList.Cursor())
	}
	return nil
}

func (q *queueTab) scroll(m *Model, delta int) { m.queueList.Move(delta) }

func (q *queueTab) handle(m *Model, a keymap.Action) (bool, tea.Cmd) {
	l := m.queueList
	if navigate(l, a) {
		return true, nil
	}

	switch a.Name {
	case keymap.Play, keymap.Confirm:
		if l.Len() > 0 {
			return true, m.playIndex(l.Cursor())
		}
	case keymap.Delete:
		cursor := l.Cursor()
		removedCurrent := m.queue.RemoveIndices(l.Targets())
		m.syncQueue()
		l.SetCursor(cursor)
		if removedCurrent {
			return true, m.currentDeleted()
		}
	case keymap.DeleteAll:
		m.queue.Clear()
		m.syncQueue()
		m.deps.Player.Stop()
		m.playGen = 0
		return true, m.showArt(nil)
	case keymap.MoveUp, keymap.MoveDown:
		delta := 1
		if a.Name == keymap.MoveUp {
			delta = -1
		}
		i := l.Cursor()
		if m.queue.Swap(i, i+delta) {
			m.syncQueue()
			l.SetCursor(i + delta)
		}
	case keymap.JumpToCurrent:
		if _, i, ok := m.queue.Current(); ok {
			l.SetCursor(i)
		}
	case keymap.Shuffle:
		m.queue.Shuffle(m.rng)
		m.syncQueue()
		m.status.infof("queue shuffled")
	case keymap.Save:
		m.askSavePlaylist(targets(l))
	case keymap.SaveAll:
		m.askSavePlaylist(m.queue.Items())
	case keymap.ShowInfo:
		if t, ok := l.Current(); ok {
			m.modal = newInfoModal(t)
		}
	case keymap.Close:
		l.ClearSelection()
		return false, nil
	default:
		return false, nil
	}
	return true, nil
}

func (q *queueTab) view(m *Model, w, h int) []string {
	st := m.st
	artW, queueW := queueSplit(m, w)

	current := m.queue.CurrentIndex()
	isCurrent := func(i int, _ domain.Track) bool { return i == current }
	body := st.trackTable(m.queueList, queueW-2, isCurrent, "QUEUE EMPTY · ADD TRACKS WITH  a  FROM ANY BROWSER TAB")

	title := fmt.Sprintf("queue · %d tracks · %s", m.queue.Len(), duration(totalDuration(m.queue.Items())))
	queuePanel := st.panel(title, body, queueW, h, true)
	if artW == 0 {
		return queuePanel
	}
	return hjoin(st.panel("visual", m.artBody(artW-2, h-2), artW, h, false), queuePanel)
}
