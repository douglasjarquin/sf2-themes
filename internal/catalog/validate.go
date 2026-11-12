package catalog

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/douglasjarquin/sf2-themes/v2/internal/pycompat"
)

// Contrast minimums.
const (
	PrimaryContrast  = 7.0
	TextContrast     = 4.5
	NontextContrast  = 3.0
	contrastEpsilon  = 1e-9
	expectedFamily   = "sf2"
	ambiguousAlias   = "bison"
	lightSuffix      = "-light"
	mainID           = "main"
	mainLightID      = "main-light"
	chromaticStart   = 1
	chromaticEnd     = 7
	distinctAnsiSlot = 8
)

// ExpectedDarkIDs is the exact dark roster; each has a `-light` sibling.
var ExpectedDarkIDs = []string{
	"main", "akuma", "balrog", "blanka", "cammy", "chun-li", "dee-jay", "dhalsim", "e-honda",
	"fei-long", "guile", "ken", "m-bison", "ryu", "sagat", "t-hawk", "vega", "zangief",
}

// Severity is error or warning.
type Severity string

// Severities.
const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Issue is one validation finding.
type Issue struct {
	Severity Severity
	Message  string
}

func errorIssue(format string, args ...any) Issue {
	return Issue{SeverityError, fmt.Sprintf(format, args...)}
}

// RelativeLuminance is the WCAG relative luminance of a #rrggbb color.
func RelativeLuminance(color HexColor) float64 {
	value := strings.TrimPrefix(string(color), "#")
	linear := func(offset int) float64 {
		channel, _ := strconv.ParseUint(value[offset:offset+2], 16, 8)
		srgb := float64(channel) / 255.0
		if srgb <= 0.04045 {
			return srgb / 12.92
		}
		return math.Pow((srgb+0.055)/1.055, 2.4)
	}
	return 0.2126*linear(0) + 0.7152*linear(2) + 0.0722*linear(4)
}

// ContrastRatio is the WCAG contrast ratio between two colors.
func ContrastRatio(foreground, background HexColor) float64 {
	first, second := RelativeLuminance(foreground), RelativeLuminance(background)
	lighter, darker := first, second
	if first < second {
		lighter, darker = second, first
	}
	return (lighter + 0.05) / (darker + 0.05)
}

func pairIssue(themeID, label string, fg, bg HexColor, minimum float64) (Issue, bool) {
	ratio := ContrastRatio(fg, bg)
	if ratio+contrastEpsilon >= minimum {
		return Issue{}, false
	}
	return errorIssue("%s: %s contrast %.2f:1 is below %.1f:1 (%s on %s)", themeID, label, ratio, minimum, fg, bg), true
}

// ValidateTheme returns errors and warnings for one fully parsed theme.
func ValidateTheme(theme Theme) []Issue {
	ui := theme.UI
	themeID := theme.Metadata.ID
	var issues []Issue
	required := []struct {
		label   string
		fg, bg  HexColor
		minimum float64
	}{
		{"foreground/background", ui.Foreground, ui.Background, PrimaryContrast},
		{"foreground/surface0", ui.Foreground, ui.Surface0(), TextContrast},
		{"foreground/surface1", ui.Foreground, ui.Surface1(), TextContrast},
		{"foreground/panel_bg", ui.Foreground, ui.PanelBg(), TextContrast},
		{"foreground/sidebar_bg", ui.Foreground, ui.SidebarBg(), TextContrast},
		{"foreground/active_row_bg", ui.Foreground, ui.ActiveRowBg(), TextContrast},
		{"foreground/navigate_row_bg", ui.Foreground, ui.NavigateRowBg(), TextContrast},
		{"selection", ui.SelectionFg(), ui.SelectionBg(), TextContrast},
		{"cursor", ui.CursorFg(), ui.CursorBg(), TextContrast},
		{"accent/background", ui.Accent, ui.Background, NontextContrast},
		{"accent_secondary/background", ui.AccentSecondary, ui.Background, NontextContrast},
	}
	for _, pair := range required {
		if issue, ok := pairIssue(themeID, pair.label, pair.fg, pair.bg, pair.minimum); ok {
			issues = append(issues, issue)
		}
	}
	for _, secondary := range []struct {
		label string
		color HexColor
	}{{"muted", ui.Muted}, {"subtle", ui.Subtle}} {
		if ratio := ContrastRatio(secondary.color, ui.Background); ratio < TextContrast {
			issues = append(issues, Issue{
				SeverityWarning,
				fmt.Sprintf("%s: %s on background is %.2f:1 (secondary only)", themeID, secondary.label, ratio),
			})
		}
	}
	issues = append(issues, projectionIssues(theme)...)
	issues = append(issues, semanticIssues(theme)...)
	issues = append(issues, ansiIssues(theme)...)
	var order []HexColor
	seen := map[HexColor][]string{}
	for _, slot := range allSlots(theme) {
		if _, ok := seen[slot.color]; !ok {
			order = append(order, slot.color)
		}
		seen[slot.color] = append(seen[slot.color], slot.name)
	}
	for _, color := range order {
		if slots := seen[color]; len(slots) > 1 {
			issues = append(issues, Issue{
				SeverityWarning,
				fmt.Sprintf("%s: %s reused by %s", themeID, color, strings.Join(slots, ", ")),
			})
		}
	}
	return issues
}

func distinctCount(colors []HexColor) int {
	set := map[HexColor]bool{}
	for _, color := range colors {
		set[color] = true
	}
	return len(set)
}

func ansiIssues(theme Theme) []Issue {
	var issues []Issue
	themeID := theme.Metadata.ID
	normal := theme.AnsiNormal.Colors()
	bright := theme.AnsiBright.Colors()
	if distinctCount(normal) != distinctAnsiSlot {
		issues = append(issues, errorIssue("%s: ansi.normal has duplicate colors", themeID))
	}
	if distinctCount(bright) != distinctAnsiSlot {
		issues = append(issues, errorIssue("%s: ansi.bright has duplicate colors", themeID))
	}
	for i, name := range ANSIOrder {
		if normal[i] == bright[i] {
			issues = append(issues, errorIssue("%s: ansi.bright.%s matches ansi.normal.%s", themeID, name, name))
		}
	}
	for _, row := range []struct {
		name   string
		colors []HexColor
	}{{"normal", normal}, {"bright", bright}} {
		for i := chromaticStart; i < chromaticEnd; i++ {
			label := fmt.Sprintf("ansi.%s.%s/background", row.name, ANSIOrder[i])
			if issue, ok := pairIssue(themeID, label, row.colors[i], theme.UI.Background, TextContrast); ok {
				issues = append(issues, issue)
			}
		}
	}
	return issues
}

func projectionIssues(theme Theme) []Issue {
	ui := theme.UI
	projection := ProjectAdapterColors(ui)
	expected := []struct {
		role      string
		projected HexColor
		source    HexColor
	}{
		{"cursor_bg", projection.CursorBg, ui.Cursor},
		{"cursor_fg", projection.CursorFg, ui.CursorText},
		{"selection_bg", projection.SelectionBg, ui.SelectionBackground},
		{"selection_fg", projection.SelectionFg, ui.SelectionForeground},
		{"panel_bg", projection.PanelBg, ui.Surface},
		{"sidebar_bg", projection.SidebarBg, ui.Background},
		{"active_row_bg", projection.ActiveRowBg, ui.SelectionBackground},
		{"navigate_row_bg", projection.NavigateRowBg, ui.Overlay},
		{"surface_dim", projection.SurfaceDim, ui.Background},
		{"surface0", projection.Surface0, ui.Surface},
		{"surface1", projection.Surface1, ui.Overlay},
		{"overlay0", projection.Overlay0, ui.Border},
		{"overlay1", projection.Overlay1, ui.Muted},
		{"subtext", projection.Subtext, ui.Subtle},
	}
	var issues []Issue
	for _, item := range expected {
		if item.projected != item.source {
			issues = append(issues, errorIssue(
				"%s: adapter projection %s does not match canonical source", theme.Metadata.ID, item.role))
		}
	}
	return issues
}

func semanticIssues(theme Theme) []Issue {
	semantic := theme.Semantic.Colors()
	normal := theme.AnsiNormal.Colors()
	var issues []Issue
	for i := chromaticStart; i < chromaticEnd; i++ {
		// SemanticFields starts at red, one slot after ANSIOrder's black.
		if semantic[i-1] != normal[i] {
			name := ANSIOrder[i]
			issues = append(issues, errorIssue(
				"%s: semantic.%s must match ansi.normal.%s", theme.Metadata.ID, name, name))
		}
	}
	return issues
}

type namedColor struct {
	name  string
	color HexColor
}

func allSlots(theme Theme) []namedColor {
	var slots []namedColor
	for i, color := range theme.UI.Colors() {
		slots = append(slots, namedColor{"ui." + UIColorFields[i], color})
	}
	for i, color := range theme.Semantic.Colors() {
		slots = append(slots, namedColor{"semantic." + SemanticFields[i], color})
	}
	normal, bright := theme.AnsiNormal.Colors(), theme.AnsiBright.Colors()
	for i, name := range ANSIOrder {
		slots = append(slots,
			namedColor{"ansi.normal." + name, normal[i]},
			namedColor{"ansi.bright." + name, bright[i]},
		)
	}
	return slots
}

func errorMessages(issues []Issue) []string {
	var messages []string
	for _, issue := range issues {
		if issue.Severity == SeverityError {
			messages = append(messages, issue.Message)
		}
	}
	return messages
}

// RequireValid returns one error holding every error-severity finding for the theme, one per line.
func RequireValid(theme Theme) error {
	if messages := errorMessages(ValidateTheme(theme)); len(messages) > 0 {
		return errorLines(messages)
	}
	return nil
}

// ValidateCatalog checks the exact roster, paired metadata, ids, and aliases.
func ValidateCatalog(themes []Theme) []Issue {
	var issues []Issue
	claimed := map[string]string{}
	byID := map[string]Theme{}
	for _, theme := range themes {
		byID[theme.Metadata.ID] = theme
	}
	expected := map[string]bool{}
	for _, darkID := range ExpectedDarkIDs {
		expected[darkID] = true
		expected[darkID+lightSuffix] = true
	}
	for _, id := range sortedKeys(expected, func(id string) bool { _, ok := byID[id]; return !ok }) {
		issues = append(issues, errorIssue("missing expected theme %s", id))
	}
	for _, id := range sortedKeys(byID, func(id string) bool { return !expected[id] }) {
		issues = append(issues, errorIssue("unexpected theme %s", id))
	}
	for _, theme := range themes {
		meta := theme.Metadata
		expectedVariant := VariantDark
		if strings.HasSuffix(meta.ID, lightSuffix) {
			expectedVariant = VariantLight
		}
		if meta.Variant != expectedVariant {
			issues = append(issues, errorIssue("%s: variant must be %s", meta.ID, expectedVariant))
		}
		if meta.Family != expectedFamily {
			issues = append(issues, errorIssue("%s: family must be sf2", meta.ID))
		}
		expectedKind := KindCharacter
		if meta.ID == mainID || meta.ID == mainLightID {
			expectedKind = KindMain
		}
		if meta.Kind != expectedKind {
			issues = append(issues, errorIssue("%s: kind must be %s", meta.ID, expectedKind))
		}
		keys := theme.LookupKeys()
		for _, key := range keys {
			if owner, ok := claimed[key]; ok {
				issues = append(issues, errorIssue(
					"duplicate lookup key %s on %s and %s", pycompat.Repr(key), owner, meta.ID))
			} else {
				claimed[key] = meta.ID
			}
		}
		if slices.Contains(keys, ambiguousAlias) {
			issues = append(issues, errorIssue("%s: alias 'bison' is regionally ambiguous", meta.ID))
		}
	}
	for _, darkID := range ExpectedDarkIDs {
		dark, hasDark := byID[darkID]
		light, hasLight := byID[darkID+lightSuffix]
		if hasDark && hasLight {
			issues = append(issues, pairingIssues(dark, light)...)
		}
	}
	return issues
}

func sortedKeys[V any](m map[string]V, keep func(string) bool) []string {
	var keys []string
	for key := range m {
		if keep(key) {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	return keys
}

func pairingIssues(dark, light Theme) []Issue {
	darkMeta, lightMeta := dark.Metadata, light.Metadata
	sameCharacter := (darkMeta.Character == nil) == (lightMeta.Character == nil) &&
		(darkMeta.Character == nil || *darkMeta.Character == *lightMeta.Character)
	checks := []struct {
		field   string
		matches bool
	}{
		{"display_name", lightMeta.DisplayName == darkMeta.DisplayName+" Light"},
		{"kind", lightMeta.Kind == darkMeta.Kind},
		{"introduced_in", lightMeta.IntroducedIn == darkMeta.IntroducedIn},
		{"character", sameCharacter},
		{"name", lightMeta.Name == darkMeta.Name},
		{"family", lightMeta.Family == darkMeta.Family},
		{"stage", lightMeta.Stage == darkMeta.Stage},
	}
	var issues []Issue
	for _, check := range checks {
		if !check.matches {
			issues = append(issues, errorIssue("%s: %s must match dark variant", lightMeta.ID, check.field))
		}
	}
	if len(lightMeta.Aliases) > 0 {
		issues = append(issues, errorIssue("%s: light variant aliases must be empty", lightMeta.ID))
	}
	if light.UI.Background == dark.UI.Background {
		issues = append(issues, errorIssue("%s: background must differ from dark variant", lightMeta.ID))
	}
	if light.UI.Foreground == dark.UI.Foreground {
		issues = append(issues, errorIssue("%s: foreground must differ from dark variant", lightMeta.ID))
	}
	return issues
}
