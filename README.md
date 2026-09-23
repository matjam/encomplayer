# EncomPlayer

A terminal music player styled after ENCOM OS-12 from *TRON: Legacy*, with
rmpc's keybindings. No MPD required.

```
encomplayer [--art auto|kitty|iterm|blocks|off] [--config path] [music-dir]
```

The music folder comes from the argument, then `music_dir` in the config, then
`~/Music`.

## Screnshot

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

`~/.config/encomplayer/config.json` (or `$XDG_CONFIG_HOME`):

```json
{
  "music_dir": "~/Music",
  "album_art": "auto",
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
