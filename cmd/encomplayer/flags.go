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
	listViz    bool
	paths      bool
	reload     bool

	// command controls a running player, e.g. "next" or "seek +10".
	command     string
	commandArgs []string
	json        bool
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
	fs.BoolVar(&opts.listViz, "list-visualizers", false, "list the visualizers and exit")
	fs.BoolVar(&opts.paths, "paths", false, "print where config, themes, playlists, cache and state live, and exit")
	fs.BoolVar(&opts.reload, "reload", false, "make the running player reread its config and theme (same as the reload command)")
	fs.BoolVar(&opts.json, "json", false, "with the status command, print JSON")
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
	for name, on := range map[string]bool{"--help": o.help, "--version": o.version, "--list-themes": o.listThemes, "--list-visualizers": o.listViz, "--paths": o.paths, "--reload": o.reload} {
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

// commandArity is how many arguments each control command takes.
var commandArity = map[string][2]int{
	"status": {0, 0}, "play": {0, 0}, "pause": {0, 0}, "toggle": {0, 0},
	"stop": {0, 0}, "next": {0, 0}, "prev": {0, 0},
	"seek": {1, 1}, "volume": {1, 1},
	"repeat": {0, 1}, "random": {0, 1}, "single": {0, 1}, "consume": {0, 1},
	"shuffle": {0, 0}, "shuffle-all": {0, 0}, "add": {1, 1}, "reload": {0, 0},
	"viz": {0, 1},
}

// valueFlags take a separate value, which must not be mistaken for a
// command name.
var valueFlags = []string{"-a", "--art", "-t", "--theme", "-c", "--config"}

// splitCommand finds a control command: the first positional argument, if
// it names one. Everything after it belongs to the command, so "seek -30"
// is not read as a flag. "--" ends the search, so "-- next" opens a folder
// called next.
func splitCommand(args []string) (flagArgs []string, cmd string, cmdArgs []string) {
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			return args, "", nil
		case slices.Contains(valueFlags, a):
			i++
		case strings.HasPrefix(a, "-") && len(a) > 1:
		default:
			if _, ok := commandArity[a]; ok {
				return args[:i], a, args[i+1:]
			}
			return args, "", nil
		}
	}
	return args, "", nil
}

// parseArgs parses args, which exclude the program name.
func parseArgs(args []string, defaultConfig string) (options, error) {
	var opts options
	fs := newFlagSet(&opts, defaultConfig)
	flagArgs, cmd, cmdArgs := splitCommand(upgradeLegacy(args))

	// "status --json" is the natural spelling, so accept the flag after
	// the command too.
	if cmd == "status" {
		if i := slices.Index(cmdArgs, "--json"); i >= 0 {
			opts.json = true
			cmdArgs = slices.Delete(slices.Clone(cmdArgs), i, i+1)
		}
	}
	if err := fs.Parse(flagArgs); err != nil {
		return opts, usageError{err}
	}
	switch rest := fs.Args(); len(rest) {
	case 0:
	case 1:
		opts.musicDir = rest[0]
	default:
		return opts, usageError{fmt.Errorf("expected at most one music folder, got %d arguments", len(rest))}
	}
	if opts.reload && cmd == "" {
		cmd = "reload"
		opts.reload = false
	}
	if len(cmdArgs) == 0 {
		cmdArgs = nil
	}
	opts.command, opts.commandArgs = cmd, cmdArgs
	return opts, opts.validate()
}

func (o options) validate() error {
	if cmds := o.exclusive(); len(cmds) > 1 || (len(cmds) == 1 && o.command != "") {
		all := cmds
		if o.command != "" {
			all = append(all, o.command)
		}
		return usageError{fmt.Errorf("%s cannot be combined", strings.Join(all, " and "))}
	}
	if o.json && o.command != "status" {
		return usageError{errors.New("--json only applies to the status command")}
	}
	if o.command == "" {
		return nil
	}
	if o.art != "" || o.theme != "" || o.shuffle || o.rescan || o.noMouse || o.musicDir != "" {
		return usageError{fmt.Errorf("%s controls a running player; options for starting one do not apply", o.command)}
	}
	arity := commandArity[o.command]
	if n := len(o.commandArgs); n < arity[0] || n > arity[1] {
		return usageError{fmt.Errorf("%s takes %s", o.command, describeArity(arity))}
	}
	return nil
}

func describeArity(a [2]int) string {
	switch {
	case a[1] == 0:
		return "no arguments"
	case a[0] == a[1]:
		return "one argument"
	default:
		return "at most one argument"
	}
}

// printUsage writes the help text.
func printUsage(w io.Writer, defaultConfig string) {
	var opts options
	fs := newFlagSet(&opts, defaultConfig)
	fmt.Fprintf(w, `EncomPlayer: a terminal music player styled after ENCOM OS-12.

Usage:
  encomplayer [options] [music-folder]    start the player
  encomplayer <command> [argument]        control the running player

The music folder defaults to music_dir in the config file, then ~/Music.
Options that apply to one run (--art, --theme, --no-mouse, --shuffle,
--rescan) are never written to the config file.

Commands:
  status [--json]               current track, position, volume and modes
  play | pause | toggle | stop
  next | prev
  seek +N | -N | SECONDS | M:SS relative or absolute
  volume N | +N | -N            0-100, or a step
  repeat | random | single | consume [on|off]   toggles without an argument
  shuffle                       shuffle the queue
  shuffle-all                   play the whole library shuffled
  add PATH                      append a file or folder to the queue
  reload                        reread config.json and the theme
  viz [NAME | next | prev]      switch visualizer; alone, toggle full screen

A folder named like a command opens with ./name or -- name.

Options:
%s
Examples:
  encomplayer -s                  shuffle everything and start playing
  encomplayer -t dracula ~/Music  one run with the dracula theme
  encomplayer seek +30            skip ahead in the running player
  encomplayer status --json | jq .title

Press ? inside the player for keys, or oc for the config screen.
`, fs.FlagUsages())
}

// isUsage reports whether err came from a bad command line.
func isUsage(err error) bool {
	var u usageError
	return errors.As(err, &u)
}
