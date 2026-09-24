# AGENTS.md

Working agreement for agents and contributors in this repository.

## Project

EncomPlayer is a terminal music player styled after ENCOM OS-12 (*TRON:
Legacy*), with rmpc's keybindings and no MPD dependency. It is an application,
not a library: nothing imports its packages.

## Verification contract

Run all of these before pushing. CI runs the same commands.

```sh
test -z "$(gofmt -l .)"
go vet ./...
go test -race ./...
go run golang.org/x/vuln/cmd/govulncheck@v1.8.0 ./...
go run github.com/goreleaser/goreleaser/v2@v2.18.2 release --snapshot --clean
```

The snapshot builds every archive, `.deb`, `.rpm` and Arch package into
`dist/` without publishing. Check `sh install.sh` after touching the release
layout; it must handle both the current archives and older ones.

```sh
# optional: install and run a package in a clean container
docker run --rm -v "$PWD/dist":/pkg debian:stable-slim sh -c 'apt-get -qq update && apt-get install -y /pkg/*_arm64.deb && encomplayer --version'
```

The decoder integration tests in `internal/audio` generate audio with ffmpeg
and skip when it is missing. Install ffmpeg to run them.

Compiling is not verifying. For UI or playback changes, run the binary in a
real terminal and exercise the change.

## Layout

| Path | Role |
|---|---|
| `cmd/encomplayer` | Flags, config and wiring. The only place that picks concrete adapters. |
| `internal/domain` | Track, playlist, generic `Queue[T]`, playback modes. No dependencies. |
| `internal/collection` | Generic `List[T]`, `Browser[T]`, `GroupBy`. |
| `internal/registry` | Generic `Registry[T]` for pluggable implementations. |
| `internal/keymap` | rmpc notation parser, action names, chord resolver. |
| `internal/audio` | Decoder port, player, spectrum analyser. |
| `internal/audio/beepdec`, `ffmpegdec` | Decoder adapters. |
| `internal/library` | Scanner, incremental cache, tag reader chain, search. |
| `internal/library/ffprobe` | Fallback tag reader. |
| `internal/art` | Album art loading and the kitty, iTerm2 and half-block renderers. |
| `internal/playlist` | M3U8 playlist store. |
| `internal/config` | Config file and saved session state. |
| `internal/theme` | Built-in palettes and custom JSON themes, mapped onto UI colour roles. |
| `internal/ui` | bubbletea model, tabs, modals, config screen, view. Colours come from the model's `styles`, never package globals. |

## Conventions

- Ports and adapters. Interfaces live in the consuming package; adapters
  depend on ports, never the reverse.
- Extend through the registries: `audio.Decoder` per extension,
  `library.TagReader` in the tagger chain, `art.Renderer` per protocol.
- Keep `CGO_ENABLED=0`. Every release binary is a static cross-compile from
  Linux; a cgo dependency breaks that.
- Keybinding defaults follow rmpc. EncomPlayer-only actions go on keys rmpc
  leaves free, and are marked as additions in `internal/keymap`.
- Files stay under about 300 lines. Tests are table-driven and co-located.
- Errors are wrapped with `%w` and surfaced once, in the status line.
- Workflows pin every action to a full commit SHA with the version in a
  trailing comment (`uses: actions/checkout@<sha> # v7.0.1`), and pin tools
  to exact versions. Tags can be moved; commits cannot.

## Commits and versioning

Commits and PR titles follow Conventional Commits. release-please reads them
to cut releases:

| Prefix | Effect |
|---|---|
| `fix:` | Patch release (1.2.3 → 1.2.4) |
| `feat:` | Minor release (1.2.3 → 1.3.0) |
| `chore:`, `docs:`, `ci:`, `refactor:`, `test:` | No release on their own |

**EncomPlayer is pinned to 1.x.** It is an application, so a "breaking
change" has no consumer to break. Never use `!` after the type or a
`BREAKING CHANGE:` footer. `.github/scripts/no-breaking.sh` rejects both in
PR checks and again before release-please runs on `main`, so a marker cannot
produce a 2.0.0. Moving to 2.x is a decision for the maintainer alone.

## Releases

1. Merging to `main` makes release-please open or update a release PR with
   the next version and changelog.
2. Merging that PR tags `vX.Y.Z` and publishes a GitHub release.
3. The release workflow then runs GoReleaser (`.goreleaser.yaml`) on that
   tag. It attaches archives for linux, darwin and windows on amd64 and
   arm64, `.deb`, `.rpm` and Arch packages, and `SHA256SUMS`, and pushes the
   cask to `matjam/homebrew-tap` using the `HOMEBREW_TAP_TOKEN` secret
   (fine-grained, Contents read/write on the tap only).

Never tag or publish releases by hand.
