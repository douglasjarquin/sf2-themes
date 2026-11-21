// Package wezterm writes WezTerm color-scheme files and the appearance-following pointer, and integrates the pointer
// into wezterm.lua when the ownership gate allows it.
package wezterm

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/douglasjarquin/sf2-themes/v2/internal/adapters"
	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
	"github.com/douglasjarquin/sf2-themes/v2/internal/fsys"
	"github.com/douglasjarquin/sf2-themes/v2/internal/plan"
	"github.com/douglasjarquin/sf2-themes/v2/internal/pycompat"
)

// label names WezTerm in summaries and in the multi-line refusals printed for a user to act on.
const label = "WezTerm"

// App is the WezTerm adapter.
var App = adapters.App{
	Label:   label,
	Inspect: Inspect,
	Plan:    Plan,
	Target:  target,
	Pair:    true,
	Pointer: true,
	Reload:  "restart WezTerm, or let a config reload pick up the managed pointer",
	Disowns: disowns,
}

// disowns reports whether path is a selection pointer other than the target's own: one an earlier release shared, or
// one a different target owns. sf2 never writes or removes such a file for this target.
func disowns(configDir, path string) bool {
	if !pointerName.MatchString(pycompat.Name(path)) {
		return false
	}
	own, err := pointerPath(configDir)
	return err == nil && !fsys.SamePath(own, path)
}

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
	spelled, err := spelledLuaPath(configDir)
	if err != nil {
		return "", err
	}
	_, err = fsys.Resolve(spelled)
	return spelled, err
}

// target is the target's one identity: the wezterm.lua the filesystem opens, through any symlink. Inspection reads,
// the commit writes, the pointer is named after, and the manifest records this one file.
func target(configDir string) (string, error) {
	spelled, err := spelledLuaPath(configDir)
	if err != nil {
		return "", err
	}
	return fsys.Resolve(spelled)
}

func spelledLuaPath(configDir string) (string, error) {
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

// readLua returns wezterm.lua as written, line breaks included, so an edit changes only the lines sf2 writes; or ""
// when it does not exist yet.
func readLua(path string) (string, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !utf8.Valid(data) {
		return "", fmt.Errorf("%s: %w", path, pycompat.ErrInvalidUTF8)
	}
	return string(data), nil
}

func colorsDir(configDir string) (string, error) {
	dir, err := weztermDir(configDir)
	if err != nil {
		return "", err
	}
	colors := pycompat.Path(dir, "colors")
	_, err = fsys.Resolve(colors)
	return colors, err
}

// pointerPath is the target's own selection pointer: `sf2-theme/wezterm-current-<digest>.lua` beside the
// wezterm.lua the filesystem opens, named after that file's canonical path.
//
// A target is the file itself, however --config-dir, WEZTERM_CONFIG_FILE, a symlink, `..`, or letter case on a
// case-insensitive volume spelled it, so each spelling of one file shares one pointer, distinct files never share
// one, and WEZTERM_CONFIG_DIR does not move it. No pointer is ever a `wezterm-current.lua` that earlier releases shared
// between targets, so writing a target's pointer never changes what another target loads.
func pointerPath(configDir string) (string, error) {
	resolved, err := target(configDir)
	if err != nil {
		return "", err
	}
	pointer := pycompat.Path(pycompat.Parent(resolved), "sf2-theme", "wezterm-current-"+fsys.Fingerprint(resolved)[:12]+".lua")
	if strings.IndexFunc(pointer, unicode.IsControl) >= 0 {
		return "", fmt.Errorf("refusing %s: its path holds a control character sf2-themes will not write into Lua", resolved)
	}
	return pointer, nil
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

// Inspect takes ownership from wezterm.lua and the selection from the pointer its recognized integration loads: the
// target's own, or one an earlier release wrote, until sf2 moves the integration to the target's own. A config whose
// scheme sf2 does not set selects nothing sf2 reports.
func Inspect(themes []catalog.Theme, configDir string) (plan.Inspection, error) {
	pointer, err := pointerPath(configDir)
	if err != nil {
		return plan.Inspection{}, err
	}
	lua, err := luaPath(configDir)
	if err != nil {
		return plan.Inspection{}, err
	}
	existing, err := readLua(lua)
	if err != nil {
		return plan.Inspection{}, err
	}
	analysis := analyze(existing, pointer, lua)
	selection := plan.Inspection{Ownership: analysis.ownership}
	isFile := false
	if analysis.selector != "" {
		if isFile, err = fsys.IsFile(analysis.selector); err != nil {
			return plan.Inspection{}, err
		}
	}
	if isFile {
		text, err := pycompat.ReadText(analysis.selector)
		if err != nil {
			return plan.Inspection{}, err
		}
		id, found := plan.RecordedID(text, idComment)
		selection = plan.Recorded(analysis.ownership, id, found, themes)
	}
	selection.Detail = analysis.manual
	return selection, nil
}

// Plan returns the scheme files and the pointer, and the wezterm.lua integration when slot allows it. An
// integration sf2 cannot write safely is refused here, before anything is written.
func Plan(theme catalog.Theme, themes []catalog.Theme, configDir string, slot bool) ([]plan.Step, error) {
	target, err := colorsDir(configDir)
	if err != nil {
		return nil, err
	}
	steps := make([]plan.Step, 0, len(themes)+2)
	for _, entry := range themes {
		steps = append(steps, plan.Write(pycompat.Path(target, schemeFilename(entry)), renderScheme(entry)))
	}
	dark, light, err := catalog.ThemePair(theme, themes)
	if err != nil {
		return nil, err
	}
	pointer, err := pointerPath(configDir)
	if err != nil {
		return nil, err
	}
	steps = append(steps, plan.Write(pointer, renderPointer(dark, light)))
	if slot {
		lua, err := luaPath(configDir)
		if err != nil {
			return nil, err
		}
		existing, err := readLua(lua)
		if err != nil {
			return nil, err
		}
		integrated, err := integrate(existing, pointer, lua)
		if err != nil {
			return nil, errors.New(label + " config was left unchanged and nothing was written.\n" + err.Error() +
				"\n\nThen run the command again.")
		}
		steps = append(steps, plan.Write(lua, integrated))
	}
	return steps, nil
}
