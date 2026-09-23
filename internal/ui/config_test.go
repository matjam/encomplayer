package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matjam/encomplayer/internal/config"
	"github.com/matjam/encomplayer/internal/keymap"
)

func openConfig(t *testing.T, m *Model, tab string) *configModal {
	t.Helper()
	press(m, "oc")
	c, ok := m.modal.(*configModal)
	if !ok {
		t.Fatal("oc did not open the config screen")
	}
	for c.tabs[c.active].name != tab {
		press(m, "l")
	}
	return c
}

func savedConfig(t *testing.T, m *Model) config.Config {
	t.Helper()
	cfg, err := config.Load(m.deps.Paths.Config)
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestConfigThemePicker(t *testing.T) {
	m, _ := newTestModel(t)
	openConfig(t, m, "Appearance")
	if got := screen(t, m); !strings.Contains(got, "APPEARANCE") || !strings.Contains(got, "Theme") {
		t.Fatalf("appearance tab not shown:\n%s", got)
	}

	press(m, "<CR>") // open the theme list on "encom"
	press(m, "j")
	previewed := m.st.theme.Name
	if previewed == "encom" {
		t.Fatal("moving through the theme list did not preview")
	}
	press(m, "<Esc>")
	if m.st.theme.Name != "encom" {
		t.Errorf("cancel left theme %q, want encom", m.st.theme.Name)
	}

	press(m, "<CR>j<CR>")
	if m.deps.Config.Theme != previewed || m.st.theme.Name != previewed {
		t.Errorf("applied theme = %q (styles %q), want %q", m.deps.Config.Theme, m.st.theme.Name, previewed)
	}
	if got := savedConfig(t, m).Theme; got != previewed {
		t.Errorf("saved theme = %q, want %q", got, previewed)
	}
	screen(t, m)
}

func TestConfigEditsFields(t *testing.T) {
	m, _ := newTestModel(t)
	c := openConfig(t, m, "General")
	c.rows.SetCursor(2) // volume step

	press(m, "<CR>")
	press(m, "<BS>")
	typeText(m, "99")
	press(m, "<CR>")
	if c.input == nil || !m.status.isError {
		t.Fatal("out-of-range value should be rejected and keep the input open")
	}

	press(m, "<BS><BS>")
	typeText(m, "7")
	press(m, "<CR>")
	if m.deps.Config.VolumeStep != 7 || savedConfig(t, m).VolumeStep != 7 {
		t.Errorf("volume step = %d (saved %d), want 7", m.deps.Config.VolumeStep, savedConfig(t, m).VolumeStep)
	}

	press(m, "l<CR>") // Appearance tab is next; Mouse after that
	press(m, "<Esc>")
	press(m, "l<CR>")
	if m.deps.Config.EnableMouse {
		t.Error("Enter on Enable mouse did not toggle it off")
	}
	if m.View().MouseMode != 0 {
		t.Error("mouse reporting still on after disabling it")
	}
}

func TestConfigKeybindings(t *testing.T) {
	m, _ := newTestModel(t)
	c := openConfig(t, m, "Keys")
	for i, row := range c.rows.Items() {
		if k := row.(keyRow); k.ctx == keymap.Global && k.action == keymap.TogglePause {
			c.rows.SetCursor(i)
		}
	}

	press(m, "<CR>")
	typeText(m, "<C-p>")
	press(m, "<CR>")
	if got := m.deps.Config.Keybinds["global"]["<C-p>"]; got != keymap.TogglePause {
		t.Fatalf("binding not stored: %v", m.deps.Config.Keybinds)
	}
	if !strings.Contains(c.rows.Items()[c.rows.Cursor()].display(m), "ctrl+p") {
		t.Error("new key not shown on the row")
	}

	press(m, "d")
	if len(m.deps.Config.Keybinds["global"]) != 0 {
		t.Errorf("d left bindings %v", m.deps.Config.Keybinds["global"])
	}
	press(m, "d")
	if !m.status.isError {
		t.Error("removing a default key should explain it cannot")
	}
}

func TestCustomThemeAndReload(t *testing.T) {
	m, _ := newTestModel(t)
	dir := m.deps.Paths.Themes
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(body string) {
		if err := os.WriteFile(filepath.Join(dir, "mine.json"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write(`{"extends": "dracula", "colors": {"accent": "#00ff00"}}`)
	press(m, ":")
	typeText(m, "theme mine")
	press(m, "<CR>")
	if m.st.theme.Name != "mine" || m.st.theme.Accent != "#00ff00" {
		t.Fatalf("custom theme not applied: %+v", m.st.theme)
	}

	// Editing the file and signalling reload applies the change.
	write(`{"extends": "dracula", "colors": {"accent": "#0000ff"}}`)
	m.Update(ReloadMsg{})
	if m.st.theme.Accent != "#0000ff" {
		t.Errorf("reload kept accent %s, want #0000ff", m.st.theme.Accent)
	}

	cfg := m.deps.Config
	cfg.Theme = "nord"
	if err := config.Save(m.deps.Paths.Config, cfg); err != nil {
		t.Fatal(err)
	}
	m.Update(ReloadMsg{})
	if m.st.theme.Name != "nord" {
		t.Errorf("reload after editing config.json gave theme %q, want nord", m.st.theme.Name)
	}
}

func TestEveryThemeRenders(t *testing.T) {
	m, _ := newTestModel(t)
	for _, name := range m.deps.Themes.Names() {
		if err := m.applyTheme(name); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		screen(t, m)
	}
}
