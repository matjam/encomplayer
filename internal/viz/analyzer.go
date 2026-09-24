package viz

import (
	"math"
	"math/cmplx"
	"time"
)

// Window is how many samples per channel a frame carries. At 44.1 kHz it is
// 46 ms of audio, and it gives the FFT 21.5 Hz bins.
const Window = 2048

// Beat detection compares bass energy with its recent average.
const (
	beatAverage  = time.Second
	beatRatio    = 1.45
	beatMinGap   = 200 * time.Millisecond
	beatMinPower = 1e-4
)

// Analyzer turns raw samples into Frames. It keeps the running averages
// that beat detection needs, so use one per stream of frames.
type Analyzer struct {
	hann []float64
	buf  []complex128

	clock, lastBeat time.Duration
	bassAverage     float64
	primed          bool
}

// NewAnalyzer returns an analyzer with no history.
func NewAnalyzer() *Analyzer {
	a := &Analyzer{hann: make([]float64, Window), buf: make([]complex128, Window), lastBeat: -time.Hour}
	for i := range Window {
		a.hann[i] = 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/(Window-1))
	}
	return a
}

// Analyze builds a frame from the latest samples. left and right may be
// shorter than Window, or empty when nothing plays; they are padded with
// silence at the start. dt is the time since the previous call.
func (a *Analyzer) Analyze(left, right []float64, rate int, dt time.Duration) *Frame {
	a.clock += dt
	f := &Frame{
		SampleRate: rate,
		Left:       padded(left),
		Right:      padded(right),
		Spectrum:   make([]float64, Window/2),
	}
	if rate > 0 {
		f.BinHz = float64(rate) / Window
	}

	var sumL, sumR float64
	for i := range Window {
		l, r := f.Left[i], f.Right[i]
		sumL += l * l
		sumR += r * r
		f.Peak = math.Max(f.Peak, math.Max(math.Abs(l), math.Abs(r)))
		a.buf[i] = complex((l+r)/2*a.hann[i], 0)
	}
	f.LevelLeft = loudness(math.Sqrt(sumL / Window))
	f.LevelRight = loudness(math.Sqrt(sumR / Window))
	f.Level = loudness(math.Sqrt((sumL + sumR) / (2 * Window)))

	fft(a.buf)
	var bassPower float64
	for i := range f.Spectrum {
		// A full-scale sine peaks near Window/4 after the Hann window.
		mag := cmplx.Abs(a.buf[i]) / (Window / 4)
		f.Spectrum[i] = clamp01((20*math.Log10(mag+1e-9) + 60) / 60)
		if hz := float64(i) * f.BinHz; hz >= 30 && hz <= 150 {
			bassPower += mag * mag
		}
	}
	f.Bass = f.meanLevel(40, 250)
	f.Mid = f.meanLevel(250, 4000)
	f.Treble = f.meanLevel(4000, 16000)
	a.detectBeat(f, bassPower, dt)
	return f
}

func (a *Analyzer) detectBeat(f *Frame, power float64, dt time.Duration) {
	if !a.primed {
		a.bassAverage, a.primed = power, true
		return
	}
	avg := a.bassAverage
	if power > beatMinPower && power > avg*beatRatio && a.clock-a.lastBeat >= beatMinGap {
		f.Beat = true
		f.BeatStrength = clamp01((power/math.Max(avg, beatMinPower) - 1) / 3)
		a.lastBeat = a.clock
	}
	a.bassAverage += (power - avg) * math.Min(1, dt.Seconds()/beatAverage.Seconds())
}

func (f *Frame) meanLevel(lo, hi float64) float64 {
	if f.BinHz == 0 {
		return 0
	}
	i0 := max(1, int(lo/f.BinHz))
	i1 := min(len(f.Spectrum), max(i0+1, int(hi/f.BinHz)))
	var sum float64
	for i := i0; i < i1; i++ {
		sum += f.Spectrum[i]
	}
	return sum / float64(i1-i0)
}

// loudness maps an RMS amplitude onto [0, 1] across 48 dB.
func loudness(rms float64) float64 {
	return clamp01((20*math.Log10(rms+1e-9) + 48) / 48)
}

// padded returns exactly Window samples: the newest of s, with silence
// before them if s is short.
func padded(s []float64) []float64 {
	out := make([]float64, Window)
	if len(s) > Window {
		s = s[len(s)-Window:]
	}
	copy(out[Window-len(s):], s)
	return out
}

// fft is an in-place iterative radix-2 Cooley-Tukey transform. len(x) must
// be a power of two.
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
