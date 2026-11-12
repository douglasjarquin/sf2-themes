package herdr

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
	"github.com/douglasjarquin/sf2-themes/v2/internal/fsys"
	"github.com/douglasjarquin/sf2-themes/v2/internal/plan"
)

// Expected values were recorded from sf2_theme.adapters.herdr and sf2_theme.filesystem.commit under Python 3.11;
// testdata/herdr-main.toml is a copy of the reference's tests/snapshots.

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

func TestRenderBlockMatchesTheSnapshot(t *testing.T) {
	theme, themes := loadTheme(t, "main")
	dark, light, err := catalog.ThemePair(theme, themes)
	if err != nil {
		t.Fatal(err)
	}
	if got := renderBlock(dark, light); got != readTestdata(t, "herdr-main.toml") {
		t.Errorf("rendered block differs from herdr-main.toml:\n%s", got)
	}
}

func TestMergeTheme(t *testing.T) {
	main, themes := loadTheme(t, "main")
	cases := []struct {
		name     string
		existing string
		want     string
	}{
		{"empty", "", "merge-empty.toml"},
		{"unrelated sections are preserved", "[ui]\ntheme = \"follow-system\"\n\n[keys]\nprefix = \"ctrl+b\"\n", "merge-unrelated.toml"},
		{"an adopted theme table is replaced", "[ui]\nconfirm_close = true\n\n[theme]\nname = \"dracula\"\n", "merge-adopted.toml"},
		{"an adopted top-level theme is replaced", "theme = \"dracula\"\n\n[ui]\nsidebar = true\n", "merge-top-level.toml"},
	}
	for _, tc := range cases {
		got, err := mergeTheme(tc.existing, main, themes)
		if err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if want := readTestdata(t, tc.want); got != want {
			t.Errorf("%s: merged\n%s", tc.name, got)
		}
		if again, err := mergeTheme(got, main, themes); err != nil || again != got {
			t.Errorf("%s: a second merge changed the block (%v)", tc.name, err)
		}
	}

	ryu, _ := loadTheme(t, "ryu")
	got, err := mergeTheme(readTestdata(t, "merge-mid-input.toml"), ryu, themes)
	if err != nil || got != readTestdata(t, "merge-mid.toml") {
		t.Errorf("a managed block between tables was not replaced in place (%v):\n%s", err, got)
	}

	dark, _ := loadTheme(t, "chun-li")
	light, _ := loadTheme(t, "chun-li-light")
	fromDark, errDark := mergeTheme("", dark, themes)
	fromLight, errLight := mergeTheme("", light, themes)
	if errDark != nil || errLight != nil || fromDark != fromLight {
		t.Errorf("selecting either sibling wrote different blocks")
	}
}

func TestMergeThemeParsesTheResult(t *testing.T) {
	main, themes := loadTheme(t, "main")
	// A dotted top-level `theme.x` is not an unmarked theme, so the block is appended and redeclares `theme`.
	if got, err := mergeTheme("theme.x = 1\n", main, themes); err == nil {
		t.Errorf("an invalid merge was returned:\n%s", got)
	}
}

func TestInspect(t *testing.T) {
	kenLight, themes := loadTheme(t, "ken-light")
	managed, err := mergeTheme("", kenLight, themes)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name       string
		content    string
		ownership  plan.Ownership
		current    string
		unreadable bool
		detail     string
	}{
		{"unmarked theme table", "[theme]\nname = \"catppuccin\"\n", plan.Foreign, "", false, ""},
		{"unmarked top-level theme", "theme = \"x\"\n", plan.Foreign, "", false, ""},
		{"theme key inside another table", "[ui]\ntheme = \"x\"\n", plan.Absent, "", false, ""},
		{"dotted top-level theme key", "theme.x = 1\n", plan.Absent, "", false, ""},
		{"managed pair", managed, plan.Managed, "ken", false, ""},
		{"managed id that does not resolve", "# >>> sf2-themes managed theme\n# sf2-themes: sf2-nope\n# <<< sf2-themes managed theme\n", plan.Managed, "", true, ""},
		{"incomplete markers", "# >>> sf2-themes managed theme\n[theme]\nname = \"x\"\n", plan.Broken, "", false, "{D}/config.toml has incomplete sf2-themes markers"},
	}
	for _, tc := range cases {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(tc.content), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := Inspect(themes, dir)
		if err != nil {
			t.Fatal(err)
		}
		current := ""
		if got.Current != nil {
			current = got.Current.Metadata.ID
		}
		detail := strings.ReplaceAll(tc.detail, "{D}", dir)
		if got.Ownership != tc.ownership || current != tc.current || got.Unreadable != tc.unreadable || got.Detail != detail {
			t.Errorf("%s: %+v", tc.name, got)
		}
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("[theme\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Inspect(themes, dir)
	if err != nil || got.Ownership != plan.Broken || !strings.HasPrefix(got.Detail, dir+"/config.toml is not valid TOML: ") {
		t.Errorf("invalid TOML: %+v, %v", got, err)
	}
}

func TestACRLFConfigIsMergedAsLF(t *testing.T) {
	main, themes := loadTheme(t, "main")
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte("[ui]\r\nsidebar = true\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	steps, err := Plan(main, themes, dir, true)
	if err != nil {
		t.Fatal(err)
	}
	results, err := fsys.Commit(steps, fsys.Options{})
	if err != nil {
		t.Fatal(err)
	}
	want := readTestdata(t, "merge-crlf.toml")
	added := strings.SplitAfter(strings.TrimPrefix(want, "[ui]\nsidebar = true\n"), "\n")
	diff := "--- " + path + "\n+++ " + path + "\n@@ -1,2 +1,53 @@\n [ui]\n sidebar = true\n+" +
		strings.Join(added[:len(added)-1], "+")
	if len(results) != 1 || results[0].Path != path || results[0].Action != fsys.Updated || results[0].Diff != diff {
		t.Errorf("results: %+v", results)
	}
	if written, err := os.ReadFile(path); err != nil || string(written) != want {
		t.Errorf("written:\n%s", written)
	}
	backups, err := filepath.Glob(path + ".bak.*")
	if err != nil || len(backups) != 1 {
		t.Fatalf("backups: %v", backups)
	}
	if held, err := os.ReadFile(backups[0]); err != nil || string(held) != "[ui]\nsidebar = true\n" {
		t.Errorf("backup holds %q", held)
	}
}

func TestPlanWritesNothingWithoutTheSlot(t *testing.T) {
	main, themes := loadTheme(t, "main")
	if steps, err := Plan(main, themes, t.TempDir(), false); err != nil || len(steps) != 0 {
		t.Errorf("steps: %+v, %v", steps, err)
	}
}
