-- Flying down a chequered tunnel. Each pixel's angle and depth are
-- computed once per canvas size.

local max, min, floor = math.max, math.min, math.floor
local sqrt, atan, pi = math.sqrt, math.atan, math.pi
local SHADES = 256

local width, height = 0, 0
local angle, depth = f64(0), f64(0)
local shades = i32(SHADES * 2) -- dark squares, then light ones
local travel, turn, flash = 0, 0, 0

local function resize(w, h)
  width, height = w, h
  angle, depth = f64(w * h), f64(w * h)
  local cx, cy = w / 2, h / 2
  local s = 2 / math.min(w, h)
  for py = 0, h - 1 do
    for px = 0, w - 1 do
      local x, y = (px - cx) * s, (py - cy) * s
      local i = py * w + px
      angle[i] = atan(y, x) / pi * 4
      depth[i] = 1 / max(0.02, sqrt(x * x + y * y))
    end
  end
end

return {
  description = "Flying down a chequered tunnel that twists with the music",

  render = function(f, c)
    local w, h = c.pw, c.ph
    if w == 0 or h == 0 then return end
    if w ~= width or h ~= height then resize(w, h) end
    local dt = step(f)
    travel = travel + dt * (0.3 + 1.8 * f.level)
    turn = turn + dt * (0.1 + 0.6 * f.mid)
    flash = max(f.beat_strength, flash * fade(dt, 0.15))

    -- Colour by shade (how near the pixel is) and square: a table per
    -- frame instead of blending colours per pixel.
    local bg = palette.background
    for check = 0, 1 do
      for i = 0, SHADES - 1 do
        local shade = i / (SHADES - 1)
        local col = ramp(shade * (0.35 + 0.5 * check) + flash * 0.3)
        shades[check * SHADES + i] = lerp(bg, col, shade + flash * 0.2)
      end
    end

    local pixels = c.pixels
    for i = 0, w * h - 1 do
      local d = depth[i]
      local check = (floor(angle[i] + turn) + floor((d + travel) * 4)) % 2
      -- Pixels near the centre are far away, so they fade into the dark.
      local shade = min(1.6 / d, 1.0)
      pixels[i] = shades[check * SHADES + floor(shade * 255)]
    end
  end,
}
