package audio

import (
	"math"
	"math/cmplx"
	"sync"

	"github.com/gopxl/beep/v2"
)

const windowSize = 2048

// Analyzer passes samples through unchanged and keeps the most recent window
// for spectrum display.
type Analyzer struct {
	streamer beep.Streamer
	rate     beep.SampleRate

	mu   sync.Mutex
	ring [windowSize]float64
	pos  int
}

// NewAnalyzer wraps s, which streams at rate.
func NewAnalyzer(s beep.Streamer, rate beep.SampleRate) *Analyzer {
	return &Analyzer{streamer: s, rate: rate}
}

// Stream implements beep.Streamer.
func (a *Analyzer) Stream(samples [][2]float64) (int, bool) {
	n, ok := a.streamer.Stream(samples)
	a.mu.Lock()
	for _, s := range samples[:n] {
		a.ring[a.pos] = (s[0] + s[1]) / 2
		a.pos = (a.pos + 1) % windowSize
	}
	a.mu.Unlock()
	return n, ok
}

// Err implements beep.Streamer.
func (a *Analyzer) Err() error { return a.streamer.Err() }

// Spectrum returns bands levels in [0, 1], spaced logarithmically from 40 Hz
// to 16 kHz.
func (a *Analyzer) Spectrum(bands int) []float64 {
	buf := make([]complex128, windowSize)
	a.mu.Lock()
	for i := range windowSize {
		hann := 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/(windowSize-1))
		buf[i] = complex(a.ring[(a.pos+i)%windowSize]*hann, 0)
	}
	a.mu.Unlock()

	fft(buf)

	const lo, hi = 40.0, 16000.0
	binHz := float64(a.rate) / windowSize
	out := make([]float64, bands)
	for b := range bands {
		f0 := lo * math.Pow(hi/lo, float64(b)/float64(bands))
		f1 := lo * math.Pow(hi/lo, float64(b+1)/float64(bands))
		i0 := max(1, int(f0/binHz))
		i1 := max(i0+1, int(f1/binHz))

		var peak float64
		for i := i0; i < i1 && i < windowSize/2; i++ {
			peak = max(peak, cmplx.Abs(buf[i]))
		}

		// Map roughly -60..0 dB onto 0..1.
		db := 20 * math.Log10(peak/(windowSize/4)+1e-9)
		out[b] = math.Max(0, math.Min(1, (db+60)/60))
	}
	return out
}

// fft is an in-place iterative radix-2 Cooley-Tukey transform. len(x) must be
// a power of two.
func fft(x []complex128) {
	n := len(x)
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			x[i], x[j] = x[j], x[i]
		}
	}
	for size := 2; size <= n; size <<= 1 {
		step := cmplx.Exp(complex(0, -2*math.Pi/float64(size)))
		for start := 0; start < n; start += size {
			w := complex(1, 0)
			for k := range size / 2 {
				even, odd := x[start+k], x[start+k+size/2]*w
				x[start+k] = even + odd
				x[start+k+size/2] = even - odd
				w *= step
			}
		}
	}
}
