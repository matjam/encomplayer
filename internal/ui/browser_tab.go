package ui

import (
	"errors"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/matjam/encomplayer/internal/collection"
	"github.com/matjam/encomplayer/internal/domain"
	"github.com/matjam/encomplayer/internal/keymap"
)

// browserTab shows a hierarchy as Miller columns: parent, current, preview.
type browserTab struct {
	baseTab
	name      string
	hint      string
	browser   *collection.Browser[entry]
	playlists PlaylistStore
}

func newBrowserTab(name, hint string, p entryProvider) *browserTab {
	return &browserTab{name: name, hint: hint, browser: collection.NewBrowser[entry](p)}
}

// newPlaylistsTab browses saved playlists and supports rename and delete.
func newPlaylistsTab(m *Model) *browserTab {
	store := m.deps.Playlists
	names := func() []string {
		names, err := store.List()
		if err != nil {
			m.status.errorf("%v", err)
		}
		return names
	}
	load := func(name string) []domain.Track {
		p, err := store.Load(name)
		if err != nil {
			m.status.errorf("%v", err)
			return nil
		}
		return m.resolveTracks(p.Paths)
	}
	t := newBrowserTab("Playlists", "playlist › track", playlistsProvider(m.allTracks, names, load))
	t.playlists = store
	return t
}

func (t *browserTab) title() string { return t.name }

func (t *browserTab) reload() { t.browser.SetProvider(t.browser.Provider()) }

func (t *browserTab) resize(_ *Model, _, h int) {
	t.browser.Current().SetHeight(h - 2)
	if p := t.browser.Parent(); p != nil {
		p.SetHeight(h - 2)
	}
}

func (t *browserTab) find(query string, forward, include bool) bool {
	q := strings.ToLower(query)
	return t.browser.Current().Find(func(e entry) bool {
		return strings.Contains(strings.ToLower(e.label), q)
	}, forward, include)
}

// click works column by column: the parent column goes up a level, the
// current column moves the cursor (double click opens or plays), and the
// preview column opens the clicked child.
func (t *browserTab) click(m *Model, x, y int, double bool) tea.Cmd {
	parentW, currentW, _ := browserSplit(m, m.width)
	row := y - 1 // below the panel border

	switch {
	case x < parentW:
		if p := t.browser.Parent(); p != nil {
			if i, ok := p.IndexAtRow(row); ok {
				t.browser.Leave()
				t.browser.Current().SetCursor(i)
			}
		}
	case x < parentW+currentW:
		if listClick(t.browser.Current(), row) && double {
			_, cmd := t.handle(m, keymap.Action{Name: keymap.Confirm})
			return cmd
		}
	default:
		if row >= 0 && row < len(t.browser.Preview()) && t.browser.Enter() {
			t.browser.Current().SetCursor(row)
		}
	}
	return nil
}

func (t *browserTab) scroll(_ *Model, delta int) { t.browser.Current().Move(delta) }

func (t *browserTab) handle(m *Model, a keymap.Action) (bool, tea.Cmd) {
	l := t.browser.Current()
	if navigate(l, a) {
		return true, nil
	}

	switch a.Name {
	case keymap.Right:
		t.browser.Enter()
	case keymap.Left:
		t.browser.Leave()
	case keymap.Confirm:
		e, ok := l.Current()
		if !ok {
			return true, nil
		}
		if e.isLeaf() {
			return true, m.enqueueAndPlay([]domain.Track{e.track})
		}
		t.browser.Enter()
	case keymap.Add:
		m.enqueue(collection.FlatMap(targets(l), entry.allTracks))
		l.ClearSelection()
	case keymap.AddAll:
		m.enqueue(collection.FlatMap(l.Items(), entry.allTracks))
	case keymap.ShufflePlay:
		return true, m.playShuffled(collection.FlatMap(targets(l), entry.allTracks))
	case keymap.Save:
		m.askSavePlaylist(collection.FlatMap(targets(l), entry.allTracks))
	case keymap.SaveAll:
		m.askSavePlaylist(collection.FlatMap(l.Items(), entry.allTracks))
	case keymap.ShowInfo:
		if e, ok := l.Current(); ok && e.isLeaf() {
			m.modal = newInfoModal(e.track)
		}
	case keymap.Close:
		l.ClearSelection()
		return false, nil
	case keymap.Delete:
		return t.playlists != nil, t.delete(m)
	case keymap.Rename:
		return t.playlists != nil, t.rename(m)
	default:
		return false, nil
	}
	return true, nil
}

// delete removes playlists at the top level, or tracks inside a playlist.
func (t *browserTab) delete(m *Model) tea.Cmd {
	if t.playlists == nil {
		return nil
	}
	l := t.browser.Current()
	if t.browser.Depth() == 0 {
		var names []string
		for _, e := range targets(l) {
			if e.key != allMusicKey {
				names = append(names, e.key)
			}
		}
		if len(names) == 0 {
			return nil
		}
		m.modal = newConfirmModal(fmt.Sprintf("DELETE PLAYLIST %s?", strings.Join(names, ", ")), func() {
			var errs []error
			for _, n := range names {
				errs = append(errs, t.playlists.Delete(n))
			}
			if err := errors.Join(errs...); err != nil {
				m.status.errorf("%v", err)
			}
			t.reload()
		})
		return nil
	}

	parent, ok := t.browser.Parent().Current()
	if !ok || parent.key == allMusicKey {
		return nil
	}
	drop := map[int]bool{}
	for _, i := range l.Targets() {
		drop[i] = true
	}
	var keep []domain.Track
	for i, e := range l.Items() {
		if !drop[i] {
			keep = append(keep, e.track)
		}
	}
	if err := t.playlists.Save(parent.key, keep); err != nil {
		m.status.errorf("%v", err)
	}
	t.reload()
	return nil
}

func (t *browserTab) rename(m *Model) tea.Cmd {
	if t.playlists == nil || t.browser.Depth() != 0 {
		return nil
	}
	e, ok := t.browser.Current().Current()
	if !ok || e.key == allMusicKey {
		return nil
	}
	m.modal = newInputModal("RENAME PLAYLIST", e.key, func(name string) {
		if err := t.playlists.Rename(e.key, name); err != nil {
			m.status.errorf("%v", err)
			return
		}
		t.reload()
	})
	return nil
}

func (t *browserTab) view(m *Model, w, h int) []string {
	parentW, currentW, previewW := browserSplit(m, w)
	playing := m.playingPath()

	var parent []string
	parentTitle := "ROOT"
	if p := t.browser.Parent(); p != nil {
		parent = renderEntries(p, parentW-2, false, playing)
		if e, ok := p.Current(); ok {
			parentTitle = e.label
		}
	} else {
		parent = []string{stDim.Render(t.hint)}
	}

	title := t.name
	if path := t.browser.Path(); len(path) > 0 {
		title = path[len(path)-1].label
	}
	current := renderEntries(t.browser.Current(), currentW-2, true, playing)
	if t.browser.Current().Len() == 0 {
		current = []string{stDim.Render(emptyMessage(m))}
	}

	var preview []string
	if e, ok := t.browser.Current().Current(); ok && e.isLeaf() {
		preview = trackDetails(e.track, previewW-2)
	} else {
		preview = renderPlain(t.browser.Preview(), previewW-2, h-2, playing)
	}

	return hjoin(
		panel(parentTitle, parent, parentW, h, false),
		panel(title, current, currentW, h, true),
		panel("preview", preview, previewW, h, false),
	)
}

func emptyMessage(m *Model) string {
	if m.scan.running {
		return "INDEXING SECTOR…"
	}
	if m.lib == nil {
		return "NO SECTOR MOUNTED · :scan <dir>"
	}
	return "EMPTY"
}

func renderEntries(l *collection.List[entry], w int, focused bool, playing string) []string {
	var out []string
	for i, e := range l.Visible() {
		row := entryRow(e, w, l.IsSelected(i), playing)
		if i == l.Cursor() {
			style := stDimCur
			if focused {
				style = stCursor
			}
			row = style.Render(fit(stripStyles(entryRow(e, w, l.IsSelected(i), "")), w))
		}
		out = append(out, row)
	}
	return out
}

func renderPlain(items []entry, w, h int, playing string) []string {
	var out []string
	for i, e := range items {
		if i >= h {
			break
		}
		out = append(out, entryRow(e, w, false, playing))
	}
	return out
}

func entryRow(e entry, w int, selected bool, playing string) string {
	mark := "  "
	switch {
	case selected:
		mark = stAccent.Render("◆ ")
	case e.isLeaf() && e.track.Path == playing && playing != "":
		mark = stAccent.Render("▶ ")
	}
	label := stText.Render(e.label)
	if !e.isLeaf() {
		label = stBright.Render(e.label)
	}
	detail := stDim.Render(e.detail)
	dw := len(e.detail)
	return mark + fit(label, max(1, w-2-dw-1)) + " " + detail
}
