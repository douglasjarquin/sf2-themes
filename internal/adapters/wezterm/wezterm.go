// Package wezterm writes WezTerm color-scheme files and the appearance-following pointer, and integrates the pointer
// into wezterm.lua when the ownership gate allows it.
package wezterm

import (
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/douglasjarquin/sf2-themes/v2/internal/adapters"
	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
	"github.com/douglasjarquin/sf2-themes/v2/internal/fsys"
	"github.com/douglasjarquin/sf2-themes/v2/internal/plan"
	"github.com/douglasjarquin/sf2-themes/v2/internal/pycompat"
)

// App is the WezTerm adapter.
var App = adapters.App{
	Label:                   "WezTerm",
	Inspect:                 Inspect,
	Plan:                    Plan,
	Pair:                    true,
	OwnsFilesOnUnknownShape: true,
	Reload:                  "restart WezTerm, or let a config reload pick up the managed pointer",
}

const legacySchemeFilePrefix = "street-fighter-ii"

var idComment = regexp.MustCompile(`^-- sf2-themes:` + pycompat.Space + `*(` + pycompat.NotSpace + `+)` + pycompat.Space + `*$`)

func configRoot() (string, error) {
	if configured := os.Getenv("XDG_CONFIG_HOME"); configured != "" {
		return pycompat.ExpandUser(configured)
	}
	home, err := pycompat.Home()
	return pycompat.Path(home, ".config"), err
}

func weztermDir(configDir string) (string, error) {
	if configDir != "" {
		return configDir, nil
	}
	if configured := os.Getenv("WEZTERM_CONFIG_DIR"); configured != "" {
		return pycompat.ExpandUser(configured)
	}
	root, err := configRoot()
	return pycompat.Path(root, "wezterm"), err
}

// luaPath is the wezterm.lua path to read or create.
//
// XDG_CONFIG_HOME and WEZTERM_CONFIG_FILE pin the write target even when the file does not exist yet, so tests and
// custom prefixes never fall through to the real home directory.
func luaPath(configDir string) (string, error) {
	if configDir != "" {
		return pycompat.Path(configDir, "wezterm.lua"), nil
	}
	if configured := os.Getenv("WEZTERM_CONFIG_FILE"); configured != "" {
		return pycompat.ExpandUser(configured)
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		root, err := pycompat.ExpandUser(xdg)
		return pycompat.Path(root, "wezterm", "wezterm.lua"), err
	}
	home, err := pycompat.Home()
	if err != nil {
		return "", err
	}
	defaultPath := pycompat.Path(home, ".config", "wezterm", "wezterm.lua")
	homeLua := pycompat.Path(home, ".wezterm.lua")
	defaultExists, err := fsys.Exists(defaultPath)
	if err != nil {
		return "", err
	}
	if defaultExists {
		return defaultPath, nil
	}
	homeExists, err := fsys.Exists(homeLua)
	if err != nil || !homeExists {
		return defaultPath, err
	}
	return homeLua, nil
}

func colorsDir(configDir string) (string, error) {
	dir, err := weztermDir(configDir)
	return pycompat.Path(dir, "colors"), err
}

func pointerPath() (string, error) {
	root, err := configRoot()
	return pycompat.Path(root, "sf2-theme", "wezterm-current.lua"), err
}

func schemeFilename(theme catalog.Theme) string {
	return theme.Metadata.SelectableID() + ".toml"
}

func quotedColors(colors []catalog.HexColor) string {
	quoted := make([]string, len(colors))
	for i, color := range colors {
		quoted[i] = `"` + string(color) + `"`
	}
	return strings.Join(quoted, ", ")
}

func tabStyle(header string, bg, fg catalog.HexColor) []string {
	return []string{
		header,
		`bg_color = "` + string(bg) + `"`,
		`fg_color = "` + string(fg) + `"`,
		`intensity = "Normal"`,
		"italic = false",
		"strikethrough = false",
		`underline = "None"`,
		"",
	}
}

// renderScheme renders a WezTerm color scheme TOML file.
//
// It covers every key Catppuccin-style schemes ship in TOML: core chrome, ANSI, indexed extras, compose/bell cues,
// and retro tab_bar. Fancy-tab window_frame colors are Lua-only and stay out of scheme files.
func renderScheme(theme catalog.Theme) string {
	ui := theme.UI
	adapter := catalog.ProjectAdapterColors(ui)
	semantic := theme.Semantic
	q := func(key string, color catalog.HexColor) string { return key + ` = "` + string(color) + `"` }
	lines := []string{
		"[colors]",
		q("foreground", ui.Foreground),
		q("background", ui.Background),
		q("cursor_bg", adapter.CursorBg),
		q("cursor_fg", adapter.CursorFg),
		q("cursor_border", adapter.CursorBg),
		q("selection_bg", adapter.SelectionBg),
		q("selection_fg", adapter.SelectionFg),
		q("split", adapter.Surface1),
		q("scrollbar_thumb", ui.Muted),
		q("compose_cursor", ui.AccentSecondary),
		q("visual_bell", ui.Overlay),
		"ansi = [" + quotedColors(theme.AnsiNormal.Colors()) + "]",
		"brights = [" + quotedColors(theme.AnsiBright.Colors()) + "]",
		"",
		"[colors.indexed]",
		q("16", semantic.Orange),
		q("17", ui.Accent),
		"",
		"[colors.tab_bar]",
		q("background", ui.Background),
		q("inactive_tab_edge", ui.Border),
		"",
	}
	lines = append(lines, tabStyle("[colors.tab_bar.active_tab]", ui.Accent, ui.CursorText)...)
	lines = append(lines, tabStyle("[colors.tab_bar.inactive_tab]", ui.Surface, ui.Muted)...)
	lines = append(lines, tabStyle("[colors.tab_bar.inactive_tab_hover]", ui.Overlay, ui.Foreground)...)
	lines = append(lines, tabStyle("[colors.tab_bar.new_tab]", ui.Surface, ui.Muted)...)
	lines = append(lines, tabStyle("[colors.tab_bar.new_tab_hover]", ui.Overlay, ui.Foreground)...)
	lines = append(lines, "[metadata]", `name = "`+theme.Metadata.SelectableID()+`"`)
	aliases := append([]string{}, theme.Metadata.Aliases...)
	if theme.Metadata.ID == "main" && !slices.Contains(aliases, "street-fighter-2") {
		aliases = append([]string{"street-fighter-2"}, aliases...)
	}
	rendered := make([]string, len(aliases))
	for i, alias := range aliases {
		rendered[i] = `"` + catalog.SelectableID(alias) + `"`
	}
	lines = append(lines, "aliases = ["+strings.Join(rendered, ", ")+"]", "")
	return strings.Join(lines, "\n")
}

func renderPointer(dark, light catalog.Theme) string {
	return strings.Join([]string{
		"-- sf2-themes: " + dark.Metadata.SelectableID(),
		`local wezterm = require("wezterm")`,
		`local appearance = wezterm.gui and wezterm.gui.get_appearance() or "Dark"`,
		`if appearance:find("Dark") then`,
		`  return "` + dark.Metadata.SelectableID() + `"`,
		"end",
		`return "` + light.Metadata.SelectableID() + `"`,
		"",
	}, "\n")
}

// Inspect takes ownership from wezterm.lua and the selection from the sf2-owned pointer.
func Inspect(themes []catalog.Theme, configDir string) (plan.Inspection, error) {
	pointer, err := pointerPath()
	if err != nil {
		return plan.Inspection{}, err
	}
	lua, err := luaPath(configDir)
	if err != nil {
		return plan.Inspection{}, err
	}
	existing, err := plan.ReadExisting(lua)
	if err != nil {
		return plan.Inspection{}, err
	}
	ownership := classify(existing, pointer)
	selection := plan.Inspection{Ownership: ownership}
	isFile, err := fsys.IsFile(pointer)
	if err != nil {
		return plan.Inspection{}, err
	}
	if isFile {
		text, err := pycompat.ReadText(pointer)
		if err != nil {
			return plan.Inspection{}, err
		}
		id, found := plan.RecordedID(text, idComment)
		selection = plan.Recorded(ownership, id, found, themes)
	}
	if ownership == plan.UnknownShape {
		selection.Detail = "Add this integration to " + lua + ":\n\n" + integrationSnippet(pointer, "config")
	}
	return selection, nil
}

// Plan returns the scheme files and the pointer, and the wezterm.lua integration when slot allows it.
func Plan(theme catalog.Theme, themes []catalog.Theme, configDir string, slot bool) ([]plan.Step, error) {
	target, err := colorsDir(configDir)
	if err != nil {
		return nil, err
	}
	steps := make([]plan.Step, 0, 2*len(themes)+2)
	for _, entry := range themes {
		steps = append(steps, plan.Remove(pycompat.Path(target, legacySchemeFilePrefix+"-"+entry.Metadata.ID+".toml")))
	}
	for _, entry := range themes {
		steps = append(steps, plan.Write(pycompat.Path(target, schemeFilename(entry)), renderScheme(entry)))
	}
	dark, light, err := catalog.ThemePair(theme, themes)
	if err != nil {
		return nil, err
	}
	pointer, err := pointerPath()
	if err != nil {
		return nil, err
	}
	steps = append(steps, plan.Write(pointer, renderPointer(dark, light)))
	if slot {
		lua, err := luaPath(configDir)
		if err != nil {
			return nil, err
		}
		existing, err := plan.ReadExisting(lua)
		if err != nil {
			return nil, err
		}
		steps = append(steps, plan.Write(lua, integrate(existing, pointer)))
	}
	return steps, nil
}
