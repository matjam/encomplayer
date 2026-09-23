package keymap

import "strings"

// Result is the outcome of feeding one keystroke to a Resolver.
type Result struct {
	// Action is set when Matched is true.
	Action Action

	// Matched is true when a complete sequence resolved to an action.
	Matched bool

	// Pending is true when the keystroke started or extended a sequence
	// that needs more keys. The caller should call Flush after a timeout
	// with the returned Generation.
	Pending bool

	// Generation identifies the pending sequence for Flush.
	Generation uint64
}

// Resolver turns keystrokes into actions, buffering multi-key sequences.
// Contexts passed to Feed are searched in priority order, so a queue binding
// can shadow a navigation binding, which can shadow a global one.
type Resolver struct {
	keymap     *Keymap
	pending    []string
	generation uint64
}

// NewResolver returns a resolver over km.
func NewResolver(km *Keymap) *Resolver {
	return &Resolver{keymap: km}
}

// Pending returns the buffered keys for display, such as "g" or "ctrl+w".
func (r *Resolver) Pending() string {
	return strings.Join(r.pending, " ")
}

// Reset discards any buffered keys.
func (r *Resolver) Reset() {
	r.pending = nil
	r.generation++
}

// Feed adds a keystroke and resolves it against contexts.
func (r *Resolver) Feed(key string, contexts ...Context) Result {
	seq := append(r.pending[:len(r.pending):len(r.pending)], key)

	action, exact, more := r.lookup(seq, contexts)
	if more {
		r.pending = seq
		r.generation++
		return Result{Pending: true, Generation: r.generation}
	}

	r.Reset()
	if exact {
		return Result{Action: action, Matched: true}
	}

	// The buffered prefix led nowhere; the last key may still start a
	// binding of its own.
	if len(seq) > 1 {
		return r.Feed(key, contexts...)
	}
	return Result{}
}

// Flush resolves a pending sequence once its timeout expires. It fires the
// exact binding for the buffered keys, if there is one, and ignores stale
// generations.
func (r *Resolver) Flush(generation uint64, contexts ...Context) Result {
	if generation != r.generation || len(r.pending) == 0 {
		return Result{}
	}
	seq := r.pending
	r.Reset()
	if action, exact, _ := r.lookup(seq, contexts); exact {
		return Result{Action: action, Matched: true}
	}
	return Result{}
}

func (r *Resolver) lookup(seq []string, contexts []Context) (Action, bool, bool) {
	var (
		found    Action
		hasExact bool
		hasMore  bool
	)
	for _, ctx := range contexts {
		action, exact, more := r.keymap.match(ctx, seq)
		if exact && !hasExact {
			found, hasExact = action, true
		}
		hasMore = hasMore || more
	}
	return found, hasExact, hasMore
}
