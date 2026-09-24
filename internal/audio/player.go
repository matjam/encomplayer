package audio

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/effects"
	"github.com/gopxl/beep/v2/speaker"
)

// State is the transport state of the player.
type State int

// Transport states.
const (
	Stopped State = iota
	Playing
	Paused
)

const (
	// outputRate is the device sample rate. Tracks at other rates are
	// resampled.
	outputRate = beep.SampleRate(44100)

	// outputBuffer is the device buffer. It only has to absorb CPU hiccups,
	// because tracks decode from memory rather than disk or network.
	outputBuffer = 100 * time.Millisecond
)

// Player plays one track at a time. It owns beep's process-wide speaker, so a
// program must create at most one.
//
// Tracks are read into memory in the background as they play, and the next
// track can be preloaded, so a slow disk or network share never stalls the
// audio callback.
//
// Lock order is p.mu, then speaker.Lock. The end-of-track callback runs under
// the speaker lock and only does a non-blocking channel send, so it never
// waits on p.mu.
type Player struct {
	decoders *Decoders
	ended    chan uint64
	cache    *trackCache

	mu      sync.Mutex
	stream  beep.StreamSeekCloser
	path    string
	format  beep.Format
	ctrl    *beep.Ctrl
	volume  *effects.Volume
	tap     *Tap
	gen     uint64
	percent int
	state   State
}

// NewPlayer opens the audio device.
func NewPlayer(decoders *Decoders, volume int) (*Player, error) {
	if err := speaker.Init(outputRate, outputRate.N(outputBuffer)); err != nil {
		return nil, fmt.Errorf("open audio device: %w", err)
	}
	return &Player{
		decoders: decoders,
		ended:    make(chan uint64, 4),
		cache:    newTrackCache(),
		percent:  clampPercent(volume),
	}, nil
}

// Ended delivers the generation of each track that played to the end.
func (p *Player) Ended() <-chan uint64 { return p.ended }

// Play stops the current track and starts path at offset start. It returns
// the generation that identifies this playback in Ended.
func (p *Player) Play(ctx context.Context, path string, start time.Duration) (uint64, error) {
	p.cache.keep(path)
	stream, format, err := Open(ctx, p.decoders, p.source(path))
	if err != nil {
		p.cache.keep()
		return 0, err
	}

	// Seek before the stream reaches the speaker, so a resumed track never
	// plays its opening first. A position past the end starts from the top.
	if pos := format.SampleRate.N(start); pos > 0 && pos < stream.Len() {
		if err := stream.Seek(pos); err != nil {
			stream.Close()
			return 0, fmt.Errorf("resume %s at %v: %w", path, start, err)
		}
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.stopLocked()
	p.gen++
	gen := p.gen

	var src beep.Streamer = stream
	if format.SampleRate != outputRate {
		src = beep.Resample(4, format.SampleRate, outputRate, stream)
	}
	done := beep.Callback(func() {
		select {
		case p.ended <- gen:
		default:
		}
	})

	p.ctrl = &beep.Ctrl{Streamer: beep.Seq(src, done)}
	// The tap sits before the volume control, so visualisers see the
	// music at full level however quietly it plays.
	p.tap = NewTap(p.ctrl)
	p.volume = &effects.Volume{Streamer: p.tap, Base: 2}
	p.applyVolumeLocked()
	p.stream, p.format, p.state, p.path = stream, format, Playing, path

	speaker.Play(p.volume)
	return gen, nil
}

// Preload starts reading path into memory so it can start instantly and play
// without touching the disk. Only the playing track and the most recent
// preload are kept.
func (p *Player) Preload(path string) {
	if !readsInProcess(p.decoders, path) || p.cache.has(path) {
		return
	}
	p.mu.Lock()
	current := p.path
	p.mu.Unlock()

	p.cache.keep(current, path)
	// A failed preload is retried, and reported, when the track plays.
	_, _ = p.cache.get(path)
}

// source returns path backed by memory when a decoder can use it.
func (p *Player) source(path string) Source {
	src := FileSource(path)
	if !readsInProcess(p.decoders, path) {
		return src
	}
	if f, err := p.cache.get(path); err == nil {
		src.Open = f.open
	}
	return src
}

// TogglePause pauses or resumes playback.
func (p *Player) TogglePause() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.state == Stopped {
		return
	}
	speaker.Lock()
	p.ctrl.Paused = !p.ctrl.Paused
	speaker.Unlock()

	p.state = Playing
	if p.ctrl.Paused {
		p.state = Paused
	}
}

// Stop halts playback and releases the track and any preload.
func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopLocked()
	p.cache.keep()
}

// Seek moves the play position by d, clamped to the track.
func (p *Player) Seek(d time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.stream == nil {
		return nil
	}
	speaker.Lock()
	defer speaker.Unlock()

	pos := p.stream.Position() + p.format.SampleRate.N(d)
	pos = max(0, min(pos, p.stream.Len()-1))
	if err := p.stream.Seek(pos); err != nil {
		return fmt.Errorf("seek: %w", err)
	}
	return nil
}

// Progress returns the play position and track length.
func (p *Player) Progress() (pos, length time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.stream == nil {
		return 0, 0
	}
	speaker.Lock()
	defer speaker.Unlock()
	rate := p.format.SampleRate
	return rate.D(p.stream.Position()), rate.D(p.stream.Len())
}

// SetVolume sets the volume as a percentage.
func (p *Player) SetVolume(percent int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.percent = clampPercent(percent)
	if p.volume != nil {
		speaker.Lock()
		p.applyVolumeLocked()
		speaker.Unlock()
	}
}

// Volume returns the volume as a percentage.
func (p *Player) Volume() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.percent
}

// State returns the transport state.
func (p *Player) State() State {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}

// Samples returns the most recent samples of each channel, oldest first,
// and their rate. Both are empty unless a track is playing.
func (p *Player) Samples() (left, right []float64, rate int) {
	p.mu.Lock()
	t, state := p.tap, p.state
	p.mu.Unlock()

	if t == nil || state != Playing {
		return nil, nil, int(outputRate)
	}
	left, right = t.Window()
	return left, right, int(outputRate)
}

// Close stops playback and releases the audio device.
func (p *Player) Close() {
	p.Stop()
	speaker.Close()
}

func (p *Player) stopLocked() {
	speaker.Clear()
	if p.stream != nil {
		// The stream is already detached from the speaker, so a close
		// failure cannot affect playback and there is nothing to recover.
		_ = p.stream.Close()
	}
	p.stream, p.ctrl, p.volume, p.tap = nil, nil, nil, nil
	p.state, p.path = Stopped, ""
}

// applyVolumeLocked maps the percentage onto a squared gain curve, which
// tracks perceived loudness better than a linear one.
func (p *Player) applyVolumeLocked() {
	p.volume.Silent = p.percent == 0
	if !p.volume.Silent {
		p.volume.Volume = 2 * math.Log2(float64(p.percent)/100)
	}
}

func clampPercent(v int) int { return max(0, min(v, 100)) }
