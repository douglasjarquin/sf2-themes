package catalog

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/douglasjarquin/sf2-themes/v2/themes"
)

// The expected messages below were recorded from the Python reference (src/sf2_theme parse_theme, and the
// standalone's embedded catalog load) on Python 3.11, applying the same edits to the same TOML text.

func catalogFile(t *testing.T, name string) string {
	t.Helper()
	data, err := fs.ReadFile(themes.FS, name)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func replaceLine(text, pattern, replacement string) string {
	return regexp.MustCompile(`(?m)`+pattern).ReplaceAllLiteralString(text, replacement)
}

func TestParseThemeRejectsMalformedThemesWithTheReferenceMessage(t *testing.T) {
	ryu := catalogFile(t, "characters/ryu.toml")
	cases := []struct {
		name, text, want string
	}{
		{"unknown ui key", strings.Replace(ryu, "[ui]\n", "[ui]\npanel_bg = \"#123456\"\n", 1), "unknown malformed.toml.ui keys: panel_bg"},
		{"unknown top-level key", strings.Replace(ryu, "[ui]\n", "[extra]\n[ui]\n", 1), "malformed.toml: unknown top-level keys: extra"},
		{"unknown meta keys", strings.Replace(ryu, "[meta]\n", "[meta]\nzeta = 1\nalpha = 2\n", 1), "malformed.toml: unknown meta keys: alpha, zeta"},
		{"bad hex", strings.Replace(ryu, `background = "#`, `background = "#GG`, 1), "color must be #RRGGBB, got '#GG141a23'"},
		{"non-hex digit", strings.Replace(ryu, `background = "#141a23"`, `background = "#141A2Z"`, 1), "color must be #RRGGBB, got '#141A2Z'"},
		{"missing ui field", replaceLine(ryu, `^accent_secondary = .*\n`, ""), "missing malformed.toml.ui.accent_secondary"},
		{"missing meta name", replaceLine(ryu, `^name = .*\n`, ""), "missing malformed.toml.meta.name"},
		{"blank stage", replaceLine(ryu, `^stage = .*$`, `stage = " \t "`), "malformed.toml.meta.stage must not be empty"},
		{"bad era", strings.Replace(ryu, `introduced_in = "world-warrior"`, `introduced_in = "it's"`, 1), `malformed.toml: unknown introduced_in "it's"`},
		{"bad variant", strings.Replace(ryu, `variant = "dark"`, `variant = "dim"`, 1), "malformed.toml: variant must be dark or light"},
		{"bad kind", strings.Replace(ryu, `kind = "character"`, `kind = "boss"`, 1), "malformed.toml: kind must be main or character"},
		{"main with character", strings.Replace(ryu, `kind = "character"`, `kind = "main"`, 1), "malformed.toml: character is required for character themes and forbidden for main"},
		{"schema 2", strings.Replace(ryu, "schema_version = 1", "schema_version = 2", 1), "malformed.toml: unsupported schema_version 2"},
		{"schema float", strings.Replace(ryu, "schema_version = 1", "schema_version = 1.0", 1), "malformed.toml.schema_version must be an integer"},
		{"schema bool", strings.Replace(ryu, "schema_version = 1", "schema_version = true", 1), "malformed.toml.schema_version must be an integer"},
		{"aliases not a list", strings.Replace(ryu, "aliases = []", `aliases = "x"`, 1), "malformed.toml.meta.aliases must be an array of strings"},
		{"alias not a string", strings.Replace(ryu, "aliases = []", `aliases = ["a", 2]`, 1), "malformed.toml.meta.aliases[1] must be a string"},
		{"ui color not a string", strings.Replace(ryu, `background = "#141a23"`, "background = 1", 1), "malformed.toml.ui.background must be a string"},
		{"unknown ansi key", strings.Replace(ryu, "[ansi.normal]", "[ansi.dim]\n[ansi.normal]", 1), "malformed.toml: unknown ansi keys: dim"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.text == ryu {
				t.Fatal("the edit did not apply")
			}
			raw, err := DecodeTOML(tc.text)
			if err != nil {
				t.Fatalf("DecodeTOML: %v", err)
			}
			_, err = ParseTheme(raw, "malformed.toml")
			if err == nil || err.Error() != tc.want {
				t.Errorf("ParseTheme error = %v, want %q", err, tc.want)
			}
		})
	}
}

func catalogFS(t *testing.T) fstest.MapFS {
	t.Helper()
	files := fstest.MapFS{}
	err := fs.WalkDir(themes.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(themes.FS, path)
		files[path] = &fstest.MapFile{Data: data}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func TestLoadFSRejectsABrokenCatalogWithTheReferenceMessage(t *testing.T) {
	cases := []struct {
		name string
		edit func(files fstest.MapFS)
		want string
	}{
		{"duplicate lookup key", editFile("characters/ryu.toml", "aliases = []", `aliases = ["claw"]`),
			"duplicate lookup key 'claw' on ryu and vega"},
		{"unknown key names the embedded file", editFile("characters/ryu.toml", "[ui]\n", "[ui]\nglow = \"#000000\"\n"),
			"unknown characters/ryu.toml.ui keys: glow"},
		{"bad hex color", editFile("characters/ken.toml", `foreground = "#`, `foreground = "#zz`),
			"color must be #RRGGBB, got '#zzdecdcc'"},
		{"missing main.toml", func(files fstest.MapFS) { delete(files, "main.toml") },
			"missing theme file: main.toml"},
		{"missing main-light.toml", func(files fstest.MapFS) { delete(files, "main-light.toml") },
			"missing expected theme main-light"},
		{"low contrast", func(files fstest.MapFS) {
			text := replaceLine(string(files["characters/ryu.toml"].Data), `^foreground = .*$`, `foreground = "#141a24"`)
			files["characters/ryu.toml"] = &fstest.MapFile{Data: []byte(text)}
		}, strings.Join([]string{
			"ryu: foreground/background contrast 1.00:1 is below 7.0:1 (#141a24 on #141a23)",
			"ryu: foreground/surface0 contrast 1.13:1 is below 4.5:1 (#141a24 on #1f252f)",
			"ryu: foreground/surface1 contrast 1.37:1 is below 4.5:1 (#141a24 on #2b333f)",
			"ryu: foreground/panel_bg contrast 1.13:1 is below 4.5:1 (#141a24 on #1f252f)",
			"ryu: foreground/sidebar_bg contrast 1.00:1 is below 4.5:1 (#141a24 on #141a23)",
			"ryu: foreground/active_row_bg contrast 1.47:1 is below 4.5:1 (#141a24 on #522d2c)",
			"ryu: foreground/navigate_row_bg contrast 1.37:1 is below 4.5:1 (#141a24 on #2b333f)",
		}, "\n")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files := catalogFS(t)
			tc.edit(files)
			_, err := LoadFS(files)
			if err == nil || err.Error() != tc.want {
				t.Errorf("LoadFS error = %v, want %q", err, tc.want)
			}
		})
	}
}

func editFile(name, old, replacement string) func(fstest.MapFS) {
	return func(files fstest.MapFS) {
		text := string(files[name].Data)
		edited := strings.Replace(text, old, replacement, 1)
		if edited == text {
			panic("the edit did not apply to " + name)
		}
		files[name] = &fstest.MapFile{Data: []byte(edited)}
	}
}

// catalogIDs is the reference `themes` order: main.toml, main-light.toml, then sorted character files.
var catalogIDs = []string{
	"main", "main-light", "akuma-light", "akuma", "balrog-light", "balrog", "blanka-light", "blanka",
	"cammy-light", "cammy", "chun-li-light", "chun-li", "dee-jay-light", "dee-jay", "dhalsim-light", "dhalsim",
	"e-honda-light", "e-honda", "fei-long-light", "fei-long", "guile-light", "guile", "ken-light", "ken",
	"m-bison-light", "m-bison", "ryu-light", "ryu", "sagat-light", "sagat", "t-hawk-light", "t-hawk",
	"vega-light", "vega", "zangief-light", "zangief",
}

func loadCatalog(t *testing.T) []Theme {
	t.Helper()
	catalog, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return catalog
}

func TestEmbeddedCatalogLoadsAllThirtySixThemesInOrder(t *testing.T) {
	catalog := loadCatalog(t)
	var ids []string
	for _, theme := range catalog {
		ids = append(ids, theme.Metadata.ID)
	}
	if strings.Join(ids, " ") != strings.Join(catalogIDs, " ") {
		t.Errorf("catalog order = %v, want %v", ids, catalogIDs)
	}
}

func TestGetThemeResolvesIDsAliasesAndDisplayNames(t *testing.T) {
	catalog := loadCatalog(t)
	for name, want := range map[string]string{
		"main": "main", "street-fighter-2": "main", "street-fighter-ii-main": "main",
		"boxer": "balrog", "claw": "vega", "dictator": "m-bison", "RYU-LIGHT": "ryu-light",
		"Street Fighter II - Ken": "ken",
	} {
		theme, err := GetTheme(name, catalog)
		if err != nil || theme.Metadata.ID != want {
			t.Errorf("GetTheme(%q) = %q, %v; want %q", name, theme.Metadata.ID, err, want)
		}
	}
	if _, err := GetTheme("bison", catalog); err == nil || err.Error() != "unknown theme: bison" {
		t.Errorf("GetTheme(bison) error = %v, want unknown theme: bison", err)
	}
}

func TestThemePairResolvesDarkAndLightFromEitherSelection(t *testing.T) {
	catalog := loadCatalog(t)
	for _, name := range []string{"ryu", "ryu-light"} {
		selected, _ := GetTheme(name, catalog)
		dark, light, err := ThemePair(selected, catalog)
		if err != nil || dark.Metadata.ID != "ryu" || light.Metadata.ID != "ryu-light" {
			t.Errorf("ThemePair(%s) = %s, %s, %v", name, dark.Metadata.ID, light.Metadata.ID, err)
		}
	}
}

func TestInstalledThemeAcceptsPrefixedAndShortIDs(t *testing.T) {
	catalog := loadCatalog(t)
	for name, want := range map[string]string{"sf2-ken": "ken", "ken": "ken", "boxer": "balrog"} {
		theme, err := InstalledTheme(name, catalog)
		if err != nil || theme.Metadata.ID != want {
			t.Errorf("InstalledTheme(%q) = %q, %v; want %q", name, theme.Metadata.ID, err, want)
		}
	}
}

func TestSelectableIDPrefixesOnce(t *testing.T) {
	if got := SelectableID("ryu"); got != "sf2-ryu" {
		t.Errorf("SelectableID(ryu) = %q", got)
	}
	if got := SelectableID("sf2-ryu"); got != "sf2-ryu" {
		t.Errorf("SelectableID(sf2-ryu) = %q", got)
	}
}
