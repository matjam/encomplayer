package ui

import (
	"fmt"
	"testing"
	"time"

	"github.com/matjam/encomplayer/internal/domain"
)

func manyTracks(n int) []domain.Track {
	out := make([]domain.Track, n)
	for i := range out {
		out[i] = domain.Track{Path: fmt.Sprintf("/music/%03d.mp3", i), Title: fmt.Sprintf("Track %d", i), Artist: "Daft Punk", Duration: time.Minute}
	}
	return out
}

func TestQueueFollowsPlayingTrack(t *testing.T) {
	tests := []struct {
		name    string
		touched time.Duration // how long ago the queue was last worked in
		advance func(m *Model)
		follows bool
	}{
		{name: "track ends while idle", touched: 2 * queueFollowIdle, advance: func(m *Model) { m.Update(endedMsg(m.playGen)) }, follows: true},
		{name: "next pressed while idle", touched: 2 * queueFollowIdle, advance: func(m *Model) { press(m, ">") }, follows: true},
		{name: "track ends while browsing", touched: queueFollowIdle / 2, advance: func(m *Model) { m.Update(endedMsg(m.playGen)) }, follows: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m, _ := newTestModel(t)
			m.enqueue(manyTracks(100))
			h := m.queueList.Height()
			last := h - h/4 - 1 // last row the playing track may sit on
			m.playIndex(last)
			m.queueList.SetCursor(0)
			m.queueTouched = time.Now().Add(-tc.touched)

			tc.advance(m)
			want := last + 1
			if got := m.queue.CurrentIndex(); got != want {
				t.Fatalf("current = %d, want %d", got, want)
			}
			if !tc.follows {
				if m.queueList.Cursor() != 0 {
					t.Errorf("cursor moved to %d while browsing", m.queueList.Cursor())
				}
				return
			}
			if m.queueList.Cursor() != want {
				t.Errorf("cursor = %d, want playing track %d", m.queueList.Cursor(), want)
			}
			if first, _ := m.queueList.IndexAtRow(0); first != 1 {
				t.Errorf("first visible = %d, want 1 (scrolled one row, %d rows below kept)", first, h/4)
			}
		})
	}
}

func TestQueueNavigationMarksTouched(t *testing.T) {
	m, _ := newTestModel(t)
	m.enqueue(manyTracks(10))
	press(m, "j")
	if time.Since(m.queueTouched) > time.Second {
		t.Errorf("moving the queue cursor did not count as working in the queue")
	}

	m.queueTouched = time.Time{}
	press(m, ">")
	if !m.queueTouched.IsZero() {
		t.Errorf("next-track counted as working in the queue")
	}
}
