-- The spectrum as rays around a sun that swells with the bass. Each band
-- appears twice, mirrored, so the rays are symmetric.

local max, min, sin, cos, pi = math.max, math.min, math.sin, math.cos, math.pi
local BANDS = 48

local levels = f64(BANDS)
local spin = 0

return {
  description = "Spectrum radiating from a sun that swells with the bass",

  render = function(f, c)
    local w, h = c.dw, c.dh
    if w == 0 or h == 0 then return end
    local dt = step(f)
    local b = bands(BANDS)
    local keep = fade(dt, 0.15)
    for i = 0, BANDS - 1 do levels[i] = max(b[i], levels[i] * keep) end
    spin = spin + dt * 0.15

    local cx, cy = w / 2, h / 2
    local limit = min(cx, cy)
    local core = limit * (0.18 + 0.12 * f.bass)

    local rays = 2 * BANDS
    for i = 0, rays - 1 do
      local band = i < BANDS and i or rays - 1 - i
      local v = levels[band]
      local a = spin + 2 * pi * i / rays
      local s, co = sin(a), cos(a)
      local outer = core + (limit - core) * v
      line(cx + co * core, cy + s * core, cx + co * outer, cy + s * outer, ramp(0.3 + 0.7 * v))
    end
    for i = 0, 119 do
      local a = 2 * pi * i / 120
      dot(cx + cos(a) * core, cy + sin(a) * core, palette.accent)
    end
  end,
}
