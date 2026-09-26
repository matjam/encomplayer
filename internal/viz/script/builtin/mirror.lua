-- The spectrum mirrored about the centre line, bass in the middle.

local levels, count = f64(0), 0

return {
  description = "Spectrum mirrored about the centre line, bass in the middle",

  render = function(f, c)
    local w, h = c.w, c.ph
    local half = (w + 1) // 2
    if half < 1 then return end
    local b = bands(half)
    if count ~= half then levels, count = f64(half), half end
    local keep = fade(step(f), 0.15)
    for i = 0, half - 1 do
      local v = levels[i] * keep
      if b[i] > v then v = b[i] end
      levels[i] = v
    end

    local mid = h / 2
    local span = mid > 1 and mid or 1
    local pixels = c.pixels
    for x = 0, w - 1 do
      -- The bass sits in the centre and treble spreads to both edges.
      local d = x - w // 2
      if d < 0 then d = -d - 1 end
      if d > half - 1 then d = half - 1 end
      local reach = levels[d] * mid
      for py = 0, h - 1 do
        local dist = py + 0.5 - mid
        if dist < 0 then dist = -dist end
        if dist <= reach then
          pixels[py * w + x] = ramp(0.3 + 0.7 * dist / span)
        end
      end
    end
  end,
}
