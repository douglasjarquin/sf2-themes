package wezterm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
	"github.com/douglasjarquin/sf2-themes/v2/internal/plan"
)

// Expected values were recorded from sf2_theme.adapters.wezterm and wezterm_lua under Python 3.11;
// testdata/wezterm-{main,ryu}.toml and wezterm-previous-sf2.lua are copies of the reference's tests/snapshots and
// tests/fixtures.

func loadTheme(t *testing.T, id string) (catalog.Theme, []catalog.Theme) {
	t.Helper()
	themes, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	theme, err := catalog.GetTheme(id, themes)
	if err != nil {
		t.Fatal(err)
	}
	return theme, themes
}

func readTestdata(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestRenderSchemeMatchesTheSnapshots(t *testing.T) {
	for id, snapshot := range map[string]string{"main": "wezterm-main.toml", "ryu": "wezterm-ryu.toml"} {
		theme, _ := loadTheme(t, id)
		if got, want := renderScheme(theme), readTestdata(t, snapshot); got != want {
			t.Errorf("%s: rendered scheme differs from %s:\n%s", id, snapshot, got)
		}
	}
}

func TestRenderPointerFollowsHostAppearance(t *testing.T) {
	theme, themes := loadTheme(t, "ryu-light")
	dark, light, err := catalog.ThemePair(theme, themes)
	if err != nil {
		t.Fatal(err)
	}
	want := "-- sf2-themes: sf2-ryu\nlocal wezterm = require(\"wezterm\")\n" +
		"local appearance = wezterm.gui and wezterm.gui.get_appearance() or \"Dark\"\n" +
		"if appearance:find(\"Dark\") then\n  return \"sf2-ryu\"\nend\nreturn \"sf2-ryu-light\"\n"
	if got := renderPointer(dark, light); got != want {
		t.Errorf("pointer:\n%s", got)
	}
}

const testPointer = "/tmp/xdg/sf2-theme/wezterm-current.lua"

func integrated(builder, rest string) string {
	return "local wezterm = require(\"wezterm\")\nlocal " + builder + " = wezterm.config_builder()\n" +
		rest +
		"local sf2_current = \"/tmp/xdg/sf2-theme/wezterm-current.lua\"\n" +
		"wezterm.add_to_config_reload_watch_list(sf2_current)\n" +
		"local sf2_scheme = dofile(sf2_current)\n" +
		builder + ".color_scheme = sf2_scheme\n" +
		"local sf2_env = " + builder + ".set_environment_variables or {}\n" +
		"sf2_env.TERM_THEME = (type(sf2_scheme) == \"string\" and sf2_scheme:find(\"-light\", 1, true)) and \"light\" or \"dark\"\n" +
		builder + ".set_environment_variables = sf2_env\n" +
		"return " + builder + "\n"
}

func TestClassifyAndIntegrate(t *testing.T) {
	const builder = "local wezterm = require(\"wezterm\")\nlocal config = wezterm.config_builder()\n"
	const commentedIntegration = "-- local sf2_current = \"/tmp/xdg/sf2-theme/wezterm-current.lua\" -- dofile(sf2_current) TERM_THEME\n"
	const unguarded = "local sf2_current = \"/tmp/xdg/sf2-theme/wezterm-current.lua\"\n" +
		"wezterm.add_to_config_reload_watch_list(sf2_current)\nconfig.color_scheme = dofile(sf2_current)\n"
	const pasted = "local wezterm = require(\"wezterm\")\nlocal sf2_current = \"/tmp/xdg/sf2-theme/wezterm-current.lua\"\n" +
		"local sf2_scheme = dofile(sf2_current)\nTERM_THEME\nreturn { color_scheme = sf2_scheme }\n"
	cases := []struct {
		name      string
		existing  string
		ownership plan.Ownership
		want      string
	}{
		{"blank", "  \n", plan.Absent, integrated("config", "")},
		{"builder", builder + "return config\n", plan.Absent, integrated("config", "")},
		{"commented-out integration is not integrated", builder + commentedIntegration + "return config\n", plan.Absent, integrated("config", commentedIntegration)},
		{"commented-out guard does not count", builder + unguarded + "-- TERM_THEME\nreturn config\n", plan.Legacy,
			"local wezterm = require(\"wezterm\")\nlocal config = wezterm.config_builder()\n" +
				"local sf2_current = \"/tmp/xdg/sf2-theme/wezterm-current.lua\"\n" +
				"wezterm.add_to_config_reload_watch_list(sf2_current)\n" +
				"local sf2_scheme = dofile(sf2_current)\nconfig.color_scheme = sf2_scheme\n" +
				"local sf2_env = config.set_environment_variables or {}\n" +
				"sf2_env.TERM_THEME = (type(sf2_scheme) == \"string\" and sf2_scheme:find(\"-light\", 1, true)) and \"light\" or \"dark\"\n" +
				"config.set_environment_variables = sf2_env\n-- TERM_THEME\nreturn config\n"},
		{"foreign scheme", builder + "config.color_scheme = \"Catppuccin Mocha\"\nreturn config\n", plan.Foreign, integrated("config", "")},
		{"legacy literal scheme", builder + "config.color_scheme = \"street-fighter-2\"\nreturn config\n", plan.Legacy, integrated("config", "")},
		{"builder name mismatch", builder + "return cfg\n", plan.UnknownShape, builder + "return cfg\n"},
		{"plain table", "return {\n  font_size = 13,\n}\n", plan.UnknownShape, "return {\n  font_size = 13,\n}\n"},
		{"discovered builder name", "local wezterm = require(\"wezterm\")\nlocal cfg = wezterm.config_builder()\nreturn cfg\n", plan.Absent, integrated("cfg", "")},
		{"pasted snippet", pasted, plan.Managed, pasted},
	}
	for _, tc := range cases {
		if got := classify(tc.existing, testPointer); got != tc.ownership {
			t.Errorf("%s: classify = %q, want %q", tc.name, got, tc.ownership)
		}
		if got := integrate(tc.existing, testPointer); got != tc.want {
			t.Errorf("%s: integrate =\n%s\nwant\n%s", tc.name, got, tc.want)
		}
	}
}

func TestPreviousSF2SchemeAssignmentIsUpgraded(t *testing.T) {
	existing := readTestdata(t, "wezterm-previous-sf2.lua")
	const pointer = "/home/fighter/.config/sf2-theme/wezterm-current.lua"
	if got := classify(existing, pointer); got != plan.Legacy {
		t.Errorf("classify = %q, want legacy", got)
	}
	if got, want := integrate(existing, pointer), readTestdata(t, "wezterm-previous-sf2-integrated.lua"); got != want {
		t.Errorf("integrate =\n%s", got)
	}
}

func TestIntegrationSnippetEscapesThePointerPath(t *testing.T) {
	want := "local sf2_current = \"/tmp/a\\\\b\\\"c/wezterm-current.lua\"\n" +
		"wezterm.add_to_config_reload_watch_list(sf2_current)\n" +
		"local sf2_scheme = dofile(sf2_current)\ncfg.color_scheme = sf2_scheme\n" +
		"local sf2_env = cfg.set_environment_variables or {}\n" +
		"sf2_env.TERM_THEME = (type(sf2_scheme) == \"string\" and sf2_scheme:find(\"-light\", 1, true)) and \"light\" or \"dark\"\n" +
		"cfg.set_environment_variables = sf2_env\n"
	if got := integrationSnippet(`/tmp/a\b"c/wezterm-current.lua`, "cfg"); got != want {
		t.Errorf("snippet:\n%s", got)
	}
}

func TestInspectReadsThePointerCommentWithZeroOrMoreSpaces(t *testing.T) {
	_, themes := loadTheme(t, "main")
	cases := []struct {
		pointer    string
		current    string
		unreadable bool
	}{
		{"-- sf2-themes:sf2-ryu\n", "ryu", false},
		{"-- sf2-themes:   sf2-ryu  \n", "ryu", false},
		{"--sf2-themes:sf2-ryu\n", "", true},
		{"-- sf2-themes: nope\n", "", true},
		{"return 1\n", "", true},
	}
	for _, tc := range cases {
		dir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "xdg"))
		writeFile(t, filepath.Join(dir, "xdg", "sf2-theme", "wezterm-current.lua"), tc.pointer)
		got, err := Inspect(themes, filepath.Join(dir, "wezterm"))
		if err != nil {
			t.Fatal(err)
		}
		current := ""
		if got.Current != nil {
			current = got.Current.Metadata.ID
		}
		if got.Ownership != plan.Absent || current != tc.current || got.Unreadable != tc.unreadable || got.Detail != "" {
			t.Errorf("%q: %+v", tc.pointer, got)
		}
	}
}

func TestInspectCarriesTheSnippetForAnUnknownShape(t *testing.T) {
	_, themes := loadTheme(t, "main")
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir+"/xdg")
	writeFile(t, dir+"/wezterm/wezterm.lua", "return {}\n")
	got, err := Inspect(themes, dir+"/wezterm")
	if err != nil {
		t.Fatal(err)
	}
	want := "Add this integration to " + dir + "/wezterm/wezterm.lua:\n\n" +
		"local sf2_current = \"" + dir + "/xdg/sf2-theme/wezterm-current.lua\"\n" +
		"wezterm.add_to_config_reload_watch_list(sf2_current)\n" +
		"local sf2_scheme = dofile(sf2_current)\nconfig.color_scheme = sf2_scheme\n" +
		"local sf2_env = config.set_environment_variables or {}\n" +
		"sf2_env.TERM_THEME = (type(sf2_scheme) == \"string\" and sf2_scheme:find(\"-light\", 1, true)) and \"light\" or \"dark\"\n" +
		"config.set_environment_variables = sf2_env\n"
	if got.Ownership != plan.UnknownShape || got.Detail != want {
		t.Errorf("inspection: %+v", got)
	}
}

func TestPlanLeavesWeztermLuaAloneWithoutTheSlot(t *testing.T) {
	theme, themes := loadTheme(t, "main")
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir+"/xdg")
	for _, slot := range []bool{false, true} {
		steps, err := Plan(theme, themes, dir+"/wezterm", slot)
		if err != nil {
			t.Fatal(err)
		}
		want := 2*len(themes) + 1
		if slot {
			want++
		}
		if len(steps) != want {
			t.Fatalf("slot %v: %d steps, want %d", slot, len(steps), want)
		}
		if !steps[0].Remove || steps[0].Path != dir+"/wezterm/colors/street-fighter-ii-main.toml" {
			t.Errorf("first step: %+v", steps[0])
		}
		if pointer := steps[2*len(themes)]; pointer.Path != dir+"/xdg/sf2-theme/wezterm-current.lua" {
			t.Errorf("pointer step: %+v", pointer)
		}
		if slot && steps[len(steps)-1].Path != dir+"/wezterm/wezterm.lua" {
			t.Errorf("integration step: %+v", steps[len(steps)-1])
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
