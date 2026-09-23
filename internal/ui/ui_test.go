package ui

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/matjam/encomplayer/internal/audio"
	"github.com/matjam/encomplayer/internal/config"
	"github.com/matjam/encomplayer/internal/domain"
	"github.com/matjam/encomplayer/internal/keymap"
	"github.com/matjam/encomplayer/internal/library"
	"github.com/matjam/encomplayer/internal/playlist"
)

// fakePlayer records calls. The embedded interface covers methods the tests
// never reach.
type fakePlayer struct {
	Player
	played []string
	state  audio.State
	volume int
	ended  chan uint64
}

func (f *fakePlayer) Play(_ context.Context, path string) (uint64, error) {
	f.played = append(f.played, path)
	f.state = audio.Playing
	return uint64(len(f.played)), nil
}
func (f *fakePlayer) Stop()                    { f.state = audio.Stopped }
func (f *fakePlayer) TogglePause()             {}
func (f *fakePlayer) Seek(time.Duration) error { return nil }
func (f *fakePlayer) Progress() (time.Duration, time.Duration) {
	return 30 * time.Second, 3 * time.Minute
}
func (f *fakePlayer) SetVolume(v int)          { f.volume = max(0, min(v, 100)) }
func (f *fakePlayer) Volume() int              { return f.volume }
func (f *fakePlayer) State() audio.State       { return f.state }
func (f *fakePlayer) Spectrum(n int) []float64 { return make([]float64, n) }
func (f *fakePlayer) Ended() <-chan uint64     { return f.ended }

func testTracks(root string) []domain.Track {
	mk := func(rel, title, artist, album, genre string, n int) domain.Track {
		return domain.Track{Path: filepath.Join(root, rel), Title: title, Artist: artist, AlbumArtist: artist, Album: album, Genre: genre, TrackNo: n, Year: 2010, Duration: 3 * time.Minute}
	}
	return []domain.Track{
		mk("Daft Punk/TRON Legacy/01.flac", "Overture", "Daft Punk", "TRON: Legacy", "Electronic", 1),
		mk("Daft Punk/TRON Legacy/02.mp3", "The Grid", "Daft Punk", "TRON: Legacy", "Electronic", 2),
		mk("Daft Punk/TRON Legacy/03.ogg", "Derezzed", "Daft Punk", "TRON: Legacy", "Electronic", 3),
		mk("Wendy Carlos/TRON/01.mp3", "Anthem", "Wendy Carlos", "TRON", "Soundtrack", 1),
	}
}

func newTestModel(t *testing.T) (*Model, *fakePlayer) {
	t.Helper()
	dir := t.TempDir()
	fp := &fakePlayer{volume: 70, ended: make(chan uint64)}
	m := New(context.Background(), Deps{
		Player:      fp,
		Playlists:   playlist.NewStore(filepath.Join(dir, "playlists")),
		Keymap:      keymap.Default(),
		ArtProtocol: "off",
		Config:      config.Default(),
		State:       config.DefaultState(),
		StatePath:   filepath.Join(dir, "state.json"),
		Version:     "test",
	})
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m.boot.skip()

	root := "/music"
	m.finishScan(scanDoneMsg{root: root, snap: &library.Snapshot{Root: root, Tracks: testTracks(root)}})
	return m, fp
}

func TestShuffleAllAndAllMusic(t *testing.T) {
	m, fp := newTestModel(t)

	press(m, "S")
	if m.queue.Len() != 4 || len(fp.played) != 1 {
		t.Fatalf("S queued %d and played %d, want 4 and 1", m.queue.Len(), len(fp.played))
	}

	press(m, "6gg")
	if got := screen(t, m); !strings.Contains(got, "ALL MUSIC") {
		t.Fatalf("playlists tab missing ALL MUSIC:\n%s", got)
	}
	press(m, "D")
	if m.modal != nil {
		t.Error("ALL MUSIC offered for deletion")
	}

	press(m, "X")
	if m.queue.Len() != 4 || len(fp.played) != 2 {
		t.Errorf("X on ALL MUSIC queued %d, played %d", m.queue.Len(), len(fp.played))
	}

	press(m, ":")
	typeText(m, "saveall Everything")
	press(m, "<CR>")
	p, err := m.deps.Playlists.Load("Everything")
	if err != nil || len(p.Paths) != 4 {
		t.Errorf(":saveall wrote %v, %v", p.Paths, err)
	}
}

func TestBackgroundSyncKeepsPosition(t *testing.T) {
	m, _ := newTestModel(t)
	press(m, "3jl") // Artists › Wendy Carlos › TRON
	m.scan = scanState{running: true, background: true, root: "/music"}

	added := append(testTracks("/music"), domain.Track{Path: "/music/New/x.mp3", Title: "New", Artist: "Aaa First"})
	m.finishScan(scanDoneMsg{root: "/music", snap: &library.Snapshot{Root: "/music", Tracks: added}, changes: library.Changes{Added: 1}})

	if len(m.lib.Tracks()) != 5 {
		t.Fatalf("library has %d tracks, want 5", len(m.lib.Tracks()))
	}
	artists := m.tabs[m.active].(*browserTab)
	if parent, _ := artists.browser.Parent().Current(); parent.label != "Wendy Carlos" {
		t.Errorf("cursor moved to %q after sync", parent.label)
	}
	if !strings.Contains(m.status.text, "+1 NEW") {
		t.Errorf("status = %q", m.status.text)
	}
}

// press feeds keys written in rmpc notation.
func press(m *Model, notation string) {
	keys, err := keymap.Parse(notation)
	if err != nil {
		panic(err)
	}
	for _, k := range keys {
		m.Update(keyMsg(k))
	}
}

func typeText(m *Model, s string) {
	for _, r := range s {
		m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}

func keyMsg(k string) tea.KeyPressMsg {
	switch k {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "space":
		return tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	}
	if rest, ok := strings.CutPrefix(k, "ctrl+"); ok {
		return tea.KeyPressMsg{Code: []rune(rest)[0], Mod: tea.ModCtrl}
	}
	return tea.KeyPressMsg{Code: []rune(k)[0], Text: k}
}

func screen(t *testing.T, m *Model) string {
	t.Helper()
	content := m.View().Content
	lines := strings.Split(content, "\n")
	if len(lines) != m.height {
		t.Fatalf("view has %d lines, want %d", len(lines), m.height)
	}
	for i, l := range lines {
		if w := ansi.StringWidth(l); w != m.width {
			t.Fatalf("line %d width %d, want %d: %q", i, w, m.width, ansi.Strip(l))
		}
	}
	return ansi.Strip(content)
}

func TestEveryTabRendersAtExactSize(t *testing.T) {
	m, _ := newTestModel(t)
	for i := range m.tabs {
		press(m, string(rune('1'+i)))
		if got := screen(t, m); !strings.Contains(got, strings.ToUpper(m.tabs[i].title())) {
			t.Errorf("tab %s not shown", m.tabs[i].title())
		}
	}
}

func TestBrowseAddAndPlay(t *testing.T) {
	m, fp := newTestModel(t)

	press(m, "3")     // Artists
	press(m, "l")     // into Daft Punk
	press(m, "a")     // add the album
	press(m, "1")     // Queue
	press(m, "G<CR>") // play the last track

	if m.queue.Len() != 3 {
		t.Fatalf("queue length = %d, want 3", m.queue.Len())
	}
	if len(fp.played) != 1 || !strings.HasSuffix(fp.played[0], "03.ogg") {
		t.Fatalf("played = %v, want 03.ogg", fp.played)
	}
	if got := screen(t, m); !strings.Contains(got, "Derezzed") || !strings.Contains(got, "PLAYING") {
		t.Errorf("queue screen missing now playing:\n%s", got)
	}

	press(m, "<")
	if last := fp.played[len(fp.played)-1]; !strings.HasSuffix(last, "02.mp3") {
		t.Errorf("previous played %s, want 02.mp3", last)
	}
}

func TestTrackEndAdvancesWithConsume(t *testing.T) {
	m, fp := newTestModel(t)
	m.enqueue(testTracks("/music"))
	m.playIndex(0)
	press(m, "c")

	m.Update(endedMsg(m.playGen))
	if m.queue.Len() != 3 {
		t.Errorf("consume left %d tracks, want 3", m.queue.Len())
	}
	if last := fp.played[len(fp.played)-1]; !strings.HasSuffix(last, "02.mp3") {
		t.Errorf("auto-advance played %s, want 02.mp3", last)
	}

	stale := m.playGen - 1
	m.Update(endedMsg(stale))
	if len(fp.played) != 2 {
		t.Errorf("stale end event advanced playback")
	}
}

func TestSearchAndSavePlaylist(t *testing.T) {
	m, _ := newTestModel(t)

	press(m, "7i")
	typeText(m, "tron")
	press(m, "<CR>")
	search := m.tabs[m.active].(*searchTab)
	if search.results.Len() != 4 {
		t.Fatalf("search found %d, want 4", search.results.Len())
	}

	press(m, "A") // add all results
	press(m, "1<C-s>a")
	typeText(m, "Grid Mix")
	press(m, "<CR>")

	names, err := m.deps.Playlists.List()
	if err != nil || len(names) != 1 || names[0] != "Grid Mix" {
		t.Fatalf("playlists = %v, %v", names, err)
	}
	press(m, "6")
	if got := screen(t, m); !strings.Contains(got, "Grid Mix") {
		t.Errorf("playlists tab missing saved playlist:\n%s", got)
	}
}

func TestQueueEditing(t *testing.T) {
	m, _ := newTestModel(t)
	m.enqueue(testTracks("/music"))
	press(m, "1gg")

	press(m, "J") // move first track down
	if got, _ := m.queue.At(1); got.Title != "Overture" {
		t.Errorf("after J, index 1 = %q, want Overture", got.Title)
	}

	press(m, "<Space><Space>d")
	if m.queue.Len() != 2 {
		t.Errorf("after deleting selection, len = %d, want 2", m.queue.Len())
	}

	press(m, "X")
	press(m, "D")
	if m.queue.Len() != 0 {
		t.Errorf("DeleteAll left %d tracks", m.queue.Len())
	}
}

func TestCommandModeAndModes(t *testing.T) {
	m, fp := newTestModel(t)

	press(m, "zxv")
	if !m.modes.Repeat || !m.modes.Random || !m.modes.Single {
		t.Errorf("modes = %+v", m.modes)
	}
	press(m, "..,")
	if fp.volume != 75 {
		t.Errorf("volume = %d, want 75", fp.volume)
	}

	press(m, ":")
	typeText(m, "volume 20")
	press(m, "<CR>")
	if fp.volume != 20 {
		t.Errorf(":volume set %d, want 20", fp.volume)
	}

	press(m, "?")
	if m.modal == nil {
		t.Fatal("help modal not open")
	}
	if got := screen(t, m); !strings.Contains(got, "◆ GLOBAL") || !strings.Contains(got, "AddRandom") {
		t.Errorf("help missing global bindings:\n%s", got)
	}
	press(m, "G")
	if got := screen(t, m); !strings.Contains(got, ":shuffleall") {
		t.Errorf("help did not scroll to command mode:\n%s", got)
	}
	press(m, "<Esc>")
	if m.modal != nil {
		t.Error("help modal did not close")
	}
}
