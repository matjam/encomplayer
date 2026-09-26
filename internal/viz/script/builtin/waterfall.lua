-- Scrolling spectrogram: a ring of history rows, newest at the top.

local ROW = 0.04 -- seconds between rows

local rows, newest, width, height = {}, 0, 0, 0
local since = 0

return {
  description = "Scrolling spectrogram: time runs down, pitch runs across",

  render = function(f, c)
    local w, h = c.pw, c.ph
    if w == 0 or h == 0 then return end
    if w ~= width or h ~= height then
      rows, newest, width, height = {}, 0, w, h
    end

    since = since + step(f)
    if since >= ROW or not rows[newest] then
      since = 0
      newest = (newest + 1) % h
      local b = bands(w)
      local row = rows[newest] or f64(w)
      for x = 0, w - 1 do
        row[x] = tilt(b[x], x / (w > 1 and w - 1 or 1), 0.25)
      end
      rows[newest] = row
    end

    local pixels = c.pixels
    for age = 0, h - 1 do
      local row = rows[(newest - age) % h]
      if not row then break end
      local base = age * w
      for x = 0, w - 1 do
        local v = row[x]
        if v > 0.05 then pixels[base + x] = heat(v) end
      end
    end
  end,
}
