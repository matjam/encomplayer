-- Conway's Game of Life on a torus. Beats drop an R-pentomino under the
-- loudest band, and louder music runs the generations faster.

local random = math.random
local STEP = 0.12 -- seconds between generations at silence

math.randomseed(0x454E434F4D)

local width, height = 0, 0
local age, fresh, glow = i32(0), i32(0), f64(0) -- age 0 is dead
local live, next_live = i32(0), i32(0) -- 1 for a live cell, else 0
local since = 0

local R_PENTOMINO = { { 1, 0 }, { 2, 0 }, { 0, 1 }, { 1, 1 }, { 1, 2 } }

local function birth(i)
  age[i], live[i] = 1, 1
end

local function seed(x, y)
  for _, d in ipairs(R_PENTOMINO) do
    birth((y + d[2]) % height * width + (x + d[1]) % width)
  end
end

-- generation runs one step on a torus. A cell's neighbours are the live
-- cells in the three columns of three around it, less itself, and each
-- column's count carries over to the next cell.
local function generation()
  local w, h = width, height
  local a, na, l, nl = age, fresh, live, next_live
  for y = 0, h - 1 do
    local up, row, down = (y + h - 1) % h * w, y * w, (y + 1) % h * w
    local x = w - 1
    local left = l[up + x] + l[row + x] + l[down + x]
    local cur = l[up] + l[row] + l[down]
    for x = 0, w - 1 do
      local rx = x + 1
      if rx == w then rx = 0 end
      local right = l[up + rx] + l[row + rx] + l[down + rx]
      local i = row + x
      local n = left + cur + right - l[i]
      local was = a[i]
      if n == 3 or n == 2 and was > 0 then
        if was < 1000 then was = was + 1 end
        na[i], nl[i] = was, 1
      else
        na[i], nl[i] = 0, 0
      end
      left, cur = cur, right
    end
  end
  age, fresh, live, next_live = fresh, age, next_live, live
end

return {
  description = "Conway's Game of Life, seeded by beats and the loudest bands",

  render = function(f, c)
    local w, h = c.pw, c.ph
    if w == 0 or h == 0 then return end
    if w ~= width or h ~= height then
      width, height = w, h
      age, fresh, glow = i32(w * h), i32(w * h), f64(w * h)
      live, next_live = i32(w * h), i32(w * h)
      for _ = 1, w * h // 6 do birth(random(0, w * h - 1)) end
    end
    local dt = step(f)

    if f.beat then
      local b, loudest = bands(w), 0
      for x = 1, w - 1 do
        if b[x] > b[loudest] then loudest = x end
      end
      seed(loudest, random(0, h - 1))
    end
    since = since + dt * (1 + 3 * f.level)
    while since >= STEP do
      generation()
      since = since - STEP
    end

    -- Newborn cells flare and old ones settle into the text colour; dead
    -- cells leave a fading glow.
    local ages = {}
    for i = 1, 12 do ages[i] = lerp(palette.accent, palette.text, i / 12) end
    local keep = fade(dt, 0.4)
    local bg, dim = palette.background, palette.dim
    local a, gl, pixels = age, glow, c.pixels
    for i = 0, w * h - 1 do
      local n = a[i]
      if n > 0 then
        gl[i] = 1.0
        pixels[i] = ages[n < 12 and n or 12]
      else
        local v = gl[i] * keep
        gl[i] = v
        if v > 0.1 then pixels[i] = lerp(bg, dim, v) end
      end
    end
  end,
}
