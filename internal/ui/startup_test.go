package ui

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/matjam/encomplayer/internal/config"
	"github.com/matjam/encomplayer/internal/domain"
	"github.com/matjam/encomplayer/internal/library"
	"github.com/matjam/encomplayer/internal/tea"
)

// recordingScanner remembers the options it was scanned with.
type recordingScanner struct {
	Scanner
	opts []library.Options
}

func (r *recordingScanner) Scan(_ context.Context, root string, _ *library.Snapshot, opts library.Options) (*library.Snapshot, library.Changes, error) {
	r.opts = append(r.opts, opts)
	return &library.Snapshot{Root: root}, library.Changes{}, nil
}

// emptyCache has never seen a scan.
type emptyCache struct{ LibraryCache }

func (emptyCache) Load(root string) (*library.Snapshot, error) {
	return &library.Snapshot{Root: root}, nil
}

func (emptyCache) Save(*library.Snapshot) error { return nil }

func newTestModelWith(t *testing.T, s Startup) (*Model, *fakePlayer) {
	t.Helper()
	return buildTestModel(t, filepath.Join(t.TempDir(), "state.json"), s)
}

func TestStartupTheme(t *testing.T) {
	m, _ := newTestModelWith(t, Startup{Theme: "dracula"})
	if m.st.theme.Name != "dracula" {
		t.Fatalf("theme = %q, want dracula", m.st.theme.Name)
	}

	// Saving an unrelated setting must not persist the one-off theme.
	m.updateConfig(func(c *config.Config) { c.VolumeStep = 9 })
	if got := savedConfig(t, m).Theme; got != "encom" {
		t.Errorf("saved theme = %q, want encom", got)
	}
}

func TestStartupNoMouse(t *testing.T) {
	m, _ := newTestModelWith(t, Startup{NoMouse: true})
	if m.View().MouseMode != 0 {
		t.Error("--no-mouse still enabled mouse reporting")
	}
	click(m, tabX(m, 2), tabBarRow)
	if m.active == 2 {
		t.Error("click handled despite --no-mouse")
	}
	if !m.deps.Config.EnableMouse {
		t.Error("--no-mouse changed the config setting")
	}
}

func TestStartupShuffle(t *testing.T) {
	m, fp := newTestModelWith(t, Startup{Shuffle: true})
	if m.queue.Len() != len(testTracks("/music")) || len(fp.played) != 1 {
		t.Fatalf("--shuffle queued %d and played %d", m.queue.Len(), len(fp.played))
	}

	// Later library updates must not reshuffle.
	m.scan = scanState{running: true, background: true, root: "/music"}
	more := append(testTracks("/music"), domain.Track{Path: "/music/x.mp3"})
	m.finishScan(scanDoneMsg{root: "/music", snap: &library.Snapshot{Root: "/music", Tracks: more}, changes: library.Changes{Added: 1}})
	if len(fp.played) != 1 {
		t.Error("a background sync reshuffled")
	}
}

func TestStartupFullRescan(t *testing.T) {
	for _, full := range []bool{false, true} {
		m, _ := newTestModelWith(t, Startup{FullRescan: full})
		scanner := &recordingScanner{}
		m.deps.Scanner, m.deps.Cache = scanner, emptyCache{}

		cmd := m.receiveCache(cacheLoadedMsg{root: "/music", snap: &library.Snapshot{Root: "/music"}})
		drain(cmd)
		if len(scanner.opts) != 1 || scanner.opts[0].Full != full {
			t.Errorf("FullRescan=%v: scans %+v", full, scanner.opts)
		}
		m.scan.running = false
	}
}

// drain runs a command and any batch it returns, discarding the messages.
// Scans run on a goroutine, so this is how a test waits for one.
func drain(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	if batch, ok := cmd().(tea.BatchMsg); ok {
		for _, c := range batch {
			drain(c)
		}
	}
}
