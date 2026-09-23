// Package collection provides generic building blocks for browsing a music
// library: grouping, a cursor list and a hierarchical browser.
package collection

import "slices"

// Group is a bucket of items that share a key.
type Group[K comparable, V any] struct {
	Key   K
	Items []V
}

// GroupBy buckets items by key and returns the groups ordered by compare.
// Items keep their input order within each group.
func GroupBy[K comparable, V any](items []V, key func(V) K, compare func(a, b K) int) []Group[K, V] {
	index := make(map[K]int)
	var groups []Group[K, V]
	for _, item := range items {
		k := key(item)
		i, ok := index[k]
		if !ok {
			i = len(groups)
			index[k] = i
			groups = append(groups, Group[K, V]{Key: k})
		}
		groups[i].Items = append(groups[i].Items, item)
	}
	slices.SortFunc(groups, func(a, b Group[K, V]) int { return compare(a.Key, b.Key) })
	return groups
}

// Map applies fn to every element of in.
func Map[T, U any](in []T, fn func(T) U) []U {
	out := make([]U, len(in))
	for i, v := range in {
		out[i] = fn(v)
	}
	return out
}

// FlatMap applies fn to every element of in and concatenates the results.
func FlatMap[T, U any](in []T, fn func(T) []U) []U {
	var out []U
	for _, v := range in {
		out = append(out, fn(v)...)
	}
	return out
}
