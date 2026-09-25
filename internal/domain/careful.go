package domain

import "math/rand/v2"

// Careful mode keeps items in the same group, such as tracks by the same
// artist, from playing back to back. An EncomPlayer addition.

// GroupBy sets how Careful mode groups items, such as by artist, and returns
// q. Items whose key is empty belong to no group and never clash.
func (q *Queue[T]) GroupBy(key func(T) string) *Queue[T] {
	q.group = key
	return q
}

// careful reports whether m asks to keep groups apart and q can tell them.
func (q *Queue[T]) careful(m Modes) bool { return m.Careful && q.group != nil }

// groups returns each item's group key.
func (q *Queue[T]) groups() []string {
	keys := make([]string, len(q.items))
	for i, item := range q.items {
		keys[i] = q.group(item)
	}
	return keys
}

// otherGroupIndex picks a random item outside the current item's group, and
// reports false when there is none.
func (q *Queue[T]) otherGroupIndex(r *rand.Rand) (int, bool) {
	current := q.group(q.items[q.current])
	var others []int
	for i, item := range q.items {
		if i != q.current && (current == "" || q.group(item) != current) {
			others = append(others, i)
		}
	}
	if len(others) == 0 {
		return 0, false
	}
	return others[r.IntN(len(others))], true
}

// spreadOrder returns a random order of items, given each one's group key,
// in which no two neighbours share a group whenever such an order exists.
//
// Each position is drawn uniformly from the items that leave the rest
// arrangeable. With left items to follow and the one just placed in group p,
// that needs p to hold at most left/2 of them and every other group at most
// (left+1)/2. So the biggest group must be placed whenever it holds more
// than (left+1)/2; otherwise any item outside p will do.
//
// When one group is too big for any such order, it is placed whenever it is
// not the previous group, which keeps back-to-back repeats to the fewest
// possible.
func spreadOrder(keys []string, r *rand.Rand) []int {
	gid, count := numberGroups(keys)

	// sizes[c] is how many groups hold c items; top is the largest group.
	top := 0
	for _, c := range count {
		top = max(top, c)
	}
	sizes := make([]int, top+1)
	for _, c := range count {
		sizes[c]++
	}

	pool := make([]int, len(keys)) // items not yet placed
	for i := range pool {
		pool[i] = i
	}
	// draw removes and returns a random pooled item whose group ok accepts.
	// Callers make sure one exists.
	draw := func(ok func(g int) bool) int {
		for {
			j := r.IntN(len(pool))
			if item := pool[j]; ok(gid[item]) {
				pool[j] = pool[len(pool)-1]
				pool = pool[:len(pool)-1]
				return item
			}
		}
	}

	order := make([]int, 0, len(keys))
	prev := -1
	for len(pool) > 0 {
		left := len(pool) - 1
		prevCount := 0
		if prev >= 0 {
			prevCount = count[prev]
		}
		notPrev := func(g int) bool { return g != prev }

		var item int
		switch {
		case top > (left+1)/2:
			big := -1
			for _, it := range pool {
				if g := gid[it]; count[g] == top && g != prev {
					big = g
					break
				}
			}
			switch {
			case big >= 0:
				item = draw(func(g int) bool { return g == big })
			case prevCount < len(pool):
				item = draw(notPrev)
			default:
				item = draw(func(int) bool { return true })
			}
		case prevCount < len(pool):
			item = draw(notPrev)
		default:
			item = draw(func(int) bool { return true })
		}

		g := gid[item]
		sizes[count[g]]--
		count[g]--
		sizes[count[g]]++
		for top > 0 && sizes[top] == 0 {
			top--
		}
		prev = g
		order = append(order, item)
	}
	return order
}

// numberGroups gives each distinct key a group number, and each empty key a
// group of its own. It returns every item's group and each group's size.
func numberGroups(keys []string) (gid, count []int) {
	gid = make([]int, len(keys))
	ids := map[string]int{}
	for i, k := range keys {
		id, ok := ids[k]
		if !ok || k == "" {
			id = len(count)
			count = append(count, 0)
			if k != "" {
				ids[k] = id
			}
		}
		gid[i] = id
		count[id]++
	}
	return gid, count
}
