package main

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/spf13/pflag"
)

// options is the parsed command line.
type options struct {
	art      string
	theme    string
	config   string
	musicDir string

	// Per-run switches; none of them is written to the config file.
	shuffle bool
	rescan  bool
	noMouse bool

	// Commands that print or signal and then exit.
	version    bool
	help       bool
	listThemes bool
	paths      bool
	reload     bool
}

// usageError is a bad command line. main exits 2 for it, as getopt tools
// do, rather than 1 for runtime failures.
type usageError struct{ err error }

func (u usageError) Error() string { return u.err.Error() }
func (u usageError) Unwrap() error { return u.err }

// newFlagSet declares the flags with GNU-style short and long forms.
func newFlagSet(opts *options, defaultConfig string) *pflag.FlagSet {
	fs := pflag.NewFlagSet("encomplayer", pflag.ContinueOnError)
	fs.SortFlags = true
	fs.StringVarP(&opts.art, "art", "a", "", "album art protocol for this run: auto, kitty, iterm, blocks or off")
	fs.StringVarP(&opts.theme, "theme", "t", "", "theme for this run (see --list-themes)")
	fs.StringVarP(&opts.config, "config", "c", defaultConfig, "config file")
	fs.BoolVarP(&opts.shuffle, "shuffle", "s", false, "start playing the whole library shuffled")
	fs.BoolVarP(&opts.rescan, "rescan", "r", false, "reread every file's tags at startup instead of trusting the cache")
	fs.BoolVar(&opts.noMouse, "no-mouse", false, "leave the mouse to the terminal for this run, e.g. to select text")
	fs.BoolVar(&opts.listThemes, "list-themes", false, "list built-in and custom themes and exit")
	fs.BoolVar(&opts.paths, "paths", false, "print where config, themes, playlists, cache and state live, and exit")
	fs.BoolVar(&opts.reload, "reload", false, "make the running player reread its config and theme, and exit")
	fs.BoolVarP(&opts.version, "version", "v", false, "print the version and exit")
	fs.BoolVarP(&opts.help, "help", "h", false, "show this help and exit")

	// Errors are reported once by main, with a pointer to --help.
	fs.SetOutput(io.Discard)
	return fs
}

// legacyLong are the single-dash long flags v1.2 and earlier accepted.
// POSIX parsing would read "-config x" as "-c onfig" plus a folder named x,
// silently doing the wrong thing, so they are rewritten to "--" first.
var legacyLong = []string{"art", "config", "version", "help"}

// exclusive are the commands that print or signal and exit; only one may be
// given.
func (o options) exclusive() []string {
	var set []string
	for name, on := range map[string]bool{"--help": o.help, "--version": o.version, "--list-themes": o.listThemes, "--paths": o.paths, "--reload": o.reload} {
		if on {
			set = append(set, name)
		}
	}
	slices.Sort(set)
	return set
}

func upgradeLegacy(args []string) []string {
	out := make([]string, len(args))
	copy(out, args)
	for i, a := range out {
		if a == "--" {
			break
		}
		name, _, _ := strings.Cut(strings.TrimPrefix(a, "-"), "=")
		if strings.HasPrefix(a, "-") && !strings.HasPrefix(a, "--") && slices.Contains(legacyLong, name) {
			out[i] = "-" + a
		}
	}
	return out
}

// parseArgs parses args, which exclude the program name.
func parseArgs(args []string, defaultConfig string) (options, error) {
	var opts options
	fs := newFlagSet(&opts, defaultConfig)
	if err := fs.Parse(upgradeLegacy(args)); err != nil {
		return opts, usageError{err}
	}
	switch rest := fs.Args(); len(rest) {
	case 0:
	case 1:
		opts.musicDir = rest[0]
	default:
		return opts, usageError{fmt.Errorf("expected at most one music folder, got %d arguments", len(rest))}
	}
	if cmds := opts.exclusive(); len(cmds) > 1 {
		return opts, usageError{fmt.Errorf("%s cannot be combined", strings.Join(cmds, " and "))}
	}
	return opts, nil
}

// printUsage writes the help text.
func printUsage(w io.Writer, defaultConfig string) {
	var opts options
	fs := newFlagSet(&opts, defaultConfig)
	fmt.Fprintf(w, `EncomPlayer: a terminal music player styled after ENCOM OS-12.

Usage:
  encomplayer [options] [music-folder]

The music folder defaults to music_dir in the config file, then ~/Music.
Options that apply to one run (--art, --theme, --no-mouse, --shuffle,
--rescan) are never written to the config file.

Options:
%s
Examples:
  encomplayer -s                  shuffle everything and start playing
  encomplayer -t dracula ~/Music  one run with the dracula theme
  encomplayer --reload            apply edits to config.json or a theme

Press ? inside the player for keys, or oc for the config screen.
`, fs.FlagUsages())
}

// isUsage reports whether err came from a bad command line.
func isUsage(err error) bool {
	var u usageError
	return errors.As(err, &u)
}
