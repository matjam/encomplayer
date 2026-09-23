// Package registry holds named, pluggable implementations that are looked up
// by key. EncomPlayer uses it for audio decoders (keyed by file extension) and
// album art renderers (keyed by protocol), so adding a format or protocol is a
// single Register call.
package registry

import (
	"iter"
	"slices"
	"strings"
	"sync"
)

// Registry is an ordered set of implementations of T. Keys are
// case-insensitive, and Lookup returns implementations in registration order,
// so earlier registrations take priority.
type Registry[T any] struct {
	mu      sync.RWMutex
	entries []entry[T]
}

type entry[T any] struct {
	name string
	keys []string
	impl T
}

// New returns an empty registry.
func New[T any]() *Registry[T] {
	return &Registry[T]{}
}

// Register adds impl under name, reachable through each key. Registering an
// existing name replaces that entry in place and keeps its priority.
func (r *Registry[T]) Register(name string, impl T, keys ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	norm := make([]string, len(keys))
	for i, k := range keys {
		norm[i] = strings.ToLower(k)
	}

	e := entry[T]{name: name, keys: norm, impl: impl}
	for i := range r.entries {
		if r.entries[i].name == name {
			r.entries[i] = e
			return
		}
	}
	r.entries = append(r.entries, e)
}

// Lookup returns every implementation registered for key, highest priority
// first.
func (r *Registry[T]) Lookup(key string) []T {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key = strings.ToLower(key)
	var out []T
	for _, e := range r.entries {
		if slices.Contains(e.keys, key) {
			out = append(out, e.impl)
		}
	}
	return out
}

// Has reports whether any implementation is registered for key.
func (r *Registry[T]) Has(key string) bool {
	return len(r.Lookup(key)) > 0
}

// Get returns the implementation registered under name.
func (r *Registry[T]) Get(name string) (T, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, e := range r.entries {
		if e.name == name {
			return e.impl, true
		}
	}
	var zero T
	return zero, false
}

// Keys returns every distinct key, sorted.
func (r *Registry[T]) Keys() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var keys []string
	for _, e := range r.entries {
		keys = append(keys, e.keys...)
	}
	slices.Sort(keys)
	return slices.Compact(keys)
}

// All yields each registered name and implementation in priority order.
func (r *Registry[T]) All() iter.Seq2[string, T] {
	r.mu.RLock()
	snapshot := slices.Clone(r.entries)
	r.mu.RUnlock()

	return func(yield func(string, T) bool) {
		for _, e := range snapshot {
			if !yield(e.name, e.impl) {
				return
			}
		}
	}
}
