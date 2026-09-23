package registry

import (
	"slices"
	"testing"
)

func TestRegistry(t *testing.T) {
	r := New[string]()
	r.Register("beep-mp3", "beep", "mp3")
	r.Register("ffmpeg", "ffmpeg", "mp3", "M4A")

	tests := []struct {
		name string
		key  string
		want []string
	}{
		{name: "priority order", key: "mp3", want: []string{"beep", "ffmpeg"}},
		{name: "case insensitive", key: "m4a", want: []string{"ffmpeg"}},
		{name: "missing key", key: "xyz", want: nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := r.Lookup(tc.key); !slices.Equal(got, tc.want) {
				t.Errorf("Lookup(%q) = %v, want %v", tc.key, got, tc.want)
			}
		})
	}

	if got, want := r.Keys(), []string{"m4a", "mp3"}; !slices.Equal(got, want) {
		t.Errorf("Keys() = %v, want %v", got, want)
	}
}

func TestRegisterReplacesInPlace(t *testing.T) {
	r := New[int]()
	r.Register("a", 1, "x")
	r.Register("b", 2, "x")
	r.Register("a", 3, "x")

	if got, want := r.Lookup("x"), []int{3, 2}; !slices.Equal(got, want) {
		t.Errorf("Lookup = %v, want %v", got, want)
	}
	if v, ok := r.Get("a"); !ok || v != 3 {
		t.Errorf("Get(a) = %v, %v", v, ok)
	}
}
