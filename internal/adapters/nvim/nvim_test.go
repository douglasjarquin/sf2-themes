package nvim

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
	"github.com/douglasjarquin/sf2-themes/v2/internal/plan"
)

// Expected values were recorded from sf2_theme.adapters.nvim under Python 3.11; testdata/nvim-*.lua are copies of
// the reference's tests/snapshots.

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

func TestRenderSchemeMatchesTheSnapshots(t *testing.T) {
	for id, snapshot := range map[string]string{"main": "nvim-main.lua", "ryu-light": "nvim-ryu-light.lua"} {
		theme, _ := loadTheme(t, id)
		want, err := os.ReadFile(filepath.Join("testdata", snapshot))
		if err != nil {
			t.Fatal(err)
		}
		if got := renderScheme(theme); got != string(want) {
			t.Errorf("%s: rendered scheme differs from %s:\n%s", id, snapshot, got)
		}
	}
}

func TestRenderPointerSwitchesTheCharacterPair(t *testing.T) {
	theme, themes := loadTheme(t, "ryu-light")
	dark, light, err := catalog.ThemePair(theme, themes)
	if err != nil {
		t.Fatal(err)
	}
	want := "-- sf2-themes: sf2-ryu\n" +
		"local light_mode = vim.env.TERM_THEME == \"light\" or (vim.env.TERM_THEME ~= \"dark\" and vim.o.background == \"light\")\n" +
		"if light_mode then\n  vim.cmd(\"colorscheme sf2-ryu-light\")\nelse\n  vim.cmd(\"colorscheme sf2-ryu\")\nend\n"
	if got := renderPointer(dark, light); got != want {
		t.Errorf("pointer:\n%s", got)
	}
}

func TestInspectReadsThePointerCommentWithOneOrMoreSpaces(t *testing.T) {
	_, themes := loadTheme(t, "main")
	cases := []struct {
		pointer    string
		current    string
		unreadable bool
	}{
		{"-- sf2-themes:  sf2-ryu \n", "ryu", false},
		{"-- sf2-themes: ken\n", "ken", false},
		{"-- sf2-themes:sf2-ryu\n", "", true},
		{"--sf2-themes:sf2-ryu\n", "", true},
	}
	for _, tc := range cases {
		dir := t.TempDir()
		path := filepath.Join(dir, "sf2-theme", "current.lua")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(tc.pointer), 0o644); err != nil {
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
		if got.Ownership != plan.Managed || current != tc.current || got.Unreadable != tc.unreadable {
			t.Errorf("%q: %+v", tc.pointer, got)
		}
	}
	got, err := Inspect(themes, t.TempDir())
	if err != nil || got != (plan.Inspection{Ownership: plan.Absent}) {
		t.Errorf("no pointer: %+v, %v", got, err)
	}
}

func TestPlanWritesEverySchemeThePointerAndTheLoader(t *testing.T) {
	theme, themes := loadTheme(t, "main")
	dir := t.TempDir()
	steps, err := Plan(theme, themes, dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) != 2*len(themes)+2 {
		t.Fatalf("%d steps", len(steps))
	}
	if !steps[0].Remove || steps[0].Path != dir+"/colors/street-fighter-ii-main.lua" {
		t.Errorf("first step: %+v", steps[0])
	}
	if steps[len(themes)].Path != dir+"/colors/sf2-main.lua" {
		t.Errorf("first scheme: %+v", steps[len(themes)])
	}
	pointer, loader := steps[len(steps)-2], steps[len(steps)-1]
	if pointer.Path != dir+"/sf2-theme/current.lua" || loader.Path != dir+"/plugin/sf2-theme.lua" {
		t.Errorf("pointer %q, loader %q", pointer.Path, loader.Path)
	}
}
