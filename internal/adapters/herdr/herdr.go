// Package herdr owns the marked theme block in the Herdr TUI config.toml: an auto-switching dark and light pair with
// the SF2 palette as per-mode custom overlays.
package herdr

import (
	"os"
	"regexp"
	"strings"

	"github.com/douglasjarquin/sf2-themes/v2/internal/adapters"
	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
	"github.com/douglasjarquin/sf2-themes/v2/internal/plan"
	"github.com/douglasjarquin/sf2-themes/v2/internal/pycompat"
)

// App is the Herdr adapter.
var App = adapters.App{
	Label:   "Herdr",
	Inspect: Inspect,
	Plan:    Plan,
	Pair:    true,
	Reload:  "run `herdr server reload-config` to apply the selection",
}

const (
	managedStart = "# >>> sf2-themes managed theme"
	managedEnd   = "# <<< sf2-themes managed theme"
)

var (
	idComment     = regexp.MustCompile(`^# sf2-themes:` + pycompat.Space + `+(` + pycompat.NotSpace + `+)` + pycompat.Space + `*$`)
	topLevelTheme = regexp.MustCompile(`^theme` + pycompat.Space + `*=`)
)

// herdrTokens maps each Herdr custom token to its SF2 color.
//
// Herdr paints the active tab chip and the focused split border with the same `accent` token. Prefer primary accent
// for both; `name = "terminal"` left those as ANSI Blue (WezTerm's semantic blue), which is wrong for SF2 palettes.
//
// Do not map Herdr row fills to ui.selection_background: that token is a punchy terminal-selection tint and reads as
// an off-palette purple slab in the sidebar. Use surface steps instead (active = surface, navigate cursor = overlay).
//
// Herdr's overlay0/overlay1 are muted *text* roles (catppuccin #6c7086/#7f849c), not chrome borders. The shared
// adapter overlay0 is ui.border for WezTerm/nvim/codex; remap here only so dim sidebar labels stay readable.
func herdrTokens(theme catalog.Theme) [][2]string {
	adapter := catalog.ProjectAdapterColors(theme.UI)
	ui := theme.UI
	semantic := theme.Semantic
	pairs := []struct {
		token string
		color catalog.HexColor
	}{
		{"sidebar_bg", adapter.SidebarBg},
		{"panel_bg", adapter.PanelBg},
		{"active_row_bg", adapter.Surface0},
		{"selection_bg", adapter.Surface1},
		{"surface0", adapter.Surface0},
		{"surface1", adapter.Surface1},
		{"surface_dim", adapter.SurfaceDim},
		{"overlay0", ui.Muted},
		{"overlay1", ui.Subtle},
		{"text", ui.Foreground},
		{"subtext0", adapter.Subtext},
		{"accent", ui.Accent},
		{"mauve", ui.AccentSecondary},
		{"green", semantic.Green},
		{"yellow", semantic.Yellow},
		{"red", semantic.Red},
		{"blue", semantic.Blue},
		{"teal", semantic.Cyan},
		{"peach", semantic.Orange},
	}
	tokens := make([][2]string, len(pairs))
	for i, pair := range pairs {
		tokens[i] = [2]string{pair.token, string(pair.color)}
	}
	return tokens
}

// baseThemeName returns a concrete Herdr base so ANSI Blue never leaks into chrome.
func baseThemeName(theme catalog.Theme) string {
	if theme.Metadata.Variant == catalog.VariantLight {
		return "catppuccin-latte"
	}
	return "catppuccin"
}

// configPath is the Herdr config.toml path.
func configPath(configDir string) (string, error) {
	if configDir != "" {
		return pycompat.Path(configDir, "config.toml"), nil
	}
	if configured := os.Getenv("HERDR_CONFIG_PATH"); configured != "" {
		return pycompat.ExpandUser(configured)
	}
	var root string
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		expanded, err := pycompat.ExpandUser(xdg)
		if err != nil {
			return "", err
		}
		root = expanded
	} else {
		home, err := pycompat.Home()
		if err != nil {
			return "", err
		}
		root = pycompat.Path(home, ".config")
	}
	return pycompat.Path(root, "herdr", "config.toml"), nil
}

func customTokenLines(theme catalog.Theme) []string {
	tokens := herdrTokens(theme)
	lines := make([]string, len(tokens))
	for i, token := range tokens {
		lines[i] = token[0] + ` = "` + token[1] + `"`
	}
	return lines
}

// renderBlock renders a marked Herdr block that auto-switches a dark/light pair, including the SF2 character
// palette itself (requires herdr#2324).
func renderBlock(dark, light catalog.Theme) string {
	lines := []string{
		managedStart,
		"# sf2-themes: " + dark.Metadata.SelectableID(),
		"[theme]",
		`name = "` + baseThemeName(dark) + `"`,
		"auto_switch = true",
		`light_name = "` + baseThemeName(light) + `"`,
		`dark_name = "` + baseThemeName(dark) + `"`,
		"",
		"[theme.custom.dark]",
	}
	lines = append(lines, customTokenLines(dark)...)
	lines = append(lines, "", "[theme.custom.light]")
	lines = append(lines, customTokenLines(light)...)
	lines = append(lines, managedEnd, "")
	return strings.Join(lines, "\n")
}

// tableName returns the name inside a `[...]` header line, already stripped, and whether the line is one.
func tableName(stripped string) (string, bool) {
	if !strings.HasPrefix(stripped, "[") || !strings.HasSuffix(stripped, "]") {
		return "", false
	}
	return pycompat.Strip(strings.Trim(stripped, "[]")), true
}

func isThemeTable(name string) bool {
	return name == "theme" || strings.HasPrefix(name, "theme.")
}

func hasUnmarkedTheme(content string) bool {
	if strings.Contains(content, managedStart) {
		return false
	}
	current := ""
	for _, line := range pycompat.SplitLines(content) {
		stripped := pycompat.Strip(line)
		if name, ok := tableName(stripped); ok {
			current = name
			if isThemeTable(current) {
				return true
			}
			continue
		}
		if current == "" && topLevelTheme.MatchString(stripped) {
			return true
		}
	}
	return false
}

func stripThemeSections(content string) string {
	var kept []string
	skip := false
	current := ""
	for _, line := range pycompat.SplitLines(content) {
		stripped := pycompat.Strip(line)
		if name, ok := tableName(stripped); ok {
			current = name
			skip = isThemeTable(current)
		}
		topLevel := current == "" && topLevelTheme.MatchString(stripped)
		if skip || topLevel {
			continue
		}
		kept = append(kept, line)
	}
	return pycompat.RStrip(strings.Join(kept, "\n"))
}

// markers returns the managed block's start and end offsets, -1 for a missing one, and false when both are absent.
func markers(content string) (start, end int, present bool) {
	if !strings.Contains(content, managedStart) && !strings.Contains(content, managedEnd) {
		return 0, 0, false
	}
	return strings.Index(content, managedStart), strings.Index(content, managedEnd), true
}

// mergeTheme inserts or replaces the managed Herdr block, replacing any unmarked theme, and parses the result so
// preservation never produces invalid Herdr configuration.
func mergeTheme(existing string, theme catalog.Theme, themes []catalog.Theme) (string, error) {
	dark, light, err := catalog.ThemePair(theme, themes)
	if err != nil {
		return "", err
	}
	block := renderBlock(dark, light)
	var merged string
	if start, end, present := markers(existing); present {
		after := end + len(managedEnd)
		if after < len(existing) && existing[after] == '\n' {
			after++
		}
		merged = existing[:start] + block + existing[after:]
	} else if hasUnmarkedTheme(existing) {
		merged = block
		if preserved := stripThemeSections(existing); preserved != "" {
			merged = preserved + "\n\n" + block
		}
	} else if pycompat.Strip(existing) != "" {
		merged = pycompat.RStrip(existing) + "\n\n" + block
	} else {
		merged = block
	}
	if _, err := catalog.DecodeTOML(merged); err != nil {
		return "", err
	}
	if !strings.HasSuffix(merged, "\n") {
		merged += "\n"
	}
	return merged, nil
}

// Inspect classifies the theme namespace of config.toml.
func Inspect(themes []catalog.Theme, configDir string) (plan.Inspection, error) {
	path, err := configPath(configDir)
	if err != nil {
		return plan.Inspection{}, err
	}
	existing, err := plan.ReadExisting(path)
	if err != nil {
		return plan.Inspection{}, err
	}
	if _, err := catalog.DecodeTOML(existing); err != nil {
		return plan.Inspection{Ownership: plan.Broken, Detail: path + " is not valid TOML: " + err.Error()}, nil
	}
	if start, end, present := markers(existing); present {
		if start == -1 || end == -1 || end < start {
			return plan.Inspection{Ownership: plan.Broken, Detail: path + " has incomplete sf2-themes markers"}, nil
		}
		id, found := plan.RecordedID(existing, idComment)
		return plan.Recorded(plan.Managed, id, found, themes), nil
	}
	if hasUnmarkedTheme(existing) {
		return plan.Inspection{Ownership: plan.Foreign}, nil
	}
	return plan.Inspection{Ownership: plan.Absent}, nil
}

// Plan returns the managed theme block for a dark/light pair, or nothing when slot keeps the setting.
func Plan(theme catalog.Theme, themes []catalog.Theme, configDir string, slot bool) ([]plan.Step, error) {
	if !slot {
		return nil, nil
	}
	path, err := configPath(configDir)
	if err != nil {
		return nil, err
	}
	existing, err := plan.ReadExisting(path)
	if err != nil {
		return nil, err
	}
	merged, err := mergeTheme(existing, theme, themes)
	if err != nil {
		return nil, err
	}
	return []plan.Step{plan.Write(path, merged)}, nil
}
