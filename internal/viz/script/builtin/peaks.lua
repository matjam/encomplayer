-- Winamp-style bars, two cells wide with a one-cell gap, and peak caps.

local max, min = math.max, math.min

local HOLD = 0.5 -- seconds a cap hangs before falling
local FALL = 0.9 -- fraction of the height a cap falls per second
local CAP = utf8.codepoint("▔")

local levels, caps, hold, count = {}, {}, {}, 0

return {
  description = "Winamp-style bars with peak caps that hang, then fall",

  render = function(f, c)
    local w, h = c.w, c.h
    local n = max(1, (w + 1) // 3)
    local b = bands(n)
    if count ~= n then
      levels, caps, hold, count = {}, {}, {}, n
      for i = 0, n - 1 do levels[i], caps[i], hold[i] = 0, 0, 0 end
    end
    local dt = step(f)
    local keep = fade(dt, 0.12)
    local steps = h * 8

    for i = 0, n - 1 do
      local level = max(b[i], levels[i] * keep)
      levels[i] = level
      if level >= caps[i] then
        caps[i], hold[i] = level, HOLD
      else
        hold[i] = hold[i] - dt
        if hold[i] < 0 then caps[i] = max(level, caps[i] - FALL * dt) end
      end

      local lit = (level * steps + 0.5) // 1
      local cap_row = h - 1 - min(h - 1, (caps[i] * h) // 1)
      for y = 0, h - 1 do
        local from_bottom = h - 1 - y
        local col = ramp(0.35 + 0.65 * (from_bottom + 1) / h)
        local cell_lit = lit - from_bottom * 8
        for dx = 0, 1 do
          local x = i * 3 + dx
          if cell_lit > 0 then
            set(x, y, EIGHTHS[min(cell_lit, 8)], col)
          elseif y == cap_row and caps[i] > 0.02 then
            set(x, y, CAP, palette.accent)
          end
        end
      end
    end
  end,
}
