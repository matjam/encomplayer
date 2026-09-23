package domain

import (
	"math/rand/v2"
	"slices"
)

// Queue is an ordered play queue with a marker on the current item.
type Queue[T any] struct {
	items   []T
	current int
}

// NewQueue returns a queue holding items with nothing current.
func NewQueue[T any](items ...T) *Queue[T] {
	return &Queue[T]{items: slices.Clone(items), current: -1}
}

// Len returns the number of items.
func (q *Queue[T]) Len() int { return len(q.items) }

// Items returns the queue contents. Callers must not modify the slice.
func (q *Queue[T]) Items() []T { return q.items }

// At returns the item at i.
func (q *Queue[T]) At(i int) (T, bool) {
	if i < 0 || i >= len(q.items) {
		var zero T
		return zero, false
	}
	return q.items[i], true
}

// Current returns the current item and its index.
func (q *Queue[T]) Current() (T, int, bool) {
	v, ok := q.At(q.current)
	if !ok {
		return v, -1, false
	}
	return v, q.current, true
}

// CurrentIndex returns the index of the current item, or -1.
func (q *Queue[T]) CurrentIndex() int { return q.current }

// SetCurrent marks index i as current. An out-of-range index clears it.
func (q *Queue[T]) SetCurrent(i int) bool {
	if i < 0 || i >= len(q.items) {
		q.current = -1
		return false
	}
	q.current = i
	return true
}

// Append adds items to the end of the queue.
func (q *Queue[T]) Append(items ...T) {
	q.items = append(q.items, items...)
}

// Remove deletes the item at i. Removing the current item clears the marker.
func (q *Queue[T]) Remove(i int) {
	if i < 0 || i >= len(q.items) {
		return
	}
	q.items = slices.Delete(q.items, i, i+1)
	switch {
	case i == q.current:
		q.current = -1
	case i < q.current:
		q.current--
	}
}

// Clear empties the queue.
func (q *Queue[T]) Clear() {
	q.items = nil
	q.current = -1
}

// Swap exchanges the items at i and j and keeps the current marker on the
// same item.
func (q *Queue[T]) Swap(i, j int) bool {
	if i < 0 || j < 0 || i >= len(q.items) || j >= len(q.items) {
		return false
	}
	q.items[i], q.items[j] = q.items[j], q.items[i]
	switch q.current {
	case i:
		q.current = j
	case j:
		q.current = i
	}
	return true
}

// Shuffle randomises the order and keeps the current marker on the same item.
func (q *Queue[T]) Shuffle(r *rand.Rand) {
	order := r.Perm(len(q.items))
	shuffled := make([]T, len(q.items))
	current := -1
	for dst, src := range order {
		shuffled[dst] = q.items[src]
		if src == q.current {
			current = dst
		}
	}
	q.items = shuffled
	q.current = current
}

// Advance moves to the next item under the given modes and returns it. auto
// is true when the current track finished on its own, which is when Single and
// Consume apply. It returns false when playback should stop.
func (q *Queue[T]) Advance(m Modes, auto bool, r *rand.Rand) (T, bool) {
	prev := q.current
	next := q.nextIndex(m, auto, r)

	if auto && m.Consume && prev >= 0 && next != prev {
		q.items = slices.Delete(q.items, prev, prev+1)
		if next > prev {
			next--
		}
	}

	if !q.SetCurrent(next) {
		var zero T
		return zero, false
	}
	return q.items[next], true
}

// Retreat moves to the previous item and returns it.
func (q *Queue[T]) Retreat(m Modes) (T, bool) {
	var zero T
	if len(q.items) == 0 {
		return zero, false
	}
	prev := q.current - 1
	if prev < 0 {
		prev = 0
		if m.Repeat {
			prev = len(q.items) - 1
		}
	}
	q.current = prev
	return q.items[prev], true
}

func (q *Queue[T]) nextIndex(m Modes, auto bool, r *rand.Rand) int {
	n := len(q.items)
	switch {
	case n == 0:
		return -1
	case auto && m.Single && m.Repeat:
		return max(q.current, 0)
	case auto && m.Single:
		return -1
	case m.Random:
		return q.randomIndex(r)
	case q.current+1 < n:
		return q.current + 1
	case m.Repeat:
		return 0
	default:
		return -1
	}
}

func (q *Queue[T]) randomIndex(r *rand.Rand) int {
	n := len(q.items)
	if n == 1 || q.current < 0 {
		return r.IntN(n)
	}

	// Draw from the other n-1 items so the current one never repeats.
	i := r.IntN(n - 1)
	if i >= q.current {
		i++
	}
	return i
}
