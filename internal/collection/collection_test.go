package collection

import (
	"slices"
	"strings"
	"testing"
)

func TestGroupBy(t *testing.T) {
	words := []string{"banana", "apple", "blueberry", "avocado", "cherry"}
	groups := GroupBy(words, func(s string) string { return s[:1] }, strings.Compare)

	var keys []string
	for _, g := range groups {
		keys = append(keys, g.Key)
	}
	if want := []string{"a", "b", "c"}; !slices.Equal(keys, want) {
		t.Fatalf("keys = %v, want %v", keys, want)
	}
	if want := []string{"banana", "blueberry"}; !slices.Equal(groups[1].Items, want) {
		t.Errorf("b items = %v, want %v", groups[1].Items, want)
	}
}

func TestListNavigation(t *testing.T) {
	tests := []struct {
		name       string
		ops        func(l *List[int])
		wantCursor int
		wantFirst  int
	}{
		{name: "down scrolls viewport", ops: func(l *List[int]) { l.Move(4) }, wantCursor: 4, wantFirst: 2},
		{name: "clamps at bottom", ops: func(l *List[int]) { l.Move(100) }, wantCursor: 9, wantFirst: 7},
		{name: "clamps at top", ops: func(l *List[int]) { l.Move(-5) }, wantCursor: 0, wantFirst: 0},
		{name: "bottom then top", ops: func(l *List[int]) { l.Bottom(); l.Top() }, wantCursor: 0, wantFirst: 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			l := NewList([]int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9})
			l.SetHeight(3)
			tc.ops(l)
			if l.Cursor() != tc.wantCursor {
				t.Errorf("cursor = %d, want %d", l.Cursor(), tc.wantCursor)
			}
			for i := range l.Visible() {
				if i != tc.wantFirst {
					t.Errorf("first visible = %d, want %d", i, tc.wantFirst)
				}
				break
			}
		})
	}
}

func TestListSelectionAndFind(t *testing.T) {
	l := NewList([]string{"alpha", "beta", "gamma", "delta"})

	if got := l.Targets(); !slices.Equal(got, []int{0}) {
		t.Fatalf("targets without selection = %v", got)
	}

	l.ToggleSelect()
	l.ToggleSelect()
	if got := l.Targets(); !slices.Equal(got, []int{0, 1}) {
		t.Fatalf("targets = %v, want [0 1]", got)
	}

	l.InvertSelection()
	if got := l.Targets(); !slices.Equal(got, []int{2, 3}) {
		t.Fatalf("inverted targets = %v, want [2 3]", got)
	}

	hasA := func(s string) bool { return strings.Contains(s, "a") }
	l.SetCursor(3)
	if !l.Find(hasA, true, false) || l.Cursor() != 0 {
		t.Errorf("forward find wrapped to %d, want 0", l.Cursor())
	}
	if !l.Find(hasA, false, false) || l.Cursor() != 3 {
		t.Errorf("backward find = %d, want 3", l.Cursor())
	}
}

type tree map[string][]string

func (t tree) Root() []string             { return t[""] }
func (t tree) Children(n string) []string { return t[n] }
func (t tree) IsLeaf(n string) bool       { return len(t[n]) == 0 }
func (t tree) Key(n string) string        { return n }

func TestBrowser(t *testing.T) {
	data := tree{"": {"a", "b"}, "a": {"a1", "a2"}, "b": {"b1"}}
	b := NewBrowser[string](data)

	if got := b.Preview(); !slices.Equal(got, []string{"a1", "a2"}) {
		t.Fatalf("preview = %v", got)
	}
	if !b.Enter() {
		t.Fatal("enter a failed")
	}
	b.Current().SetCursor(1)
	if b.Enter() {
		t.Fatal("entered a leaf")
	}
	if got := b.Path(); !slices.Equal(got, []string{"a"}) {
		t.Fatalf("path = %v", got)
	}

	grown := tree{"": {"0", "a", "b"}, "a": {"a0", "a1", "a2"}, "b": {"b1"}}
	b.SetProvider(grown)
	if b.Depth() != 1 {
		t.Fatalf("depth after reload = %d, want 1", b.Depth())
	}
	if cur, _ := b.Current().Current(); cur != "a2" {
		t.Errorf("cursor after reload = %q, want a2", cur)
	}

	if !b.Leave() || b.Leave() {
		t.Error("leave should succeed once then stop at root")
	}
}
