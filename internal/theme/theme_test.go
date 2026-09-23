package theme

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestBuiltinsAreValid(t *testing.T) {
	names := BuiltinNames()
	if len(names) < 30 {
		t.Errorf("only %d built-in themes", len(names))
	}
	for _, name := range names {
		th, _ := Builtin(name)
		if th.Name != name {
			t.Errorf("theme %q registered as %q", th.Name, name)
		}
		if err := th.Validate(); err != nil {
			t.Error(err)
		}
	}
	if _, ok := Builtin(Default); !ok {
		t.Fatalf("default theme %q missing", Default)
	}
}

func TestCustomThemes(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name+".json"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("neon", `{"extends": "nord", "colors": {"accent": "#ff00ff", "background": ""}}`)
	write("nord", `{"extends": "nord", "colors": {"text": "#ffffff"}}`)
	write("broken", `{"colors": {"accent": "pink"}}`)
	write("orphan", `{"extends": "no-such-theme"}`)
	s := NewStore(dir)

	tests := []struct {
		name    string
		check   func(Theme) bool
		wantErr error
	}{
		{name: "neon", check: func(th Theme) bool { return th.Accent == "#ff00ff" && th.Background == "" && th.Text == "#d8dee9" }},
		{name: "nord", check: func(th Theme) bool { return th.Text == "#ffffff" }},
		{name: "dracula", check: func(th Theme) bool { return th.Background == "#282a36" }},
		{name: "broken", wantErr: nil},
		{name: "orphan", wantErr: ErrUnknown},
		{name: "missing", wantErr: ErrUnknown},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			th, err := s.Load(tc.name)
			switch {
			case tc.check == nil && err == nil:
				t.Fatalf("Load(%q) succeeded, want error", tc.name)
			case tc.wantErr != nil && !errors.Is(err, tc.wantErr):
				t.Fatalf("Load(%q) error = %v, want %v", tc.name, err, tc.wantErr)
			case tc.check != nil && err != nil:
				t.Fatalf("Load(%q): %v", tc.name, err)
			case tc.check != nil && !tc.check(th):
				t.Errorf("Load(%q) = %+v", tc.name, th)
			}
		})
	}

	names := s.Names()
	for _, want := range []string{"neon", "broken", "dracula"} {
		if !slices.Contains(names, want) {
			t.Errorf("Names() missing %q", want)
		}
	}
	if n := len(slices.DeleteFunc(slices.Clone(names), func(s string) bool { return s != "nord" })); n != 1 {
		t.Errorf("nord listed %d times", n)
	}
}
