package tea

// EncomPlayer addition: tests for SetScrollOptimization.

import (
	"bytes"
	"fmt"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// scrolls matches every sequence hard scrolling uses to move lines: DECSTBM,
// SU, SD, IL, DL, RI and IND. Terminals such as foot move images for all of
// them.
var scrolls = regexp.MustCompile(`\x1b\[\d+;\d+r|\x1b\[\d*[STLM]|\x1b[MD]`)

// shiftedFrames renders a screen whose top rows stay put while the rows below
// them move up by one, the way a scrolling panel under a fixed one does, and
// returns what the second frame wrote.
func shiftedFrames(t *testing.T, s *cursedRenderer) string {
	t.Helper()
	frame := func(first int) View {
		lines := make([]string, 12)
		for i := range lines {
			if i < 4 {
				lines[i] = fmt.Sprintf("fixed %02d stays exactly where it is", i)
				continue
			}
			lines[i] = fmt.Sprintf("line %02d with enough text to be worth moving", first+i)
		}
		return View{Content: strings.Join(lines, "\n"), AltScreen: true}
	}
	s.render(frame(0))
	if err := s.flush(false); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	s.w = &out
	s.render(frame(1))
	if err := s.flush(false); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestSetScrollOptimization(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("scroll optimisation is always off on Windows")
	}
	tests := []struct {
		name       string
		apply      func(s *cursedRenderer)
		wantScroll bool
	}{
		{name: "on by default", apply: func(*cursedRenderer) {}, wantScroll: true},
		{name: "turned off", apply: func(s *cursedRenderer) { s.setScrollOptim(false) }, wantScroll: false},
		{name: "off survives reset", apply: func(s *cursedRenderer) { s.setScrollOptim(false); s.reset() }, wantScroll: false},
		{name: "turned back on", apply: func(s *cursedRenderer) { s.setScrollOptim(false); s.setScrollOptim(true) }, wantScroll: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newCursedRenderer(&bytes.Buffer{}, []string{"TERM=xterm-256color"}, 60, 12)
			tc.apply(s)
			out := shiftedFrames(t, s)
			if got := scrolls.MatchString(out); got != tc.wantScroll {
				t.Errorf("scrolled = %t, want %t; output %q", got, tc.wantScroll, out)
			}
		})
	}
}

func TestSetScrollOptimizationCmd(t *testing.T) {
	for _, on := range []bool{true, false} {
		if got := SetScrollOptimization(on)(); got != scrollOptimMsg(on) {
			t.Errorf("SetScrollOptimization(%t)() = %#v", on, got)
		}
	}
}
