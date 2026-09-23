package beepdec

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestFLACProbe(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	dir := t.TempDir()
	plain := filepath.Join(dir, "tone.flac")
	gen := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "sine=duration=3", "-ar", "48000", plain)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("ffmpeg: %v: %s", err, out)
	}

	// A 10-byte ID3v2 header declaring a 5-byte body, then the FLAC.
	data, err := os.ReadFile(plain)
	if err != nil {
		t.Fatal(err)
	}
	prefixed := filepath.Join(dir, "id3.flac")
	id3 := append([]byte("ID3\x04\x00\x00\x00\x00\x00\x05"), bytes.Repeat([]byte{0}, 5)...)
	if err := os.WriteFile(prefixed, append(id3, data...), 0o644); err != nil {
		t.Fatal(err)
	}

	notFLAC := filepath.Join(dir, "fake.flac")
	if err := os.WriteFile(notFLAC, []byte("RIFF0000WAVEfmt "), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		path    string
		want    time.Duration
		wantErr bool
	}{
		{name: "plain", path: plain, want: 3 * time.Second},
		{name: "id3 prefix", path: prefixed, want: 3 * time.Second},
		{name: "not flac", path: notFLAC, wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := flacDecoder{}.Probe(context.Background(), tc.path)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("Probe = %v, want error", got)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Errorf("Probe = %v, %v; want %v", got, err, tc.want)
			}
		})
	}
}
