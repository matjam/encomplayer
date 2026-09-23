// Package ffprobe reads tags with ffprobe, covering formats the pure-Go
// reader cannot, such as WAV INFO chunks, AIFF and some Ogg streams.
package ffprobe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/matjam/encomplayer/internal/library"
)

// ErrNotInstalled means ffprobe is not on PATH.
var ErrNotInstalled = errors.New("ffprobe not installed")

// Reader runs ffprobe per file.
type Reader struct {
	bin string
}

// New locates ffprobe.
func New() (*Reader, error) {
	bin, err := exec.LookPath("ffprobe")
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNotInstalled, err)
	}
	return &Reader{bin: bin}, nil
}

type probeOutput struct {
	Format struct {
		Tags map[string]string `json:"tags"`
	} `json:"format"`
	Streams []struct {
		Tags map[string]string `json:"tags"`
	} `json:"streams"`
}

// ReadTags implements library.TagReader. Container and stream tags are
// merged because Ogg keeps them on the stream and most others on the
// container.
func (r *Reader) ReadTags(ctx context.Context, path string) (library.Tags, error) {
	out, err := exec.CommandContext(ctx, r.bin,
		"-v", "error",
		"-show_entries", "format_tags:stream_tags",
		"-select_streams", "a:0",
		"-of", "json",
		path,
	).Output()
	if err != nil {
		return library.Tags{}, fmt.Errorf("ffprobe %s: %w", path, err)
	}

	var p probeOutput
	if err := json.Unmarshal(out, &p); err != nil {
		return library.Tags{}, fmt.Errorf("ffprobe %s: %w", path, err)
	}

	tags := map[string]string{}
	merge := func(m map[string]string) {
		for k, v := range m {
			tags[normalise(k)] = v
		}
	}
	for _, s := range p.Streams {
		merge(s.Tags)
	}
	merge(p.Format.Tags)

	return library.Tags{
		Title:       tags["title"],
		Artist:      tags["artist"],
		AlbumArtist: first(tags, "albumartist", "album_artist"),
		Album:       tags["album"],
		Genre:       tags["genre"],
		Year:        leadingInt(first(tags, "date", "year", "originaldate")),
		Track:       leadingInt(first(tags, "track", "tracknumber")),
		Disc:        leadingInt(first(tags, "disc", "discnumber")),
	}, nil
}

func normalise(k string) string {
	return strings.ReplaceAll(strings.ToLower(k), " ", "")
}

func first(tags map[string]string, keys ...string) string {
	for _, k := range keys {
		if v := tags[k]; v != "" {
			return v
		}
	}
	return ""
}

// leadingInt parses "3/12" as 3 and "2010-05-17" as 2010.
func leadingInt(s string) int {
	end := 0
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	n, _ := strconv.Atoi(s[:end])
	return n
}
