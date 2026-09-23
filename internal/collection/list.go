package collection

import "iter"

// List is a scrollable list with a cursor and a multi-selection. It holds no
// rendering logic, so every tab shares the same navigation behaviour.
type List[T any] struct {
	items    []T
	cursor   int
	offset   int
	height   int
	selected map[int]struct{}
}

// NewList returns a list over items with the cursor on the first item.
func NewList[T any](items []T) *List[T] {
	return &List[T]{items: items, height: 1, selected: map[int]struct{}{}}
}

// SetItems replaces the contents, clamps the cursor and clears the selection.
func (l *List[T]) SetItems(items []T) {
	l.items = items
	l.selected = map[int]struct{}{}
	l.SetCursor(l.cursor)
}

// Items returns the contents. Callers must not modify the slice.
func (l *List[T]) Items() []T { return l.items }

// Len returns the number of items.
func (l *List[T]) Len() int { return len(l.items) }

// Cursor returns the cursor index, or 0 for an empty list.
func (l *List[T]) Cursor() int { return l.cursor }

// Current returns the item under the cursor.
func (l *List[T]) Current() (T, bool) {
	if l.cursor < 0 || l.cursor >= len(l.items) {
		var zero T
		return zero, false
	}
	return l.items[l.cursor], true
}

// SetCursor moves the cursor to i, clamped to the list bounds.
func (l *List[T]) SetCursor(i int) {
	l.cursor = max(0, min(i, len(l.items)-1))
	l.scroll()
}

// Move shifts the cursor by delta items.
func (l *List[T]) Move(delta int) { l.SetCursor(l.cursor + delta) }

// Top moves the cursor to the first item.
func (l *List[T]) Top() { l.SetCursor(0) }

// Bottom moves the cursor to the last item.
func (l *List[T]) Bottom() { l.SetCursor(len(l.items) - 1) }

// Height returns the number of visible rows.
func (l *List[T]) Height() int { return l.height }

// SetHeight sets the number of visible rows.
func (l *List[T]) SetHeight(h int) {
	l.height = max(1, h)
	l.scroll()
}

// Visible yields the index and item of each row in the viewport.
func (l *List[T]) Visible() iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		end := min(l.offset+l.height, len(l.items))
		for i := l.offset; i < end; i++ {
			if !yield(i, l.items[i]) {
				return
			}
		}
	}
}

// ToggleSelect flips the selection of the item under the cursor and moves
// down, so repeated presses select a run of items.
func (l *List[T]) ToggleSelect() {
	if len(l.items) == 0 {
		return
	}
	if _, ok := l.selected[l.cursor]; ok {
		delete(l.selected, l.cursor)
	} else {
		l.selected[l.cursor] = struct{}{}
	}
	l.Move(1)
}

// InvertSelection selects every unselected item and deselects the rest.
func (l *List[T]) InvertSelection() {
	next := make(map[int]struct{}, len(l.items)-len(l.selected))
	for i := range l.items {
		if _, ok := l.selected[i]; !ok {
			next[i] = struct{}{}
		}
	}
	l.selected = next
}

// IsSelected reports whether index i is selected.
func (l *List[T]) IsSelected(i int) bool {
	_, ok := l.selected[i]
	return ok
}

// ClearSelection deselects everything.
func (l *List[T]) ClearSelection() { l.selected = map[int]struct{}{} }

// Targets returns the selected indices in order, or the cursor index when
// nothing is selected. Actions such as add and delete apply to these.
func (l *List[T]) Targets() []int {
	if len(l.items) == 0 {
		return nil
	}
	if len(l.selected) == 0 {
		return []int{l.cursor}
	}
	var out []int
	for i := range l.items {
		if _, ok := l.selected[i]; ok {
			out = append(out, i)
		}
	}
	return out
}

// Find moves the cursor to the next item matching match, searching forward or
// backward from the cursor and wrapping. It reports whether a match exists.
func (l *List[T]) Find(match func(T) bool, forward bool, includeCursor bool) bool {
	n := len(l.items)
	step := 1
	if !forward {
		step = -1
	}
	start := 1
	if includeCursor {
		start = 0
	}
	for k := start; k < n+start; k++ {
		i := ((l.cursor+step*k)%n + n) % n
		if match(l.items[i]) {
			l.SetCursor(i)
			return true
		}
	}
	return false
}

func (l *List[T]) scroll() {
	if l.cursor < l.offset {
		l.offset = l.cursor
	}
	if l.cursor >= l.offset+l.height {
		l.offset = l.cursor - l.height + 1
	}
	l.offset = max(0, min(l.offset, len(l.items)-l.height))
}
