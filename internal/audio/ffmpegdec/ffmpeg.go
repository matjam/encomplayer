// Package ffmpegdec decodes any format ffmpeg understands by streaming
// signed 16-bit PCM from an ffmpeg subprocess. It covers formats without a
// pure-Go decoder, such as AAC, ALAC and Opus, and backs up the pure-Go
// decoders for files they reject.
package ffmpegdec

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gopxl/beep/v2"

	"github.com/matjam/encomplayer/internal/audio"
)

// ErrNotInstalled means ffmpeg or ffprobe is not on PATH.
var ErrNotInstalled = errors.New("ffmpeg not installed")

// Extensions are the formats handed to ffmpeg.
var Extensions = []string{
	"m4a", "m4b", "mp4", "aac", "alac", "opus", "ogg", "oga", "wma",
	"aiff", "aif", "ape", "wv", "mka", "webm", "ac3", "dsf", "mp3", "flac", "wav",
}

const (
	rate     = 44100
	channels = 2
	frame    = channels * 2
)

// Decoder runs ffmpeg for each track.
type Decoder struct {
	ffmpeg  string
	ffprobe string
}

// New locates ffmpeg and ffprobe on PATH.
func New() (*Decoder, error) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNotInstalled, err)
	}
	ffprobe, err := exec.LookPath("ffprobe")
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNotInstalled, err)
	}
	return &Decoder{ffmpeg: ffmpeg, ffprobe: ffprobe}, nil
}

// Register adds d to r for every extension in Extensions. Register it after
// the pure-Go decoders so they keep priority.
func Register(r *audio.Decoders, d *Decoder) {
	r.Register("ffmpeg", d, Extensions...)
}

// Decode implements audio.Decoder.
func (d *Decoder) Decode(ctx context.Context, path string) (beep.StreamSeekCloser, beep.Format, error) {
	length, err := d.probe(ctx, path)
	if err != nil {
		return nil, beep.Format{}, err
	}
	s := &stream{ffmpeg: d.ffmpeg, path: path, length: length}
	if err := s.start(0); err != nil {
		return nil, beep.Format{}, err
	}
	format := beep.Format{SampleRate: rate, NumChannels: channels, Precision: 2}
	return s, format, nil
}

// Probe implements audio.Prober.
func (d *Decoder) Probe(ctx context.Context, path string) (time.Duration, error) {
	n, err := d.probe(ctx, path)
	return beep.SampleRate(rate).D(n), err
}

func (d *Decoder) probe(ctx context.Context, path string) (int, error) {
	out, err := exec.CommandContext(ctx, d.ffprobe,
		"-v", "error",
		"-select_streams", "a:0",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		path,
	).Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe %s: %w", path, err)
	}
	secs, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
	if err != nil {
		return 0, fmt.Errorf("ffprobe %s: no audio duration: %w", path, err)
	}
	return beep.SampleRate(rate).N(time.Duration(secs * float64(time.Second))), nil
}

// stream reads PCM from a running ffmpeg. Seeking restarts ffmpeg at the new
// offset, which is cheap because ffmpeg seeks the input before decoding.
type stream struct {
	ffmpeg string
	path   string
	length int

	cmd *exec.Cmd
	out io.ReadCloser
	r   *bufio.Reader
	pos int
	buf []byte
	err error
}

func (s *stream) start(pos int) error {
	s.stop()
	offset := beep.SampleRate(rate).D(pos).Seconds()

	cmd := exec.Command(s.ffmpeg,
		"-nostdin", "-v", "error",
		"-ss", strconv.FormatFloat(offset, 'f', 3, 64),
		"-i", s.path,
		"-vn", "-f", "s16le", "-acodec", "pcm_s16le",
		"-ac", strconv.Itoa(channels), "-ar", strconv.Itoa(rate),
		"-",
	)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("ffmpeg %s: %w", s.path, err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("ffmpeg %s: %w", s.path, err)
	}
	s.cmd, s.out, s.r, s.pos = cmd, out, bufio.NewReaderSize(out, 64<<10), pos
	return nil
}

// Stream implements beep.Streamer.
func (s *stream) Stream(samples [][2]float64) (int, bool) {
	if s.r == nil {
		return 0, false
	}
	need := len(samples) * frame
	if cap(s.buf) < need {
		s.buf = make([]byte, need)
	}
	buf := s.buf[:need]

	read, err := io.ReadFull(s.r, buf)
	n := read / frame
	for i := range n {
		b := buf[i*frame:]
		samples[i][0] = float64(int16(uint16(b[0])|uint16(b[1])<<8)) / 32768
		samples[i][1] = float64(int16(uint16(b[2])|uint16(b[3])<<8)) / 32768
	}
	s.pos += n

	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		s.err = fmt.Errorf("ffmpeg %s: %w", s.path, err)
	}
	return n, n > 0
}

// Err implements beep.Streamer.
func (s *stream) Err() error { return s.err }

// Len implements beep.StreamSeeker.
func (s *stream) Len() int { return s.length }

// Position implements beep.StreamSeeker.
func (s *stream) Position() int { return s.pos }

// Seek implements beep.StreamSeeker.
func (s *stream) Seek(p int) error { return s.start(max(0, min(p, s.length))) }

// Close implements beep.StreamSeekCloser.
func (s *stream) Close() error {
	s.stop()
	return nil
}

// stop kills ffmpeg. Its exit status is irrelevant because the process was
// killed on purpose.
func (s *stream) stop() {
	if s.cmd == nil {
		return
	}
	_ = s.cmd.Process.Kill()
	_ = s.out.Close()
	_ = s.cmd.Wait()
	s.cmd, s.out, s.r = nil, nil, nil
}
