-- Left and right channels, each triggered like a bench scope.

local SPAN = 1024 -- samples one trace shows, about 23 ms at 44.1 kHz

local track = gain()

local function trigger(s, limit)
  for i = 1, limit - 1 do
    if s[i - 1] < 0 and s[i] >= 0 then return i end
  end
  return limit > 0 and limit or 0
end

-- trace plots samples in the band of dot rows [top, top+height).
local function trace(c, s, g, top, height, col)
  local w = c.dw
  if w == 0 or height <= 0 then return end
  local mid = top + (height - 1) / 2
  for x = 0, w - 1, 4 do dot(x, mid, palette.grid) end

  local n = #s
  local start = trigger(s, n - SPAN)
  local span = n - start
  if span > SPAN then span = SPAN end
  local px, py
  for x = 0, w - 1 do
    local y = mid - s[start + x * span // w] * g * (height - 1) / 2
    if px then line(px, py, x, y, col) end
    px, py = x, y
  end
end

return {
  description = "Left and right channels traced one above the other",

  render = function(f, c)
    local g = track(f.peak, step(f))
    local half = c.dh // 2
    trace(c, f.left, g, 0, half, palette.bright)
    trace(c, f.right, g, half, c.dh - half, palette.accent)
  end,
}
