package collection

// Provider describes a hierarchy that a Browser can walk, such as directories
// or artist → album → track.
type Provider[T any] interface {
	// Root returns the top level.
	Root() []T

	// Children returns the level below node. Leaves return nil.
	Children(node T) []T

	// IsLeaf reports whether node has no level below it.
	IsLeaf(node T) bool

	// Key identifies node among its siblings, so Reload can restore the path.
	Key(node T) string
}

// Browser keeps a stack of lists, one per level descended, in the style of
// ranger or rmpc's Miller columns.
type Browser[T any] struct {
	provider Provider[T]
	stack    []*List[T]
}

// NewBrowser returns a browser positioned at the provider's root.
func NewBrowser[T any](p Provider[T]) *Browser[T] {
	return &Browser[T]{provider: p, stack: []*List[T]{NewList(p.Root())}}
}

// Provider returns the hierarchy being browsed.
func (b *Browser[T]) Provider() Provider[T] { return b.provider }

// Current returns the list at the current level.
func (b *Browser[T]) Current() *List[T] { return b.stack[len(b.stack)-1] }

// Parent returns the list one level up, or nil at the root.
func (b *Browser[T]) Parent() *List[T] {
	if len(b.stack) < 2 {
		return nil
	}
	return b.stack[len(b.stack)-2]
}

// Depth returns how many levels below the root the browser is.
func (b *Browser[T]) Depth() int { return len(b.stack) - 1 }

// Preview returns the children of the highlighted node, or nil for a leaf.
func (b *Browser[T]) Preview() []T {
	node, ok := b.Current().Current()
	if !ok || b.provider.IsLeaf(node) {
		return nil
	}
	return b.provider.Children(node)
}

// Enter descends into the highlighted node. It reports false for leaves.
func (b *Browser[T]) Enter() bool {
	node, ok := b.Current().Current()
	if !ok || b.provider.IsLeaf(node) {
		return false
	}
	next := NewList(b.provider.Children(node))
	next.SetHeight(b.Current().Height())
	b.stack = append(b.stack, next)
	return true
}

// Leave ascends one level. It reports false at the root.
func (b *Browser[T]) Leave() bool {
	if len(b.stack) < 2 {
		return false
	}
	b.stack = b.stack[:len(b.stack)-1]
	return true
}

// Path returns the highlighted node at each level above the current one.
func (b *Browser[T]) Path() []T {
	var path []T
	for _, l := range b.stack[:len(b.stack)-1] {
		if node, ok := l.Current(); ok {
			path = append(path, node)
		}
	}
	return path
}

// SetProvider swaps the hierarchy and re-walks the previous path by key, so a
// rescan keeps the user where they were when the nodes still exist.
func (b *Browser[T]) SetProvider(p Provider[T]) {
	keys := make([]string, 0, len(b.stack))
	cursors := make([]int, 0, len(b.stack))
	for _, l := range b.stack {
		key := ""
		if node, ok := l.Current(); ok {
			key = b.provider.Key(node)
		}
		keys = append(keys, key)
		cursors = append(cursors, l.Cursor())
	}

	height := b.Current().Height()
	b.provider = p
	b.stack = []*List[T]{NewList(p.Root())}
	b.stack[0].SetHeight(height)

	for depth, key := range keys {
		l := b.Current()
		if !l.Find(func(n T) bool { return p.Key(n) == key }, true, true) {
			l.SetCursor(cursors[depth])
			return
		}
		if depth == len(keys)-1 || !b.Enter() {
			return
		}
	}
}
