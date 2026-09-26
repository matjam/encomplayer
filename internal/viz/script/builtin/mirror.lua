-- The spectrum mirrored about the centre line, bass in the middle.

local max, min, abs = math.max, math.min, math.abs

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
    for i = 0, half - 1 do levels[i] = max(b[i], levels[i] * keep) end

    local mid = h / 2
    local span = max(1, mid)
    local pixels = c.pixels
    for x = 0, w - 1 do
      -- The bass sits in the centre and treble spreads to both edges.
      local d = x - w // 2
      if d < 0 then d = -d - 1 end
      local reach = levels[min(d, half - 1)] * mid
      for py = 0, h - 1 do
        local dist = abs(py + 0.5 - mid)
        if dist <= reach then
          pixels[py * w + x] = ramp(0.3 + 0.7 * dist / span)
        end
      end
    end
  end,
}
