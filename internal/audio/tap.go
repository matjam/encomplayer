package audio

import (
	"sync"

	"github.com/gopxl/beep/v2"
)

// tapSize is how many recent samples per channel the tap keeps.
const tapSize = 2048

// Tap passes samples through unchanged and keeps the most recent window of
// each channel for visualisers.
type Tap struct {
	streamer beep.Streamer

	mu          sync.Mutex
	left, right [tapSize]float64
	pos         int
}

// NewTap wraps s.
func NewTap(s beep.Streamer) *Tap { return &Tap{streamer: s} }

// Stream implements beep.Streamer.
func (t *Tap) Stream(samples [][2]float64) (int, bool) {
	n, ok := t.streamer.Stream(samples)
	t.mu.Lock()
	for _, s := range samples[:n] {
		t.left[t.pos], t.right[t.pos] = s[0], s[1]
		t.pos = (t.pos + 1) % tapSize
	}
	t.mu.Unlock()
	return n, ok
}

// Err implements beep.Streamer.
func (t *Tap) Err() error { return t.streamer.Err() }

// Window copies the most recent samples, oldest first.
func (t *Tap) Window() (left, right []float64) {
	left, right = make([]float64, tapSize), make([]float64, tapSize)
	t.mu.Lock()
	defer t.mu.Unlock()
	n := copy(left, t.left[t.pos:])
	copy(left[n:], t.left[:t.pos])
	copy(right, t.right[t.pos:])
	copy(right[n:], t.right[:t.pos])
	return left, right
}
