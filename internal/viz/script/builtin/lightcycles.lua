-- Light cycles racing the Grid. The Grid is a floor plane in world units;
-- the camera rides above it, moving forward along +z.

local abs, ceil, random = math.abs, math.ceil, math.random

local SPACING = 2.0
local CAM_HEIGHT = 2.6
local FAR = 60.0
local TRAIL = 90

math.randomseed(0x454E434F4D)

local cam_z = 0
local cycles

-- face points the cycle along (dx, dz), leaving a corner in its trail.
local function face(cy, dx, dz)
  if cy.dx == dx and cy.dz == dz then return end
  cy.dx, cy.dz = dx, dz
  table.insert(cy.trail, #cy.trail, { cy.x, cy.z })
end

-- turn swings the cycle 90° left or right.
local function turn(cy, left)
  if left then face(cy, -cy.dz, cy.dx) else face(cy, cy.dz, -cy.dx) end
end

-- steer keeps a cycle in view. It rides forward when it falls behind,
-- stops heading for the edge when it strays, and cuts back towards the
-- middle once it has room ahead.
local function steer(cy)
  local ahead = cy.z - cam_z
  local forward = cy.dz > 0
  local outward = cy.dx * cy.x > 0
  if ahead < 4 and not forward then
    face(cy, 0, 1)
  elseif abs(cy.x) > 7 and outward then
    face(cy, 0, 1)
  elseif forward and ahead > 16 then
    turn(cy, cy.x > 0)
  elseif forward and ahead > 9 and abs(cy.x) > 4 then
    turn(cy, cy.x > 0)
  end
end

return {
  description = "Light cycles racing the Grid, turning hard on the beat",

  render = function(f, c)
    local w, h = c.dw, c.dh
    if w == 0 or h == 0 then return end
    if not cycles then
      cycles = {}
      for i, x in ipairs({ -3, 0, 3 }) do
        local z = 5 + (i - 1) * 2
        -- The trail's last point follows the cycle; the first stays where
        -- it started.
        cycles[i] = { x = x, z = z, dx = 0, dz = 1, trail = { { x, z }, { x, z } }, colour = i }
      end
    end
    local dt = step(f)
    local speed = 5 + 14 * f.level
    cam_z = cam_z + speed * dt

    if f.beat then turn(cycles[random(#cycles)], random(2) == 1) end
    for _, cy in ipairs(cycles) do
      steer(cy)
      cy.x = cy.x + cy.dx * speed * 1.15 * dt
      cy.z = cy.z + cy.dz * speed * 1.15 * dt
      local trail = cy.trail
      trail[#trail] = { cy.x, cy.z }
      while #trail > TRAIL do table.remove(trail, 1) end
    end

    local horizon = h * 0.25
    local focal = w * 0.35
    local function project(x, z)
      z = z - cam_z
      if z < 0.3 then return nil end
      return w / 2 + x / z * focal, horizon + CAM_HEIGHT / z * focal
    end

    -- Cross lines scroll towards the camera; rails converge on the horizon.
    local z = ceil(cam_z / SPACING) * SPACING
    while z < cam_z + FAR do
      local x0, y = project(-40, z)
      local x1 = project(40, z)
      if x0 and x1 then
        line(x0, y, x1, y, lerp(palette.grid, palette.dim, 1 - (z - cam_z) / FAR))
      end
      z = z + SPACING
    end
    for x = -40, 40, SPACING do
      local x0, y0 = project(x, cam_z + 0.5)
      local x1, y1 = project(x, cam_z + FAR)
      line(x0, y0, x1, y1, palette.grid)
    end
    line(0, horizon, w, horizon, palette.text)

    local colours = { palette.accent, palette.bright, palette.error }
    for _, cy in ipairs(cycles) do
      local col = colours[(cy.colour - 1) % #colours + 1]
      local trail = cy.trail
      for i = 2, #trail do
        -- Short pieces, so the parts behind the camera drop out and the
        -- rest keeps its perspective.
        local a, b = trail[i - 1], trail[i]
        local px, py = project(a[1], a[2])
        for k = 1, 8 do
          local t = k / 8
          local x, y = project(a[1] + (b[1] - a[1]) * t, a[2] + (b[2] - a[2]) * t)
          if x and px then line(px, py, x, y, col) end
          px, py = x, y
        end
      end
      local x, y = project(cy.x, cy.z)
      if x then line(x - 1, y - 1, x + 1, y - 1, palette.bright) end
    end
  end,
}
