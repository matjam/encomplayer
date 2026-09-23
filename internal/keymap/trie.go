package keymap

// Trie maps key sequences to values.
type Trie[V any] struct {
	root node[V]
}

type node[V any] struct {
	children map[string]*node[V]
	value    V
	bound    bool
}

// Insert binds seq to v, replacing any existing binding.
func (t *Trie[V]) Insert(seq []string, v V) {
	n := &t.root
	for _, k := range seq {
		if n.children == nil {
			n.children = map[string]*node[V]{}
		}
		child, ok := n.children[k]
		if !ok {
			child = &node[V]{}
			n.children[k] = child
		}
		n = child
	}
	n.value, n.bound = v, true
}

// Match returns the value bound to seq, whether one is bound, and whether any
// longer sequence starts with seq.
func (t *Trie[V]) Match(seq []string) (v V, exact, more bool) {
	n := &t.root
	for _, k := range seq {
		child, ok := n.children[k]
		if !ok {
			return v, false, false
		}
		n = child
	}
	return n.value, n.bound, len(n.children) > 0
}

// Walk calls fn for every bound sequence.
func (t *Trie[V]) Walk(fn func(seq []string, v V)) {
	var visit func(n *node[V], prefix []string)
	visit = func(n *node[V], prefix []string) {
		if n.bound {
			fn(prefix, n.value)
		}
		for k, child := range n.children {
			visit(child, append(prefix[:len(prefix):len(prefix)], k))
		}
	}
	visit(&t.root, nil)
}
