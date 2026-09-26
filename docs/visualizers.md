# Writing a visualizer

Every EncomPlayer visualizer, built-in or yours, is a Lua script. The player
runs it with [apogee](https://github.com/matjam/apogee), a Lua 5.5 VM with a
JIT compiler, and calls it once a frame with the music it should draw.

- [Quick start](#quick-start)
- [How a script runs](#how-a-script-runs)
- [The frame](#the-frame)
- [The canvas](#the-canvas)
- [Colours](#colours)
- [Buffers](#buffers)
- [Helpers](#helpers)
- [Examples](#examples)
- [Performance](#performance)
- [Adding a built-in](#adding-a-built-in)
- [Troubleshooting](#troubleshooting)

## Quick start

Save this as `~/.config/encomplayer/visualizers/pulse.lua`
(`encomplayer --paths` shows the folder on your system):

```lua
-- A ring that swells with the bass and flashes on the beat.
local sin, cos, pi = math.sin, math.cos, math.pi
local flash = 0

return {
  description = "A ring that swells with the bass",

  render = function(f, c)
    flash = math.max(f.beat_strength, flash * fade(step(f), 0.2))
    local r = math.min(c.dw, c.dh) / 2 * (0.3 + 0.6 * f.bass)
    local col = lerp(palette.text, palette.accent, flash)
    for i = 0, 179 do
      local a = 2 * pi * i / 180
      dot(c.dw / 2 + cos(a) * r, c.dh / 2 + sin(a) * r, col)
    end
  end,
}
```

Then pick it: `encomplayer viz pulse` from a shell, `:viz pulse` inside the
player, or `[` and `]` to step to it. It is listed as `pulse` because the
file is `pulse.lua`. A file named after a built-in, such as `spectrum.lua`,
replaces that built-in.

Edit the file and run `encomplayer reload` (or `:reload`, or
`pkill -USR1 encomplayer`) to see the change without restarting.

The built-ins in
[`internal/viz/script/builtin`](../internal/viz/script/builtin) are working
examples of every technique below; copying one is often the fastest start.

## How a script runs

A script is a file that returns a table:

| Field | |
|---|---|
| `description` | One line for `--list-visualizers` and the status line. Optional. |
| `render` | `function(f, c)`, called every frame with the [frame](#the-frame) `f` and the [canvas](#the-canvas) `c`. Required. |

The player runs the whole file each time the visualizer is selected, in a
Lua state of its own, and closes that state when you switch away. So:

- Code at the top of the file runs once per selection. Build tables there,
  and keep state between frames in local variables, as `flash` does above.
- `render` runs every frame, 30 times a second by default
  (`visualizer_fps` in the config sets it, from 5 to 60).
- The canvas is blank when `render` starts, and may change size between
  frames. Check its size each frame, and rebuild anything sized to it when
  the size changes.
- Selecting the visualizer again starts it fresh.

**Errors.** A script that fails to load is left out of the list, and its
error appears in the status line, or on stderr for
`encomplayer --list-visualizers`. A script that fails while drawing shows
its error, with file and line, in the status line, and the player switches
to `spectrum`.

**Limits.** Loading may take one second and 256 MB; each frame may take
250 ms and 64 MB. Past a limit, the script stops with an error as above.

**Sandbox.** Scripts get Lua's basic functions and the `string`, `math`,
`table` and `utf8` libraries. There is no `io`, `os`, `require`, `load`,
`dofile` or `loadfile`, so a script cannot read or write files, run
programs, or load other code. `print` does nothing, since the terminal
belongs to the player; draw with `text` to see a value.

## The frame

`f` describes the audio just before this frame, taken before the volume
control, so visualizers stay lively at low volume. When nothing plays, the
samples and levels are zero but every field is present.

| Field | |
|---|---|
| `time` | Seconds since the visualizer started |
| `dt` | Seconds since the previous frame. `step(f)` is the same, capped at 0.25 s, so a pause does not make an animation jump. |
| `playing` | `true` while music plays, `false` when stopped or paused |
| `rate` | The sample rate, such as 44100 |
| `left`, `right` | [Buffers](#buffers) of the latest 2,048 samples of each channel, oldest first, from -1 to 1 |
| `spectrum` | A buffer of 1,024 FFT bins of the mono mix, each from 0 to 1 across 60 dB |
| `bin_hz` | The width of a bin: bin `i` is centred on `i × bin_hz` Hz |
| `level`, `level_left`, `level_right` | Loudness, from 0 for silence to 1 near full scale, across 48 dB |
| `peak` | The largest absolute sample in the window |
| `bass`, `mid`, `treble` | The mean spectrum over 40–250 Hz, 250 Hz–4 kHz and 4–16 kHz |
| `beat` | `true` on the frame a kick or other bass onset is detected |
| `beat_strength` | How far the beat stood out, from 0 to 1 |
| `title`, `artist`, `album` | The playing track, or empty strings |
| `position`, `duration` | Seconds into the track, and its length |

`bands(n)` groups the spectrum into `n` bands spaced as the ear hears pitch,
logarithmically from 40 Hz to 16 kHz. Each band is its loudest bin, from 0
to 1. It returns a buffer, and the same `n` returns the same buffer every
frame, refilled, so keeping it is cheap. Music carries much less energy in
the treble than the bass; `tilt` (see [Helpers](#helpers)) evens that out
for effects spread across the screen.

## The canvas

`c` is the grid of terminal cells the visualizer fills, in three
resolutions:

| | Size | Draw with |
|---|---|---|
| Cells | `c.w` × `c.h` | `set`, `cell`, `text` |
| Half-block pixels, two per cell, stacked | `c.pw` × `c.ph` | `pixel`, `c.pixels` |
| Braille dots, 2 × 4 per cell | `c.dw` × `c.dh` | `dot`, `line` |

| Function | Draws |
|---|---|
| `set(x, y, codepoint, fg)` | A glyph, by code point, in cell (x, y) |
| `cell(x, y, glyph, fg[, bg])` | A glyph, as a string or code point, with an optional background colour |
| `text(x, y, s, fg)` | The string `s` from cell (x, y) rightwards; every character must be single-width |
| `pixel(x, y, col)` | Half-block pixel (x, y) |
| `dot(x, y, col)` | Braille dot (x, y) |
| `line(x0, y0, x1, y1, col)` | A braille line between two points in dot coordinates |

Coordinates count from the top left, starting at 0, and round down, so
fractional positions are fine. Anything off the canvas is dropped.

A cell shows one resolution at a time: whatever drew into it last. Drawing
a dot into a cell that held a glyph replaces the glyph. Several dots, or the
two pixels of a cell, share it; a braille cell takes the colour of the last
dot drawn into it.

**The pixel buffer.** `c.pixels` is a [buffer](#buffers) of `c.pw × c.ph`
colours, one per half-block pixel, row by row: pixel (x, y) is
`c.pixels[y * c.pw + x]`. It starts every frame at -1, which draws
nothing. It lies beneath everything else drawn this frame, so a cell that
`set`, `dot` or `pixel` also drew into shows that instead. For effects that
colour most of the screen, filling the buffer is much faster than calling
`pixel` for each pixel.

## Colours

A colour is a number, `0xRRGGBB`: `0xff8000` is orange. A negative colour
draws nothing.

`palette` holds the current theme's colours, so a visualizer follows
whatever theme is chosen: `palette.background`, `text`, `bright`, `dim`,
`grid`, `accent` and `error`. It is filled before the first `render`, so
read it inside `render`, not at the top of the file.

| Function | Returns |
|---|---|
| `rgb(r, g, b)` | A colour from channels of 0 to 255 |
| `hex("#rrggbb")` | A colour from a hex string; `"#rgb"` works too |
| `lerp(a, b, t)` | The blend from `a` towards `b`, with `t` from 0 to 1 |
| `scale(col, f)` | Each channel multiplied by `f`, capped at white |
| `ramp(t)` | The theme from faint to hot: grid, dim, text, bright, accent, for `t` from 0 to 1 |
| `heat(t)` | Background through error and accent to bright, for fire and heat maps |
| `cycle(t)` | Around a loop of the theme's vivid colours, repeating every 1 |

`background` is black when the theme keeps the terminal's own background;
fading towards it makes trails disappear.

## Buffers

A buffer is an array of numbers that the player and the script share
without copying: `f.left`, `f.right`, `f.spectrum`, `bands(n)` and
`c.pixels`. Make your own with `f64(n)`, which holds floats, or `i32(n)`,
which holds 32-bit integers; both start at zero and hold up to 16,777,216
elements.

- Buffers count from 0, not 1: `buf[0]` to `buf[#buf - 1]`.
- `#buf` is the length, which never changes.
- Reading outside the buffer gives `nil`; writing there is an error.
- An `i32` buffer stores integers, and floats with a whole value such as
  `3.0`; storing `3.5` is an error. Use `math.floor` first.

The JIT reads and writes buffers without calling out of the compiled loop,
so per-pixel state (a heat map, the previous frame, precomputed angles)
belongs in buffers rather than tables.

## Helpers

The player defines these globals in every script:

| Helper | |
|---|---|
| `step(f)` | `f.dt`, capped at 0.25 s |
| `fade(dt, half_life)` | The factor that halves a value every `half_life` seconds: `v = v * fade(dt, 0.3)` |
| `clamp(v, lo, hi)` | `v` limited to [lo, hi] |
| `tilt(v, t, boost)` | Band value `v` at position `t` (0 to 1) across the spectrum, lifted by up to `boost` towards the treble |
| `gain()` | Returns a function `(peak, dt) -> gain` that follows the signal's peak, to scale a trace so quiet passages still fill the screen |
| `EIGHTHS[0]` … `EIGHTHS[8]` | The code points of the block glyphs that fill a cell from the bottom, from empty to full |

## Examples

**Bars from glyphs**, in cells. Bars jump up at once and fall back slowly,
so they do not flicker:

```lua
local N = 32
local levels = f64(N)

return {
  description = "Chunky bars",
  render = function(f, c)
    local b = bands(N)
    local keep = fade(step(f), 0.2)
    for i = 0, N - 1 do levels[i] = math.max(b[i], levels[i] * keep) end

    for x = 0, c.w - 1 do
      local v = levels[x * N // c.w]
      local lit = math.floor(v * c.h * 8) -- eighths of a cell
      for y = c.h - 1, 0, -1 do
        local here = math.min(lit, 8)
        if here <= 0 then break end
        set(x, y, EIGHTHS[here], ramp(1 - y / c.h))
        lit = lit - 8
      end
    end
  end,
}
```

**A trace in braille dots.** `line` joins each sample to the next:

```lua
local track = gain()

return {
  description = "A plain scope",
  render = function(f, c)
    local g = track(f.peak, step(f))
    local mid, n = c.dh / 2, #f.left
    local px, py
    for x = 0, c.dw - 1 do
      local k = x * n // c.dw
      local y = mid - (f.left[k] + f.right[k]) / 2 * g * mid
      if px then line(px, py, x, y, palette.bright) end
      px, py = x, y
    end
  end,
}
```

**A full-screen effect in the pixel buffer.** Colours are worked out once
per frame into a table of shades, so the per-pixel loop is only arithmetic
and buffer reads and writes:

```lua
local sin, floor = math.sin, math.floor
local shades = i32(256)
local t = 0

return {
  description = "Rippling rings",
  render = function(f, c)
    t = t + step(f) * (0.5 + 2 * f.level)
    for i = 0, 255 do shades[i] = cycle(i / 256 + t * 0.05) end

    local w, h, pixels = c.pw, c.ph, c.pixels
    for y = 0, h - 1 do
      for x = 0, w - 1 do
        local dx, dy = x - w / 2, (y - h / 2) * 2
        local v = sin(math.sqrt(dx * dx + dy * dy) * 0.3 - t * 3)
        pixels[y * w + x] = shades[floor((v + 1) * 127.5)]
      end
    end
  end,
}
```

The built-ins show more: feedback trails that warp the previous frame
(`milkdrop`), a cellular automaton (`life`), 3D projection (`wireframe`,
`starfield`, `lightcycles`, `outrun`), particles (`particles`) and
analogue meters (`vu`).

## Performance

A full-screen 200 × 50 canvas has 20,000 pixels, and a frame should stay
well under 2 ms. apogee compiles an inner loop of arithmetic, comparisons,
buffer reads and writes, and calls to `math.floor`, `ceil`, `abs`, `min`,
`max`, `sqrt`, `sin` and `cos` to machine code with its variables in
registers, at about a nanosecond an operation. For the per-pixel loop:

- Keep per-pixel state in buffers, not tables.
- Work out anything that does not change per pixel before the loop: a
  table of shades, per-row or per-column values, geometry that only
  depends on the canvas size.
- Calls to the drawing and colour functions cost more than arithmetic; in
  a loop over every pixel, write colours into `c.pixels` from a table made
  once per frame.
- Write `min(x, 1.0)`, not `min(x, 1)`, for a float `x`. When it clamps,
  `min(x, 1)` returns the integer `1`, and a value whose type changes from
  one pixel to the next slows the loop.
- Clamp a `floor` result with `if` rather than `min` or `max`; the two
  together run slowly in apogee 1.1
  ([matjam/apogee#140](https://github.com/matjam/apogee/issues/140)).
- `math.random`, and `%` of floats by anything but a power of two, call
  into Go on every use. The built-in `fire` uses an inline random number
  generator instead.

On Windows, and other platforms apogee does not compile for, scripts are
interpreted. Per-pixel effects then run two to four times slower.

## Adding a built-in

A built-in visualizer is a script in
[`internal/viz/script/builtin`](../internal/viz/script/builtin), embedded in
the binary; no Go changes are needed. The tests pick it up automatically:

```sh
go test ./internal/viz/script
go test -run '^$' -bench . ./internal/viz/script
```

The tests run every built-in at sizes from 0 × 0 to 120 × 40, in music and
in silence, and fail if it raises an error, leaves a row wider or narrower
than the canvas, or draws nothing while music plays. The benchmark reports each one's cost for a
200 × 50 frame. Add the new name to the table in the README's Visualizers
section.

If you change what scripts can use, keep this guide in step with
[`api.go`](../internal/viz/script/api.go),
[`frame.go`](../internal/viz/script/frame.go) and
[`prelude.lua`](../internal/viz/script/prelude.lua).

## Troubleshooting

| Symptom | Cause |
|---|---|
| The script is missing from the list | It failed to load. `encomplayer --list-visualizers` prints the error on stderr. |
| `must return a table` | The file does not end with `return { ... }`. |
| `render must be a function` | The returned table has no `render`, or it is misspelt. |
| The player switches to `spectrum` | `render` raised an error; the status line shows it, with the line number. |
| `interrupted!` | A frame took longer than 250 ms, often an endless loop. |
| `not enough memory` | A frame allocated more than 64 MB; build tables once, not every frame. |
| Colours are all nil at startup | `palette` is read at the top of the file; read it inside `render`. |
| An edit does not show | Run `encomplayer reload`; the player reads the file when the visualizer starts. |
