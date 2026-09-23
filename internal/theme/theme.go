// Package theme defines EncomPlayer's colour themes: the built-in palettes
// and custom themes loaded from JSON files.
//
// A theme assigns colours to UI roles rather than to widgets, so every
// palette maps onto the same small set of decisions.
package theme

import (
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
)

// Default is the theme used when none is configured.
const Default = "encom"

// ErrUnknown means no built-in or custom theme has the requested name.
var ErrUnknown = errors.New("unknown theme")

// Theme is a named palette. Colours are "#rrggbb" strings; an empty
// Background keeps the terminal's own background.
type Theme struct {
	Name string `json:"name"`

	// Background fills the screen behind everything.
	Background string `json:"background"`

	// Text is ordinary list and table text.
	Text string `json:"text"`

	// Bright emphasises titles, folder names and the current track.
	Bright string `json:"bright"`

	// Dim is secondary text such as counts, hints and column headers.
	Dim string `json:"dim"`

	// Grid draws faint rules, inactive mode flags and empty meter segments.
	Grid string `json:"grid"`

	// Border outlines unfocused panels. Focused panels use Text.
	Border string `json:"border"`

	// Accent marks the playing track, selections and active modes.
	Accent string `json:"accent"`

	// Error colours error messages.
	Error string `json:"error"`

	// Selection and SelectionText colour the cursor row and active tab.
	Selection     string `json:"selection"`
	SelectionText string `json:"selection_text"`
}

var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Validate reports colours that are missing or not "#rrggbb".
func (t Theme) Validate() error {
	var errs []error
	for role, c := range t.colors() {
		if role == "background" && c == "" {
			continue
		}
		if !hexColor.MatchString(c) {
			errs = append(errs, fmt.Errorf("%s: %q is not a #rrggbb colour", role, c))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("theme %q: %w", t.Name, errors.Join(errs...))
	}
	return nil
}

func (t Theme) colors() map[string]string {
	return map[string]string{
		"background":     t.Background,
		"text":           t.Text,
		"bright":         t.Bright,
		"dim":            t.Dim,
		"grid":           t.Grid,
		"border":         t.Border,
		"accent":         t.Accent,
		"error":          t.Error,
		"selection":      t.Selection,
		"selection_text": t.SelectionText,
	}
}

// overlay returns t with every non-empty colour in o applied on top.
func (t Theme) overlay(o map[string]string) Theme {
	set := map[string]*string{
		"background":     &t.Background,
		"text":           &t.Text,
		"bright":         &t.Bright,
		"dim":            &t.Dim,
		"grid":           &t.Grid,
		"border":         &t.Border,
		"accent":         &t.Accent,
		"error":          &t.Error,
		"selection":      &t.Selection,
		"selection_text": &t.SelectionText,
	}
	for role, c := range o {
		if p, ok := set[role]; ok {
			*p = c
		}
	}
	return t
}

// Builtin returns the built-in theme called name.
func Builtin(name string) (Theme, bool) {
	t, ok := builtins[name]
	return t, ok
}

// BuiltinNames lists the built-in themes, sorted.
func BuiltinNames() []string {
	return slices.Sorted(maps.Keys(builtins))
}
