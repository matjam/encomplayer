-- Stereo goniometer: each sample pair is a point, with phosphor that glows
-- for a moment after the beam passes.

local min, floor = math.min, math.floor
local ROOT2 = math.sqrt(2)

local phosphor, width, height = f64(0), 0, 0
local track = gain()

return {
  description = "Stereo goniometer: mono is a vertical line, wide stereo a cloud",

  render = function(f, c)
    local w, h = c.dw, c.dh
    if w == 0 or h == 0 then return end
    if w ~= width or h ~= height then
      phosphor, width, height = f64(w * h), w, h
    end
    local ph = phosphor
    local keep = fade(step(f), 0.08)
    for i = 0, w * h - 1 do ph[i] = ph[i] * keep end

    local cx, cy = w / 2, h / 2
    local radius = min(cx, cy) * 0.95
    local g = track(f.peak, step(f))
    local left, right = f.left, f.right
    for i = 0, #left - 1 do
      local l, r = left[i] * g, right[i] * g
      -- Rotate 45° so the mid signal points up and the side signal
      -- spreads sideways.
      local px = floor(cx + (l - r) / ROOT2 * radius)
      local py = floor(cy - (l + r) / ROOT2 * radius)
      if px >= 0 and py >= 0 and px < w and py < h then
        local k = py * w + px
        ph[k] = min(ph[k] + 0.35, 1.0)
      end
    end

    -- The L and R axes are the diagonals.
    local d = radius * 0.7
    line(cx - d, cy - d, cx + d, cy + d, palette.grid)
    line(cx - d, cy + d, cx + d, cy - d, palette.grid)
    for py = 0, h - 1 do
      for px = 0, w - 1 do
        local b = ph[py * w + px]
        if b > 0.05 then dot(px, py, ramp(0.3 + 0.7 * b)) end
      end
    end
  end,
}
