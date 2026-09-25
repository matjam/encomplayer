# internal/tea

A copy of [bubbletea](https://github.com/charmbracelet/bubbletea) and the
parts of [bubbles](https://github.com/charmbracelet/bubbles) EncomPlayer
uses, kept in the repository so we can change them. Both are MIT licensed by
Charmbracelet, Inc.; see `LICENSE` and `LICENSE.bubbles`.

| Path | Upstream |
|---|---|
| `internal/tea` | `charm.land/bubbletea/v2` v2.0.10 |
| `internal/tea/textinput`, `cursor`, `key`, `internal/runeutil` | `charm.land/bubbles/v2` v2.2.1 |

## Why

bubbletea's renderer always uses hard scroll optimisation (except on
Windows): when lines shift, it scrolls a region of the screen instead of
redrawing them. foot, and possibly other terminals, move sixel images
whenever any region scrolls, even images outside it, so album art drifted
with the visualiser. bubbletea has no option to turn this off.

A `replace` directive pointing at a fork would have worked too, but
`go install github.com/matjam/encomplayer/cmd/encomplayer@latest` refuses
modules with replace directives.

`textinput` and its helpers come along because textinput type-switches on
bubbletea's own key messages, so it has to be built against this copy.

## Changes from upstream

Every change is marked `EncomPlayer addition`; grep for it.

- `SetScrollOptimization(on bool) Cmd` (`scroll.go`) turns the renderer's
  scroll optimisation on or off at run time. The setting survives renderer
  resets. Tests are in `scroll_test.go`.
- Import paths in the bubbles packages point at this copy.

## Updating

1. Copy the new upstream `.go` files, `testdata` and licences over these
   directories from the module cache (`go mod download
   charm.land/bubbletea/v2@vX.Y.Z`, then `go list -m -f '{{.Dir}}' ...`).
2. Reapply every `EncomPlayer addition` and the import paths.
3. Update the versions above and run the full verification contract from
   `AGENTS.md`. The upstream tests run with ours.

govulncheck cannot see copied code. Watch the upstream repositories'
security advisories when updating.
