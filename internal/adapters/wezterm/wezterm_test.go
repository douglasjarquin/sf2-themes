package wezterm

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
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
	const unguarded = "local sf2_current = \"/tmp/xdg/sf2-theme/wezterm-current.lua\"\n" +
		"wezterm.add_to_config_reload_watch_list(sf2_current)\nconfig.color_scheme = dofile(sf2_current)\n"
	full := strings.TrimSuffix(strings.TrimPrefix(integrated("config", ""), builder), "return config\n")
	cases := []struct {
		name      string
		existing  string
		ownership plan.Ownership
		want      string
	}{
		{"blank", "  \n", plan.Absent, integrated("config", "")},
		{"managed", integrated("config", "config.font_size = 13\n"), plan.Managed, integrated("config", "config.font_size = 13\n")},
		{"unguarded form is replaced as a whole", builder + unguarded + "-- TERM_THEME\nreturn config\n", plan.Legacy,
			builder + full + "-- TERM_THEME\nreturn config\n"},
		{"literal form is replaced as a whole", builder + "config.color_scheme = \"street-fighter-2\"\nconfig.font_size = 13\nreturn config\n",
			plan.Legacy, builder + full + "config.font_size = 13\nreturn config\n"},
		{"adoptable builder", builder + "-- note\nconfig.font_size = 13\nconfig.font = wezterm.font(\"Monaspace Neon\")\nreturn config\n",
			plan.Foreign, integrated("config", "-- note\nconfig.font_size = 13\nconfig.font = wezterm.font(\"Monaspace Neon\")\n")},
		{"discovered builder name", "local wezterm = require(\"wezterm\")\nlocal cfg = wezterm.config_builder()\nreturn cfg\n", plan.Foreign, integrated("cfg", "")},
		{"foreign scheme is never removed", builder + "config.color_scheme = \"Catppuccin Mocha\"\nreturn config\n", plan.Foreign, ""},
		{"builder name mismatch", builder + "return cfg\n", plan.UnknownShape, ""},
		{"plain table", "return {\n  font_size = 13,\n}\n", plan.UnknownShape, ""},
		{"a function is more than sf2 verifies for adoption", builder + "local function f() return 1 end\nreturn config\n", plan.UnknownShape, ""},
		{"invalid Lua", builder + "local invalid =\nreturn config\n", plan.Broken, ""},
		{"unterminated string", builder + "config.font = \"Monaspace\nreturn config\n", plan.Broken, ""},
	}
	for _, tc := range cases {
		if got := classify(tc.existing, testPointer); got != tc.ownership {
			t.Errorf("%s: classify = %q, want %q", tc.name, got, tc.ownership)
		}
		got, err := integrate(tc.existing, testPointer, "wezterm.lua")
		if tc.want == "" {
			if err == nil {
				t.Errorf("%s: integrated a config sf2 must not edit:\n%s", tc.name, got)
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
	if got, err := integrate(existing, pointer, "wezterm.lua"); err != nil || got != readTestdata(t, "wezterm-previous-sf2-integrated.lua") {
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
	want := "Add this integration to " + dir + "/wezterm/wezterm.lua at the top level, after config is created and " +
		"before the file returns it:\n\n" +
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
	pointerOf := func(configDir, configFile, configRoot string) (string, error) {
		t.Helper()
		t.Setenv("WEZTERM_CONFIG_FILE", configFile)
		t.Setenv("WEZTERM_CONFIG_DIR", configRoot)
		return pointerPath(configDir)
	}
	digest := func(lua string) string {
		sum := sha256.Sum256([]byte(lua))
		return hex.EncodeToString(sum[:])[:12]
	}
	writeFile(t, dir+"/a/wezterm.lua", "")
	writeFile(t, dir+"/xdg/wezterm/.keep", "")
	if err := os.Symlink(dir+"/xdg", dir+"/alias"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(dir+"/a/wezterm.lua", dir+"/linked.lua"); err != nil {
		t.Fatal(err)
	}

	// The pointer sits beside the file the filesystem opens, named after its path.
	if got, err := pointerOf("", "", ""); err != nil || got != dir+"/xdg/wezterm/sf2-theme/wezterm-current-"+digest(dir+"/xdg/wezterm/wezterm.lua")+".lua" {
		t.Errorf("default: %s %v", got, err)
	}
	want, err := pointerOf(dir+"/a", "", "")
	if err != nil {
		t.Fatal(err)
	}
	for name, spelling := range map[string][3]string{
		"WEZTERM_CONFIG_FILE":        {"", dir + "/a/wezterm.lua", ""},
		"followed file symlink":      {"", dir + "/linked.lua", ""},
		"another WEZTERM_CONFIG_DIR": {"", dir + "/a/wezterm.lua", dir + "/elsewhere"},
	} {
		if got, err := pointerOf(spelling[0], spelling[1], spelling[2]); err != nil || got != want {
			t.Errorf("%s: %s %v, want %s", name, got, err, want)
		}
	}
	if got, err := pointerOf(dir+"/alias", "", ""); err != nil || got != dir+"/xdg/sf2-theme/wezterm-current-"+digest(dir+"/xdg/wezterm.lua")+".lua" {
		t.Errorf("symlinked XDG_CONFIG_HOME: %s %v", got, err)
	}
	// A spelling that is not the path on disk is refused, never reconciled.
	for _, spelling := range [][3]string{
		{dir + "/a/../a", "", ""},
		{"", dir + "/absent/../profile.lua", ""},
	} {
		if got, err := pointerOf(spelling[0], spelling[1], spelling[2]); err == nil {
			t.Errorf("%v: accepted as %s", spelling, got)
		}
	}
	t.Setenv("WEZTERM_CONFIG_DIR", dir+"/x/../colors")
	if got, err := colorsDir(""); err == nil {
		t.Errorf("colors through `..` accepted as %s", got)
	}
	// A path holding a control character is refused before any Lua could carry it.
	writeFile(t, dir+"/line\nbreak/profile.lua", "")
	if got, err := pointerOf("", dir+"/line\nbreak/profile.lua", ""); err == nil {
		t.Errorf("control character accepted: %q", got)
	}
}

func TestResolveFollowsTheFilesystem(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir+"/real/file.lua", "")
	writeFile(t, dir+"/other/nested/x", "")
	link := func(target, at string) {
		t.Helper()
		if err := os.Symlink(target, at); err != nil {
			t.Fatal(err)
		}
	}
	link(dir+"/other/nested", dir+"/real/link")
	link("../../real/file.lua", dir+"/other/nested/relative.lua")
	link("absent.lua", dir+"/real/dangling.lua")
	cases := map[string]string{
		dir + "/real/file.lua":          dir + "/real/file.lua",
		dir + "/real/./missing/new.lua": dir + "/real/missing/new.lua",
		dir + "/real/link/x":            dir + "/other/nested/x",
		dir + "/real/link/relative.lua": dir + "/real/file.lua",
		dir + "/real/dangling.lua":      dir + "/real/absent.lua",
	}
	for in, want := range cases {
		if got, err := fsys.Resolve(in); err != nil || got != want {
			t.Errorf("Resolve(%s) = %s, %v; want %s", in, got, err, want)
		}
	}
	if !fsys.SamePath(dir+"/real/link/x", dir+"/other/nested/x") {
		t.Error("a symlinked spelling compared distinct")
	}
	// A symlink whose target leaves a missing directory with `..` opens nothing; a lexical join would name file.lua.
	link("missing/../file.lua", dir+"/real/escape.lua")
	link(dir+"/real/escape.lua", dir+"/real/escape-again.lua")
	link(dir+"/real/missing/../link", dir+"/real/escape-dir")
	refused := []string{
		dir + "/real/link/../file.lua", dir + "/absent/../real/file.lua", dir + "/real/../real/file.lua",
		dir + "/real/escape.lua", dir + "/real/escape-again.lua", dir + "/real/escape-dir/x",
	}
	for _, path := range refused {
		if got, err := fsys.Resolve(path); err == nil {
			t.Errorf("Resolve(%s) accepted as %s", path, got)
		}
	}
	// Another letter case or Unicode normalization of an existing name is refused where the volume would open it,
	// whether the path names it or a symlink it follows does.
	writeFile(t, dir+"/caf\u00e9.lua", "")
	for i, alias := range []string{dir + "/REAL/file.lua", dir + "/real/FILE.LUA", dir + "/cafe\u0301.lua"} {
		if _, err := os.Stat(alias); err != nil {
			continue
		}
		if _, err := fsys.Resolve(alias); err == nil {
			t.Errorf("Resolve(%q) accepted another spelling of an existing file", alias)
		}
		through := fmt.Sprintf("%s/alias-%d.lua", dir, i)
		link(alias, through)
		if _, err := fsys.Resolve(through); err == nil {
			t.Errorf("Resolve(%q) accepted a symlink to another spelling, %q", through, alias)
		}
	}
}

const testBuilder = "local wezterm = require(\"wezterm\")\nlocal config = wezterm.config_builder()\n"

func TestOnlySF2sOwnFormsAtTheTopLevelAreRecognized(t *testing.T) {
	const legacy = "/tmp/xdg-legacy/sf2-theme/wezterm-current.lua"
	marker := "local sf2_current = \"" + testPointer + "\"\n" + watchLine + "\n" + schemeLine + "\n"
	snippet := integrationSnippet(testPointer, "config")
	notManaged := map[string]string{
		// A pasted three-line marker is no longer recognized: only whole generated forms are sf2's.
		"adapted pasted marker":          "local wezterm = require(\"wezterm\")\n" + marker + "return { color_scheme = sf2_scheme }\n",
		"marker read as a field":         testBuilder + marker + "local unused = other.sf2_scheme\nreturn config\n",
		"block inside a function":        testBuilder + "local function later()\n" + snippet + "end\nreturn config\n",
		"block inside a long comment":    testBuilder + "--[=[\n" + snippet + "]=]\nreturn config\n",
		"block inside a long string":     testBuilder + "local doc = [==[\n" + snippet + "]==]\nreturn config\n",
		"dormant one-line function":      testBuilder + "local function inactive() local sf2_current = \"" + legacy + "\"; return dofile(sf2_current) end\nlocal TERM_THEME = \"untouched\"\nreturn config\n",
		"two integrations":               testBuilder + snippet + snippet + "return config\n",
		"another scheme after the block": strings.Replace(integrated("config", ""), "return config", "config.color_scheme = \"Builtin\"\nreturn config", 1),
		"a non-pointer file":             strings.Replace(integrated("config", ""), testPointer, "/tmp/elsewhere/theme.lua", 1),
		"single-quoted integration":      testBuilder + strings.Replace(integrationSnippet(legacy, "config"), `"`+legacy+`"`, `'`+legacy+`'`, 1) + "return config\n",
		"block before its builder":       "local wezterm = require(\"wezterm\")\n" + snippet + "local config = wezterm.config_builder()\nreturn config\n",
		"no final return of its builder": testBuilder + snippet + "return { }\n",
	}
	for name, existing := range notManaged {
		a := analyze(existing, testPointer, "wezterm.lua")
		if a.ownership == plan.Managed || a.ownership == plan.Legacy || a.selector != "" {
			t.Errorf("%s: %q with selector %q", name, a.ownership, a.selector)
		}
		if got, err := integrate(existing, testPointer, "wezterm.lua"); err == nil {
			t.Errorf("%s: integrated\n%s", name, got)
		}
	}
	// The refusal says what to do by hand, including removing an older loader.
	a := analyze(notManaged["single-quoted integration"], testPointer, "wezterm.lua")
	if !strings.Contains(a.manual, snippet) || !strings.Contains(a.manual, "older sf2-themes loader") || !strings.Contains(a.manual, "other color_scheme") {
		t.Errorf("manual steps:\n%s", a.manual)
	}
}

func TestMalformedLuaIsNeverRecognizedOrEdited(t *testing.T) {
	for name, block := range map[string]string{
		"incomplete local":         "local invalid =\n",
		"escape above 2^31-1":      "local s = \"\\u{80000000}\"\n",
		"long escape above 2^31-1": "local s = \"\\u{00FFFFFFFF}\"\n",
		"unterminated string":      "config.font = \"Monaspace\n",
		"unterminated comment":     "--[[ never closed\n",
		"unbalanced block":         "if true then\n",
		"bad escape":               "local s = \"\\q\"\n",
		"bad number":               "local n = 3..2\n",
		"stray character":          "local n = 1 $\n",
		"expression statement":     "config.font_size\n",
		"break outside a loop":     "break\n",
		"vararg in a function":     "local function f() return ... end\n",
		"goto is unverified":       "goto done\n::done::\n",
		"attribute is unverified":  "local x <const> = 1\n",
	} {
		existing := testBuilder + block + integrationSnippet(testPointer, "config") + "return config\n"
		if got := classify(existing, testPointer); got != plan.Broken {
			t.Errorf("%s: classify = %q, want broken", name, got)
		}
	}
	// Valid Lua the reviewers used as lexical traps parses, and comments or strings never count.
	for name, block := range map[string]string{
		"escaped quote":         "local note = \"say \\\"config.color_scheme = 1\\\"\"\n",
		"z continuation":        "local note = \"a\\z\n  config.color_scheme = 'x'\"\n",
		"nested lookalike":      "--[==[\n]=] ]] local sf2_current = 1\n]==]\n",
		"escapes":               "local s = \"\\65\\x41\\u{41}\\255\\u{000000041}\\u{7FFFFFFF}\"\n",
		"numbers":               "local n = 0x1p4 + 1e-3 + .5 + 3 // 2 + (1 << 2) ~ 1\n",
		"method and call forms": "local s = (\"x\"):rep(2) .. tostring{1} .. ('y'):upper()\n",
		"control flow":          "for i = 1, 3 do if i == 2 then break end end\nwhile false do end\nrepeat local x = 1 until x\n",
		"functions":             "local function f(a, ...) return select('#', ...) end\nlocal t = { f = function() end, [1] = 2; x = 3, }\n",
	} {
		existing := testBuilder + block + integrationSnippet(testPointer, "config") + "return config\n"
		if _, err := parseLua(existing); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestAFormerPointerIsReplacedWithTheWholeBlock(t *testing.T) {
	const legacy = "/tmp/xdg-legacy/sf2-theme/wezterm-current.lua"
	indented := strings.ReplaceAll(strings.TrimSuffix(integrationSnippet(legacy, "config"), "\n"), "\n", "\n  ")
	for name, existing := range map[string]string{
		"full form":     testBuilder + "--[=[ " + legacy + " ]=]\n" + integrationSnippet(legacy, "config") + "return config\n",
		"indented form": testBuilder + "do\nend\n  " + indented + "\nreturn config\n",
	} {
		a := analyze(existing, testPointer, "wezterm.lua")
		if a.ownership != plan.Managed || a.selector != legacy {
			t.Errorf("%s: %q selector %q", name, a.ownership, a.selector)
		}
		want := strings.Replace(existing, `local sf2_current = "`+legacy+`"`, `local sf2_current = "`+testPointer+`"`, 1)
		if got, err := integrate(existing, testPointer, "wezterm.lua"); err != nil || got != want {
			t.Errorf("%s: integrate = %v\n%s\nwant\n%s", name, err, got, want)
		}
	}
}

func TestAdoptionInsertsOnlyIntoAFileSf2CanVerify(t *testing.T) {
	for name, existing := range map[string]string{
		"closer then two statements": testBuilder + "--[=[\nkeep this note\n]=] config.color_scheme = \"Builtin User Scheme\"; config.font_size = 21\nreturn config\n",
		"user scheme":                testBuilder + "config.color_scheme = \"Builtin\" -- my favourite\nreturn config\n",
		"nested scheme":              testBuilder + "if wezterm.gui then\n  config.color_scheme = \"A\"\nend\nreturn config\n",
		"table value":                testBuilder + "config.keys = {}\nreturn config\n",
		"trailing comment":           testBuilder + "config.font_size = 12 -- big\nreturn config\n",
		"long comment":               testBuilder + "--[[ note ]]\nreturn config\n",
	} {
		if got, err := integrate(existing, testPointer, "wezterm.lua"); err == nil {
			t.Errorf("%s: integrated\n%s", name, got)
		}
	}
}
