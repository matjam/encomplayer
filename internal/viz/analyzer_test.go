package viz

import (
	"math"
	"slices"
	"testing"
	"time"
)

const rate = 44100

func tone(hz, amp float64, n int) []float64 {
	s := make([]float64, n)
	for i := range s {
		s[i] = amp * math.Sin(2*math.Pi*hz*float64(i)/rate)
	}
	return s
}

func TestAnalyzeTone(t *testing.T) {
	a := NewAnalyzer()
	s := tone(1000, 0.5, Window)
	f := a.Analyze(s, s, rate, 33*time.Millisecond)

	peak := 0
	for i, v := range f.Spectrum {
		if v > f.Spectrum[peak] {
			peak = i
		}
	}
	if hz := float64(peak) * f.BinHz; math.Abs(hz-1000) > f.BinHz {
		t.Errorf("loudest bin at %.0f Hz, want 1000", hz)
	}
	if f.Mid <= f.Bass || f.Mid <= f.Treble {
		t.Errorf("1 kHz should be mid: bass %.2f mid %.2f treble %.2f", f.Bass, f.Mid, f.Treble)
	}
	if f.Level < 0.7 || math.Abs(f.Peak-0.5) > 0.01 {
		t.Errorf("level %.2f peak %.2f for a -6 dB tone", f.Level, f.Peak)
	}

	bands := f.Bands(16)
	loudest := slices.Index(bands, slices.Max(bands))
	if loudest < 6 || loudest > 10 {
		t.Errorf("1 kHz lands in band %d of 16", loudest)
	}
	if &f.Bands(16)[0] != &bands[0] {
		t.Error("Bands is not cached")
	}
}

func TestAnalyzeSilence(t *testing.T) {
	f := NewAnalyzer().Analyze(nil, nil, rate, 0)
	if len(f.Left) != Window || len(f.Right) != Window || len(f.Spectrum) != Window/2 {
		t.Fatalf("lengths %d %d %d", len(f.Left), len(f.Right), len(f.Spectrum))
	}
	if f.Level != 0 || f.Peak != 0 || f.Beat || slices.Max(f.Spectrum) != 0 {
		t.Errorf("silence produced level %v peak %v beat %v", f.Level, f.Peak, f.Beat)
	}
	for _, v := range f.Bands(10) {
		if math.IsNaN(v) {
			t.Fatal("NaN band")
		}
	}
}

// A kick drum every 500 ms should be found as beats, and a steady tone
// should not.
func TestBeats(t *testing.T) {
	a := NewAnalyzer()
	const frame = 20 * time.Millisecond
	steps := int(4 * time.Second / frame)
	beats := 0
	for i := range steps {
		now := time.Duration(i+1) * frame
		s := make([]float64, Window)
		for j := range s {
			at := now.Seconds() - float64(Window-j)/rate
			if since := math.Mod(at, 0.5); at > 0 && since < 0.06 {
				s[j] = 0.9 * math.Sin(2*math.Pi*55*since) * (1 - since/0.06)
			}
			s[j] += 0.05 * math.Sin(2*math.Pi*2000*at)
		}
		if a.Analyze(s, s, rate, frame).Beat {
			beats++
		}
	}
	if beats < 6 || beats > 9 {
		t.Errorf("found %d beats in 4 s of kicks every 500 ms, want about 8", beats)
	}

	steady := NewAnalyzer()
	s := tone(60, 0.8, Window)
	for i := range 100 {
		if steady.Analyze(s, s, rate, frame).Beat && i > 1 {
			t.Fatalf("steady tone detected as a beat at frame %d", i)
		}
	}
}
