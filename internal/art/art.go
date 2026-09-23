// Package art finds album art and renders it into terminal cells. Each
// graphics protocol is a Renderer registered by name, so supporting another
// protocol means registering one more implementation.
package art

import (
	"image"

	"github.com/matjam/encomplayer/internal/registry"
)

// Protocol names a terminal image protocol.
type Protocol = string

// Supported protocols.
const (
	Auto   Protocol = "auto"
	Kitty  Protocol = "kitty"
	ITerm  Protocol = "iterm"
	Blocks Protocol = "blocks"
	Off    Protocol = "off"
)

// Frame is a rendered image ready to place in the layout.
type Frame struct {
	// Setup is written to the terminal once before the image first shows,
	// such as a kitty image upload.
	Setup string

	// Lines fill the art box in the layout, one string per row.
	Lines []string

	// Place, when set, returns a sequence that draws the image with its top
	// left corner at cell (x, y). It is for protocols that draw over the
	// text grid rather than through it.
	Place func(x, y int) string

	// Erase, when set, returns a sequence that clears what Place drew.
	Erase func(x, y int) string

	// Cleanup releases terminal resources once the frame is replaced.
	Cleanup string
}

// Renderer draws an image into a box of cols × rows cells.
type Renderer interface {
	Render(img image.Image, cols, rows int) (Frame, error)
}

// Renderers is the registry of image protocols.
type Renderers = registry.Registry[Renderer]

// DefaultRenderers registers every built-in protocol.
func DefaultRenderers() *Renderers {
	r := registry.New[Renderer]()
	r.Register(Kitty, NewKitty(), Kitty)
	r.Register(ITerm, ITermRenderer{}, ITerm)
	r.Register(Blocks, BlockRenderer{}, Blocks)
	return r
}
