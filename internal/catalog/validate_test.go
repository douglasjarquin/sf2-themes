package catalog

import (
	"slices"
	"strings"
	"testing"
)

// Expected values come from tests/test_validation.py and tests/test_catalog.py in the Python reference.

func ryuTheme(t *testing.T) Theme {
	t.Helper()
	raw, err := DecodeTOML(catalogFile(t, "characters/ryu.toml"))
	if err != nil {
		t.Fatal(err)
	}
	theme, err := ParseTheme(raw, "canonical.toml")
	if err != nil {
		t.Fatal(err)
	}
	return theme
}

func TestParseHexAcceptsRRGGBBAndLowercases(t *testing.T) {
	if got, err := ParseHex("#FFF4D6"); err != nil || got != "#fff4d6" {
		t.Errorf("ParseHex = %q, %v", got, err)
	}
	for _, raw := range []string{"#fff", "#ffffffff", "red", "fff4d6", "#GGG000", "#ffffff\n"} {
		if _, err := ParseHex(raw); err == nil || !strings.Contains(err.Error(), "#RRGGBB") {
			t.Errorf("ParseHex(%q) error = %v", raw, err)
		}
	}
}

func TestContrastRatios(t *testing.T) {
	if r := ContrastRatio("#fff4d6", "#101a3a"); r < 7.0 {
		t.Errorf("cream on navy = %v", r)
	}
	if r := ContrastRatio("#fff4d6", "#18a6a6"); r >= 4.5 {
		t.Errorf("cream on teal = %v", r)
	}
	if r := ContrastRatio("#101a3a", "#f2b134"); r < 4.5 {
		t.Errorf("navy on gold = %v", r)
	}
}

func TestCanonicalThemeHasNoErrors(t *testing.T) {
	if errs := errorMessages(ValidateTheme(ryuTheme(t))); len(errs) != 0 {
		t.Errorf("errors = %v", errs)
	}
}

func TestIdenticalBrightAndNormalIsAnError(t *testing.T) {
	theme := ryuTheme(t)
	theme.AnsiBright.Black = theme.AnsiNormal.Black
	if !slices.Contains(errorMessages(ValidateTheme(theme)), "ryu: ansi.bright.black matches ansi.normal.black") {
		t.Error("no ansi.bright.black matches error")
	}
}

func TestSemanticSlotMustMatchItsAnsiMeaning(t *testing.T) {
	theme := ryuTheme(t)
	theme.Semantic.Red = "#123456"
	if !slices.Contains(errorMessages(ValidateTheme(theme)), "ryu: semantic.red must match ansi.normal.red") {
		t.Error("no semantic.red error")
	}
}

func TestSemanticFieldsStartOneSlotAfterAnsiBlack(t *testing.T) {
	if SemanticFields[0] != "red" || ANSIOrder[1] != "red" {
		t.Errorf("SemanticFields[0] = %q, ANSIOrder[1] = %q, want both red", SemanticFields[0], ANSIOrder[1])
	}
}

func TestAnsiChromaticSlotRequiresTextContrast(t *testing.T) {
	theme := ryuTheme(t)
	theme.AnsiNormal.Red = "#151b24"
	found := false
	for _, message := range errorMessages(ValidateTheme(theme)) {
		found = found || strings.HasPrefix(message, "ryu: ansi.normal.red/background contrast")
	}
	if !found {
		t.Error("no ansi.normal.red/background contrast error")
	}
}

func TestDuplicateLookupKeyIsAnError(t *testing.T) {
	theme := ryuTheme(t)
	if !slices.Contains(errorMessages(ValidateCatalog([]Theme{theme, theme})), "duplicate lookup key 'ryu' on ryu and ryu") {
		t.Error("no duplicate lookup key error")
	}
}

func replaceTheme(catalog []Theme, id string, edit func(*Theme)) []Theme {
	edited := slices.Clone(catalog)
	for i := range edited {
		if edited[i].Metadata.ID == id {
			edit(&edited[i])
		}
	}
	return edited
}

func TestCatalogRejectsLightIDWithDarkVariant(t *testing.T) {
	catalog := replaceTheme(loadCatalog(t), "ryu-light", func(theme *Theme) { theme.Metadata.Variant = VariantDark })
	if !slices.Contains(errorMessages(ValidateCatalog(catalog)), "ryu-light: variant must be light") {
		t.Error("no variant error")
	}
}

func TestCatalogRejectsMissingAndUnexpectedIDs(t *testing.T) {
	catalog := replaceTheme(loadCatalog(t), "ryu-light", func(theme *Theme) { theme.Metadata.ID = "ryu-preview" })
	messages := errorMessages(ValidateCatalog(catalog))
	for _, want := range []string{"missing expected theme ryu-light", "unexpected theme ryu-preview"} {
		if !slices.Contains(messages, want) {
			t.Errorf("missing %q in %v", want, messages)
		}
	}
}

func TestCatalogRejectsLightMetadataDrift(t *testing.T) {
	catalog := replaceTheme(loadCatalog(t), "ryu-light", func(theme *Theme) {
		theme.Metadata.Stage = "Wrong stage"
		theme.Metadata.Aliases = []string{"ryu-alt"}
	})
	messages := errorMessages(ValidateCatalog(catalog))
	for _, want := range []string{"ryu-light: stage must match dark variant", "ryu-light: light variant aliases must be empty"} {
		if !slices.Contains(messages, want) {
			t.Errorf("missing %q in %v", want, messages)
		}
	}
}

func TestShippedCatalogHasNoCatalogErrors(t *testing.T) {
	if errs := errorMessages(ValidateCatalog(loadCatalog(t))); len(errs) != 0 {
		t.Errorf("errors = %v", errs)
	}
}

func TestAdapterProjectionCoversEveryRole(t *testing.T) {
	ui := ryuTheme(t).UI
	got := ProjectAdapterColors(ui)
	want := AdapterColors{
		CursorBg: ui.Cursor, CursorFg: ui.CursorText, SelectionBg: ui.SelectionBackground,
		SelectionFg: ui.SelectionForeground, PanelBg: ui.Surface, SidebarBg: ui.Background,
		ActiveRowBg: ui.SelectionBackground, NavigateRowBg: ui.Overlay, SurfaceDim: ui.Background,
		Surface0: ui.Surface, Surface1: ui.Overlay, Overlay0: ui.Border, Overlay1: ui.Muted, Subtext: ui.Subtle,
	}
	if got != want {
		t.Errorf("projection = %+v, want %+v", got, want)
	}
}

func TestCanonicalTOMLRoundTrips(t *testing.T) {
	theme := ryuTheme(t)
	raw, err := DecodeTOML(ThemeToTOML(theme))
	if err != nil {
		t.Fatal(err)
	}
	back, err := ParseTheme(raw, "round-trip.toml")
	if err != nil {
		t.Fatal(err)
	}
	if ThemeToTOML(back) != ThemeToTOML(theme) {
		t.Error("round trip changed the theme")
	}
}
