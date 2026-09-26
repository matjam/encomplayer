-- A synthwave sunset over ridges of mountains. Each ridge takes its
-- heights from the spectrum when it rises on the horizon.

local max, min, sqrt, fmod = math.max, math.min, math.sqrt, math.fmod

local COUNT = 18
local POINTS = 32
local SPACING = 1.0
local DEPTH = COUNT * SPACING

local ridges -- far to near: { z = ..., heights = { [0] = ... } }

-- mountains shapes a ridge: a flat road in the middle and peaks rising
-- towards the sides, where the bass lifts the tallest ones.
local function mountains()
  local half = POINTS // 2
  local b = bands(half)
  local out = {}
  for j = 0, POINTS - 1 do
    local d = j - half
    if d < 0 then d = -d - 1 end
    local side = clamp((d - 2) / (half - 2), 0, 1)
    local k = half - 1 - d
    local v = tilt(b[k], k / (half - 1), 0.2)
    out[j] = v * v * side * 5
  end
  return out
end

local function flat()
  local out = {}
  for j = 0, POINTS - 1 do out[j] = 0 end
  return out
end

-- advance moves every ridge towards the viewer and raises new ones on the
-- horizon.
local function advance(travel)
  if not ridges then
    ridges = {}
    for i = 0, COUNT - 1 do
      ridges[#ridges + 1] = { z = DEPTH - i * SPACING + 0.5, heights = flat() }
    end
  end
  for _, r in ipairs(ridges) do r.z = r.z - travel end
  while #ridges > 0 and ridges[#ridges].z < 0.5 do ridges[#ridges] = nil end
  while #ridges < COUNT do
    local far = #ridges > 0 and ridges[1].z + SPACING or DEPTH + 0.5
    table.insert(ridges, 1, { z = far, heights = mountains() })
  end
end

-- sun draws a striped sun sinking into the horizon.
local function sun(cx, horizon, radius, bass)
  local r = radius * (0.9 + 0.1 * bass)
  local y = horizon - r
  while y < horizon do
    local t = (y - (horizon - r)) / r
    -- Stripes widen towards the horizon, as in every synthwave poster.
    if not (t > 0.45 and fmod(y, 4 + t * 6) < 1 + t * 4) then
      local half = sqrt(max(0, r * r - (y - horizon) * (y - horizon)))
      line(cx - half, y, cx + half, y, lerp(palette.accent, palette.error, t))
    end
    y = y + 1
  end
end

return {
  description = "Synthwave sunset over mountains raised by the spectrum",

  render = function(f, c)
    local w, h = c.dw, c.dh
    if w == 0 or h == 0 then return end
    advance(step(f) * (1.5 + 4 * f.level))

    local horizon = h * 0.45
    sun(w / 2, horizon, min(w, h) * 0.3, f.bass)

    local focal = w * 0.45
    local prev_x, prev_y = {}, {}
    local xs, ys = {}, {}
    for i, r in ipairs(ridges) do
      local col = lerp(palette.accent, palette.grid, r.z / DEPTH)
      for j = 0, POINTS - 1 do
        local x = (j / (POINTS - 1) - 0.5) * 14
        xs[j] = w / 2 + x / r.z * focal
        ys[j] = horizon + (1.2 - r.heights[j]) / r.z * focal
        if j > 0 then line(xs[j - 1], ys[j - 1], xs[j], ys[j], col) end
        if i > 1 then line(prev_x[j], prev_y[j], xs[j], ys[j], col) end
      end
      prev_x, xs = xs, prev_x
      prev_y, ys = ys, prev_y
    end
  end,
}
