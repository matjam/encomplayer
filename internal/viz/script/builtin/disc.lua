-- The ENCOM identity disc: rim segments lit by the spectrum, rings turning
-- with the music, and a core that swells on each beat.

local max, min, sin, cos, pi = math.max, math.min, math.sin, math.cos, math.pi
local SEGMENTS = 32

local levels = f64(SEGMENTS)
local spin, pulse = 0, 0

-- arc dots a circle of radius r around (cx, cy) from angle a to b.
local function arc(cx, cy, r, a, b, col)
  local steps = max(2, ((b - a) * r) // 1)
  for i = 0, steps do
    local t = a + (b - a) * i / steps
    dot(cx + cos(t) * r, cy + sin(t) * r, col)
  end
end

return {
  description = "ENCOM identity disc: its segments light with the spectrum",

  render = function(f, c)
    local w, h = c.dw, c.dh
    if w == 0 or h == 0 then return end
    local dt = step(f)
    local b = bands(SEGMENTS)
    local keep = fade(dt, 0.2)
    for i = 0, SEGMENTS - 1 do levels[i] = max(b[i], levels[i] * keep) end
    spin = spin + dt * (0.3 + 2 * f.level)
    pulse = max(f.beat_strength, pulse * fade(dt, 0.12))

    local cx, cy = w / 2, h / 2
    local outer = min(cx, cy) * 0.95

    -- Outer rim: one segment per band, lit by its level.
    local seg = 2 * pi / SEGMENTS
    for i = 0, SEGMENTS - 1 do
      local v = levels[i]
      local from = i * seg + seg * 0.12
      local col = v > 0.25 and ramp(0.4 + 0.6 * v) or palette.grid
      local r = outer * 0.84
      while r <= outer do
        arc(cx, cy, r, from, from + seg * 0.76, col)
        r = r + 1
      end
    end

    -- Middle ring: dashes turning with the music.
    for i = 0, 11 do
      local from = spin + i * pi / 6
      arc(cx, cy, outer * 0.68, from, from + pi / 12, palette.text)
    end
    -- Inner ring turns the other way.
    for i = 0, 5 do
      local from = -spin * 1.6 + i * pi / 3
      arc(cx, cy, outer * 0.5, from, from + pi / 5, palette.bright)
    end

    -- The core glows and swells on each beat.
    local core = outer * (0.22 + 0.12 * pulse)
    local r = 1
    while r <= core do
      arc(cx, cy, r, 0, 2 * pi, lerp(palette.accent, palette.bright, pulse * (1 - r / core)))
      r = r + 1
    end

    local label = f.title:upper()
    if label ~= "" and c.h >= 6 then
      local n = utf8.len(label) or #label
      if n > c.w then
        label = label:sub(1, (utf8.offset(label, c.w + 1) or #label + 1) - 1)
        n = c.w
      end
      text((c.w - n) // 2, c.h - 1, label, palette.dim)
    end
  end,
}
