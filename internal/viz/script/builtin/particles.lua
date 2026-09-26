-- Fountains of sparks. Positions are fractions of the canvas, so resizing
-- keeps them.

local random, sin, cos, pi = math.random, math.sin, math.cos, math.pi
local MAX = 1500
local GRAVITY = 1.6 -- canvas heights per second squared

math.randomseed(0x454E434F4D)

-- Particle i lives at index i of each buffer; the first count are alive.
local px, py, vx, vy = f64(MAX), f64(MAX), f64(MAX), f64(MAX)
local life, hue = f64(MAX), f64(MAX)
local count = 0

-- burst throws n sparks upwards from the bottom at x, with speed in canvas
-- heights per second.
local function burst(x, n, speed)
  local h = random()
  for _ = 1, n do
    if count >= MAX then return end
    local a = -pi / 2 + (random() - 0.5) * 0.9
    local v = speed * (0.6 + 0.6 * random())
    local i = count
    px[i], py[i] = x, 1
    vx[i], vy[i] = cos(a) * v, sin(a) * v * 1.4
    life[i], hue[i] = 0.7 + 0.3 * random(), h + random() * 0.15
    count = count + 1
  end
end

return {
  description = "Fountains of sparks thrown up on every beat",

  render = function(f, c)
    local w, h = c.dw, c.dh
    if w == 0 or h == 0 then return end
    local dt = step(f)

    if f.beat then
      burst(random() * 0.8 + 0.1, 40 + (160 * f.beat_strength) // 1, 0.9 + f.beat_strength)
    end
    -- A steady trickle from the middle follows the level.
    local trickle = (f.level * 300 * dt) // 1
    if trickle > 0 then burst(0.5, trickle, 0.5 + f.level * 0.6) end

    local aspect = h / w
    local bg = palette.background
    local live = 0
    for i = 0, count - 1 do
      local ox, oy = px[i], py[i]
      local nvy = vy[i] + GRAVITY * dt
      local x, y = ox + vx[i] * dt * aspect, oy + nvy * dt
      local l = life[i] - dt * 0.6
      if l > 0 and y <= 1 and x >= 0 and x <= 1 then
        line(ox * w, oy * h, x * w, y * h, lerp(cycle(hue[i]), bg, 1 - l))
        px[live], py[live], vx[live], vy[live] = x, y, vx[i], nvy
        life[live], hue[live] = l, hue[i]
        live = live + 1
      end
    end
    count = live
  end,
}
