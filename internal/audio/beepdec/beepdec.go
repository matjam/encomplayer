// Package beepdec registers pure-Go decoders for MP3, FLAC, Ogg Vorbis and WAV.
package beepdec

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/vorbis"
	"github.com/gopxl/beep/v2/wav"

	"github.com/matjam/encomplayer/internal/audio"
)

// Register adds the beep decoders to r.
func Register(r *audio.Decoders) {
	r.Register("beep-mp3", decoder("mp3", mp3.Decode), "mp3")
	r.Register("beep-flac", flacDecoder{}, "flac")
	r.Register("beep-vorbis", decoder("vorbis", vorbis.Decode), "ogg", "oga")
	r.Register("beep-wav", decoder("wav", readerDecode(wav.Decode)), "wav")
}

type decodeFunc func(io.ReadCloser) (beep.StreamSeekCloser, beep.Format, error)

// readerDecode adapts decoders that take an io.Reader. They close the reader
// themselves when it implements io.Closer.
func readerDecode(fn func(io.Reader) (beep.StreamSeekCloser, beep.Format, error)) decodeFunc {
	return func(rc io.ReadCloser) (beep.StreamSeekCloser, beep.Format, error) { return fn(rc) }
}

func decoder(name string, fn decodeFunc) audio.Decoder {
	return audio.DecoderFunc(func(_ context.Context, path string) (beep.StreamSeekCloser, beep.Format, error) {
		f, err := os.Open(path)
		if err != nil {
			return nil, beep.Format{}, fmt.Errorf("%s: %w", name, err)
		}
		s, format, err := fn(f)
		if err != nil {
			f.Close()
			return nil, beep.Format{}, fmt.Errorf("%s: %w", name, err)
		}
		return s, format, nil
	})
}
