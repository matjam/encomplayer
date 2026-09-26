-- A warp-speed starfield. Each star streaks from where it was, so speed
-- reads as length.

local max, min, random = math.max, math.min, math.random
local COUNT = 400

math.randomseed(0x454E434F4D)

-- Stars live in three buffers: position in the plane, and depth.
local sx, sy, sz = f64(COUNT), f64(COUNT), f64(COUNT)

local function spawn(i, far)
  sx[i] = random() * 4 - 2
  sy[i] = random() * 4 - 2
  sz[i] = 0.1 + random() * (far - 0.1 + 0.01)
end

for i = 0, COUNT - 1 do spawn(i, random()) end

local boost = 0

return {
  description = "Warp-speed starfield; louder music, faster travel",

  render = function(f, c)
    local w, h = c.dw, c.dh
    if w == 0 or h == 0 then return end
    local dt = step(f)
    boost = max(f.beat_strength * 2, boost * fade(dt, 0.25))
    local speed = 0.15 + 1.2 * f.level + boost

    local cx, cy = w / 2, h / 2
    local focal = min(cx, cy)
    for i = 0, COUNT - 1 do
      local old = sz[i]
      local z = old - speed * dt
      sz[i] = z
      if z <= 0.02 then
        spawn(i, 1)
      else
        local x, y = sx[i], sy[i]
        local x1, y1 = cx + x / z * focal, cy + y / z * focal
        if x1 < 0 or y1 < 0 or x1 >= w or y1 >= h then
          spawn(i, 1)
        else
          line(cx + x / old * focal, cy + y / old * focal, x1, y1, ramp(1 - z))
        end
      end
    end
  end,
}
