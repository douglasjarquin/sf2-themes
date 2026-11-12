// Package catalog turns the TOML theme catalog into typed, validated Theme values.
package catalog

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/douglasjarquin/sf2-themes/v2/internal/pycompat"
)

// HexColor is a lowercase #rrggbb color.
type HexColor string

// ANSIOrder is the ANSI slot order: black, red, green, yellow, blue, magenta, cyan, white.
var ANSIOrder = []string{"black", "red", "green", "yellow", "blue", "magenta", "cyan", "white"}

// UIColorFields names the canonical designer palette fields in source order.
var UIColorFields = []string{
	"background",
	"surface",
	"overlay",
	"border",
	"foreground",
	"muted",
	"subtle",
	"accent",
	"accent_secondary",
	"cursor",
	"cursor_text",
	"selection_background",
	"selection_foreground",
}

// SemanticFields names the semantic hue fields in source order.
var SemanticFields = []string{"red", "green", "yellow", "blue", "magenta", "cyan", "orange"}

// SelectablePrefix prefixes every adapter-installed theme identity.
const SelectablePrefix = "sf2-"

var hexPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// ThemeKind is main or character.
type ThemeKind string

// Theme kinds.
const (
	KindMain      ThemeKind = "main"
	KindCharacter ThemeKind = "character"
)

// IntroducedIn is the game release a theme belongs to.
type IntroducedIn string

// Game releases.
const (
	IntroducedMain            IntroducedIn = "main"
	IntroducedWorldWarrior    IntroducedIn = "world-warrior"
	IntroducedChampionEdition IntroducedIn = "champion-edition"
	IntroducedSuper           IntroducedIn = "super"
	IntroducedSuperTurbo      IntroducedIn = "super-turbo"
)

// ThemeVariant is dark or light.
type ThemeVariant string

// Theme variants.
const (
	VariantDark  ThemeVariant = "dark"
	VariantLight ThemeVariant = "light"
)

// ThemeMetadata is the identity and catalog lookup keys for one theme.
type ThemeMetadata struct {
	ID           string
	DisplayName  string
	Kind         ThemeKind
	IntroducedIn IntroducedIn
	// Character is nil for main themes.
	Character *string
	Aliases   []string
	Name      string
	Variant   ThemeVariant
	Family    string
	Stage     string
}

// SelectableID is the prefixed identity used by installed adapters.
func (m ThemeMetadata) SelectableID() string {
	return SelectableID(m.ID)
}

// UIColors is the canonical designer palette.
type UIColors struct {
	Background          HexColor
	Surface             HexColor
	Overlay             HexColor
	Border              HexColor
	Foreground          HexColor
	Muted               HexColor
	Subtle              HexColor
	Accent              HexColor
	AccentSecondary     HexColor
	Cursor              HexColor
	CursorText          HexColor
	SelectionBackground HexColor
	SelectionForeground HexColor
}

// Colors returns the palette in UIColorFields order.
func (u UIColors) Colors() []HexColor {
	return []HexColor{
		u.Background, u.Surface, u.Overlay, u.Border, u.Foreground, u.Muted, u.Subtle,
		u.Accent, u.AccentSecondary, u.Cursor, u.CursorText, u.SelectionBackground, u.SelectionForeground,
	}
}

func uiColorsFrom(c map[string]HexColor) UIColors {
	return UIColors{
		Background:          c["background"],
		Surface:             c["surface"],
		Overlay:             c["overlay"],
		Border:              c["border"],
		Foreground:          c["foreground"],
		Muted:               c["muted"],
		Subtle:              c["subtle"],
		Accent:              c["accent"],
		AccentSecondary:     c["accent_secondary"],
		Cursor:              c["cursor"],
		CursorText:          c["cursor_text"],
		SelectionBackground: c["selection_background"],
		SelectionForeground: c["selection_foreground"],
	}
}

// CursorBg is the adapter cursor background role.
func (u UIColors) CursorBg() HexColor { return ProjectAdapterColors(u).CursorBg }

// CursorFg is the adapter cursor foreground role.
func (u UIColors) CursorFg() HexColor { return ProjectAdapterColors(u).CursorFg }

// SelectionBg is the adapter selection background role.
func (u UIColors) SelectionBg() HexColor { return ProjectAdapterColors(u).SelectionBg }

// SelectionFg is the adapter selection foreground role.
func (u UIColors) SelectionFg() HexColor { return ProjectAdapterColors(u).SelectionFg }

// PanelBg is the adapter panel background role.
func (u UIColors) PanelBg() HexColor { return ProjectAdapterColors(u).PanelBg }

// SidebarBg is the adapter sidebar background role.
func (u UIColors) SidebarBg() HexColor { return ProjectAdapterColors(u).SidebarBg }

// ActiveRowBg is the adapter active row background role.
func (u UIColors) ActiveRowBg() HexColor { return ProjectAdapterColors(u).ActiveRowBg }

// NavigateRowBg is the adapter navigate row background role.
func (u UIColors) NavigateRowBg() HexColor { return ProjectAdapterColors(u).NavigateRowBg }

// SurfaceDim is the adapter dim surface role.
func (u UIColors) SurfaceDim() HexColor { return ProjectAdapterColors(u).SurfaceDim }

// Surface0 is the adapter surface0 role.
func (u UIColors) Surface0() HexColor { return ProjectAdapterColors(u).Surface0 }

// Surface1 is the adapter surface1 role.
func (u UIColors) Surface1() HexColor { return ProjectAdapterColors(u).Surface1 }

// Overlay0 is the adapter overlay0 role.
func (u UIColors) Overlay0() HexColor { return ProjectAdapterColors(u).Overlay0 }

// Overlay1 is the adapter overlay1 role.
func (u UIColors) Overlay1() HexColor { return ProjectAdapterColors(u).Overlay1 }

// Subtext is the adapter subtext role.
func (u UIColors) Subtext() HexColor { return ProjectAdapterColors(u).Subtext }

// AdapterColors holds the operational roles derived from the canonical designer palette.
type AdapterColors struct {
	CursorBg      HexColor
	CursorFg      HexColor
	SelectionBg   HexColor
	SelectionFg   HexColor
	PanelBg       HexColor
	SidebarBg     HexColor
	ActiveRowBg   HexColor
	NavigateRowBg HexColor
	SurfaceDim    HexColor
	Surface0      HexColor
	Surface1      HexColor
	Overlay0      HexColor
	Overlay1      HexColor
	Subtext       HexColor
}

// ProjectAdapterColors maps canonical tokens to every adapter role deterministically.
func ProjectAdapterColors(ui UIColors) AdapterColors {
	return AdapterColors{
		CursorBg:      ui.Cursor,
		CursorFg:      ui.CursorText,
		SelectionBg:   ui.SelectionBackground,
		SelectionFg:   ui.SelectionForeground,
		PanelBg:       ui.Surface,
		SidebarBg:     ui.Background,
		ActiveRowBg:   ui.SelectionBackground,
		NavigateRowBg: ui.Overlay,
		SurfaceDim:    ui.Background,
		Surface0:      ui.Surface,
		Surface1:      ui.Overlay,
		Overlay0:      ui.Border,
		Overlay1:      ui.Muted,
		Subtext:       ui.Subtle,
	}
}

// SemanticColors are the hue families that keep their terminal meanings.
type SemanticColors struct {
	Red     HexColor
	Green   HexColor
	Yellow  HexColor
	Blue    HexColor
	Magenta HexColor
	Cyan    HexColor
	Orange  HexColor
}

// Colors returns the hues in SemanticFields order.
func (s SemanticColors) Colors() []HexColor {
	return []HexColor{s.Red, s.Green, s.Yellow, s.Blue, s.Magenta, s.Cyan, s.Orange}
}

func semanticColorsFrom(c map[string]HexColor) SemanticColors {
	return SemanticColors{
		Red: c["red"], Green: c["green"], Yellow: c["yellow"], Blue: c["blue"],
		Magenta: c["magenta"], Cyan: c["cyan"], Orange: c["orange"],
	}
}

// AnsiColors are the named ANSI slots.
type AnsiColors struct {
	Black   HexColor
	Red     HexColor
	Green   HexColor
	Yellow  HexColor
	Blue    HexColor
	Magenta HexColor
	Cyan    HexColor
	White   HexColor
}

// Colors returns the slots in ANSIOrder.
func (a AnsiColors) Colors() []HexColor {
	return []HexColor{a.Black, a.Red, a.Green, a.Yellow, a.Blue, a.Magenta, a.Cyan, a.White}
}

func ansiColorsFrom(c map[string]HexColor) AnsiColors {
	return AnsiColors{
		Black: c["black"], Red: c["red"], Green: c["green"], Yellow: c["yellow"],
		Blue: c["blue"], Magenta: c["magenta"], Cyan: c["cyan"], White: c["white"],
	}
}

// Theme is a fully resolved Street Fighter II theme.
type Theme struct {
	Metadata   ThemeMetadata
	UI         UIColors
	Semantic   SemanticColors
	AnsiNormal AnsiColors
	AnsiBright AnsiColors
}

// LookupKeys returns the id and aliases that resolve to this theme.
func (t Theme) LookupKeys() []string {
	return append([]string{t.Metadata.ID}, t.Metadata.Aliases...)
}

// SelectableID prefixes an adapter-facing identity while leaving catalog ids unchanged.
func SelectableID(themeID string) string {
	if strings.HasPrefix(themeID, SelectablePrefix) {
		return themeID
	}
	return SelectablePrefix + themeID
}

// ParseHex parses an exact #RRGGBB color into a lowercase HexColor.
func ParseHex(raw string) (HexColor, error) {
	if !hexPattern.MatchString(raw) {
		return "", fmt.Errorf("color must be #RRGGBB, got %s", pycompat.Repr(raw))
	}
	return HexColor(strings.ToLower(raw)), nil
}

// ThemeToTOML renders a theme back to canonical TOML for `show`.
func ThemeToTOML(theme Theme) string {
	meta := theme.Metadata
	lines := []string{
		"schema_version = 1",
		"",
		"[meta]",
		fmt.Sprintf(`id = "%s"`, meta.ID),
		fmt.Sprintf(`display_name = "%s"`, meta.DisplayName),
		fmt.Sprintf(`kind = "%s"`, meta.Kind),
		fmt.Sprintf(`introduced_in = "%s"`, meta.IntroducedIn),
	}
	if meta.Character != nil {
		lines = append(lines, fmt.Sprintf(`character = "%s"`, *meta.Character))
	}
	aliases := make([]string, len(meta.Aliases))
	for i, alias := range meta.Aliases {
		aliases[i] = `"` + alias + `"`
	}
	lines = append(lines,
		fmt.Sprintf("aliases = [%s]", strings.Join(aliases, ", ")),
		fmt.Sprintf(`name = "%s"`, meta.Name),
		fmt.Sprintf(`variant = "%s"`, meta.Variant),
		fmt.Sprintf(`family = "%s"`, meta.Family),
		fmt.Sprintf(`stage = "%s"`, meta.Stage),
	)
	section := func(header string, names []string, colors []HexColor) {
		lines = append(lines, "", header)
		for i, name := range names {
			lines = append(lines, fmt.Sprintf(`%s = "%s"`, name, colors[i]))
		}
	}
	section("[ui]", UIColorFields, theme.UI.Colors())
	section("[semantic]", SemanticFields, theme.Semantic.Colors())
	section("[ansi.normal]", ANSIOrder, theme.AnsiNormal.Colors())
	section("[ansi.bright]", ANSIOrder, theme.AnsiBright.Colors())
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}
