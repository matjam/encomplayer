// Command encomplayer is a terminal music player styled after ENCOM OS-12.
//
// Usage:
//
//	encomplayer [options] [music-folder]
//
// Run encomplayer --help for the options.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/charmbracelet/x/term"

	"github.com/matjam/encomplayer/internal/art"
	"github.com/matjam/encomplayer/internal/audio"
	"github.com/matjam/encomplayer/internal/audio/beepdec"
	"github.com/matjam/encomplayer/internal/audio/ffmpegdec"
	"github.com/matjam/encomplayer/internal/config"
	"github.com/matjam/encomplayer/internal/keymap"
	"github.com/matjam/encomplayer/internal/library"
	"github.com/matjam/encomplayer/internal/library/ffprobe"
	"github.com/matjam/encomplayer/internal/playlist"
	"github.com/matjam/encomplayer/internal/tea"
	"github.com/matjam/encomplayer/internal/theme"
	"github.com/matjam/encomplayer/internal/ui"
	"github.com/matjam/encomplayer/internal/viz"
)

// version is set at build time with -ldflags "-X main.version=v1.2.3".
var version = "dev"

func main() {
	err := run(os.Args[1:])
	switch {
	case err == nil:
	case isUsage(err):
		fmt.Fprintf(os.Stderr, "encomplayer: %v\nRun 'encomplayer --help' for usage.\n", err)
		os.Exit(2)
	default:
		fmt.Fprintf(os.Stderr, "encomplayer: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	paths, err := config.DefaultPaths()
	if err != nil {
		return err
	}

	opts, err := parseArgs(args, paths.Config)
	if err != nil {
		return err
	}
	if opts.help {
		printUsage(os.Stdout, paths.Config)
		return nil
	}

	paths.Config = opts.config
	cfg, err := config.Load(paths.Config)

	// Informational commands run even with a broken config file, falling
	// back to defaults, so they stay useful for diagnosing it.
	tty := term.IsTerminal(os.Stdout.Fd())
	switch {
	case opts.version:
		showVersion(opts, cfg, paths)
		return nil
	case opts.paths:
		printPaths(os.Stdout, paths)
		return nil
	case opts.listThemes:
		current := cfg.Theme
		if opts.theme != "" {
			current = opts.theme
		}
		printThemes(os.Stdout, tty, theme.NewStore(paths.Themes), current)
		return nil
	case opts.listViz:
		printVisualizers(os.Stdout, tty, viz.Builtins.List(), cfg.Visualizer)
		return nil
	case opts.command != "":
		return runControl(os.Stdout, socketPath(paths), opts)
	}
	if err != nil {
		return err
	}
	if opts.theme != "" {
		if _, err := theme.NewStore(paths.Themes).Load(opts.theme); err != nil {
			return usageError{fmt.Errorf("--theme: %w; see --list-themes", err)}
		}
	}

	km, err := keymap.WithOverrides(cfg.Keybinds)
	if err != nil {
		return fmt.Errorf("config keybinds: %w", err)
	}

	decoders := audio.NewDecoders()
	beepdec.Register(decoders)
	formats := "MP3 FLAC OGG WAV"
	if ff, err := ffmpegdec.New(); err == nil {
		ffmpegdec.Register(decoders, ff)
		formats += " + FFMPEG"
	} else if !errors.Is(err, ffmpegdec.ErrNotInstalled) {
		return err
	}

	readers := []library.TagReader{library.NativeReader{}}
	if fp, err := ffprobe.New(); err == nil {
		readers = append(readers, fp)
	}
	tagger := library.NewTagger(readers...)

	// The --art flag applies to this run only; it is not written back to
	// the config file.
	artSetting := cfg.AlbumArt
	if opts.art != "" {
		artSetting = opts.art
	}
	renderer, protocol, err := art.Choose(artSetting, os.Getenv)
	if err != nil {
		return err
	}

	state := config.LoadState(paths.State)
	player, err := audio.NewPlayer(decoders, state.Volume)
	if err != nil {
		return err
	}
	defer player.Close()

	probe := func(ctx context.Context, path string) (time.Duration, error) {
		return audio.Duration(ctx, decoders, path)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	model := ui.New(ctx, ui.Deps{
		Player:      player,
		Scanner:     library.NewScanner(decoders.Has, tagger, probe),
		Tags:        tagger,
		Cache:       library.NewCache(paths.Cache),
		Playlists:   playlist.NewStore(paths.Playlists),
		Keymap:      km,
		Art:         renderer,
		ArtProtocol: protocol,
		Formats:     formats,
		Themes:      theme.NewStore(paths.Themes),
		Startup: ui.Startup{
			Theme:      opts.theme,
			NoMouse:    opts.noMouse,
			FullRescan: opts.rescan,
			Shuffle:    opts.shuffle,
		},
		Config:   cfg,
		State:    state,
		Paths:    paths,
		MusicDir: musicDir(opts.musicDir, cfg),
		Version:  version,
	})

	program := tea.NewProgram(model, tea.WithContext(ctx))
	stopReload := notifyReload(func() { program.Send(ui.ReloadMsg{}) })
	defer stopReload()

	stopRemote := serveRemote(socketPath(paths), program)
	defer stopRemote()

	if _, err := program.Run(); err != nil && !errors.Is(err, tea.ErrProgramKilled) {
		return fmt.Errorf("run ui: %w", err)
	}
	return nil
}

// showVersion prints the version banner using the configured theme.
func showVersion(opts options, cfg config.Config, paths config.Paths) {
	tty := term.IsTerminal(os.Stdout.Fd())
	if opts.theme != "" {
		cfg.Theme = opts.theme
	}
	b := banner{version: version}
	if tty {
		t, err := theme.NewStore(paths.Themes).Load(cfg.Theme)
		if err != nil {
			t, _ = theme.Builtin(theme.Default)
		}
		artSetting := cfg.AlbumArt
		if opts.art != "" {
			artSetting = opts.art
		}
		b.theme = t
		b.terminal = detectTerminal(os.Getenv)
		b.artProto = artProtocolName(artSetting, os.Getenv)
		b.ffmpeg = ffmpegVersion()
		b.musicDir = musicDir(opts.musicDir, cfg)
	}
	printVersion(os.Stdout, tty, b)
}

// musicDir picks the folder to scan: the argument, then the config file,
// then ~/Music when it exists.
func musicDir(arg string, cfg config.Config) string {
	if arg != "" {
		return arg
	}
	if cfg.MusicDir != "" {
		return cfg.MusicDir
	}
	if home, err := os.UserHomeDir(); err == nil {
		dir := filepath.Join(home, "Music")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	return ""
}
