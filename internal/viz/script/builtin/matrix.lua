-- Digital rain: drops of glyphs fall down each column, more of them and
-- faster as the music gets louder.

local random = math.random

-- Single-width glyphs: half-width katakana, digits and symbols.
local GLYPHS = {}
for _, cp in utf8.codes("ｱｲｳｴｵｶｷｸｹｺｻｼｽｾｿﾀﾁﾂﾃﾄﾅﾆﾇﾈﾉﾊﾋﾌﾍﾎﾏﾐﾑﾒﾓﾔﾕﾖﾗﾘﾙﾚﾛﾜﾝ0123456789:.=*+<>") do
  GLYPHS[#GLYPHS + 1] = cp
end

math.randomseed(0x454E434F4D)

local function glyph() return GLYPHS[random(#GLYPHS)] end

local width, height = 0, 0
local glyphs, drops = {}, {}

return {
  description = "Digital rain that pours harder with the music",

  render = function(f, c)
    local w, h = c.w, c.h
    if w == 0 or h == 0 then return end
    if w ~= width or h ~= height then
      width, height, glyphs, drops = w, h, {}, {}
      for i = 0, w * h - 1 do glyphs[i] = glyph() end
    end
    local dt = step(f)
    local pace = 0.4 + 1.8 * f.level
    local spawn = dt * (0.3 + 3 * f.level + 6 * f.beat_strength)

    for x = 0, w - 1 do
      local d = drops[x]
      if not d then
        if random() < spawn / 4 then
          drops[x] = { y = -1, speed = 6 + random() * 14, length = 4 + random(0, h - 1) }
        end
      else
        d.y = d.y + d.speed * pace * dt
        local head = d.y // 1
        if head - d.length > h then
          drops[x] = nil
        else
          for i = 0, d.length - 1 do
            local y = head - i
            if y >= 0 and y < h then
              local k = y * w + x
              if random() < 0.02 then glyphs[k] = glyph() end
              local col = i == 0 and palette.bright or ramp(0.85 - 0.7 * i / d.length)
              set(x, y, glyphs[k], col)
            end
          end
        end
      end
    end
  end,
}
