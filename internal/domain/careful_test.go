package domain

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
)

// artistOf groups test items written as "artist:title". Items without a
// colon belong to no group.
func artistOf(s string) string {
	artist, _, ok := strings.Cut(s, ":")
	if !ok {
		return ""
	}
	return artist
}

// tracks returns count items for each artist, as "artist:n".
func tracks(counts map[string]int) []string {
	var out []string
	for artist, n := range counts {
		for i := range n {
			out = append(out, fmt.Sprintf("%s:%d", artist, i))
		}
	}
	slices.Sort(out)
	return out
}

// repeats counts neighbours in the same group.
func repeats(items []string) int {
	n := 0
	for i := 1; i < len(items); i++ {
		if g := artistOf(items[i]); g != "" && g == artistOf(items[i-1]) {
			n++
		}
	}
	return n
}

func TestShuffleCarefully(t *testing.T) {
	tests := []struct {
		name        string
		items       []string
		wantRepeats int
	}{
		{name: "even artists", items: tracks(map[string]int{"a": 3, "b": 3, "c": 3})},
		{name: "one artist at exactly half", items: tracks(map[string]int{"a": 5, "b": 2, "c": 2})},
		{name: "one artist just over half of an odd count", items: tracks(map[string]int{"a": 5, "b": 4})},
		{name: "one big artist among many", items: tracks(map[string]int{"a": 10, "b": 3, "c": 3, "d": 3, "e": 3, "f": 3, "g": 3})},
		{name: "many artists", items: tracks(map[string]int{"a": 7, "b": 6, "c": 5, "d": 4, "e": 3, "f": 2, "g": 1, "h": 1})},
		{name: "ungrouped items never clash", items: []string{"x", "y", "z", "a:1", "a:2"}},
		// No careful order exists: keep back-to-back repeats to the minimum,
		// the dominant artist's count minus everything else plus one.
		{name: "dominant artist", items: tracks(map[string]int{"a": 6, "b": 2}), wantRepeats: 3},
		{name: "one artist only", items: tracks(map[string]int{"a": 3}), wantRepeats: 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for seed := range uint64(200) {
				q := NewQueue(tc.items...).GroupBy(artistOf)
				q.Shuffle(Modes{Careful: true}, rand.New(rand.NewPCG(seed, 7)))
				got := q.Items()
				if !slices.Equal(slices.Sorted(slices.Values(got)), slices.Sorted(slices.Values(tc.items))) {
					t.Fatalf("seed %d: %v is not a permutation of %v", seed, got, tc.items)
				}
				if r := repeats(got); r != tc.wantRepeats {
					t.Fatalf("seed %d: %v has %d back-to-back repeats, want %d", seed, got, r, tc.wantRepeats)
				}
			}
		})
	}
}

func TestShuffleCarefullyIsRandom(t *testing.T) {
	items := tracks(map[string]int{"a": 3, "b": 3, "c": 3, "d": 3})
	firsts := map[string]bool{}
	orders := map[string]bool{}
	for seed := range uint64(100) {
		q := NewQueue(items...).GroupBy(artistOf)
		q.Shuffle(Modes{Careful: true}, rand.New(rand.NewPCG(seed, 9)))
		firsts[q.Items()[0]] = true
		orders[strings.Join(q.Items(), ",")] = true
	}
	if len(firsts) < 8 {
		t.Errorf("only %d different first tracks in 100 shuffles", len(firsts))
	}
	if len(orders) < 95 {
		t.Errorf("only %d different orders in 100 shuffles", len(orders))
	}
}

func TestShuffleCarefullyKeepsCurrent(t *testing.T) {
	q := NewQueue(tracks(map[string]int{"a": 3, "b": 3})...).GroupBy(artistOf)
	q.SetCurrent(4)
	q.Shuffle(Modes{Careful: true}, rand.New(rand.NewPCG(1, 1)))
	if cur, _, _ := q.Current(); cur != "b:1" {
		t.Fatalf("current = %q, want b:1", cur)
	}
}

func TestShuffleWithoutGroupingIgnoresCareful(t *testing.T) {
	items := tracks(map[string]int{"a": 3, "b": 3})
	plain, careful := NewQueue(items...), NewQueue(items...)
	plain.Shuffle(Modes{}, rand.New(rand.NewPCG(5, 5)))
	careful.Shuffle(Modes{Careful: true}, rand.New(rand.NewPCG(5, 5)))
	if !slices.Equal(plain.Items(), careful.Items()) {
		t.Errorf("an ungrouped queue shuffled differently in careful mode: %v vs %v", careful.Items(), plain.Items())
	}
}

func TestArtistKey(t *testing.T) {
	tests := []struct {
		name  string
		track Track
		want  string
	}{
		{name: "track artist", track: Track{Artist: "Daft Punk", AlbumArtist: "Various Artists"}, want: "daft punk"},
		{name: "album artist when untagged", track: Track{AlbumArtist: "Wendy Carlos"}, want: "wendy carlos"},
		{name: "case does not matter", track: Track{Artist: "DAFT PUNK"}, want: "daft punk"},
		{name: "no artist groups with nothing", track: Track{Title: "Untitled"}, want: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.track.ArtistKey(); got != tc.want {
				t.Errorf("ArtistKey = %q, want %q", got, tc.want)
			}
		})
	}
}

// BenchmarkShuffleCarefully shuffles a 10,000-track library whose artists
// range from one track to a few hundred.
func BenchmarkShuffleCarefully(b *testing.B) {
	r := rand.New(rand.NewPCG(1, 2))
	counts := map[string]int{}
	for n := 0; n < 10_000; {
		c := min(1+int(r.ExpFloat64()*20), 10_000-n)
		counts[fmt.Sprintf("artist %d", len(counts))] = c
		n += c
	}
	items := tracks(counts)
	for b.Loop() {
		q := NewQueue(items...).GroupBy(artistOf)
		q.Shuffle(Modes{Careful: true}, r)
	}
}

func TestRandomCarefullyAvoidsCurrentArtist(t *testing.T) {
	tests := []struct {
		name    string
		items   []string
		current int
		allowed []string
	}{
		{name: "picks another artist", items: []string{"a:1", "a:2", "a:3", "b:1", "c:1"}, current: 0, allowed: []string{"b:1", "c:1"}},
		{name: "only one artist left falls back to any other track", items: []string{"a:1", "a:2", "a:3"}, current: 1, allowed: []string{"a:1", "a:3"}},
		{name: "ungrouped current clashes with nothing", items: []string{"x", "a:1", "a:2"}, current: 0, allowed: []string{"a:1", "a:2"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for seed := range uint64(100) {
				r := rand.New(rand.NewPCG(seed, 3))
				q := NewQueue(tc.items...).GroupBy(artistOf)
				q.SetCurrent(tc.current)
				modes := Modes{Random: true, Careful: true}
				peek, _ := q.PeekNext(modes, r)
				got, _ := q.Advance(modes, true, r)
				if got != peek || !slices.Contains(tc.allowed, got) {
					t.Fatalf("seed %d: peeked %q, played %q, want one of %v", seed, peek, got, tc.allowed)
				}
			}
		})
	}
}
