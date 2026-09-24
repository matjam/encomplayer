package domain

import (
	"math/rand/v2"
	"slices"
	"testing"
)

func TestQueueAdvance(t *testing.T) {
	tests := []struct {
		name      string
		items     []string
		current   int
		modes     Modes
		auto      bool
		wantItem  string
		wantOK    bool
		wantItems []string
	}{
		{name: "sequential", items: []string{"a", "b", "c"}, current: 0, wantItem: "b", wantOK: true, wantItems: []string{"a", "b", "c"}},
		{name: "start from nothing", items: []string{"a", "b"}, current: -1, wantItem: "a", wantOK: true, wantItems: []string{"a", "b"}},
		{name: "end stops", items: []string{"a", "b"}, current: 1, wantOK: false, wantItems: []string{"a", "b"}},
		{name: "end repeats", items: []string{"a", "b"}, current: 1, modes: Modes{Repeat: true}, wantItem: "a", wantOK: true, wantItems: []string{"a", "b"}},
		{name: "single stops when auto", items: []string{"a", "b"}, current: 0, modes: Modes{Single: true}, auto: true, wantOK: false, wantItems: []string{"a", "b"}},
		{name: "single ignored when manual", items: []string{"a", "b"}, current: 0, modes: Modes{Single: true}, wantItem: "b", wantOK: true, wantItems: []string{"a", "b"}},
		{name: "single repeat replays", items: []string{"a", "b"}, current: 0, modes: Modes{Single: true, Repeat: true}, auto: true, wantItem: "a", wantOK: true, wantItems: []string{"a", "b"}},
		{name: "consume removes finished", items: []string{"a", "b", "c"}, current: 0, modes: Modes{Consume: true}, auto: true, wantItem: "b", wantOK: true, wantItems: []string{"b", "c"}},
		{name: "consume last empties", items: []string{"a"}, current: 0, modes: Modes{Consume: true}, auto: true, wantOK: false, wantItems: []string{}},
		{name: "empty queue", items: nil, current: -1, wantOK: false, wantItems: []string{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q := NewQueue(tc.items...)
			q.SetCurrent(tc.current)

			got, ok := q.Advance(tc.modes, tc.auto, rand.New(rand.NewPCG(1, 2)))
			if ok != tc.wantOK || (ok && got != tc.wantItem) {
				t.Fatalf("Advance = %q, %v; want %q, %v", got, ok, tc.wantItem, tc.wantOK)
			}
			if items := q.Items(); !slices.Equal(items, tc.wantItems) && !(len(items) == 0 && len(tc.wantItems) == 0) {
				t.Errorf("items = %v, want %v", items, tc.wantItems)
			}
			if cur, _, ok := q.Current(); ok && cur != tc.wantItem {
				t.Errorf("current = %q, want %q", cur, tc.wantItem)
			}
		})
	}
}

func TestQueueRandomAvoidsCurrent(t *testing.T) {
	q := NewQueue("a", "b", "c", "d")
	r := rand.New(rand.NewPCG(7, 7))
	for range 50 {
		prev, _, _ := q.Current()
		next, ok := q.Advance(Modes{Random: true}, false, r)
		if !ok || next == prev {
			t.Fatalf("random advance returned %q after %q", next, prev)
		}
	}
}

func TestPeekNextMatchesAdvance(t *testing.T) {
	tests := []struct {
		name   string
		modes  Modes
		edit   func(q *Queue[string])
		wantOK bool
	}{
		{name: "sequential", wantOK: true},
		{name: "random", modes: Modes{Random: true}, wantOK: true},
		{name: "random with consume", modes: Modes{Random: true, Consume: true}, wantOK: true},
		{name: "single stops", modes: Modes{Single: true}, wantOK: false},
		{name: "single repeat replays", modes: Modes{Single: true, Repeat: true}, wantOK: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := rand.New(rand.NewPCG(9, 9))
			for range 30 {
				q := NewQueue("a", "b", "c", "d", "e", "f")
				q.SetCurrent(2)
				peek, ok := q.PeekNext(tc.modes, r)
				if ok != tc.wantOK {
					t.Fatalf("PeekNext ok = %v, want %v", ok, tc.wantOK)
				}
				got, _ := q.Advance(tc.modes, true, r)
				if ok && got != peek {
					t.Fatalf("Advance played %q, PeekNext promised %q", got, peek)
				}
			}
		})
	}
}

func TestQueueEditDropsRandomPlan(t *testing.T) {
	q := NewQueue("a", "b", "c")
	q.SetCurrent(0)
	r := rand.New(rand.NewPCG(1, 1))
	q.PeekNext(Modes{Random: true}, r)
	q.RemoveIndices([]int{1})
	if q.planned != -1 {
		t.Error("RemoveIndices kept a stale random plan")
	}
}

func TestQueueRemoveIndices(t *testing.T) {
	tests := []struct {
		name        string
		current     int
		remove      []int
		wantItems   []string
		wantCurrent string
		wantRemoved bool
	}{
		{name: "current passes to its follower", current: 1, remove: []int{1}, wantItems: []string{"a", "c", "d"}, wantCurrent: "c", wantRemoved: true},
		{name: "run including current", current: 1, remove: []int{0, 1, 2}, wantItems: []string{"d"}, wantCurrent: "d", wantRemoved: true},
		{name: "unsorted with duplicates", current: 2, remove: []int{3, 2, 2, 0}, wantItems: []string{"b"}, wantCurrent: "", wantRemoved: true},
		{name: "last item has no follower", current: 3, remove: []int{3}, wantItems: []string{"a", "b", "c"}, wantCurrent: "", wantRemoved: true},
		{name: "earlier item shifts current", current: 2, remove: []int{0}, wantItems: []string{"b", "c", "d"}, wantCurrent: "c"},
		{name: "later item leaves current", current: 1, remove: []int{3}, wantItems: []string{"a", "b", "c"}, wantCurrent: "b"},
		{name: "nothing current", current: -1, remove: []int{0}, wantItems: []string{"b", "c", "d"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q := NewQueue("a", "b", "c", "d")
			q.SetCurrent(tc.current)

			removed := q.RemoveIndices(tc.remove)
			if removed != tc.wantRemoved {
				t.Errorf("removed current = %v, want %v", removed, tc.wantRemoved)
			}
			if !slices.Equal(q.Items(), tc.wantItems) {
				t.Errorf("items = %v, want %v", q.Items(), tc.wantItems)
			}
			cur, _, ok := q.Current()
			if (tc.wantCurrent == "") == ok || cur != tc.wantCurrent {
				t.Errorf("current = %q (%v), want %q", cur, ok, tc.wantCurrent)
			}
		})
	}
}

func TestQueueRetreat(t *testing.T) {
	tests := []struct {
		name    string
		current int
		modes   Modes
		want    string
	}{
		{name: "previous", current: 2, want: "b"},
		{name: "clamps at start", current: 0, want: "a"},
		{name: "wraps with repeat", current: 0, modes: Modes{Repeat: true}, want: "c"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q := NewQueue("a", "b", "c")
			q.SetCurrent(tc.current)
			if got, _ := q.Retreat(tc.modes); got != tc.want {
				t.Errorf("Retreat = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestQueueKeepsCurrentThroughEdits(t *testing.T) {
	q := NewQueue("a", "b", "c", "d")
	q.SetCurrent(2)

	q.RemoveIndices([]int{0})
	if cur, i, _ := q.Current(); cur != "c" || i != 1 {
		t.Fatalf("after RemoveIndices current = %q@%d, want c@1", cur, i)
	}

	q.Swap(1, 0)
	if cur, i, _ := q.Current(); cur != "c" || i != 0 {
		t.Fatalf("after Swap current = %q@%d, want c@0", cur, i)
	}

	q.Shuffle(rand.New(rand.NewPCG(3, 4)))
	if cur, _, _ := q.Current(); cur != "c" {
		t.Fatalf("after Shuffle current = %q, want c", cur)
	}

	// Removing the current item hands current to the one that followed it.
	i := q.CurrentIndex()
	var follower string
	if next, ok := q.At(i + 1); ok {
		follower = next
	}
	q.RemoveIndices([]int{i})
	if cur, _, ok := q.Current(); (follower != "") != ok || cur != follower {
		t.Fatalf("after removing current, current = %q (%v), want follower %q", cur, ok, follower)
	}
}
