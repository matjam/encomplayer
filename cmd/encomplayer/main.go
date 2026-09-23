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
	"github.com/matjam/encomplayer/internal/theme"
	"github.com/matjam/encomplayer/internal/ui"
)

// version is set at build time with -ldflags "-X main.version=v1.2.3".
var version = "dev"

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

	paths.Config = *configPath
	cfg, err := config.Load(paths.Config)
	if err != nil {
		return err
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
	if *artFlag != "" {
		artSetting = *artFlag
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
		Config:      cfg,
		State:       state,
		Paths:       paths,
		MusicDir:    musicDir(cfg),
		Version:     version,
	})

	program := tea.NewProgram(model, tea.WithContext(ctx))
	stopReload := notifyReload(func() { program.Send(ui.ReloadMsg{}) })
	defer stopReload()

	if _, err := program.Run(); err != nil && !errors.Is(err, tea.ErrProgramKilled) {
		return fmt.Errorf("run ui: %w", err)
	}
	return nil
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
