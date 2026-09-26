-- MilkDrop-style feedback: keep the previous frame at pixel resolution,
-- warp it towards the centre, fade it, and draw the waveform on top.
--
-- The frame is a brightness per pixel, which warps smoothly, and a colour
-- per pixel, which moves with its nearest sample. Warping one channel
-- rather than three keeps the frame within budget in Lua; the waveform's
-- colour changes slowly, so neighbouring trail pixels share a colour
-- anyway.

local max, min, abs, floor = math.max, math.min, math.abs, math.floor
local sin, cos, pi = math.sin, math.cos, math.pi

local width, height = 0, 0
-- The frame, and the next one being built.
local lum0, hue0, lum1, hue1

-- plot sets a pixel to colour col at brightness bright.
local function plot(x, y, col, bright)
  if x < 0 or y < 0 or x >= width or y >= height then return end
  local i = y * width + x
  lum0[i], hue0[i] = bright, col
end

local function stroke(x0, y0, x1, y1, col, bright)
  local steps = max(abs(x1 - x0), abs(y1 - y0), 1)
  for i = 0, steps do
    plot(x0 + (x1 - x0) * i // steps, y0 + (y1 - y0) * i // steps, col, bright)
  end
end

local function ring(cx, cy, r, col)
  for i = 0, 179 do
    local a = 2 * pi * i / 180
    plot(floor(cx + cos(a) * r), floor(cy + sin(a) * r), col, 1)
  end
end

-- warp builds the next frame by sampling the current one slightly closer
-- to the centre, rotated by swirl, so the picture flows outwards.
-- Brightness blends the four nearest pixels, so repeated warping stays
-- smooth instead of blocky.
local function warp(w, h, zoom, keep, swirl)
  local sn, cs = sin(swirl) * zoom, cos(swirl) * zoom
  local cx, cy = w / 2, h / 2
  local wm, hm = w - 1, h - 1
  for py = 0, hm do
    local y = py - cy
    -- The sample point moves by (cs, sn) per pixel along the row.
    local sx = cx - cx * cs - y * sn
    local sy = cy - cx * sn + y * cs
    local i = py * w
    for _ = 0, wm do
      local x0, y0 = floor(sx), floor(sy)
      if x0 >= 0 and y0 >= 0 and x0 < wm and y0 < hm then
        local fx, fy = sx - x0, sy - y0
        local a = y0 * w + x0
        local c = a + w
        local top = lum0[a] + (lum0[a + 1] - lum0[a]) * fx
        local bottom = lum0[c] + (lum0[c + 1] - lum0[c]) * fx
        lum1[i] = (top + (bottom - top) * fy) * keep
        if fx >= 0.5 then a = a + 1 end
        if fy >= 0.5 then a = a + w end
        hue1[i] = hue0[a]
      else
        lum1[i] = 0.0
      end
      sx, sy, i = sx + cs, sy + sn, i + 1
    end
  end
end

return {
  description = "Feedback trails: each frame zooms and swirls the last, MilkDrop-style",

  render = function(f, c)
    local w, h = c.pw, c.ph
    if w == 0 or h == 0 then return end
    if w ~= width or h ~= height then
      width, height = w, h
      lum0, hue0, lum1, hue1 = f64(w * h), i32(w * h), f64(w * h), i32(w * h)
    end
    local dt = step(f)
    local swirl = sin(f.time * 0.3) * (0.02 + 0.06 * f.mid)
    warp(w, h, 1 - dt * (0.6 + 2 * f.bass), fade(dt, 0.35), swirl)
    lum0, hue0, lum1, hue1 = lum1, hue1, lum0, hue0

    local cy = h / 2
    local col = cycle(f.time * 0.07)
    local bright = 0.6 + 0.4 * f.level
    local g = 0.5 / max(0.15, f.peak)
    local left, right, n = f.left, f.right, #f.left
    local px, py
    for x = 0, w - 1 do
      local k = x * n // w
      local y = floor(cy - (left[k] + right[k]) / 2 * g * cy)
      if px then stroke(px, py, x, y, col, bright) end
      px, py = x, y
    end
    if f.beat then ring(w / 2, cy, min(w / 2, cy) * 0.6, palette.bright) end

    -- Trails fade into the theme's background rather than to black.
    -- Brightness never passes 1, so channels need no upper clamp. The
    -- lower one is an if, not max: apogee 1.1 runs floor(max(...)) slowly
    -- (matjam/apogee#140).
    local bg = palette.background
    local br, bgg, bb = bg >> 16 & 255, bg >> 8 & 255, bg & 255
    local pixels = c.pixels
    for i = 0, w * h - 1 do
      local v = lum0[i]
      if v > 0.02 then
        local col = hue0[i]
        local r, gr, b = (col >> 16 & 255) * v, (col >> 8 & 255) * v, (col & 255) * v
        if r < br then r = br end
        if gr < bgg then gr = bgg end
        if b < bb then b = bb end
        pixels[i] = floor(r) * 65536 + floor(gr) * 256 + floor(b)
      end
    end
  end,
}
