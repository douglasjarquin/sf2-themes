package starship

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
	"github.com/douglasjarquin/sf2-themes/v2/internal/plan"
)

// Expected values were recorded from sf2_theme.adapters.starship, zsh_syntax, and the cli APPS entry under
// Python 3.11. Each testdata/<name>-input.toml was merged into testdata/<name>.toml with the main theme.

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

func TestMergeThemeKeepsThePaletteAtTopLevel(t *testing.T) {
	main, _ := loadTheme(t, "main")
	for _, name := range []string{
		"preset",              // a multi-line `format` string holding `[...]` lines
		"single-quote-preset", // the same with a literal multi-line string
		"tables",
		"comment-heads-table", // the comment above the first table stays with it
		"palette-replaced",    // a top-level palette outside the markers is replaced
		"after-tables",        // a managed block found after the tables moves to the top
		"empty",
	} {
		got, err := mergeTheme(readTestdata(t, name+"-input.toml"), main)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if want := readTestdata(t, name+".toml"); got != want {
			t.Errorf("%s: merged\n%s", name, got)
		}
		if again, err := mergeTheme(got, main); err != nil || again != got {
			t.Errorf("%s: a second merge changed the document (%v)", name, err)
		}
	}
}

func TestMergeThemeParsesTheResult(t *testing.T) {
	main, _ := loadTheme(t, "main")
	// The dotted key declares palettes.sf2, so the managed [palettes.sf2] table would declare it twice.
	if got, err := mergeTheme("palettes.sf2.x = 1\n", main); err == nil {
		t.Errorf("an invalid merge was returned:\n%s", got)
	}
}

func TestInspect(t *testing.T) {
	_, themes := loadTheme(t, "main")
	cases := []struct {
		name      string
		content   string
		ownership plan.Ownership
		current   string
		detail    string
	}{
		{"own top-level palette", "palette = \"mine\"\n\n[palettes.mine]\nblue = \"#0000ff\"\n", plan.Foreign, "", ""},
		{"managed block", readTestdata(t, "tables.toml"), plan.Managed, "main", ""},
		{"managed block after the tables", readTestdata(t, "after-tables-input.toml"), plan.Managed, "ken", ""},
		{"no palette", "[directory]\n", plan.Absent, "", ""},
		{"palette only under a table", "palettes.sf2.x = 1\n", plan.Absent, "", ""},
		{"incomplete markers", "# <<< sf2-themes managed theme\n", plan.Broken, "", "{D}/starship.toml has incomplete sf2-themes markers"},
	}
	for _, tc := range cases {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "starship.toml"), []byte(tc.content), 0o644); err != nil {
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
		if got.Ownership != tc.ownership || current != tc.current || got.Unreadable || got.Detail != detail {
			t.Errorf("%s: %+v", tc.name, got)
		}
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "starship.toml"), []byte("palette = \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := Inspect(themes, dir)
	if err != nil || got.Ownership != plan.Broken || !strings.HasPrefix(got.Detail, dir+"/starship.toml is not valid TOML: ") {
		t.Errorf("invalid TOML: %+v, %v", got, err)
	}
}

func TestPlanWritesThePaletteAndTheZshSnippet(t *testing.T) {
	vega, themes := loadTheme(t, "vega")
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir+"/xdg")
	steps, err := Plan(vega, themes, dir+"/config", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) != 2 || steps[0].Path != dir+"/config/starship.toml" || steps[1].Path != dir+"/xdg/sf2-theme/zsh-syntax-highlighting.zsh" {
		t.Fatalf("steps: %+v", steps)
	}
	if steps[1].Content != readTestdata(t, "zsh-vega.zsh") {
		t.Errorf("zsh snippet:\n%s", steps[1].Content)
	}
	if steps, err := Plan(vega, themes, dir+"/config", false); err != nil || len(steps) != 0 {
		t.Errorf("without the slot: %+v, %v", steps, err)
	}
}

func TestStderrHintMatchesTheReference(t *testing.T) {
	want := "Source the zsh highlight snippet after zsh-syntax-highlighting:\n" +
		`  [[ -f "$HOME/.config/sf2-theme/zsh-syntax-highlighting.zsh" ]] && source "$HOME/.config/sf2-theme/zsh-syntax-highlighting.zsh"` + "\n"
	if App.StderrHint != want {
		t.Errorf("hint: %q", App.StderrHint)
	}
}
