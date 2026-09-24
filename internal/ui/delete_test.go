package ui

import (
	"strings"
	"testing"

	"github.com/matjam/encomplayer/internal/audio"
)

func TestDeletingPlayingTrackAdvances(t *testing.T) {
	tests := []struct {
		name       string
		play       int
		cursor     int
		keys       string
		setup      func(m *Model, fp *fakePlayer)
		wantPlayed string // suffix of the newest Play; "" means no new Play
		wantState  audio.State
		wantQueue  int
	}{
		{name: "plays the follower", play: 1, cursor: 1, keys: "d", wantPlayed: "03.ogg", wantState: audio.Playing, wantQueue: 3},
		{name: "other track keeps playing", play: 1, cursor: 3, keys: "d", wantState: audio.Playing, wantQueue: 3},
		{name: "last track stops", play: 3, cursor: 3, keys: "d", wantState: audio.Stopped, wantQueue: 3},
		{name: "last track wraps with repeat", play: 3, cursor: 3, keys: "zd", wantPlayed: "01.flac", wantState: audio.Playing, wantQueue: 3},
		{name: "selection including current", play: 1, cursor: 0, keys: "<Space><Space>d", wantPlayed: "03.ogg", wantState: audio.Playing, wantQueue: 2},
		{
			name: "paused stops on the follower", play: 1, cursor: 1, keys: "d",
			setup:     func(_ *Model, fp *fakePlayer) { fp.state = audio.Paused },
			wantState: audio.Stopped, wantQueue: 3,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m, fp := newTestModel(t)
			m.enqueue(testTracks("/music"))
			m.playIndex(tc.play)
			if tc.setup != nil {
				tc.setup(m, fp)
			}
			plays := len(fp.played)

			press(m, "1")
			m.queueList.SetCursor(tc.cursor)
			press(m, tc.keys)

			switch {
			case tc.wantPlayed == "" && len(fp.played) != plays:
				t.Errorf("unexpected Play of %s", fp.played[len(fp.played)-1])
			case tc.wantPlayed != "" && (len(fp.played) == plays || !strings.HasSuffix(fp.played[len(fp.played)-1], tc.wantPlayed)):
				t.Errorf("played %v, want a new Play of %s", fp.played, tc.wantPlayed)
			}
			if fp.state != tc.wantState {
				t.Errorf("player state = %v, want %v", fp.state, tc.wantState)
			}
			if m.queue.Len() != tc.wantQueue {
				t.Errorf("queue length = %d, want %d", m.queue.Len(), tc.wantQueue)
			}
		})
	}
}

func TestPausedDeleteResumesOnFollower(t *testing.T) {
	m, fp := newTestModel(t)
	m.enqueue(testTracks("/music"))
	m.playIndex(1)
	fp.state = audio.Paused

	press(m, "1")
	m.queueList.SetCursor(1)
	press(m, "dp")
	if last := fp.played[len(fp.played)-1]; !strings.HasSuffix(last, "03.ogg") {
		t.Errorf("p after deleting the paused track played %s, want 03.ogg", last)
	}
}
