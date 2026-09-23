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

// outputRate is the device sample rate. Tracks at other rates are resampled.
const outputRate = beep.SampleRate(44100)

// Player plays one track at a time. It owns beep's process-wide speaker, so a
// program must create at most one.
//
// Lock order is p.mu, then speaker.Lock. The end-of-track callback runs under
// the speaker lock and only does a non-blocking channel send, so it never
// waits on p.mu.
type Player struct {
	decoders *Decoders
	ended    chan uint64

	mu       sync.Mutex
	stream   beep.StreamSeekCloser
	format   beep.Format
	ctrl     *beep.Ctrl
	volume   *effects.Volume
	analyzer *Analyzer
	gen      uint64
	percent  int
	state    State
}

// NewPlayer opens the audio device.
func NewPlayer(decoders *Decoders, volume int) (*Player, error) {
	if err := speaker.Init(outputRate, outputRate.N(time.Second/20)); err != nil {
		return nil, fmt.Errorf("open audio device: %w", err)
	}
	return &Player{
		decoders: decoders,
		ended:    make(chan uint64, 4),
		percent:  clampPercent(volume),
	}, nil
}

// Ended delivers the generation of each track that played to the end.
func (p *Player) Ended() <-chan uint64 { return p.ended }

// Play stops the current track and starts path. It returns the generation
// that identifies this playback in Ended.
func (p *Player) Play(ctx context.Context, path string) (uint64, error) {
	stream, format, err := Open(ctx, p.decoders, path)
	if err != nil {
		return 0, err
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
	p.volume = &effects.Volume{Streamer: p.ctrl, Base: 2}
	p.applyVolumeLocked()
	p.analyzer = NewAnalyzer(p.volume, outputRate)
	p.stream, p.format, p.state = stream, format, Playing

	speaker.Play(p.analyzer)
	return gen, nil
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

// Stop halts playback and releases the track.
func (p *Player) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.stopLocked()
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

// Spectrum returns band levels for the playing audio, or zeros when idle.
func (p *Player) Spectrum(bands int) []float64 {
	p.mu.Lock()
	a, state := p.analyzer, p.state
	p.mu.Unlock()

	if a == nil || state != Playing {
		return make([]float64, bands)
	}
	return a.Spectrum(bands)
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
	p.stream, p.ctrl, p.volume, p.analyzer = nil, nil, nil, nil
	p.state = Stopped
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
