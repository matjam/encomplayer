package art

import "strings"

// Detect picks the best protocol for the terminal described by getenv.
//
// kitty and Ghostty support kitty's Unicode placeholders, which flow through
// the normal text layout. iTerm2 and WezTerm support iTerm2 inline images.
// foot supports sixel. Anything else gets half-block rendering. tmux and
// screen get half blocks too, because graphics passthrough depends on their
// configuration.
func Detect(getenv func(string) string) Protocol {
	term := getenv("TERM")
	program := getenv("TERM_PROGRAM")

	switch {
	case getenv("TMUX") != "" || strings.HasPrefix(term, "screen") || strings.HasPrefix(term, "tmux"):
		return Blocks
	case getenv("KITTY_WINDOW_ID") != "" || term == "xterm-kitty":
		return Kitty
	case strings.EqualFold(program, "ghostty") || term == "xterm-ghostty":
		return Kitty
	case program == "iTerm.app" || getenv("LC_TERMINAL") == "iTerm2":
		return ITerm
	case program == "WezTerm":
		return ITerm
	case term == "foot" || strings.HasPrefix(term, "foot-"):
		return Sixel
	default:
		return Blocks
	}
}
