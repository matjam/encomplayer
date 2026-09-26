-- Demoscene plasma: four sine fields summed. Three depend only on the
-- column, the row, or their sum, so they are computed once per line rather
-- than once per pixel.

local max, min, floor, sqrt, sin = math.max, math.min, math.floor, math.sqrt, math.sin
local SHADES = 256

local width, height = 0, 0
local dist, cols, rows, diag = f64(0), f64(0), f64(0), f64(0)
local shades = i32(SHADES)
local t, glow = 0, 0

-- spread is how fast the fields change across the canvas.
local function spread() return 12 / max(1, min(width, height)) end

local function resize(w, h)
  width, height = w, h
  cols, rows, diag, dist = f64(w), f64(h), f64(w + h), f64(w * h)
  local s = spread()
  for y = 0, h - 1 do
    for x = 0, w - 1 do
      local dx, dy = x - w / 2, y - h / 2
      dist[y * w + x] = sqrt(dx * dx + dy * dy) * s
    end
  end
end

return {
  description = "Demoscene plasma that churns faster as the music gets louder",

  render = function(f, c)
    local w, h = c.pw, c.ph
    if w == 0 or h == 0 then return end
    if w ~= width or h ~= height then resize(w, h) end
    local dt = step(f)
    t = t + dt * (0.4 + 2.5 * f.level)
    glow = max(f.bass, glow * fade(dt, 0.3))
    if f.beat then glow = 1 end

    local s = spread()
    for x = 0, w - 1 do cols[x] = sin((x - w / 2) * s + t) end
    for y = 0, h - 1 do rows[y] = sin((y - h / 2) * s * 0.8 + t * 1.3) end
    for d = 0, w + h - 1 do diag[d] = sin((d - (w + h) / 2) * s * 0.6 + t * 0.7) end
    local brightness = 0.35 + 0.65 * glow
    for i = 0, SHADES - 1 do
      local v = i / (SHADES - 1) * 8 - 4
      shades[i] = scale(cycle(v / 8 + t * 0.03), brightness)
    end

    local pixels = c.pixels
    for y = 0, h - 1 do
      local row, base = rows[y], y * w
      for x = 0, w - 1 do
        local v = cols[x] + row + diag[x + y] + sin(dist[base + x] * 1.4 - t * 2)
        -- An if clamp: min and max of floor's result run slowly in apogee
        -- 1.1 (matjam/apogee#140).
        local k = floor((v + 4) / 8 * (SHADES - 1))
        if k < 0 then k = 0 elseif k > SHADES - 1 then k = SHADES - 1 end
        pixels[base + x] = shades[k]
      end
    end
  end,
}
