package herdrapp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
	"github.com/douglasjarquin/sf2-themes/v2/internal/plan"
)

// Every expected value here was recorded from the reference adapter, src/sf2_theme/adapters/herdr_app.py.

func load(t *testing.T, id string) (catalog.Theme, []catalog.Theme) {
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

func writeSettings(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRenderThemeMatchesTheReference(t *testing.T) {
	ryu, _ := load(t, "ryu")
	want, err := os.ReadFile("testdata/sf2-ryu.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := RenderTheme(ryu); got != string(want) {
		t.Errorf("RenderTheme(ryu) = %q, want %q", got, want)
	}
}

func TestSelectionKeepsExistingSubKeysInPlaceAndSetsDarkAndLight(t *testing.T) {
	kenLight, themes := load(t, "ken-light")
	dir := t.TempDir()
	path := writeSettings(t, dir,
		`{"fontSize": 14, "theme": {"mode": "auto", "light": "paper", "extra": [1]}, "layout": "work"}`)
	steps, err := Plan(kenLight, themes, dir, true)
	if err != nil {
		t.Fatal(err)
	}
	last := steps[len(steps)-1]
	want := "{\n  \"fontSize\": 14,\n  \"theme\": {\n    \"mode\": \"auto\",\n    \"light\": \"sf2-ken-light\",\n" +
		"    \"extra\": [\n      1\n    ],\n    \"dark\": \"sf2-ken\"\n  },\n  \"layout\": \"work\"\n}\n"
	if last.Path != path || last.Content != want {
		t.Errorf("last step %q = %q, want %q = %q", last.Path, last.Content, path, want)
	}
	if len(steps) != len(themes)+1 {
		t.Errorf("%d steps, want one per catalog theme plus settings", len(steps))
	}
}

func TestOnlyAMatchingCatalogPairIsManaged(t *testing.T) {
	_, themes := load(t, "main")
	for content, want := range map[string]plan.Ownership{
		`{"theme": {"dark": "sf2-ryu", "light": "sf2-ryu-light", "mode": "auto"}}`: plan.Managed,
		`{"theme": {"dark": "sf2-ryu", "light": "sf2-ken-light"}}`:                 plan.Foreign,
		`{"theme": {"dark": "sf2-unknown", "light": "sf2-unknown-light"}}`:         plan.Foreign,
		`{"theme": "graphite"}`: plan.Foreign,
		`{"fontSize": 14}`:      plan.Absent,
		`{"theme": 3}`:          plan.Broken,
		"[]\n":                  plan.Broken,
	} {
		dir := t.TempDir()
		writeSettings(t, dir, content)
		got, err := Inspect(themes, dir)
		if err != nil {
			t.Fatal(err)
		}
		if got.Ownership != want {
			t.Errorf("%s: %s, want %s", content, got.Ownership, want)
		}
		if want == plan.Managed && (got.Current == nil || got.Current.Metadata.ID != "ryu") {
			t.Errorf("%s: current %+v, want ryu", content, got.Current)
		}
	}
}

// An XDG_CONFIG_HOME that is set but empty makes the directory the relative `herdr-app`, as the reference does.
func TestEmptyXDGConfigHomeIsTheRelativeDirectory(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	got, err := home("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "herdr-app" {
		t.Errorf("home = %q, want %q", got, "herdr-app")
	}
}
