package audio_test

import (
	"context"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/matjam/encomplayer/internal/audio"
	"github.com/matjam/encomplayer/internal/audio/beepdec"
	"github.com/matjam/encomplayer/internal/audio/ffmpegdec"
)

// TestDecodeFormats generates a two-second tone in each format with ffmpeg
// and checks that the registry decodes it and reports its length.
func TestDecodeFormats(t *testing.T) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg not installed")
	}
	ff, err := ffmpegdec.New()
	if err != nil {
		t.Skip(err)
	}

	decoders := audio.NewDecoders()
	beepdec.Register(decoders)
	ffmpegdec.Register(decoders, ff)

	dir := t.TempDir()
	for _, ext := range []string{"mp3", "flac", "ogg", "wav", "m4a", "opus", "aiff"} {
		t.Run(ext, func(t *testing.T) {
			path := filepath.Join(dir, "tone."+ext)
			gen := exec.Command(ffmpeg, "-v", "error", "-f", "lavfi", "-i", "sine=frequency=440:duration=2", "-ac", "2", path)
			if out, err := gen.CombinedOutput(); err != nil {
				t.Skipf("ffmpeg cannot write %s: %v: %s", ext, err, out)
			}

			ctx := context.Background()
			length, err := audio.Duration(ctx, decoders, path)
			if err != nil {
				t.Fatalf("Duration: %v", err)
			}
			if length < 1900*time.Millisecond || length > 2200*time.Millisecond {
				t.Errorf("length = %v, want about 2s", length)
			}

			// Decode the way the player does, from the in-memory copy.
			// Duration above already exercised the on-disk source.
			src, err := audio.MemorySource(path)
			if err != nil {
				t.Fatalf("MemorySource: %v", err)
			}
			s, format, err := audio.Open(ctx, decoders, src)
			if err != nil {
				t.Fatalf("Open: %v", err)
			}
			defer s.Close()

			buf := make([][2]float64, 4096)
			var peak float64
			for range 8 {
				n, ok := s.Stream(buf)
				for _, v := range buf[:n] {
					peak = max(peak, v[0], -v[0])
				}
				if !ok {
					break
				}
			}
			// lavfi's sine source plays at 1/8 amplitude, about 0.088
			// per channel after the stereo upmix.
			if peak < 0.05 {
				t.Errorf("decoded audio is silent (peak %.3f)", peak)
			}

			if err := s.Seek(format.SampleRate.N(time.Second)); err != nil {
				t.Fatalf("Seek: %v", err)
			}
			if pos := format.SampleRate.D(s.Position()); pos < 900*time.Millisecond || pos > 1100*time.Millisecond {
				t.Errorf("position after seek = %v, want about 1s", pos)
			}
		})
	}
}
