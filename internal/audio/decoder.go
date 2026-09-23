// Package audio plays tracks through the system audio device. Decoding is
// pluggable: adapters register a Decoder per file extension, and Open tries
// them in priority order.
package audio

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gopxl/beep/v2"

	"github.com/matjam/encomplayer/internal/registry"
)

// ErrUnsupported means no registered decoder handles the file.
var ErrUnsupported = errors.New("unsupported audio format")

// Decoder opens an audio file as a seekable PCM stream.
type Decoder interface {
	Decode(ctx context.Context, path string) (beep.StreamSeekCloser, beep.Format, error)
}

// DecoderFunc adapts a function to Decoder.
type DecoderFunc func(ctx context.Context, path string) (beep.StreamSeekCloser, beep.Format, error)

// Decode calls f.
func (f DecoderFunc) Decode(ctx context.Context, path string) (beep.StreamSeekCloser, beep.Format, error) {
	return f(ctx, path)
}

// Decoders is the registry type adapters register into, keyed by extension.
type Decoders = registry.Registry[Decoder]

// NewDecoders returns an empty decoder registry.
func NewDecoders() *Decoders { return registry.New[Decoder]() }

// Open decodes path with the first registered decoder that accepts it.
func Open(ctx context.Context, decoders *Decoders, path string) (beep.StreamSeekCloser, beep.Format, error) {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	candidates := decoders.Lookup(ext)
	if len(candidates) == 0 {
		return nil, beep.Format{}, fmt.Errorf("open %s: %w", path, ErrUnsupported)
	}

	var errs []error
	for _, d := range candidates {
		s, format, err := d.Decode(ctx, path)
		if err == nil {
			return s, format, nil
		}
		errs = append(errs, err)
	}
	return nil, beep.Format{}, fmt.Errorf("open %s: %w", path, errors.Join(errs...))
}
