package codex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
	"github.com/douglasjarquin/sf2-themes/v2/internal/plan"
)

// Every expected value here was recorded from the reference adapter, src/sf2_theme/adapters/codex.py.

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

func TestRenderThemeMatchesTheReference(t *testing.T) {
	ryu, _ := load(t, "ryu")
	want, err := os.ReadFile("testdata/sf2-ryu.tmTheme")
	if err != nil {
		t.Fatal(err)
	}
	if got := RenderTheme(ryu); got != string(want) {
		t.Errorf("RenderTheme(ryu) = %q, want %q", got, want)
	}
}

func TestThemeNameEscapesAmpersandAndAngleBracketsButNotQuotes(t *testing.T) {
	theme, _ := load(t, "ryu")
	theme.Metadata.ID = `a&b"c<d>`
	want := `<key>name</key><string>sf2-a&amp;b"c&lt;d&gt;</string>`
	if !strings.Contains(RenderTheme(theme), "\n"+want+"\n") {
		t.Errorf("rendered theme lacks %q", want)
	}
}

func TestMergeConfig(t *testing.T) {
	ryu, _ := load(t, "ryu")
	for _, tc := range []struct{ name, existing, want string }{
		{
			"existing table gains the assignment at its end",
			"model = \"o3\"\n\n[tui]\nanimations = false\n\n[other]\nx = 1\n",
			"model = \"o3\"\n\n[tui]\nanimations = false\n\ntheme = \"sf2-ryu\"\n[other]\nx = 1\n",
		},
		{"inline table", "tui = { theme = \"dracula\" }\n", "tui = { theme = \"dracula\" }\n\n[tui]\ntheme = \"sf2-ryu\"\n"},
		// A header followed by a comment is not recognized, so a second [tui] table is appended. The result is
		// not valid TOML; the reference does the same.
		{"header with a comment", "[tui] # mine\ntheme = \"dracula\"\n", "[tui] # mine\ntheme = \"dracula\"\n\n[tui]\ntheme = \"sf2-ryu\"\n"},
	} {
		if got := MergeConfig(tc.existing, ryu); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestInspectClassifiesTUIShapes(t *testing.T) {
	_, themes := load(t, "main")
	for _, tc := range []struct {
		content string
		want    plan.Ownership
		detail  bool
	}{
		{"tui = { theme = \"dracula\" }\n", plan.UnknownShape, true},
		{"tui.theme = \"dracula\"\n", plan.UnknownShape, true},
		{"[tui] # mine\ntheme = \"dracula\"\n", plan.Absent, false},
		{"[tui]\ntheme = \"ryu\"\n", plan.Legacy, false},
		{"[tui]\ntheme = \"dracula\"\n", plan.Foreign, false},
		{"[tui]\ntheme = \"sf2-ryu\"\n", plan.Managed, false},
	} {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.toml")
		if err := os.WriteFile(path, []byte(tc.content), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := Inspect(themes, dir)
		if err != nil {
			t.Fatal(err)
		}
		wantDetail := ""
		if tc.detail {
			wantDetail = "Convert the inline tui value in " + path + " into a [tui] table, for example:\n\n[tui]\ntheme = \"...\""
		}
		if got.Ownership != tc.want || got.Detail != wantDetail {
			t.Errorf("%q: %s %q, want %s %q", tc.content, got.Ownership, got.Detail, tc.want, wantDetail)
		}
	}
}
