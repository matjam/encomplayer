# EncomPlayer

A terminal music player styled after ENCOM OS-12 from *TRON: Legacy*, with
rmpc's keybindings. No MPD required.

```
encomplayer [options] [music-folder]

  -a, --art string      album art protocol for this run: auto, kitty, iterm, blocks or off
  -c, --config string   config file (default ~/.config/encomplayer/config.json)
  -h, --help            show help and exit
  -v, --version         print the version and exit
```

The music folder comes from the argument, then `music_dir` in the config, then
`~/Music`. On a terminal, `--version` shows a diagnostics banner (terminal,
colour depth, album art protocol, decoders); piped, it prints one plain line.

## Install

### Script (Linux and macOS)

```sh
curl -fsSL https://raw.githubusercontent.com/matjam/encomplayer/main/install.sh | sh
```

Installs the latest release to `~/.local/bin` after checking it against the
release's `SHA256SUMS`. Set `BINDIR` to install elsewhere or `VERSION` to pick
a release, e.g. `curl -fsSL … | BINDIR=/usr/local/bin VERSION=v1.1.0 sh`.

### Homebrew (macOS)

```sh
brew install --cask matjam/tap/encomplayer
```

### Debian and Ubuntu

```sh
v=1.2.0   # the release you want, without the v
curl -fLO https://github.com/matjam/encomplayer/releases/download/v$v/encomplayer_${v}_amd64.deb
sudo apt install ./encomplayer_${v}_amd64.deb
```

### Fedora, RHEL and openSUSE

```sh
v=1.2.0
sudo dnf install https://github.com/matjam/encomplayer/releases/download/v$v/encomplayer-$v-1.x86_64.rpm
```

### Arch Linux

```sh
v=1.2.0
sudo pacman -U https://github.com/matjam/encomplayer/releases/download/v$v/encomplayer-$v-1-x86_64.pkg.tar.zst
```

On ARM, use `arm64` for the `.deb` and `aarch64` for the `.rpm` and Arch
package. Windows users can download the `.zip` from the
[releases page](https://github.com/matjam/encomplayer/releases).

### ffmpeg

MP3, FLAC, Ogg Vorbis and WAV play without anything else. For AAC/M4A, ALAC,
Opus, WavPack and other formats, install ffmpeg as well (`brew install ffmpeg`,
`sudo apt install ffmpeg`, `sudo pacman -S ffmpeg`). The Debian and Fedora
packages suggest it; the Arch package cannot declare optional dependencies,
so install it yourself there.

### From source

```sh
go install github.com/matjam/encomplayer/cmd/encomplayer@latest
```

## Screenshot

<img width="1274" height="892" alt="image" src="https://github.com/user-attachments/assets/c0e1891d-1e01-411b-b4e7-c0aa993a7fbf" />

## Features

- Plays MP3, FLAC, Ogg Vorbis and WAV in pure Go. With ffmpeg installed it also
  plays AAC/M4A, ALAC, Opus, WavPack, AIFF, WMA and anything else ffmpeg reads.
- Browses by folder, artist, album artist, album and genre, with search.
- Queue with repeat, random, single and consume modes; M3U8 playlists.
- Album art in kitty and Ghostty (Unicode placeholders), iTerm2 and WezTerm
  (inline images), and half-block rendering in any truecolor terminal.
- Instant startup from a cache, background sync, and periodic rescans that
  pick up files added while it runs.

## Keys

The defaults match rmpc. Press `?` for the full list.

| Key | Action |
|---|---|
| `1`–`8`, `Tab`, `gt`/`gT` | Switch tab |
| `j`/`k`, `gg`/`G`, `C-u`/`C-d` | Move |
| `h`/`l`, `Enter` | Up / into a level; `Enter` on a track plays it |
| `a` / `A` | Add highlighted (or selected) / add everything listed |
| `Space`, `C-Space` | Select, invert selection |
| `p`, `s`, `>`/`<` | Pause, stop, next/previous |
| `f`/`b`, `.`/`,` | Seek, volume |
| `z` `x` `v` `c` | Repeat, random, single, consume |
| `/`, `n`/`N` | Find in list |
| `C-s s` / `C-s a` | Save selection / everything to a playlist |
| **`S`** | **Play the whole library shuffled** |
| **`X`** | Queue tab: shuffle the queue. Elsewhere: play the highlighted item shuffled |
| `d`, `D`, `J`/`K`, `C` | Queue: delete, clear, move, jump to current |
| `C-u`, `C-U` | Sync library, full rescan (in command mode: `:update`, `:rescan`) |
| `:` | Command mode (`:help` lists commands) |

The Playlists tab starts with **ALL MUSIC**, so `6` `X` also shuffles
everything, and `C-s s` on it saves the library as a playlist.

The queue, current track and play position survive a restart. After
relaunching, `p` resumes the track where it stopped; choosing a track with
`Enter` starts it from the beginning, and `s` clears the saved position.

## Mouse

| Where | Click | Double-click | Wheel |
|---|---|---|---|
| Tab bar | Switch tab | | |
| Queue, browser middle column, search results | Move cursor | Play, or open a folder | Scroll |
| Browser left column | Go up a level | | |
| Browser right column | Open the clicked item | | |
| Seek bar | Seek | | |
| Volume meter, mode flags | Set volume, toggle mode | | |
| Help | | | Scroll |

Drag a divider to resize panes: the SIGNAL strip's top edge (the spectrum
grows with it), the border between album art and the queue, and the borders
between browser columns. Sizes are remembered between sessions.

## Configuration

Press `oc` (or `:config`) for the config screen. Its tabs cover General,
Appearance, Mouse, Keys and Paths; `h`/`l` switch tabs, `j`/`k` pick a
setting and `Enter` edits it. Changes apply immediately and are saved.

The settings live in `~/.config/encomplayer/config.json` (or
`$XDG_CONFIG_HOME`), which you can also edit by hand:

```json
{
  "music_dir": "~/Music",
  "album_art": "auto",
  "theme": "catppuccin-mocha",
  "volume_step": 5,
  "seek_seconds": 5,
  "rescan_seconds": 0,
  "enable_mouse": true,
  "scroll_amount": 1,
  "keybinds": {
    "global": { "<C-p>": "TogglePause" },
    "queue": { "x": "Delete" }
  }
}
```

`rescan_seconds` of 0 checks every 60 s on local disks and every 10 minutes on
network mounts. A negative value disables periodic rescans.

Playlists live in `~/.config/encomplayer/playlists`, the library cache in
`~/.cache/encomplayer`, and the saved queue in `~/.local/state/encomplayer`.

After editing the file, send `SIGUSR1` (`pkill -USR1 encomplayer`) or run
`:reload` to apply it without restarting.

## Themes

Built in: `encom` (default), `catppuccin-latte`, `-frappe`, `-macchiato`,
`-mocha`, `gruvbox-dark`, `-light`, `tokyonight-night`, `-storm`, `-day`,
`rose-pine`, `-moon`, `-dawn`, `solarized-dark`, `-light`, `kanagawa-wave`,
`-dragon`, `everforest-dark`, `-light`, `ayu-dark`, `-mirage`,
`github-dark`, `-light`, `dracula`, `nord`, `one-dark`, `monokai`,
`nightfox`, `material`, `palenight`, `synthwave-84`, `night-owl`,
`oxocarbon` and `cyberpunk`.

Pick one in the config screen, where moving through the list previews each
theme, or with `:theme <name>`.

A custom theme is a JSON file in `~/.config/encomplayer/themes/`, named for
the theme. It extends a built-in and overrides any of the colour roles:

```json
{
  "extends": "catppuccin-mocha",
  "colors": {
    "accent": "#f5c2e7",
    "background": ""
  }
}
```

Roles: `background` (empty keeps the terminal's), `text`, `bright`, `dim`,
`grid`, `border`, `accent`, `error`, `selection`, `selection_text`. Save the
file and send `SIGUSR1` to see the change live. A custom file with a
built-in's name replaces that built-in.

## Library cache

The cache records each track's tags, size and modification time, plus each
folder's listing and modification time. At startup the cached library shows
immediately and a sync runs in the background. The sync relists only folders
whose modification time changed and rereads only files whose size or time
changed, so a network library of 10,000 tracks syncs in a few seconds.

A file retagged in place without its folder changing is only noticed by a full
rescan (`C-U` or `:rescan`).

## Extending

- **Formats:** implement `audio.Decoder` and register it for its extensions.
  Decoders registered earlier take priority.
- **Tags:** implement `library.TagReader` and add it to the `Tagger` chain.
- **Image protocols:** implement `art.Renderer` and register it by name.
- **Keys:** every action can be rebound in the config.

## Building

```
go build -o bin/encomplayer ./cmd/encomplayer
go test ./...
```

Builds with `CGO_ENABLED=0`.
