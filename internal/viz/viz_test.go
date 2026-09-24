package viz

import (
	"errors"
	"slices"
	"testing"
)

type fixed string

func (fixed) Render(*Canvas, *Frame) error { return nil }

type closing struct{ closed *bool }

func (closing) Render(*Canvas, *Frame) error { return nil }
func (c closing) Close() error               { *c.closed = true; return nil }

// scripts stands in for a runtime source, such as a folder of scripts.
type scripts struct {
	names    []string
	reloaded int
}

func (s *scripts) List() []Info {
	out := make([]Info, len(s.names))
	for i, n := range s.names {
		out[i] = Info{Name: n, Description: "script", Source: "script"}
	}
	return out
}

func (s *scripts) New(name string) (Visualizer, error) { return fixed("script:" + name), nil }
func (s *scripts) Reload() error                       { s.reloaded++; return nil }

func TestCatalog(t *testing.T) {
	c := NewCatalog()
	c.Register(Info{Name: "spectrum", Description: "bars"}, func() Visualizer { return fixed("spectrum") })
	c.Register(Info{Name: "scope", Description: "wave"}, func() Visualizer { return fixed("scope") })

	src := &scripts{names: []string{"aurora", "scope"}}
	c.AddSource(src)

	if got := c.Names(); !slices.Equal(got, []string{"aurora", "scope", "spectrum"}) {
		t.Errorf("Names = %v", got)
	}

	// A source's visualiser replaces the built-in of the same name.
	v, info, err := c.New("scope")
	if err != nil || v != fixed("script:scope") || info.Source != "script" {
		t.Errorf("New(scope) = %v %+v %v", v, info, err)
	}
	if v, info, _ := c.New("spectrum"); v != fixed("spectrum") || info.Source != "builtin" {
		t.Errorf("New(spectrum) = %v %+v", v, info)
	}
	if _, _, err := c.New("nope"); !errors.Is(err, ErrUnknown) {
		t.Errorf("New(nope) err = %v", err)
	}

	if err := c.Reload(); err != nil || src.reloaded != 1 {
		t.Errorf("Reload err %v, reloaded %d times", err, src.reloaded)
	}
}

func TestStep(t *testing.T) {
	c := NewCatalog()
	for _, n := range []string{"a", "b", "c"} {
		c.Register(Info{Name: n}, func() Visualizer { return fixed(n) })
	}
	tests := []struct {
		from  string
		delta int
		want  string
	}{
		{"a", 1, "b"},
		{"c", 1, "a"},
		{"a", -1, "c"},
		{"b", 5, "a"},
		{"gone", 1, "a"},
		{"gone", -1, "c"},
	}
	for _, tc := range tests {
		if got := c.Step(tc.from, tc.delta); got != tc.want {
			t.Errorf("Step(%q, %d) = %q, want %q", tc.from, tc.delta, got, tc.want)
		}
	}
}

func TestRegisterRejectsMissingParts(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("Register with no factory did not panic")
		}
	}()
	NewCatalog().Register(Info{Name: "x"}, nil)
}

func TestClose(t *testing.T) {
	closed := false
	if err := Close(closing{&closed}); err != nil || !closed {
		t.Error("Close did not close a closer")
	}
	if err := Close(fixed("plain")); err != nil {
		t.Errorf("Close on a plain visualizer = %v", err)
	}
}
