package ui

import (
	"fmt"
	"math/rand/v2"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matjam/encomplayer/internal/config"
	"github.com/matjam/encomplayer/internal/domain"
)

func TestCarefulToggle(t *testing.T) {
	tests := []struct {
		name   string
		toggle func(t *testing.T, m *Model)
	}{
		{name: "Z key", toggle: func(_ *testing.T, m *Model) { press(m, "Z") }},
		{name: "command", toggle: func(_ *testing.T, m *Model) {
			press(m, ":")
			typeText(m, "careful")
			press(m, "<CR>")
		}},
		{name: "remote", toggle: func(t *testing.T, m *Model) {
			if r := send(m, "careful", "on"); !r.OK || !r.Status.Careful {
				t.Fatalf("remote careful on: ok %t, status careful %t, error %q", r.OK, r.Status.Careful, r.Error)
			}
		}},
		{name: "header click", toggle: func(t *testing.T, m *Model) {
			for x := range m.width {
				if i, hit := m.modeAt(x); hit && modeNames[i] == "CAREFUL" {
					click(m, x, modesRow)
					return
				}
			}
			t.Fatal("no CAREFUL flag in the header")
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			statePath := filepath.Join(t.TempDir(), "state.json")
			m, _ := newTestModelAt(t, statePath)
			tc.toggle(t, m)
			if !m.modes.Careful {
				t.Fatal("careful mode not on")
			}
			if !config.LoadState(statePath).Modes.Careful {
				t.Error("careful mode not saved")
			}
			if !strings.Contains(screen(t, m), "CAREFUL") {
				t.Error("header does not show CAREFUL")
			}
		})
	}
}

// artistTracks returns count tracks for each artist.
func artistTracks(counts map[string]int) []domain.Track {
	var out []domain.Track
	for artist, n := range counts {
		for i := range n {
			out = append(out, domain.Track{Path: fmt.Sprintf("/music/%s/%02d.flac", artist, i), Artist: artist})
		}
	}
	return out
}

func TestShuffleQueueCarefully(t *testing.T) {
	tests := []struct {
		name        string
		careful     bool
		wantRepeats bool // whether any shuffle may put an artist twice in a row
	}{
		{name: "careful keeps artists apart", careful: true, wantRepeats: false},
		{name: "plain shuffle does not", careful: false, wantRepeats: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m, _ := newTestModel(t)
			m.enqueue(artistTracks(map[string]int{"Daft Punk": 3, "Wendy Carlos": 3, "Aphex Twin": 2}))
			m.modes.Careful = tc.careful

			sawRepeat := false
			for seed := range uint64(50) {
				m.rng = rand.New(rand.NewPCG(seed, 1))
				press(m, "1X")
				items := m.queue.Items()
				for i := 1; i < len(items); i++ {
					if items[i].ArtistKey() == items[i-1].ArtistKey() {
						sawRepeat = true
					}
				}
			}
			if sawRepeat != tc.wantRepeats {
				t.Errorf("an artist played twice in a row: %t, want %t", sawRepeat, tc.wantRepeats)
			}
		})
	}
}
