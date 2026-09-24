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
	config   string
	musicDir string
	version  bool
	help     bool
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
	fs.StringVarP(&opts.config, "config", "c", defaultConfig, "config file")
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

Options:
%s
Press ? inside the player for keys, or oc for the config screen.
`, fs.FlagUsages())
}

// isUsage reports whether err came from a bad command line.
func isUsage(err error) bool {
	var u usageError
	return errors.As(err, &u)
}
