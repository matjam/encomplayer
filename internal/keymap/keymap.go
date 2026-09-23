package keymap

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Context is a group of bindings that applies in some part of the UI.
type Context string

// Binding contexts, matching rmpc's keybinds sections.
const (
	Global     Context = "global"
	Navigation Context = "navigation"
	Queue      Context = "queue"
)

// Keymap holds one trie of bindings per context.
type Keymap struct {
	tries map[Context]*Trie[Action]
}

// Default returns rmpc's default bindings.
func Default() *Keymap {
	k := &Keymap{tries: map[Context]*Trie[Action]{}}
	for ctx, binds := range defaultBindings {
		if err := k.BindAll(ctx, binds); err != nil {
			panic(fmt.Sprintf("invalid default keymap: %v", err))
		}
	}
	return k
}

// WithOverrides returns the defaults with binds applied on top, as in the
// config file's keybinds section: context → notation → action.
func WithOverrides(binds map[string]map[string]string) (*Keymap, error) {
	k := Default()
	var errs []error
	for ctx, b := range binds {
		errs = append(errs, k.BindAll(Context(ctx), b))
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return k, nil
}

// Contexts lists the binding contexts in the order the help screen shows
// them.
var Contexts = []Context{Global, Navigation, Queue}

// Bind maps a key sequence in rmpc notation to an action in ctx.
func (k *Keymap) Bind(ctx Context, notation string, action Action) error {
	seq, err := Parse(notation)
	if err != nil {
		return err
	}
	t, ok := k.tries[ctx]
	if !ok {
		t = &Trie[Action]{}
		k.tries[ctx] = t
	}
	t.Insert(seq, action)
	return nil
}

// BindAll applies a map of notation to action strings, such as the keybinds
// section of the config file.
func (k *Keymap) BindAll(ctx Context, binds map[string]string) error {
	var errs []error
	for notation, spec := range binds {
		action, err := ParseAction(spec)
		if err == nil {
			err = k.Bind(ctx, notation, action)
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("%s %q: %w", ctx, notation, err))
		}
	}
	return errors.Join(errs...)
}

// Help lists every binding in ctx as display strings, sorted by action.
func (k *Keymap) Help(ctx Context) [][2]string {
	t, ok := k.tries[ctx]
	if !ok {
		return nil
	}
	var rows [][2]string
	t.Walk(func(seq []string, a Action) {
		rows = append(rows, [2]string{strings.Join(seq, " "), a.String()})
	})
	slices.SortFunc(rows, func(a, b [2]string) int {
		if c := strings.Compare(a[1], b[1]); c != 0 {
			return c
		}
		return strings.Compare(a[0], b[0])
	})
	return rows
}

func (k *Keymap) match(ctx Context, seq []string) (Action, bool, bool) {
	t, ok := k.tries[ctx]
	if !ok {
		return Action{}, false, false
	}
	return t.Match(seq)
}
