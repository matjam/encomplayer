-- Doom-style fire: the bottom row burns as hot as the band beneath it,
-- and each step moves the heat up a row, cooling it and letting it drift.

local STEP = 1 / 40 -- seconds between steps, whatever the frame rate

local temp, width, height = f64(0), 0, 0
local since = 0
local seed = 0x454E434F4D

-- spread moves the heat up a row, cooling it and letting it drift. It
-- draws its random numbers from an inline generator (a 64-bit LCG, taking
-- high bits), as calling math.random twice a pixel costs more than the
-- rest of the frame.
local function spread(w, h)
  local hb, s = temp, seed
  local cooling = 3.0 / h / 65536
  for y = 0, h - 2 do
    local row, below = y * w, (y + 1) * w
    for x = 0, w - 1 do
      s = s * 6364136223846793005 + 1442695040888963407
      local drift = x + 1 - (s >> 40) % 3
      if drift < 0 then drift = 0 elseif drift > w - 1 then drift = w - 1 end
      local v = hb[below + x] - (s >> 48) * cooling
      if v < 0.0 then v = 0.0 end
      hb[row + drift] = v
    end
  end
  seed = s
end

return {
  description = "Doom-style fire fed by the spectrum along its base",

  render = function(f, c)
    local w, h = c.pw, c.ph
    if w == 0 or h == 0 then return end
    if w ~= width or h ~= height then
      temp, width, height = f64(w * h), w, h
    end

    local b = bands(w)
    local base = (h - 1) * w
    local span = w > 1 and w - 1 or 1
    for x = 0, w - 1 do
      temp[base + x] =clamp(tilt(b[x], x / span, 0.3) * 1.3 + f.beat_strength * 0.3, 0, 1)
    end

    since = since + step(f)
    while since >= STEP do
      spread(w, h)
      since = since - STEP
    end

    local hb, pixels = temp, c.pixels
    for i = 0, w * h - 1 do
      local v = hb[i]
      if v > 0.04 then pixels[i] = heat(v) end
    end
  end,
}
