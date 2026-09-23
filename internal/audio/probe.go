package audio

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Prober is implemented by decoders that can report a track's length without
// decoding it.
type Prober interface {
	Probe(ctx context.Context, path string) (time.Duration, error)
}

// Duration returns the length of the track at path, using the first decoder
// that can read it.
func Duration(ctx context.Context, decoders *Decoders, path string) (time.Duration, error) {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	var errs []error
	for _, d := range decoders.Lookup(ext) {
		if p, ok := d.(Prober); ok {
			length, err := p.Probe(ctx, path)
			if err == nil {
				return length, nil
			}
			errs = append(errs, err)
			continue
		}
		s, format, err := d.Decode(ctx, path)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		length := format.SampleRate.D(s.Len())
		_ = s.Close() // Read-only probe; nothing to recover on close failure.
		return length, nil
	}
	if len(errs) == 0 {
		return 0, fmt.Errorf("duration %s: %w", path, ErrUnsupported)
	}
	return 0, fmt.Errorf("duration %s: %w", path, errors.Join(errs...))
}
