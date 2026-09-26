-- A triggered waveform trace, like a bench scope.

local SPAN = 1024 -- samples one trace shows, about 23 ms at 44.1 kHz

local mono = f64(2048)
local track = gain()

-- trigger finds a rising zero crossing, leaving SPAN samples after it.
local function trigger(s, limit)
  for i = 1, limit - 1 do
    if s[i - 1] < 0 and s[i] >= 0 then return i end
  end
  return limit > 0 and limit or 0
end

return {
  description = "Triggered waveform trace, like a bench scope",

  render = function(f, c)
    local w, height = c.dw, c.dh
    if w == 0 or height == 0 then return end
    local left, right, n = f.left, f.right, #f.left
    for i = 0, n - 1 do mono[i] = (left[i] + right[i]) / 2 end
    local g = track(f.peak, step(f))

    local mid = (height - 1) / 2
    for x = 0, w - 1, 4 do dot(x, mid, palette.grid) end

    local start = trigger(mono, n - SPAN)
    local span = n - start
    if span > SPAN then span = SPAN end
    local col, px, py = palette.bright, nil, nil
    for x = 0, w - 1 do
      local y = mid - mono[start + x * span // w] * g * (height - 1) / 2
      if px then line(px, py, x, y, col) end
      px, py = x, y
    end
  end,
}
