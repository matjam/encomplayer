package ui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/matjam/encomplayer/internal/config"
)

// playAndQuit plays the second queued track, moves it to pos and quits,
// leaving the session state on disk.
func playAndQuit(t *testing.T, statePath string, pos time.Duration) {
	t.Helper()
	m, fp := newTestModelAt(t, statePath)
	m.enqueue(testTracks("/music"))
	m.playIndex(1)
	fp.pos = pos
	press(m, "q")
}

func TestResumeAfterRestart(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	playAndQuit(t, statePath, 95*time.Second)

	if got := config.LoadState(statePath).PositionSeconds; got != 95 {
		t.Fatalf("saved position = %vs, want 95s", got)
	}

	m, fp := newTestModelAt(t, statePath)
	m.Update(tickMsg{})
	if m.position != 95*time.Second {
		t.Errorf("seek bar shows %v before resuming, want 1m35s", m.position)
	}

	press(m, "1p")
	if len(fp.played) != 1 || !strings.HasSuffix(fp.played[0], "02.mp3") {
		t.Fatalf("p played %v, want 02.mp3", fp.played)
	}
	if fp.starts[0] != 95*time.Second {
		t.Errorf("p started at %v, want 1m35s", fp.starts[0])
	}
}

func TestResumeOnlyAppliesToP(t *testing.T) {
	tests := []struct {
		name      string
		keys      string
		wantStart time.Duration
	}{
		{name: "p resumes", keys: "1p", wantStart: 40 * time.Second},
		{name: "Enter on the same track starts over", keys: "1gg j<CR>", wantStart: 0},
		{name: "next track starts over", keys: "1>", wantStart: 0},
		{name: "second p after stop starts over", keys: "1ps p", wantStart: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			statePath := filepath.Join(t.TempDir(), "state.json")
			playAndQuit(t, statePath, 40*time.Second)

			m, fp := newTestModelAt(t, statePath)
			press(m, strings.ReplaceAll(tc.keys, " ", ""))
			if len(fp.starts) == 0 {
				t.Fatal("nothing played")
			}
			if got := fp.starts[len(fp.starts)-1]; got != tc.wantStart {
				t.Errorf("last play started at %v, want %v", got, tc.wantStart)
			}
		})
	}
}

func TestStopClearsSavedPosition(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	m, fp := newTestModelAt(t, statePath)
	m.enqueue(testTracks("/music"))
	m.playIndex(0)
	fp.pos = time.Minute
	press(m, "s")

	if got := config.LoadState(statePath).PositionSeconds; got != 0 {
		t.Errorf("position after stop = %vs, want 0", got)
	}
}

func TestPositionSavedWhilePlaying(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	m, fp := newTestModelAt(t, statePath)
	m.enqueue(testTracks("/music"))
	m.playIndex(0)

	fp.pos = 2 * time.Minute
	m.lastSaved = time.Now().Add(-positionSaveEvery)
	m.Update(tickMsg{})

	if got := config.LoadState(statePath).PositionSeconds; got != 120 {
		t.Errorf("periodic save wrote %vs, want 120s", got)
	}
}
