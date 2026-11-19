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
	// The snippet sf2 prints for a shape it cannot edit, pasted by hand and adapted to a table config.
	const pasted = "local wezterm = require(\"wezterm\")\nlocal sf2_current = \"/tmp/xdg/sf2-theme/wezterm-current.lua\"\n" +
		"wezterm.add_to_config_reload_watch_list(sf2_current)\nlocal sf2_scheme = dofile(sf2_current)\n" +
		"return { color_scheme = sf2_scheme }\n"
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
		{"foreign call", builder + "config.color_scheme = scheme_for(wezterm.gui.get_appearance())\nreturn config\n", plan.Foreign, integrated("config", "")},
		{"legacy literal scheme", builder + "config.color_scheme = \"street-fighter-2\"\nreturn config\n", plan.Legacy, integrated("config", "")},
		{"builder name mismatch", builder + "return cfg\n", plan.UnknownShape, ""},
		{"plain table", "return {\n  font_size = 13,\n}\n", plan.UnknownShape, ""},
		{"discovered builder name", "local wezterm = require(\"wezterm\")\nlocal cfg = wezterm.config_builder()\nreturn cfg\n", plan.Absent, integrated("cfg", "")},
		{"adapted pasted marker", pasted, plan.Managed, pasted},
	}
	for _, tc := range cases {
		if got := classify(tc.existing, testPointer); got != tc.ownership {
			t.Errorf("%s: classify = %q, want %q", tc.name, got, tc.ownership)
		}
		got, err := integrate(tc.existing, testPointer)
		if tc.want == "" {
			if err == nil {
				t.Errorf("%s: integrated a shape sf2 cannot edit:\n%s", tc.name, got)
			}
		} else if err != nil || got != tc.want {
			t.Errorf("%s: integrate = %v\n%s\nwant\n%s", tc.name, err, got, tc.want)
		}
	}
}
func TestPreviousSF2SchemeAssignmentIsUpgraded(t *testing.T) {
	existing := readTestdata(t, "wezterm-previous-sf2.lua")
	const pointer = "/home/fighter/.config/sf2-theme/wezterm-current.lua"
	if got := classify(existing, pointer); got != plan.Legacy {
		t.Errorf("classify = %q, want legacy", got)
	}
	if got, err := integrate(existing, pointer); err != nil || got != readTestdata(t, "wezterm-previous-sf2-integrated.lua") {
		t.Errorf("integrate = %v\n%s", err, got)
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
		writeFile(t, ownPointer(t, filepath.Join(dir, "wezterm")), tc.pointer)
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
		integrationSnippet(ownPointer(t, dir+"/wezterm"), "config")
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
		if pointer := steps[2*len(themes)]; pointer.Path != ownPointer(t, dir+"/wezterm") {
			t.Errorf("pointer step: %+v", pointer)
		}
		if slot && steps[len(steps)-1].Path != dir+"/wezterm/wezterm.lua" {
			t.Errorf("integration step: %+v", steps[len(steps)-1])
		}
	}
}

// ownPointer is the target's own pointer.
func ownPointer(t *testing.T, configDir string) string {
	t.Helper()
	pointer, err := pointerPath(configDir)
	if err != nil {
		t.Fatal(err)
	}
	return pointer
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
	pointer := ownPointer(t, dir+"/owned")
	if inspection, _ := currentID(t, themes, dir+"/owned"); inspection.Released != legacy {
		t.Errorf("released %q, want the legacy pointer", inspection.Released)
	}
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

func TestPointerIdentityIsTheFileTheFilesystemOpens(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", dir+"/home")
	t.Setenv("XDG_CONFIG_HOME", dir+"/xdg")
	t.Setenv("WEZTERM_CONFIG_DIR", "")
	t.Setenv("WEZTERM_CONFIG_FILE", "")
	pointerOf := func(configDir, configFile, configRoot string) string {
		t.Helper()
		t.Setenv("WEZTERM_CONFIG_FILE", configFile)
		t.Setenv("WEZTERM_CONFIG_DIR", configRoot)
		return ownPointer(t, configDir)
	}
	digest := func(lua string) string {
		sum := sha256.Sum256([]byte(lua))
		return hex.EncodeToString(sum[:])[:12]
	}
	for _, d := range []string{"xdg/wezterm", "a", "root", "other/nested"} {
		if err := os.MkdirAll(dir+"/"+d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeFile(t, dir+"/a/wezterm.lua", "")
	if err := os.Symlink(dir+"/other/nested", dir+"/root/link"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(dir+"/xdg", dir+"/alias"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(dir+"/a/wezterm.lua", dir+"/root/linked.lua"); err != nil {
		t.Fatal(err)
	}

	// The pointer sits beside the file, named after its canonical path, never a name earlier releases shared.
	if got, want := pointerOf("", "", ""), dir+"/xdg/wezterm/sf2-theme/wezterm-current-"+digest(dir+"/xdg/wezterm/wezterm.lua")+".lua"; got != want {
		t.Errorf("default: %s, want %s", got, want)
	}
	// Spellings of one file share its pointer.
	same := map[string]string{
		"--config-dir":               pointerOf(dir+"/a", "", ""),
		"WEZTERM_CONFIG_FILE":        pointerOf("", dir+"/a/wezterm.lua", ""),
		"dot-dot":                    pointerOf(dir+"/root/../a", "", ""),
		"followed file symlink":      pointerOf("", dir+"/root/linked.lua", ""),
		"another WEZTERM_CONFIG_DIR": pointerOf("", dir+"/a/wezterm.lua", dir+"/elsewhere"),
	}
	for name, got := range same {
		if got != same["--config-dir"] {
			t.Errorf("%s: %s, want %s", name, got, same["--config-dir"])
		}
	}
	// The symlinked XDG_CONFIG_HOME and its real spelling are one target.
	if a, b := pointerOf(dir+"/alias", "", ""), pointerOf(dir+"/xdg", "", ""); a != b || pycompat.Name(a) == "wezterm-current.lua" {
		t.Errorf("alias %s, real %s", a, b)
	}
	// Distinct files never share one, including `link/..`, which opens the link target's parent.
	viaLink := pointerOf("", dir+"/root/link/../profile.lua", "")
	if viaLink != dir+"/other/sf2-theme/wezterm-current-"+digest(dir+"/other/profile.lua")+".lua" {
		t.Errorf("link/..: %s", viaLink)
	}
	if beside := pointerOf("", dir+"/root/profile.lua", ""); beside == viaLink {
		t.Errorf("distinct files share %s", beside)
	}
}

func TestCanonicalFollowsTheFilesystem(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir+"/real/file.lua", "")
	writeFile(t, dir+"/other/nested/x", "")
	if err := os.Symlink(dir+"/other/nested", dir+"/real/link"); err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		dir + "/real/file.lua":                 dir + "/real/file.lua",
		dir + "/real/./missing/new.lua":        dir + "/real/missing/new.lua",
		dir + "/real/link/../file.lua":         dir + "/other/file.lua",
		dir + "/real/link/x":                   dir + "/other/nested/x",
		dir + "/real/link/../../real/file.lua": dir + "/real/file.lua",
	}
	for in, want := range cases {
		if got := fsys.Canonical(in); got != want {
			t.Errorf("Canonical(%s) = %s, want %s", in, got, want)
		}
	}
	if fsys.SamePath(dir+"/real/link/../file.lua", dir+"/real/file.lua") {
		t.Error("link/.. compared equal to a distinct file")
	}
	if !fsys.SamePath(dir+"/real/link/x", dir+"/other/nested/x") {
		t.Error("a symlinked spelling compared distinct")
	}
	// On a case-insensitive volume, a different letter case opens the same file and gets its on-disk spelling.
	if info, err := os.Stat(dir + "/REAL/FILE.LUA"); err == nil {
		real, _ := os.Stat(dir + "/real/file.lua")
		if os.SameFile(info, real) {
			if got := fsys.Canonical(dir + "/REAL/FILE.LUA"); got != dir+"/real/file.lua" {
				t.Errorf("case: %s", got)
			}
			if !fsys.SamePath(dir+"/REAL", dir+"/real") {
				t.Error("case spellings compared distinct")
			}
		}
	}
}

const testBuilder = "local wezterm = require(\"wezterm\")\nlocal config = wezterm.config_builder()\n"

// marker is the three lines sf2 recognizes as an integration a person pasted and adapted.
func marker(pointer string) string {
	return "local sf2_current = \"" + pointer + "\"\n" + watchLine + "\n" + schemeLine + "\n"
}

func TestOnlyGeneratedIntegrationAtTheTopLevelIsRecognized(t *testing.T) {
	const legacy = "/tmp/xdg-legacy/sf2-theme/wezterm-current.lua"
	notManaged := map[string]string{
		"marker inside a function": testBuilder + "local function later()\n" + marker(testPointer) +
			"  config.color_scheme = sf2_scheme\nend\nreturn config\n",
		"marker inside a long comment":  testBuilder + "--[=[\n" + marker(testPointer) + "config.color_scheme = sf2_scheme\n]=]\nreturn config\n",
		"marker inside a long string":   testBuilder + "local doc = [==[\n" + marker(testPointer) + "]==]\nreturn config\n",
		"marker with sf2_scheme unused": testBuilder + marker(testPointer) + "return config\n",
		"marker with sf2_scheme only in a comment": testBuilder + marker(testPointer) +
			"-- config.color_scheme = sf2_scheme\nreturn config\n",
		"marker with sf2_scheme reassigned": testBuilder + marker(testPointer) +
			"sf2_scheme = \"Builtin\"\nconfig.color_scheme = sf2_scheme\nreturn config\n",
		"dormant one-line function": testBuilder + "local function inactive() local sf2_current = \"" + legacy +
			"\"; return dofile(sf2_current) end\nlocal TERM_THEME = \"untouched\"\nreturn config\n",
		"unterminated string":               testBuilder + "config.font = \"Monaspace\nreturn config\n",
		"unterminated long comment":         testBuilder + "--[[ never closed\nreturn config\n",
		"unbalanced block":                  testBuilder + "if true then\nreturn config\n",
		"two integrations":                  testBuilder + integrationSnippet(testPointer, "config") + integrationSnippet(testPointer, "config") + "return config\n",
		"integration of a non-pointer file": strings.Replace(integrated("config", ""), testPointer, "/tmp/elsewhere/theme.lua", 1),
	}
	for name, existing := range notManaged {
		if got := classify(existing, testPointer); got == plan.Managed || got == plan.Legacy {
			t.Errorf("%s: classify = %q", name, got)
		}
		if _, ok := assignedPointer(existing, testPointer); ok {
			t.Errorf("%s: reported a selector", name)
		}
	}

	// Comments and strings never count, whatever they hold, and the integration goes in around them untouched.
	inactive := "local sf2_current = \"" + legacy + "\"\nlocal sf2_scheme = dofile(sf2_current)\n" +
		"config.color_scheme = sf2_scheme\nlocal TERM_THEME = \"untouched\"\n"
	for name, block := range map[string]string{
		"long comment":         "--[[\n" + inactive + "]]\n",
		"level-1 long comment": "--[=[\n" + inactive + "]] ]=]\n",
		"nested lookalike":     "--[==[\n]=] ]] " + inactive + "]==]\n",
		"long string":          "local example = [==[\n" + inactive + "]==]\n",
		"inline comments":      "config.font_size = 12 -- local sf2_current = \"" + legacy + "\"\n-- dofile(sf2_current) TERM_THEME\n",
		"escaped quote":        "local note = \"say \\\"config.color_scheme = 1\\\"\"\n",
		"z continuation":       "local note = \"a\\z\n  config.color_scheme = 'x'\"\n",
		"backslash line break": "local note = \"a\\\nconfig.color_scheme = 'x'\"\n",
	} {
		existing := testBuilder + block + "return config\n"
		if got := classify(existing, testPointer); got != plan.Absent {
			t.Errorf("%s: classify = %q, want absent", name, got)
		}
		if got, err := integrate(existing, testPointer); err != nil || got != integrated("config", block) {
			t.Errorf("%s: integrate = %v\n%s", name, err, got)
		}
	}
}

func TestAFormerPointerMovesByRewritingOnlyItsLiteral(t *testing.T) {
	const legacy = "/tmp/xdg-legacy/sf2-theme/wezterm-current.lua"
	for name, existing := range map[string]string{
		"full block":     testBuilder + "--[=[ " + legacy + " ]=]\n" + integrationSnippet(legacy, "config") + "return config\n",
		"adapted marker": testBuilder + marker(legacy) + "config.color_scheme = sf2_scheme -- mine\nreturn config\n",
		"indented block": testBuilder + "do\nend\n  " + strings.ReplaceAll(strings.TrimSuffix(integrationSnippet(legacy, "config"), "\n"), "\n", "\n  ") + "\nreturn config\n",
	} {
		if named, ok := assignedPointer(existing, testPointer); !ok || named != legacy {
			t.Errorf("%s: selector %q, %v", name, named, ok)
		}
		want := strings.Replace(existing, `local sf2_current = "`+legacy+`"`, `local sf2_current = "`+testPointer+`"`, 1)
		if got, err := integrate(existing, testPointer); err != nil || got != want {
			t.Errorf("%s: integrate = %v\n%s\nwant\n%s", name, err, got, want)
		}
	}
}

func TestAdoptionKeepsNeighbouringTokensOrRefuses(t *testing.T) {
	// The assignment shares its line with a long comment's closer: only the statement goes.
	closer := testBuilder + "--[=[\nkeep this note\n]=] config.color_scheme = \"Builtin User Scheme\"\nreturn config\n"
	got, err := integrate(closer, testPointer)
	if want := integrated("config", "--[=[\nkeep this note\n]=]\n"); err != nil || got != want {
		t.Errorf("closer: %v\n%s\nwant\n%s", err, got, want)
	}
	trailing := testBuilder + "config.color_scheme = \"Builtin\" -- my favourite\nreturn config\n"
	if got, err := integrate(trailing, testPointer); err != nil || got != integrated("config", " -- my favourite\n") {
		t.Errorf("trailing comment: %v\n%s", err, got)
	}
	// Assignments sf2 cannot remove without guessing are refused, not half-edited.
	for name, existing := range map[string]string{
		"nested":           testBuilder + "if wezterm.gui then\n  config.color_scheme = \"A\"\nend\nreturn config\n",
		"continued":        testBuilder + "config.color_scheme = a\n  or \"B\"\nreturn config\n",
		"open expression":  testBuilder + "config.color_scheme = pick(\n  \"A\")\nreturn config\n",
		"second statement": testBuilder + "config.font_size = 12; config.color_scheme = \"A\"\nreturn config\n",
		"indexed":          testBuilder + "config[\"color_scheme\"] = \"A\"\nreturn config\n",
	} {
		if got := classify(existing, testPointer); got != plan.Foreign {
			t.Errorf("%s: classify = %q, want foreign", name, got)
		}
		if got, err := integrate(existing, testPointer); err == nil {
			t.Errorf("%s: integrated\n%s", name, got)
		}
	}
	// A single-quoted integration was never generated by sf2: it is the user's own scheme.
	single := testBuilder + strings.Replace(integrationSnippet("/tmp/x/sf2-theme/wezterm-current.lua", "config"), `"/tmp/x/sf2-theme/wezterm-current.lua"`, `'/tmp/x/sf2-theme/wezterm-current.lua'`, 1) + "return config\n"
	if got := classify(single, testPointer); got != plan.Foreign {
		t.Errorf("single quotes: %q", got)
	}
}
