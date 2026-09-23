// Package audio plays tracks through the system audio device. Decoding is
// pluggable: adapters register a Decoder per file extension, and Open tries
// them in priority order.
package audio

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/gopxl/beep/v2"

	"github.com/matjam/encomplayer/internal/registry"
)

// ErrUnsupported means no registered decoder handles the file.
var ErrUnsupported = errors.New("unsupported audio format")

// Source is an audio file to decode. Open returns a fresh reader positioned
// at the start; the player backs it with memory, the scanner with the disk.
type Source struct {
	Path string
	Open func() (io.ReadSeekCloser, error)
}

// FileSource reads path straight from disk.
func FileSource(path string) Source {
	return Source{Path: path, Open: func() (io.ReadSeekCloser, error) { return os.Open(path) }}
}

// Decoder opens an audio file as a seekable PCM stream.
type Decoder interface {
	Decode(ctx context.Context, src Source) (beep.StreamSeekCloser, beep.Format, error)
}

// PathReader is implemented by decoders that read the file by path in
// another process, such as ffmpeg. The player does not load those files into
// memory because the decoder would never read the copy.
type PathReader interface {
	ReadsPath()
}

// DecoderFunc adapts a function to Decoder.
type DecoderFunc func(ctx context.Context, src Source) (beep.StreamSeekCloser, beep.Format, error)

// Decode calls f.
func (f DecoderFunc) Decode(ctx context.Context, src Source) (beep.StreamSeekCloser, beep.Format, error) {
	return f(ctx, src)
}

// Decoders is the registry type adapters register into, keyed by extension.
type Decoders = registry.Registry[Decoder]

// NewDecoders returns an empty decoder registry.
func NewDecoders() *Decoders { return registry.New[Decoder]() }

// Open decodes src with the first registered decoder that accepts it.
func Open(ctx context.Context, decoders *Decoders, src Source) (beep.StreamSeekCloser, beep.Format, error) {
	candidates := decoders.Lookup(extOf(src.Path))
	if len(candidates) == 0 {
		return nil, beep.Format{}, fmt.Errorf("open %s: %w", src.Path, ErrUnsupported)
	}

	var errs []error
	for _, d := range candidates {
		s, format, err := d.Decode(ctx, src)
		if err == nil {
			return s, format, nil
		}
		errs = append(errs, err)
	}
	return nil, beep.Format{}, fmt.Errorf("open %s: %w", src.Path, errors.Join(errs...))
}

// readsInProcess reports whether any decoder for path reads through
// Source.Open, which is when loading the file into memory helps.
func readsInProcess(decoders *Decoders, path string) bool {
	for _, d := range decoders.Lookup(extOf(path)) {
		if _, ok := d.(PathReader); !ok {
			return true
		}
	}
	return false
}

func extOf(path string) string {
	return strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
}
