// Command encomplayer is a terminal music player styled after ENCOM OS-12.
//
// Usage:
//
//	encomplayer [flags] [music-dir]
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/matjam/encomplayer/internal/art"
	"github.com/matjam/encomplayer/internal/audio"
	"github.com/matjam/encomplayer/internal/audio/beepdec"
	"github.com/matjam/encomplayer/internal/audio/ffmpegdec"
	"github.com/matjam/encomplayer/internal/config"
	"github.com/matjam/encomplayer/internal/keymap"
	"github.com/matjam/encomplayer/internal/library"
	"github.com/matjam/encomplayer/internal/library/ffprobe"
	"github.com/matjam/encomplayer/internal/playlist"
	"github.com/matjam/encomplayer/internal/ui"
)

var version = "v0.1.0"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "encomplayer: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	paths, err := config.DefaultPaths()
	if err != nil {
		return err
	}

	artFlag := flag.String("art", "", "album art protocol: auto, kitty, iterm, blocks or off")
	configPath := flag.String("config", paths.Config, "config file")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "usage: encomplayer [flags] [music-dir]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	if *showVersion {
		fmt.Println("encomplayer", version)
		return nil
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	if *artFlag != "" {
		cfg.AlbumArt = *artFlag
	}

	km := keymap.Default()
	for ctx, binds := range cfg.Keybinds {
		if err := km.BindAll(keymap.Context(ctx), binds); err != nil {
			return fmt.Errorf("config keybinds: %w", err)
		}
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

	renderer, protocol, err := chooseArt(cfg.AlbumArt)
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
		Config:      cfg,
		State:       state,
		StatePath:   paths.State,
		MusicDir:    musicDir(cfg),
		Version:     version,
	})

	if _, err := tea.NewProgram(model, tea.WithContext(ctx)).Run(); err != nil && !errors.Is(err, tea.ErrProgramKilled) {
		return fmt.Errorf("run ui: %w", err)
	}
	return nil
}

// chooseArt resolves the configured protocol to a renderer. "off" returns a
// nil renderer.
func chooseArt(setting string) (art.Renderer, string, error) {
	protocol := strings.ToLower(setting)
	if protocol == "" || protocol == art.Auto {
		protocol = art.Detect(os.Getenv)
	}
	if protocol == art.Off {
		return nil, art.Off, nil
	}
	r, ok := art.DefaultRenderers().Get(protocol)
	if !ok {
		return nil, "", fmt.Errorf("unknown album art protocol %q", setting)
	}
	return r, protocol, nil
}

// musicDir picks the folder to scan: the argument, then the config file,
// then ~/Music when it exists.
func musicDir(cfg config.Config) string {
	if flag.NArg() > 0 {
		return flag.Arg(0)
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
