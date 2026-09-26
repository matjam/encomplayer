-- A pair of analogue VU meters. The scale sweeps from MIN to MAX radians,
-- anticlockwise from the right. As on a real meter, the needle moves with
-- voltage, 0 VU sits at -18 dBFS RMS, full scale is +3 VU, and the red zone
-- starts at 0 VU.

local min, sin, cos, exp, pi = math.min, math.sin, math.cos, math.exp, math.pi

local RESPONSE = 0.3 -- seconds for a needle to reach its reading
local MIN, MAX = pi * 0.83, pi * 0.17
local REFERENCE, FULL_SCALE = -18, 3
local RED = 10 ^ (-FULL_SCALE / 20) -- where 0 VU falls on the scale

-- needle converts a frame level, which spans 48 dB below full scale, to
-- the needle's place on the scale.
local function needle(level)
  if level <= 0 then return 0 end
  local vu = level * 48 - 48 - REFERENCE
  return clamp(10 ^ ((vu - FULL_SCALE) / 20), 0, 1)
end

-- meter draws one meter in the dot columns [x0, x0+w).
local function meter(c, x0, w, level, label)
  local h = c.dh
  if w < 8 or h < 8 then return end
  local cx, cy = x0 + w / 2, h - 2
  local radius = min(w / 2 - 2, h - 4)

  for i = 0, 60 do
    local t = i / 60
    local a = MIN + (MAX - MIN) * t
    local col = t > RED and palette.error or palette.dim
    local x, y = cx + cos(a) * radius, cy - sin(a) * radius
    dot(x, y, col)
    if i % 10 == 0 then
      line(x, y, cx + cos(a) * radius * 0.9, cy - sin(a) * radius * 0.9, col)
    end
  end

  local a = MIN + (MAX - MIN) * clamp(level, 0, 1)
  local col = level > RED and palette.accent or palette.bright
  line(cx, cy, cx + cos(a) * radius * 1.02, cy - sin(a) * radius * 1.02, col)

  text(cx // 1 // 2, (cy // 1 - (radius * 0.45) // 1) // 4, label, palette.text)
  if w // 2 >= 6 then text(cx // 1 // 2 - 1, c.h - 1, "VU", palette.dim) end
end

local left, right = 0, 0

return {
  description = "Pair of analogue VU meters with swinging needles",

  render = function(f, c)
    local follow = 1 - exp(-step(f) / RESPONSE * 3)
    left = left + (needle(f.level_left) - left) * follow
    right = right + (needle(f.level_right) - right) * follow

    local w = c.dw // 2
    meter(c, 0, w, left, "L")
    meter(c, w, c.dw - w, right, "R")
  end,
}
