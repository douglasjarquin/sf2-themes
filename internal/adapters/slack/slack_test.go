package slack

import (
	"testing"

	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
)

// The import lines were recorded from the reference's tests/test_slack.py.
func TestRenderThemeIsTheFourColorImportLine(t *testing.T) {
	themes, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]string{
		"ken":       "#221616,#de6767,#549657,#c76e67",
		"ken-light": "#faeeed,#af3c40,#2d7633,#a94a46",
		"main":      "#131927,#ad8705,#4e975b,#c86e67",
	} {
		theme, err := catalog.GetTheme(id, themes)
		if err != nil {
			t.Fatal(err)
		}
		if got := RenderTheme(theme); got != want {
			t.Errorf("%s: %q, want %q", id, got, want)
		}
	}
}
