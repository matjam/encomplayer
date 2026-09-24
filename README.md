# EncomPlayer

A terminal music player styled after ENCOM OS-12 from *TRON: Legacy*, with
rmpc's keybindings. No MPD required.

<img width="1274" height="892" alt="EncomPlayer playing TRON: Legacy with album art, the queue and the spectrum analyser" src="https://github.com/user-attachments/assets/c0e1891d-1e01-411b-b4e7-c0aa993a7fbf" />

## Features

- Plays MP3, FLAC, Ogg Vorbis and WAV in pure Go. With ffmpeg installed it also
  plays AAC/M4A, ALAC, Opus, WavPack, AIFF, WMA and anything else ffmpeg reads.
- Browses by folder, artist, album artist, album and genre, with search.
- Queue with repeat, random, single and consume modes; M3U8 playlists.
- Album art in kitty and Ghostty (Unicode placeholders), iTerm2 and WezTerm
  (inline images), and half-block rendering in any truecolor terminal.
- Instant startup from a cache, background sync, and periodic rescans that
  pick up files added while it runs.
- Control a running player from the shell, e.g. from a window manager
  keybinding or a status bar.
- 34 built-in themes, custom themes, and a config screen that applies changes
  live.

## Install

<details>
<summary><b>macOS and Linux: install script</b></summary>

```sh
curl -fsSL https://raw.githubusercontent.com/matjam/encomplayer/main/install.sh | sh
```

Installs the latest release to `~/.local/bin` after checking it against the
release's `SHA256SUMS`. Set `BINDIR` to install elsewhere or `VERSION` to pick
a release:

```sh
curl -fsSL https://raw.githubusercontent.com/matjam/encomplayer/main/install.sh | BINDIR=/usr/local/bin VERSION=v1.2.0 sh
```

</details>

<details>
<summary><b>macOS: Homebrew</b></summary>

```sh
brew install --cask matjam/tap/encomplayer
```

</details>

<details>
<summary><b>Debian and Ubuntu</b></summary>

```sh
v=1.2.0   # the release you want, without the v
curl -fLO https://github.com/matjam/encomplayer/releases/download/v$v/encomplayer_${v}_amd64.deb
sudo apt install ./encomplayer_${v}_amd64.deb
```

On ARM, replace `amd64` with `arm64`.

</details>

<details>
<summary><b>Fedora, RHEL and openSUSE</b></summary>

```sh
v=1.2.0
sudo dnf install https://github.com/matjam/encomplayer/releases/download/v$v/encomplayer-$v-1.x86_64.rpm
```

On ARM, replace `x86_64` with `aarch64`.

</details>

<details>
<summary><b>Arch Linux</b></summary>

```sh
v=1.2.0
curl -fLO https://github.com/matjam/encomplayer/releases/download/v$v/encomplayer-$v-1-x86_64.pkg.tar.zst
sudo pacman -U ./encomplayer-$v-1-x86_64.pkg.tar.zst
```

Download the file first, because `pacman -U <url>` requires a detached
signature for remote packages and these are not signed. On ARM, replace
`x86_64` with `aarch64`.

</details>

<details>
<summary><b>Windows</b></summary>

Download the `.zip` from the
[releases page](https://github.com/matjam/encomplayer/releases) and put
`encomplayer.exe` on your `PATH`. Remote control needs Windows 10 1803 or
later.

</details>

<details>
<summary><b>From source</b></summary>

```sh
go install github.com/matjam/encomplayer/cmd/encomplayer@latest
```

Builds with `CGO_ENABLED=0`, so no C toolchain is needed.

</details>

MP3, FLAC, Ogg Vorbis and WAV need nothing else. For other formats, also
install ffmpeg (`brew install ffmpeg`, `sudo apt install ffmpeg`,
`sudo pacman -S ffmpeg`). The Debian and Fedora packages suggest it, but the
Arch package cannot declare optional dependencies.

## Usage

```
encomplayer [options] [music-folder]    start the player
encomplayer <command> [argument]        control the running player
```

The music folder comes from the argument, then `music_dir` in the config, then
`~/Music`.

| Option | Effect |
|---|---|
| `-a, --art PROTOCOL` | Album art for this run: `auto`, `kitty`, `iterm`, `blocks` or `off` |
| `-t, --theme NAME` | Theme for this run |
| `-s, --shuffle` | Start playing the whole library shuffled |
| `-r, --rescan` | Reread every file's tags instead of trusting the cache |
| `--no-mouse` | Leave the mouse to the terminal, e.g. to select text |
| `-c, --config FILE` | Config file (default `~/.config/encomplayer/config.json`) |
| `--list-themes` | List built-in and custom themes |
| `--paths` | Print where config, themes, playlists, cache, state and the socket live |
| `-v, --version` | Print the version and a diagnostics banner |
| `-h, --help` | Show help |

`--art`, `--theme`, `--no-mouse`, `--shuffle` and `--rescan` apply to one run
and are never written to the config file. On a terminal, `--version` and
`--list-themes` print in colour; piped, they print plain text for scripts.

## Remote control

With a player running, these commands control it from any shell:

| Command | Effect |
|---|---|
| `status` | Print the current track, position, queue position, volume and modes |
| `status --json` | The same as JSON |
| `play`, `pause`, `toggle`, `stop` | Transport; `play` resumes a paused or stopped track |
| `next`, `prev` | Skip forward or back in the queue |
| `seek +N`, `seek -N` | Seek relative to the current position, in seconds |
| `seek SECONDS`, `seek M:SS`, `seek H:MM:SS` | Seek to an absolute position |
| `volume N` | Set the volume, 0–100 |
| `volume +N`, `volume -N` | Step the volume |
| `repeat`, `random`, `single`, `consume` | Toggle a mode, or pass `on` or `off` |
| `shuffle` | Shuffle the queue |
| `shuffle-all` | Replace the queue with the whole library, shuffled, and play it |
| `add PATH` | Append a file or folder to the queue |
| `reload` | Reread `config.json` and the theme |

```sh
encomplayer toggle
encomplayer seek +30
encomplayer volume -5
encomplayer random on
encomplayer add ~/Music/Daft\ Punk/TRON\ Legacy
encomplayer status
# ▶ Derezzed — Daft Punk  1:23 / 1:44  3/22  vol 70%  [random]
encomplayer status --json | jq -r .title
```

Commands print nothing on success. They exit 1 with a message when the
player rejects the command or no player is running, and 2 for a bad command
line. A folder named like a command opens with `./name` or `-- name`.

The player listens on a Unix socket in its state directory
(`encomplayer --paths` shows where), readable only by your user. If that path
is too long for a socket, it moves to `$XDG_RUNTIME_DIR` or the temp
directory. Only one
player can listen at a time. A second instance still plays but shows a notice
that remote control is unavailable.

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
| `oc` | Config screen |
| `:` | Command mode (`:help` lists commands) |

The Playlists tab starts with **ALL MUSIC**, so `6` `X` also shuffles
everything, and `C-s s` on it saves the library as a playlist.

The queue, current track and play position survive a restart. After
relaunching, `p` resumes the track where it stopped. Choosing a track with
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

Drag a divider to resize panes. The dividers are the SIGNAL strip's top edge
(the spectrum grows with it), the border between album art and the queue, and
the borders between browser columns. Sizes are remembered between sessions.

## Configuration

Press `oc` (or `:config`) for the config screen. Its tabs cover General,
Appearance, Mouse, Keys and Paths. `h`/`l` switch tabs, `j`/`k` pick a
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

After editing the file, apply it without restarting by running
`encomplayer reload`, `:reload` inside the player, or
`pkill -USR1 encomplayer`.

## Themes

Built in: `encom` (default), `catppuccin-latte`, `-frappe`, `-macchiato`,
`-mocha`, `gruvbox-dark`, `-light`, `tokyonight-night`, `-storm`, `-day`,
`rose-pine`, `-moon`, `-dawn`, `solarized-dark`, `-light`, `kanagawa-wave`,
`-dragon`, `everforest-dark`, `-light`, `ayu-dark`, `-mirage`,
`github-dark`, `-light`, `dracula`, `nord`, `one-dark`, `monokai`,
`nightfox`, `material`, `palenight`, `synthwave-84`, `night-owl`,
`oxocarbon` and `cyberpunk`.

Pick one in the config screen, where moving through the list previews each
theme, or with `:theme <name>`. `encomplayer --list-themes` shows a swatch of
each.

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

The roles are `background` (empty keeps the terminal's), `text`, `bright`,
`dim`, `grid`, `border`, `accent`, `error`, `selection` and `selection_text`.
Save the file and run `encomplayer reload` to see the change live. A custom
file with a built-in's name replaces that built-in.

## Files

| What | Where |
|---|---|
| Config | `~/.config/encomplayer/config.json` |
| Custom themes | `~/.config/encomplayer/themes/` |
| Playlists | `~/.config/encomplayer/playlists/` |
| Library cache | `~/.cache/encomplayer/` |
| Queue, position and layout | `~/.local/state/encomplayer/` |
| Remote control socket | `~/.local/state/encomplayer/encomplayer.sock` |

These are the defaults on Linux and macOS, and they follow the `XDG_*`
variables. Run
`encomplayer --paths` for the locations on your system.

## Library cache

The cache records each track's tags, size and modification time, plus each
folder's listing and modification time. At startup the cached library shows
immediately and a sync runs in the background. The sync relists only folders
whose modification time changed and rereads only files whose size or time
changed, so a network library of 10,000 tracks syncs in a few seconds.

A full rescan (`C-U`, `:rescan` or `--rescan`) is the only way to notice a
file retagged in place when its folder did not change.

## Extending

- **Formats:** implement `audio.Decoder` and register it for its extensions.
  Decoders registered earlier take priority.
- **Tags:** implement `library.TagReader` and add it to the `Tagger` chain.
- **Image protocols:** implement `art.Renderer` and register it by name.
- **Keys:** every action can be rebound in the config.

## Building

```sh
go build -o bin/encomplayer ./cmd/encomplayer
go test ./...
```
