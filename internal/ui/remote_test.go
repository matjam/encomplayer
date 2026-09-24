package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/matjam/encomplayer/internal/audio"
	"github.com/matjam/encomplayer/internal/remote"
)

// send runs a remote command through the event loop, as the socket does.
func send(m *Model, cmd string, args ...string) remote.Response {
	reply := make(chan remote.Response, 1)
	m.Update(RemoteMsg{Req: remote.Request{Cmd: cmd, Args: args}, Reply: reply})
	return <-reply
}

func TestRemoteCommands(t *testing.T) {
	tests := []struct {
		name  string
		setup func(m *Model, fp *fakePlayer)
		cmd   string
		args  []string
		check func(t *testing.T, m *Model, fp *fakePlayer, r remote.Response)
	}{
		{
			name: "status reports the current track", cmd: "status",
			check: func(t *testing.T, _ *Model, _ *fakePlayer, r remote.Response) {
				s := r.Status
				if s.State != "playing" || s.Title != "Overture" || s.Artist != "Daft Punk" || s.QueueLength != 4 || s.Volume != 70 {
					t.Errorf("status = %+v", s)
				}
			},
		},
		{
			name: "next advances", cmd: "next",
			check: func(t *testing.T, _ *Model, fp *fakePlayer, r remote.Response) {
				if !strings.HasSuffix(fp.played[len(fp.played)-1], "02.mp3") || r.Status.Title != "The Grid" {
					t.Errorf("played %v, status title %q", fp.played, r.Status.Title)
				}
			},
		},
		{
			name: "pause when playing pauses", cmd: "pause",
			check: func(t *testing.T, _ *Model, fp *fakePlayer, _ remote.Response) {
				if fp.toggles != 1 {
					t.Errorf("toggles = %d, want 1", fp.toggles)
				}
			},
		},
		{
			name: "play when playing does nothing", cmd: "play",
			check: func(t *testing.T, _ *Model, fp *fakePlayer, _ remote.Response) {
				if fp.toggles != 0 || len(fp.played) != 1 {
					t.Errorf("toggles %d, plays %d", fp.toggles, len(fp.played))
				}
			},
		},
		{
			name: "play when stopped starts", setup: func(_ *Model, fp *fakePlayer) { fp.state = audio.Stopped }, cmd: "play",
			check: func(t *testing.T, _ *Model, fp *fakePlayer, _ remote.Response) {
				if len(fp.played) != 2 {
					t.Errorf("plays = %d, want 2", len(fp.played))
				}
			},
		},
		{
			name: "relative seek", setup: func(_ *Model, fp *fakePlayer) { fp.pos = time.Minute }, cmd: "seek", args: []string{"-15"},
			check: func(t *testing.T, _ *Model, fp *fakePlayer, _ remote.Response) {
				if fp.seeks[0] != -15*time.Second {
					t.Errorf("seek delta = %v", fp.seeks)
				}
			},
		},
		{
			name: "absolute seek", setup: func(_ *Model, fp *fakePlayer) { fp.pos = time.Minute }, cmd: "seek", args: []string{"1:30"},
			check: func(t *testing.T, _ *Model, fp *fakePlayer, _ remote.Response) {
				if fp.seeks[0] != 30*time.Second {
					t.Errorf("seek delta = %v, want +30s", fp.seeks)
				}
			},
		},
		{
			name: "volume step", cmd: "volume", args: []string{"+5"},
			check: func(t *testing.T, _ *Model, fp *fakePlayer, r remote.Response) {
				if fp.volume != 75 || r.Status.Volume != 75 {
					t.Errorf("volume = %d", fp.volume)
				}
			},
		},
		{
			name: "volume absolute", cmd: "volume", args: []string{"20"},
			check: func(t *testing.T, _ *Model, fp *fakePlayer, _ remote.Response) {
				if fp.volume != 20 {
					t.Errorf("volume = %d", fp.volume)
				}
			},
		},
		{
			name: "mode on", cmd: "repeat", args: []string{"on"},
			check: func(t *testing.T, m *Model, _ *fakePlayer, r remote.Response) {
				if !m.modes.Repeat || !r.Status.Repeat {
					t.Error("repeat not on")
				}
			},
		},
		{
			name: "mode toggle", setup: func(m *Model, _ *fakePlayer) { m.modes.Random = true }, cmd: "random",
			check: func(t *testing.T, m *Model, _ *fakePlayer, _ remote.Response) {
				if m.modes.Random {
					t.Error("random still on after toggle")
				}
			},
		},
		{
			name: "shuffle-all replaces the queue", cmd: "shuffle-all",
			check: func(t *testing.T, m *Model, fp *fakePlayer, _ remote.Response) {
				if m.queue.Len() != 4 || len(fp.played) != 2 {
					t.Errorf("queue %d, plays %d", m.queue.Len(), len(fp.played))
				}
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m, fp := newTestModel(t)
			m.enqueue(testTracks("/music"))
			m.playIndex(0)
			if tc.setup != nil {
				tc.setup(m, fp)
			}
			r := send(m, tc.cmd, tc.args...)
			if !r.OK {
				t.Fatalf("%s failed: %s", tc.cmd, r.Error)
			}
			tc.check(t, m, fp, r)
		})
	}
}

func TestRemoteErrors(t *testing.T) {
	tests := []struct {
		cmd  string
		args []string
		want string
	}{
		{cmd: "dance", want: "unknown command"},
		{cmd: "seek", args: []string{"soon"}, want: "want +N"},
		{cmd: "seek", args: []string{"9:00"}, want: "past the end"},
		{cmd: "volume", args: []string{"loud"}, want: "want 0-100"},
		{cmd: "repeat", args: []string{"maybe"}, want: "want on, off or toggle"},
	}
	for _, tc := range tests {
		t.Run(tc.cmd+" "+strings.Join(tc.args, " "), func(t *testing.T) {
			m, _ := newTestModel(t)
			m.enqueue(testTracks("/music"))
			m.playIndex(0)
			r := send(m, tc.cmd, tc.args...)
			if r.OK || !strings.Contains(r.Error, tc.want) {
				t.Errorf("response = %+v, want an error containing %q", r, tc.want)
			}
			if r.Status == nil {
				t.Error("errors should still report status")
			}
		})
	}
}
