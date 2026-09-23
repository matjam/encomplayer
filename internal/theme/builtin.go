package theme

// palette builds a theme from colours in role order: background, text,
// bright, dim, grid, border, accent, error, selection, selection text.
func palette(name string, c ...string) Theme {
	return Theme{
		Name: name, Background: c[0], Text: c[1], Bright: c[2], Dim: c[3], Grid: c[4],
		Border: c[5], Accent: c[6], Error: c[7], Selection: c[8], SelectionText: c[9],
	}
}

// builtins maps each palette's official colours onto EncomPlayer's roles.
var builtins = func() map[string]Theme {
	themes := []Theme{
		//      name                 background text       bright     dim        grid       border     accent     error      selection  sel. text
		palette("encom", "#02090c", "#6fc3df", "#e6ffff", "#2a6475", "#123c48", "#2a6475", "#ff9a2e", "#ff4a3d", "#6fc3df", "#02090c"),

		palette("catppuccin-latte", "#eff1f5", "#4c4f69", "#1e66f5", "#9ca0b0", "#ccd0da", "#bcc0cc", "#fe640b", "#d20f39", "#1e66f5", "#eff1f5"),
		palette("catppuccin-frappe", "#303446", "#c6d0f5", "#8caaee", "#737994", "#414559", "#51576d", "#ef9f76", "#e78284", "#8caaee", "#303446"),
		palette("catppuccin-macchiato", "#24273a", "#cad3f5", "#8aadf4", "#6e738d", "#363a4f", "#494d64", "#f5a97f", "#ed8796", "#8aadf4", "#24273a"),
		palette("catppuccin-mocha", "#1e1e2e", "#cdd6f4", "#89b4fa", "#6c7086", "#313244", "#45475a", "#fab387", "#f38ba8", "#89b4fa", "#1e1e2e"),

		palette("gruvbox-dark", "#282828", "#ebdbb2", "#83a598", "#928374", "#3c3836", "#504945", "#fe8019", "#fb4934", "#83a598", "#282828"),
		palette("gruvbox-light", "#fbf1c7", "#3c3836", "#076678", "#928374", "#ebdbb2", "#d5c4a1", "#af3a03", "#9d0006", "#076678", "#fbf1c7"),

		palette("tokyonight-night", "#1a1b26", "#c0caf5", "#7aa2f7", "#565f89", "#292e42", "#3b4261", "#ff9e64", "#f7768e", "#7aa2f7", "#1a1b26"),
		palette("tokyonight-storm", "#24283b", "#c0caf5", "#7aa2f7", "#565f89", "#2f334d", "#3b4261", "#ff9e64", "#f7768e", "#7aa2f7", "#24283b"),
		palette("tokyonight-day", "#e1e2e7", "#3760bf", "#2e7de9", "#848cb5", "#c4c8da", "#a8aecb", "#b15c00", "#f52a65", "#2e7de9", "#e1e2e7"),

		palette("rose-pine", "#191724", "#e0def4", "#c4a7e7", "#6e6a86", "#26233a", "#403d52", "#f6c177", "#eb6f92", "#c4a7e7", "#191724"),
		palette("rose-pine-moon", "#232136", "#e0def4", "#c4a7e7", "#6e6a86", "#393552", "#44415a", "#f6c177", "#eb6f92", "#c4a7e7", "#232136"),
		palette("rose-pine-dawn", "#faf4ed", "#575279", "#907aa9", "#9893a5", "#f2e9e1", "#dfdad9", "#ea9d34", "#b4637a", "#907aa9", "#faf4ed"),

		palette("solarized-dark", "#002b36", "#839496", "#268bd2", "#586e75", "#073642", "#586e75", "#cb4b16", "#dc322f", "#268bd2", "#002b36"),
		palette("solarized-light", "#fdf6e3", "#657b83", "#268bd2", "#93a1a1", "#eee8d5", "#93a1a1", "#cb4b16", "#dc322f", "#268bd2", "#fdf6e3"),

		palette("kanagawa-wave", "#1f1f28", "#dcd7ba", "#7e9cd8", "#727169", "#2a2a37", "#363646", "#ffa066", "#e82424", "#7e9cd8", "#1f1f28"),
		palette("kanagawa-dragon", "#181616", "#c5c9c5", "#8ba4b0", "#737c73", "#282727", "#393836", "#b6927b", "#c4746e", "#8ba4b0", "#181616"),

		palette("everforest-dark", "#2d353b", "#d3c6aa", "#7fbbb3", "#859289", "#343f44", "#475258", "#e69875", "#e67e80", "#a7c080", "#2d353b"),
		palette("everforest-light", "#fdf6e3", "#5c6a72", "#3a94c5", "#939f91", "#f4f0d9", "#e0dcc7", "#f57d26", "#f85552", "#8da101", "#fdf6e3"),

		palette("ayu-dark", "#0b0e14", "#bfbdb6", "#39bae6", "#565b66", "#131721", "#1e222a", "#ff8f40", "#d95757", "#e6b450", "#0b0e14"),
		palette("ayu-mirage", "#1f2430", "#cccac2", "#73d0ff", "#707a8c", "#242936", "#33415e", "#ffad66", "#f28779", "#ffcc66", "#1f2430"),

		palette("github-dark", "#0d1117", "#c9d1d9", "#58a6ff", "#8b949e", "#161b22", "#30363d", "#d29922", "#f85149", "#1f6feb", "#ffffff"),
		palette("github-light", "#ffffff", "#24292f", "#0969da", "#6e7781", "#f6f8fa", "#d0d7de", "#bc4c00", "#cf222e", "#0969da", "#ffffff"),

		palette("dracula", "#282a36", "#f8f8f2", "#bd93f9", "#6272a4", "#343746", "#44475a", "#ffb86c", "#ff5555", "#bd93f9", "#282a36"),
		palette("nord", "#2e3440", "#d8dee9", "#88c0d0", "#616e88", "#3b4252", "#4c566a", "#ebcb8b", "#bf616a", "#88c0d0", "#2e3440"),
		palette("one-dark", "#282c34", "#abb2bf", "#61afef", "#5c6370", "#2c313a", "#3e4451", "#e5c07b", "#e06c75", "#61afef", "#282c34"),
		palette("monokai", "#272822", "#f8f8f2", "#66d9ef", "#75715e", "#3e3d32", "#49483e", "#fd971f", "#f92672", "#a6e22e", "#272822"),
		palette("nightfox", "#192330", "#cdcecf", "#719cd6", "#738091", "#29394f", "#39506d", "#f4a261", "#c94f6d", "#719cd6", "#192330"),
		palette("material", "#212121", "#eeffff", "#82aaff", "#545454", "#2c2c2c", "#424242", "#ffcb6b", "#f07178", "#80cbc4", "#212121"),
		palette("palenight", "#292d3e", "#a6accd", "#82aaff", "#676e95", "#32374d", "#444267", "#ffcb6b", "#ff5370", "#c792ea", "#292d3e"),
		palette("synthwave-84", "#262335", "#ffffff", "#36f9f6", "#848bbd", "#34294f", "#495495", "#fede5d", "#fe4450", "#ff7edb", "#262335"),
		palette("night-owl", "#011627", "#d6deeb", "#82aaff", "#637777", "#0b2942", "#1d3b53", "#ecc48d", "#ef5350", "#7fdbca", "#011627"),
		palette("oxocarbon", "#161616", "#f2f4f8", "#78a9ff", "#525252", "#262626", "#393939", "#ff7eb6", "#ee5396", "#33b1ff", "#161616"),
		palette("cyberpunk", "#000b1e", "#0abdc6", "#ea00d9", "#133e7c", "#091833", "#133e7c", "#f57800", "#ff003c", "#ea00d9", "#000b1e"),
	}
	m := make(map[string]Theme, len(themes))
	for _, t := range themes {
		m[t.Name] = t
	}
	return m
}()
