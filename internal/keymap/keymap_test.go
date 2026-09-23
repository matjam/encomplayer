package keymap

import (
	"slices"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		notation string
		want     []string
	}{
		{notation: "q", want: []string{"q"}},
		{notation: "gg", want: []string{"g", "g"}},
		{notation: "G", want: []string{"G"}},
		{notation: ">", want: []string{">"}},
		{notation: "<CR>", want: []string{"enter"}},
		{notation: "<Esc>", want: []string{"esc"}},
		{notation: "<Space>", want: []string{"space"}},
		{notation: "<Tab>", want: []string{"tab"}},
		{notation: "<S-Tab>", want: []string{"shift+tab"}},
		{notation: "<C-u>", want: []string{"ctrl+u"}},
		{notation: "<C-U>", want: []string{"ctrl+shift+u"}},
		{notation: "<C-w>h", want: []string{"ctrl+w", "h"}},
		{notation: "<C-Space>", want: []string{"ctrl+space"}},
		{notation: "<C-Up>", want: []string{"ctrl+up"}},
		{notation: "<PageDown>", want: []string{"pgdown"}},
		{notation: "<C-s>a", want: []string{"ctrl+s", "a"}},
		{notation: "<A-x>", want: []string{"alt+x"}},
		{notation: "<", want: []string{"<"}},
	}
	for _, tc := range tests {
		t.Run(tc.notation, func(t *testing.T) {
			got, err := Parse(tc.notation)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("Parse(%q) = %q, want %q", tc.notation, got, tc.want)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	for _, notation := range []string{"", "<X-a>", "<Bogus>"} {
		if _, err := Parse(notation); err == nil {
			t.Errorf("Parse(%q) succeeded, want error", notation)
		}
	}
}

func TestParseAction(t *testing.T) {
	tests := []struct {
		in   string
		want Action
	}{
		{in: "Quit", want: Action{Name: "Quit"}},
		{in: `SwitchToTab("Album Artists")`, want: Action{Name: "SwitchToTab", Arg: "Album Artists"}},
	}
	for _, tc := range tests {
		t.Run(tc.in, func(t *testing.T) {
			got, err := ParseAction(tc.in)
			if err != nil || got != tc.want {
				t.Errorf("ParseAction(%q) = %v, %v; want %v", tc.in, got, err, tc.want)
			}
		})
	}
}

func TestResolver(t *testing.T) {
	queueCtx := []Context{Queue, Navigation, Global}
	tests := []struct {
		name     string
		keys     []string
		contexts []Context
		want     Action
	}{
		{name: "single key", keys: []string{"p"}, contexts: queueCtx, want: Action{Name: TogglePause}},
		{name: "chord", keys: []string{"g", "g"}, contexts: queueCtx, want: Action{Name: Top}},
		{name: "chord in another context", keys: []string{"g", "t"}, contexts: queueCtx, want: Action{Name: NextTab}},
		{name: "queue shadows navigation", keys: []string{"D"}, contexts: queueCtx, want: Action{Name: DeleteAll}},
		{name: "navigation outside queue", keys: []string{"D"}, contexts: []Context{Navigation, Global}, want: Action{Name: Delete}},
		{name: "navigation shadows global", keys: []string{"ctrl+u"}, contexts: queueCtx, want: Action{Name: UpHalf}},
		{name: "dead prefix retries last key", keys: []string{"g", "p"}, contexts: queueCtx, want: Action{Name: TogglePause}},
		{name: "pane chord", keys: []string{"ctrl+w", "l"}, contexts: queueCtx, want: Action{Name: PaneRight}},
		{name: "tab argument", keys: []string{"4"}, contexts: queueCtx, want: Action{Name: SwitchToTab, Arg: "Album Artists"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := NewResolver(Default())
			var last Result
			for i, k := range tc.keys {
				last = r.Feed(k, tc.contexts...)
				if i < len(tc.keys)-1 && !last.Pending && !last.Matched {
					t.Fatalf("key %q dropped mid-sequence", k)
				}
			}
			if !last.Matched || last.Action != tc.want {
				t.Errorf("result = %+v, want %v", last, tc.want)
			}
		})
	}
}

func TestResolverFlush(t *testing.T) {
	km := Default()
	if err := km.Bind(Global, "o", Action{Name: ShowHelp}); err != nil {
		t.Fatal(err)
	}
	r := NewResolver(km)

	res := r.Feed("o", Global)
	if !res.Pending {
		t.Fatalf("o should wait for oI, got %+v", res)
	}
	if stale := r.Flush(res.Generation-1, Global); stale.Matched {
		t.Fatal("stale generation fired")
	}
	if got := r.Flush(res.Generation, Global); !got.Matched || got.Action.Name != ShowHelp {
		t.Fatalf("flush = %+v, want ShowHelp", got)
	}
}
