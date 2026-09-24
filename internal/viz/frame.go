package viz

import (
	"math"
	"time"
)

// Frame is the audio a visualiser draws from, taken just before it renders.
// Every slice is valid and full length even when nothing is playing, in
// which case the samples and levels are zero.
type Frame struct {
	// Time is how long this visualiser has been showing; Delta is the time
	// since its previous frame.
	Time, Delta time.Duration

	// Playing is false while stopped or paused.
	Playing bool

	SampleRate int

	// Left and Right are the most recent samples, oldest first, in
	// [-1, 1]. They are taken before the volume control, so quiet
	// listening still animates.
	Left, Right []float64

	// Spectrum is the magnitude of each FFT bin of the mono mix, mapped
	// from a 60 dB range onto [0, 1]. Bin i is centred on i × BinHz.
	Spectrum []float64
	BinHz    float64

	// Level is the loudness of the window, 0 for silence and 1 near full
	// scale, and Peak its largest absolute sample.
	Level, LevelLeft, LevelRight, Peak float64

	// Bass, Mid and Treble are the mean spectrum level over 40–250 Hz,
	// 250 Hz–4 kHz and 4–16 kHz.
	Bass, Mid, Treble float64

	// Beat is true on the frame a kick or other bass onset is detected.
	// BeatStrength says how far it stood out, from 0 to 1.
	Beat         bool
	BeatStrength float64

	Track Track

	bands map[int][]float64
}

// Track is what is playing.
type Track struct {
	Title, Artist, Album string
	Position, Duration   time.Duration
}

// Mono returns sample i of the mixed channels.
func (f *Frame) Mono(i int) float64 { return (f.Left[i] + f.Right[i]) / 2 }

// Bands groups the spectrum into n bands spaced logarithmically from 40 Hz
// to 16 kHz, as the ear hears pitch. Each band is the loudest bin in it.
// The result is cached, so calling Bands often is cheap.
func (f *Frame) Bands(n int) []float64 {
	if n <= 0 {
		return nil
	}
	if b, ok := f.bands[n]; ok {
		return b
	}
	out := make([]float64, n)
	if f.BinHz > 0 && len(f.Spectrum) > 1 {
		const lo, hi = 40.0, 16000.0
		for b := range n {
			f0 := lo * math.Pow(hi/lo, float64(b)/float64(n))
			f1 := lo * math.Pow(hi/lo, float64(b+1)/float64(n))
			i0 := min(len(f.Spectrum)-1, max(1, int(f0/f.BinHz)))
			i1 := min(len(f.Spectrum), max(i0+1, int(f1/f.BinHz)))
			for i := i0; i < i1; i++ {
				out[b] = math.Max(out[b], f.Spectrum[i])
			}
		}
	}
	if f.bands == nil {
		f.bands = map[int][]float64{}
	}
	f.bands[n] = out
	return out
}
