-- A rotating icosahedron whose vertices push out with the music.

local max, min, sqrt, sin, cos = math.max, math.min, math.sqrt, math.sin, math.cos

local PHI = (1 + sqrt(5)) / 2
local verts = {}
for _, v in ipairs({
  { -1, PHI, 0 }, { 1, PHI, 0 }, { -1, -PHI, 0 }, { 1, -PHI, 0 },
  { 0, -1, PHI }, { 0, 1, PHI }, { 0, -1, -PHI }, { 0, 1, -PHI },
  { PHI, 0, -1 }, { PHI, 0, 1 }, { -PHI, 0, -1 }, { -PHI, 0, 1 },
}) do
  local len = sqrt(v[1] * v[1] + v[2] * v[2] + v[3] * v[3])
  verts[#verts + 1] = { v[1] / len, v[2] / len, v[3] / len }
end

-- Edges join vertices at the shortest distance.
local edges = {}
for i = 1, #verts do
  for j = i + 1, #verts do
    local a, b = verts[i], verts[j]
    local dx, dy, dz = a[1] - b[1], a[2] - b[2], a[3] - b[3]
    if sqrt(dx * dx + dy * dy + dz * dz) < 1.1 then edges[#edges + 1] = { i, j } end
  end
end

-- rotate turns (x, y, z) around the x, y and z axes, in that order.
local function rotate(x, y, z, ax, ay, az)
  local s, c = sin(ax), cos(ax)
  y, z = y * c - z * s, y * s + z * c
  s, c = sin(ay), cos(ay)
  x, z = x * c + z * s, -x * s + z * c
  s, c = sin(az), cos(az)
  return x * c - y * s, x * s + y * c, z
end

local ax, ay, az, pulse = 0, 0, 0, 0
local px, py, pz = {}, {}, {}

local function depth(e) return pz[e[1]] + pz[e[2]] end

return {
  description = "Rotating icosahedron whose vertices push out with the music",

  render = function(f, c)
    local w, h = c.dw, c.dh
    if w == 0 or h == 0 then return end
    local dt = step(f)
    local speed = 0.3 + 1.5 * f.level
    ax, ay, az = ax + dt * speed * 0.7, ay + dt * speed, az + dt * speed * 0.3
    pulse = max(f.beat_strength, pulse * fade(dt, 0.15))

    local b = bands(#verts)
    local size = min(w, h) * 0.32 * (1 + 0.25 * pulse)
    local cx, cy = w / 2, h / 2
    for i, v in ipairs(verts) do
      local s = 1 + 0.5 * b[i - 1]
      local x, y, z = rotate(v[1] * s, v[2] * s, v[3] * s, ax, ay, az)
      local persp = 3 / (3 + z)
      px[i], py[i], pz[i] = cx + x * size * persp, cy + y * size * persp, z
    end

    -- Draw far edges first, so near ones stay on top.
    table.sort(edges, function(e1, e2) return depth(e1) > depth(e2) end)
    for _, e in ipairs(edges) do
      local near = clamp(0.5 - depth(e) / 4, 0, 1)
      line(px[e[1]], py[e[1]], px[e[2]], py[e[2]], lerp(palette.dim, palette.bright, near))
    end
    for i = 1, #verts do
      line(px[i] - 1, py[i], px[i] + 1, py[i], palette.accent)
    end
  end,
}
