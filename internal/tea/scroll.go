package tea

// EncomPlayer addition: not in upstream bubbletea. See README.md.

// SetScrollOptimization returns a command that turns the renderer's hard
// scroll optimisation on or off. It is on by default except on Windows.
//
// With it on, the renderer moves unchanged lines by scrolling regions of the
// screen. Some terminals, foot among them, also move sixel images outside the
// scrolled region, so programs that draw images over the text grid turn it
// off while an image is showing.
func SetScrollOptimization(on bool) Cmd {
	return func() Msg { return scrollOptimMsg(on) }
}

// scrollOptimMsg carries a SetScrollOptimization request to the renderer.
type scrollOptimMsg bool
