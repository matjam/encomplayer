package audio

import (
	"testing"

	"github.com/gopxl/beep/v2"
)

// counting streams sample n as (n, -n).
type counting struct{ n float64 }

func (c *counting) Stream(samples [][2]float64) (int, bool) {
	for i := range samples {
		c.n++
		samples[i] = [2]float64{c.n, -c.n}
	}
	return len(samples), true
}

func (*counting) Err() error { return nil }

var _ beep.Streamer = (*counting)(nil)

func TestTapKeepsNewestSamplesInOrder(t *testing.T) {
	tap := NewTap(&counting{})
	buf := make([][2]float64, 1000)
	for range 5 {
		tap.Stream(buf) // 5,000 samples wraps the 2,048-sample ring
	}
	left, right := tap.Window()
	if len(left) != tapSize || len(right) != tapSize {
		t.Fatalf("window lengths %d, %d", len(left), len(right))
	}
	for i := range tapSize {
		want := float64(5000 - tapSize + 1 + i)
		if left[i] != want || right[i] != -want {
			t.Fatalf("sample %d = (%v, %v), want (%v, %v)", i, left[i], right[i], want, -want)
		}
	}
	if got := buf[999]; got != [2]float64{5000, -5000} {
		t.Errorf("tap changed the samples it passed on: %v", got)
	}
}
