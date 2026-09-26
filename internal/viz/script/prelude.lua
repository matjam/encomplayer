-- Helpers for visualizer scripts. The player runs this in every script's
-- state before the script itself, so these are globals.

local min, max = math.min, math.max

-- clamp limits v to [lo, hi].
function clamp(v, lo, hi)
  if v < lo then return lo end
  if v > hi then return hi end
  return v
end

-- step is the frame's time step in seconds, capped so a long pause does
-- not make an animation jump.
function step(f)
  return min(f.dt, 0.25)
end

-- fade is the factor that halves a value every half_life seconds, over dt
-- seconds.
function fade(dt, half_life)
  return 0.5 ^ (dt / half_life)
end

-- tilt lifts band v, at position t in [0, 1] across the spectrum, by up to
-- boost at the top. Music carries far less energy in the treble than the
-- bass, so effects that map the spectrum across the screen would
-- otherwise go cold on the right.
function tilt(v, t, boost)
  if v > 0.05 then return min(1, v + boost * t) end
  return v
end

-- gain returns a function (peak, dt) -> gain that tracks the signal's peak,
-- so quiet passages still fill the screen without jumping frame to frame.
function gain()
  local held = 0
  return function(peak, dt)
    held = max(peak, held * fade(dt, 1))
    return 0.9 / max(0.2, held)
  end
end

-- EIGHTHS are the code points of the block glyphs that fill a cell from
-- the bottom, EIGHTHS[0] (empty) to EIGHTHS[8] (full).
EIGHTHS = { [0] = 0x20, 0x2581, 0x2582, 0x2583, 0x2584, 0x2585, 0x2586, 0x2587, 0x2588 }
