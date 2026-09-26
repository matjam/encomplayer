-- The waveform wrapped around a slowly turning circle.

local min, sin, cos, pi = math.min, math.sin, math.cos, math.pi
local POINTS = 360

local angle = 0
local track = gain()

return {
  description = "Waveform wrapped around a slowly turning circle",

  render = function(f, c)
    local w, h = c.dw, c.dh
    if w == 0 or h == 0 then return end
    local dt = step(f)
    angle = angle + dt * (0.2 + f.level)
    local g = track(f.peak, dt)

    local cx, cy = w / 2, h / 2
    local base = min(cx, cy) * 0.55
    local amp = min(cx, cy) * 0.4
    local col = cycle(f.time * 0.05)
    local left, right, n = f.left, f.right, #f.left

    local fx, fy, px, py
    for i = 0, POINTS - 1 do
      local k = i * n // POINTS
      local rad = base + (left[k] + right[k]) / 2 * g * amp
      local a = angle + 2 * pi * i / POINTS
      local x, y = cx + cos(a) * rad, cy + sin(a) * rad
      if i == 0 then fx, fy = x, y else line(px, py, x, y, col) end
      px, py = x, y
    end
    line(px, py, fx, fy, col)

    -- A faint inner ring marks silence.
    for i = 0, 89 do
      local a = 2 * pi * i / 90
      dot(cx + cos(a) * base * 0.5, cy + sin(a) * base * 0.5, palette.grid)
    end
  end,
}
