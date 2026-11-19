package wezterm

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
	"github.com/douglasjarquin/sf2-themes/v2/internal/fsys"
	"github.com/douglasjarquin/sf2-themes/v2/internal/plan"
	"github.com/douglasjarquin/sf2-themes/v2/internal/pycompat"
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

// testFormer are former pointers no classify case names.
var testFormer = []string{"/tmp/xdg-legacy/sf2-theme/wezterm-current.lua"}

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
		if got := classify(tc.existing, testPointer, testFormer); got != tc.ownership {
			t.Errorf("%s: classify = %q, want %q", tc.name, got, tc.ownership)
		}
		if got := integrate(tc.existing, testPointer, testFormer); got != tc.want {
			t.Errorf("%s: integrate =\n%s\nwant\n%s", tc.name, got, tc.want)
		}
	}
}

func TestPreviousSF2SchemeAssignmentIsUpgraded(t *testing.T) {
	existing := readTestdata(t, "wezterm-previous-sf2.lua")
	const pointer = "/home/fighter/.config/sf2-theme/wezterm-current.lua"
	if got := classify(existing, pointer, testFormer); got != plan.Legacy {
		t.Errorf("classify = %q, want legacy", got)
	}
	if got, want := integrate(existing, pointer, testFormer), readTestdata(t, "wezterm-previous-sf2-integrated.lua"); got != want {
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
		writeFile(t, filepath.Join(dir, "wezterm", "sf2-theme", "wezterm-current.lua"), tc.pointer)
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
		"local sf2_current = \"" + dir + "/wezterm/sf2-theme/wezterm-current.lua\"\n" +
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
		if pointer := steps[2*len(themes)]; pointer.Path != dir+"/wezterm/sf2-theme/wezterm-current.lua" {
			t.Errorf("pointer step: %+v", pointer)
		}
		if slot && steps[len(steps)-1].Path != dir+"/wezterm/wezterm.lua" {
			t.Errorf("integration step: %+v", steps[len(steps)-1])
		}
	}
}

func currentID(t *testing.T, themes []catalog.Theme, configDir string) (plan.Inspection, string) {
	t.Helper()
	got, err := Inspect(themes, configDir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Current == nil {
		return got, ""
	}
	return got, got.Current.Metadata.SelectableID()
}

func TestTwoConfigDirTargetsKeepIndependentSelections(t *testing.T) {
	_, themes := loadTheme(t, "main")
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir+"/xdg")
	for target, id := range map[string]string{"a": "ryu", "b": "ken"} {
		theme, _ := loadTheme(t, id)
		steps, err := Plan(theme, themes, dir+"/"+target, true)
		if err != nil {
			t.Fatal(err)
		}
		for _, step := range steps {
			if !step.Remove {
				writeFile(t, step.Path, step.Content)
			}
		}
	}
	for target, want := range map[string]string{"a": "sf2-ryu", "b": "sf2-ken"} {
		if got, id := currentID(t, themes, dir+"/"+target); id != want || got.Ownership != plan.Managed {
			t.Errorf("%s: %s %+v, want %s", target, id, got, want)
		}
	}
	if _, err := os.Stat(dir + "/xdg/sf2-theme"); !os.IsNotExist(err) {
		t.Errorf("a target wrote the legacy shared pointer: %v", err)
	}
}

// legacyLua is the integration earlier releases wrote, naming the shared pointer under XDG_CONFIG_HOME.
func legacyLua(legacy string) string {
	return "local wezterm = require(\"wezterm\")\nlocal config = wezterm.config_builder()\n" +
		"-- was \"" + legacy + "\"\n" + "config.font_size = 13\n" +
		pycompat.RStrip(integrationSnippet(legacy, "config")) + "\nreturn config\n"
}

func TestALegacySharedPointerMigratesOnlyToTheTargetThatNamesIt(t *testing.T) {
	theme, themes := loadTheme(t, "main")
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir+"/xdg")
	legacy := dir + "/xdg/sf2-theme/wezterm-current.lua"
	writeFile(t, legacy, "-- sf2-themes: sf2-ryu\n")
	existing := legacyLua(legacy)
	writeFile(t, dir+"/owned/wezterm.lua", existing)
	writeFile(t, dir+"/other/wezterm.lua", "")
	writeFile(t, dir+"/commented/wezterm.lua", "-- \""+legacy+"\"\n")

	got, id := currentID(t, themes, dir+"/owned")
	if got.Ownership != plan.Managed || id != "sf2-ryu" {
		t.Errorf("owned: %s %+v", id, got)
	}
	for _, target := range []string{"other", "commented"} {
		if got, id := currentID(t, themes, dir+"/"+target); id != "" || got.Unreadable {
			t.Errorf("%s adopted the legacy pointer it does not name: %s %+v", target, id, got)
		}
	}

	steps, err := Plan(theme, themes, dir+"/owned", true)
	if err != nil {
		t.Fatal(err)
	}
	pointer := dir + "/owned/sf2-theme/wezterm-current.lua"
	lua := steps[len(steps)-1]
	if steps[len(steps)-2].Path != pointer || lua.Path != dir+"/owned/wezterm.lua" {
		t.Fatalf("steps end with %+v, %+v", steps[len(steps)-2], lua)
	}
	want := strings.Replace(existing, "local sf2_current = \""+legacy+"\"", "local sf2_current = \""+pointer+"\"", 1)
	if lua.Content != want {
		t.Errorf("migrated wezterm.lua:\n%s\nwant:\n%s", lua.Content, want)
	}
	for _, step := range steps {
		if step.Path == legacy {
			t.Errorf("planned a change to the legacy pointer: %+v", step)
		}
	}

	// While wezterm.lua still loads the legacy pointer, that pointer is the selection, even beside an own pointer.
	writeFile(t, pointer, "-- sf2-themes: sf2-ken\n")
	if _, id := currentID(t, themes, dir+"/owned"); id != "sf2-ryu" {
		t.Errorf("unmigrated: %s, want the legacy pointer's sf2-ryu", id)
	}
	// Once wezterm.lua loads the target's own pointer, the legacy one no longer decides its selection.
	writeFile(t, dir+"/owned/wezterm.lua", lua.Content)
	if _, id := currentID(t, themes, dir+"/owned"); id != "sf2-ken" {
		t.Errorf("own pointer: %s", id)
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

func TestPointerPathsAreUniquePerTargetAndNeverTheLegacyPointer(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir+"/home")
	t.Setenv("XDG_CONFIG_HOME", dir+"/xdg")
	t.Setenv("WEZTERM_CONFIG_DIR", "")
	t.Setenv("WEZTERM_CONFIG_FILE", "")
	digest := func(lua string) string {
		sum := sha256.Sum256([]byte(fsys.Canonical(lua)))
		return hex.EncodeToString(sum[:])[:12]
	}
	legacy := dir + "/xdg/sf2-theme/wezterm-current.lua"
	if err := os.MkdirAll(dir+"/xdg", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(dir+"/xdg", dir+"/alias"); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name, configDir, configFile, want string
	}{
		{"default", "", "", dir + "/xdg/wezterm/sf2-theme/wezterm-current.lua"},
		{"config dir", dir + "/a", "", dir + "/a/sf2-theme/wezterm-current.lua"},
		{"config dir at XDG_CONFIG_HOME", dir + "/xdg", "", dir + "/xdg/sf2-theme/wezterm-current-" + digest(dir+"/xdg/wezterm.lua") + ".lua"},
		{"config file one", "", dir + "/one.lua", dir + "/xdg/wezterm/sf2-theme/wezterm-current-" + digest(dir+"/one.lua") + ".lua"},
		{"config file two", "", dir + "/two.lua", dir + "/xdg/wezterm/sf2-theme/wezterm-current-" + digest(dir+"/two.lua") + ".lua"},
		{"config file named wezterm.lua", "", dir + "/b/wezterm.lua", dir + "/b/sf2-theme/wezterm-current.lua"},
	}
	seen := map[string]string{}
	for _, tc := range cases {
		t.Setenv("WEZTERM_CONFIG_FILE", tc.configFile)
		got, err := pointerPath(tc.configDir)
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want || got == legacy {
			t.Errorf("%s: %s, want %s", tc.name, got, tc.want)
		}
		if other, ok := seen[got]; ok {
			t.Errorf("%s shares its pointer with %s", tc.name, other)
		}
		seen[got] = tc.name
	}

	// Every spelling of one location is one target: a dot-dot or symlinked spelling of XDG_CONFIG_HOME never gets
	// the legacy pointer, and --config-dir X and WEZTERM_CONFIG_FILE=X/wezterm.lua get the same pointer.
	t.Setenv("WEZTERM_CONFIG_FILE", "")
	for _, alias := range []string{dir + "/xdg/../xdg", dir + "/alias"} {
		got, err := pointerPath(alias)
		if err != nil {
			t.Fatal(err)
		}
		if fsys.SamePath(got, legacy) || pycompat.Name(got) != "wezterm-current-"+digest(dir+"/xdg/wezterm.lua")+".lua" {
			t.Errorf("%s: %s", alias, got)
		}
	}
	viaDir, err := pointerPath(dir + "/b")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("WEZTERM_CONFIG_FILE", dir+"/b/wezterm.lua")
	if viaFile, err := pointerPath(""); err != nil || viaFile != viaDir {
		t.Errorf("WEZTERM_CONFIG_FILE: %s, --config-dir: %s (%v)", viaFile, viaDir, err)
	}
}

func TestCommentsAndStringsNeverCountAsTheIntegration(t *testing.T) {
	const legacy = "/tmp/xdg-legacy/sf2-theme/wezterm-current.lua"
	const builder = "local wezterm = require(\"wezterm\")\nlocal config = wezterm.config_builder()\n"
	inactive := "local sf2_current = \"" + legacy + "\"\nlocal sf2_scheme = dofile(sf2_current)\n" +
		"config.color_scheme = sf2_scheme\nlocal TERM_THEME = \"untouched\"\nreturn config\n"
	cases := map[string]string{
		"level-1 long comment": "--[=[\n" + inactive + "]=]\n",
		"long comment":         "--[[\n" + strings.ReplaceAll(inactive, "]]", "") + "]]\n",
		"long string":          "local example = [==[\n" + inactive + "]==]\n",
		"inline comments":      "config.font_size = 12 -- local sf2_current = \"" + legacy + "\"\n-- dofile(sf2_current) TERM_THEME\n",
	}
	for name, block := range cases {
		existing := builder + block + "return config\n"
		if _, ok := assignedPointer(existing); ok {
			t.Errorf("%s: found an assignment", name)
		}
		if got := classify(existing, testPointer, []string{legacy}); got != plan.Absent {
			t.Errorf("%s: classify = %q, want absent", name, got)
		}
		if got, want := integrate(existing, testPointer, []string{legacy}), integrated("config", block); got != want {
			t.Errorf("%s: integrate =\n%s\nwant\n%s", name, got, want)
		}
	}

	// An active integration beside an inactive example moves only the active literal.
	active := "local sf2_current = '" + legacy + "' -- local sf2_current = \"" + legacy + "\"\n" +
		"wezterm.add_to_config_reload_watch_list(sf2_current)\nlocal sf2_scheme = dofile(sf2_current)\n" +
		"config.color_scheme = sf2_scheme\nlocal sf2_env = config.set_environment_variables or {}\n" + termThemeAssign + "\n" +
		"config.set_environment_variables = sf2_env\n"
	existing := builder + cases["level-1 long comment"] + active + "return config\n"
	if named, ok := assignedPointer(existing); !ok || named != legacy {
		t.Fatalf("assignedPointer = %q, %v", named, ok)
	}
	want := strings.Replace(existing, "local sf2_current = '"+legacy+"'", "local sf2_current = \""+testPointer+"\"", 1)
	if got := integrate(existing, testPointer, []string{legacy}); got != want {
		t.Errorf("migrated:\n%s\nwant\n%s", got, want)
	}

	// An assignment the code never loads, or a second one, proves nothing.
	for name, tail := range map[string]string{
		"never loaded":     "local sf2_current = \"" + legacy + "\"\n",
		"reassigned":       active + "sf2_current = \"/elsewhere/wezterm-current.lua\"\n",
		"part of a string": strings.Replace(active, "' --", "' .. \"x\" --", 1),
	} {
		if _, ok := assignedPointer(builder + tail + "return config\n"); ok {
			t.Errorf("%s: found an assignment", name)
		}
	}
}

func TestBlankInactiveKeepsOffsetsAndLines(t *testing.T) {
	source := "a = 1 --[[ x\n\ry\vz\f]] b = \"--[[\" -- c\n[=[\n]]\n]=] d"
	want := "a = 1       \n\r \v \f   b = \"--[[\"     \n   \n  \n    d"
	if got := blankInactive(source); got != want {
		t.Errorf("blanked %q\nwant    %q", got, want)
	}
	if len(pycompat.SplitLines(want)) != len(pycompat.SplitLines(source)) {
		t.Error("lines moved")
	}
}
