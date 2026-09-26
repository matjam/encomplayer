-- Spectrum analyser: 64 bands spread across the strip, whatever its width.

local max = math.max

local BANDS = 64
local levels = f64(BANDS)

return {
  description = "Spectrum analyser bars that glow hotter towards the top",

  render = function(f, c)
    local bands = bands(BANDS)
    -- Bars jump up at once and fall back gradually, so they do not flicker.
    local keep = fade(step(f), 0.25)
    for i = 0, BANDS - 1 do
      levels[i] = max(bands[i], levels[i] * keep)
    end

    local w, h = c.w, c.h
    local steps = h * 8
    -- Hot loops avoid math.floor, min and max, which apogee 1.0 calls in Go
    -- (matjam/apogee#120, #122).
    for y = 0, h - 1 do
      local from_bottom = h - 1 - y
      local height = (from_bottom + 1) / h
      local col = palette.text
      if height > 0.85 and h > 1 then
        col = palette.accent
      elseif height > 0.5 then
        col = palette.bright
      end
      local base = from_bottom * 8
      for x = 0, w - 1 do
        local lit = (levels[x * BANDS // w] * steps + 0.5) // 1 - base
        if lit > 0 then
          if lit > 8 then lit = 8 end
          set(x, y, EIGHTHS[lit], col)
        end
      end
    end
  end,
}
