package beepdec

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/flac"

	"github.com/matjam/encomplayer/internal/audio"
)

var errNotFLAC = errors.New("not a FLAC stream")

// flacDecoder decodes with beep and probes length from the STREAMINFO
// header. Opening a FLAC with beep builds a seek table by reading the whole
// file, which is far too slow for scanning a library over a network mount.
type flacDecoder struct{}

var _ audio.Prober = flacDecoder{}

func (flacDecoder) Decode(ctx context.Context, path string) (beep.StreamSeekCloser, beep.Format, error) {
	return decoder("flac", readerDecode(flac.Decode)).Decode(ctx, path)
}

// Probe implements audio.Prober.
func (flacDecoder) Probe(_ context.Context, path string) (time.Duration, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("flac probe: %w", err)
	}
	defer f.Close()

	d, err := streamInfoDuration(f)
	if err != nil {
		return 0, fmt.Errorf("flac probe %s: %w", path, err)
	}
	return d, nil
}

// streamInfoDuration reads the length from a FLAC stream's first metadata
// block, skipping a leading ID3v2 tag if one is present.
func streamInfoDuration(r io.ReadSeeker) (time.Duration, error) {
	var magic [4]byte
	if _, err := io.ReadFull(r, magic[:]); err != nil {
		return 0, err
	}
	if string(magic[:3]) == "ID3" {
		var hdr [6]byte
		if _, err := io.ReadFull(r, hdr[:]); err != nil {
			return 0, err
		}

		// ID3v2 sizes are syncsafe: 7 bits per byte.
		size := int64(hdr[2])<<21 | int64(hdr[3])<<14 | int64(hdr[4])<<7 | int64(hdr[5])
		if _, err := r.Seek(10+size, io.SeekStart); err != nil {
			return 0, err
		}
		if _, err := io.ReadFull(r, magic[:]); err != nil {
			return 0, err
		}
	}
	if string(magic[:]) != "fLaC" {
		return 0, errNotFLAC
	}

	// Block header (4 bytes), then STREAMINFO: sample rate is 20 bits at
	// byte 10, total samples is 36 bits ending at byte 17.
	var b [4 + 18]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, err
	}
	if b[0]&0x7F != 0 {
		return 0, errNotFLAC
	}
	si := b[4:]
	rate := int64(si[10])<<12 | int64(si[11])<<4 | int64(si[12])>>4
	total := int64(si[13]&0x0F)<<32 | int64(si[14])<<24 | int64(si[15])<<16 | int64(si[16])<<8 | int64(si[17])
	if rate == 0 || total == 0 {
		return 0, errors.New("FLAC header has no length")
	}
	return time.Duration(total * int64(time.Second) / rate), nil
}
