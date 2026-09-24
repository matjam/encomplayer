package builtin

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/matjam/encomplayer/internal/viz"
)

var testPalette = viz.Palette{
	Background: viz.Hex("#000000"), Text: viz.Hex("#6fc3df"), Bright: viz.Hex("#e6ffff"),
	Dim: viz.Hex("#3a6a7a"), Grid: viz.Hex("#16323c"), Accent: viz.Hex("#ffb000"), Error: viz.Hex("#ff4040"),
}

// music makes a frame of a 220 Hz tone, panned slightly right, with a kick
// drum every half second.
func music(a *viz.Analyzer, t time.Duration, dt time.Duration) *viz.Frame {
	const rate = 44100
	left, right := make([]float64, viz.Window), make([]float64, viz.Window)
	end := t.Seconds()
	for i := range viz.Window {
		s := end - float64(viz.Window-i)/rate
		tone := 0.4 * math.Sin(2*math.Pi*220*s)
		kick := 0.0
		if since := math.Mod(s, 0.5); since < 0.08 {
			kick = 0.8 * math.Sin(2*math.Pi*60*since) * (1 - since/0.08)
		}
		left[i], right[i] = tone*0.8+kick, tone+kick
	}
	f := a.Analyze(left, right, rate, dt)
	f.Time, f.Delta, f.Playing = t, dt, true
	f.Track = viz.Track{Title: "Derezzed", Artist: "Daft Punk"}
	return f
}

// Every registered visualiser must survive any canvas size, fill exactly
// the canvas, and draw something when music plays.
func TestEveryVisualizer(t *testing.T) {
	infos := viz.Builtins.List()
	if len(infos) < 20 {
		t.Fatalf("only %d built-in visualizers registered", len(infos))
	}
	sizes := [][2]int{{0, 0}, {1, 1}, {3, 2}, {80, 6}, {120, 40}, {40, 12}}
	for _, info := range infos {
		t.Run(info.Name, func(t *testing.T) {
			if info.Description == "" {
				t.Error("no description")
			}
			v, _, err := viz.Builtins.New(info.Name)
			if err != nil {
				t.Fatal(err)
			}
			a := viz.NewAnalyzer()
			c := viz.NewCanvas(0, 0, testPalette)
			const dt = 33 * time.Millisecond
			now := time.Duration(0)
			for _, size := range sizes {
				drewSomething := false
				for range 45 {
					now += dt
					c.Resize(size[0], size[1])
					if err := v.Render(c, music(a, now, dt)); err != nil {
						t.Fatalf("%dx%d: %v", size[0], size[1], err)
					}
					lines := c.Lines()
					if len(lines) != size[1] {
						t.Fatalf("%dx%d: %d lines", size[0], size[1], len(lines))
					}
					for i, l := range lines {
						if w := ansi.StringWidth(l); w != size[0] {
							t.Fatalf("%dx%d: line %d is %d cells wide", size[0], size[1], i, w)
						}
						drewSomething = drewSomething || strings.TrimSpace(ansi.Strip(l)) != ""
					}
				}
				if size[0] >= 40 && size[1] >= 6 && !drewSomething {
					t.Errorf("%dx%d: drew nothing in 1.5 s of music", size[0], size[1])
				}
			}
		})
	}
}

// Silence must not break anything either: no NaNs, no panics.
func TestEveryVisualizerInSilence(t *testing.T) {
	for _, info := range viz.Builtins.List() {
		v, _, _ := viz.Builtins.New(info.Name)
		a := viz.NewAnalyzer()
		c := viz.NewCanvas(60, 16, testPalette)
		for i := range 10 {
			c.Clear()
			f := a.Analyze(nil, nil, 44100, 33*time.Millisecond)
			f.Time, f.Delta = time.Duration(i)*33*time.Millisecond, 33*time.Millisecond
			if err := v.Render(c, f); err != nil {
				t.Errorf("%s: %v", info.Name, err)
			}
		}
	}
}

// BenchmarkVisualizers reports each visualiser's cost for one frame on a
// large full-screen canvas.
func BenchmarkVisualizers(b *testing.B) {
	for _, info := range viz.Builtins.List() {
		b.Run(info.Name, func(b *testing.B) {
			v, _, _ := viz.Builtins.New(info.Name)
			a := viz.NewAnalyzer()
			c := viz.NewCanvas(200, 50, testPalette)
			frames := make([]*viz.Frame, 30)
			for i := range frames {
				frames[i] = music(a, time.Duration(i+1)*33*time.Millisecond, 33*time.Millisecond)
			}
			b.ResetTimer()
			for i := range b.N {
				c.Clear()
				if err := v.Render(c, frames[i%len(frames)]); err != nil {
					b.Fatal(err)
				}
				_ = c.Lines()
			}
		})
	}
}
