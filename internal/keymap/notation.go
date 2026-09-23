// Package keymap maps key sequences written in rmpc's notation to actions and
// resolves multi-key sequences such as "gg" and "<C-w>h".
package keymap

import (
	"fmt"
	"strings"
	"unicode"
)

// namedKeys maps rmpc key names to bubbletea keystroke names.
var namedKeys = map[string]string{
	"cr":       "enter",
	"enter":    "enter",
	"esc":      "esc",
	"tab":      "tab",
	"space":    "space",
	"up":       "up",
	"down":     "down",
	"left":     "left",
	"right":    "right",
	"pageup":   "pgup",
	"pagedown": "pgdown",
	"home":     "home",
	"end":      "end",
	"bs":       "backspace",
	"del":      "delete",
	"delete":   "delete",
	"insert":   "insert",
}

// Parse converts rmpc notation such as "gg", "<C-w>h", "<S-Tab>" or "<C-U>"
// into keystrokes in the form bubbletea's KeyPressMsg.String returns.
func Parse(notation string) ([]string, error) {
	if notation == "" {
		return nil, fmt.Errorf("empty key notation")
	}

	var keys []string
	rest := notation
	for rest != "" {
		if rest[0] == '<' {
			end := strings.IndexByte(rest, '>')
			if end > 1 {
				key, err := parseBracketed(rest[1:end])
				if err != nil {
					return nil, fmt.Errorf("key %q: %w", notation, err)
				}
				keys = append(keys, key)
				rest = rest[end+1:]
				continue
			}
		}

		r := []rune(rest)[0]
		keys = append(keys, plainKey(r))
		rest = rest[len(string(r)):]
	}
	return keys, nil
}

func plainKey(r rune) string {
	if r == ' ' {
		return "space"
	}
	return string(r)
}

func parseBracketed(body string) (string, error) {
	var ctrl, alt, shift bool
	parts := strings.Split(body, "-")
	name := parts[len(parts)-1]

	// "<C-->" or "<->" style bodies leave an empty final part.
	if name == "" && len(parts) > 1 {
		name = "-"
		parts = parts[:len(parts)-1]
	}

	for _, mod := range parts[:len(parts)-1] {
		switch strings.ToUpper(mod) {
		case "C":
			ctrl = true
		case "A", "M":
			alt = true
		case "S":
			shift = true
		default:
			return "", fmt.Errorf("unknown modifier %q", mod)
		}
	}

	base, err := keyName(name, &shift)
	if err != nil {
		return "", err
	}

	// Unmodified or shift-only printable keys arrive as their text.
	if !ctrl && !alt && len([]rune(base)) == 1 {
		if shift {
			return strings.ToUpper(base), nil
		}
		return base, nil
	}

	var b strings.Builder
	if ctrl {
		b.WriteString("ctrl+")
	}
	if alt {
		b.WriteString("alt+")
	}
	if shift {
		b.WriteString("shift+")
	}
	b.WriteString(base)
	return b.String(), nil
}

// keyName resolves a key name. An upper-case letter under a modifier, as in
// rmpc's "<C-U>", means shift is held too.
func keyName(name string, shift *bool) (string, error) {
	if named, ok := namedKeys[strings.ToLower(name)]; ok {
		return named, nil
	}
	if len(name) > 1 && (name[0] == 'F' || name[0] == 'f') {
		return strings.ToLower(name), nil
	}

	runes := []rune(name)
	if len(runes) != 1 {
		return "", fmt.Errorf("unknown key %q", name)
	}
	r := runes[0]
	if unicode.IsUpper(r) {
		*shift = true
		return string(unicode.ToLower(r)), nil
	}
	return string(r), nil
}
