package art

import "strings"

// SixelAttribute is the primary device attribute (DA1) terminals report when
// they support sixel graphics.
const SixelAttribute = 4

// Terminal is what is known about the terminal: its environment, and whether
// it has said it supports sixel once asked.
type Terminal struct {
	Getenv func(string) string

	// Sixel is set when the terminal's primary device attributes include
	// SixelAttribute.
	Sixel bool
}

// Detect picks the best protocol for term.
//
// kitty and Ghostty support kitty's Unicode placeholders, which flow through
// the normal text layout. iTerm2 and WezTerm support iTerm2 inline images.
// These are recognised from the environment.
//
// Anything else that reports sixel support gets sixel. That covers foot
// whatever TERM says, since it is often run as xterm-256color. foot's own
// TERM values are recognised too, so sixel is chosen before the terminal has
// answered. Everything else gets half blocks.
//
// Under tmux and screen, kitty and iTerm2 passthrough depends on their
// configuration, so those are never chosen there. tmux reports sixel only
// when it draws sixel itself.
func Detect(term Terminal) Protocol {
	getenv := term.Getenv
	name := getenv("TERM")
	program := getenv("TERM_PROGRAM")

	switch {
	case getenv("TMUX") != "" || strings.HasPrefix(name, "screen") || strings.HasPrefix(name, "tmux"):
		return fallback(term)
	case getenv("KITTY_WINDOW_ID") != "" || name == "xterm-kitty":
		return Kitty
	case strings.EqualFold(program, "ghostty") || name == "xterm-ghostty":
		return Kitty
	case program == "iTerm.app" || getenv("LC_TERMINAL") == "iTerm2":
		return ITerm
	case program == "WezTerm":
		return ITerm
	case name == "foot" || strings.HasPrefix(name, "foot-"):
		return Sixel
	default:
		return fallback(term)
	}
}

// fallback is sixel when the terminal reports it, and half blocks otherwise.
func fallback(term Terminal) Protocol {
	if term.Sixel {
		return Sixel
	}
	return Blocks
}
