// Package lazygit installs SF2 theme fragments for Lazygit and selects one through managed blocks in config.yml.
//
// config.yml is edited line by line by indentation, with no YAML parser, so every unrelated line is kept as written.
package lazygit

import (
	"errors"
	"os"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/douglasjarquin/sf2-themes/v2/internal/adapters"
	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
	"github.com/douglasjarquin/sf2-themes/v2/internal/plan"
	"github.com/douglasjarquin/sf2-themes/v2/internal/pycompat"
)

// App is the Lazygit adapter.
var App = adapters.App{
	Label:   "Lazygit",
	Inspect: Inspect,
	Plan:    Plan,
	Reload:  "restart lazygit to render the new theme",
}

const (
	themeDirName       = "themes"
	configFileName     = "config.yml"
	themeSuffix        = ".yml"
	managedThemeStart  = "# >>> sf2-themes managed theme"
	managedThemeEnd    = "# <<< sf2-themes managed theme"
	managedAuthorStart = "# >>> sf2-themes managed author colors"
	managedAuthorEnd   = "# <<< sf2-themes managed author colors"
)

var idComment = regexp.MustCompile(`^# sf2-themes:` + pycompat.Space + `+(` + pycompat.NotSpace + `+)` + pycompat.Space + `*$`)

var errIncompleteMarkers = errors.New("incomplete sf2-themes markers")

// Dir is the Lazygit configuration directory: --config-dir, then LAZYGIT_CONFIG_DIR when set and not empty, then
// ~/Library/Application Support/lazygit on macOS, and `$XDG_CONFIG_HOME/lazygit` or ~/.config/lazygit elsewhere.
func Dir(configDir string) (string, error) {
	if configDir != "" {
		return configDir, nil
	}
	if configured := os.Getenv("LAZYGIT_CONFIG_DIR"); configured != "" {
		return pycompat.ExpandUser(configured)
	}
	if runtime.GOOS == "darwin" {
		userHome, err := pycompat.Home()
		if err != nil {
			return "", err
		}
		return pycompat.Path(userHome, "Library", "Application Support", "lazygit"), nil
	}
	var root string
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		expanded, err := pycompat.ExpandUser(xdg)
		if err != nil {
			return "", err
		}
		root = expanded
	} else {
		userHome, err := pycompat.Home()
		if err != nil {
			return "", err
		}
		root = pycompat.Path(userHome, ".config")
	}
	return pycompat.Path(root, "lazygit"), nil
}

type themeColor struct {
	key   string
	color catalog.HexColor
	bold  bool
}

func themeColors(theme catalog.Theme) []themeColor {
	adapter := catalog.ProjectAdapterColors(theme.UI)
	return []themeColor{
		{"activeBorderColor", theme.UI.Accent, true},
		{"inactiveBorderColor", adapter.Overlay0, false},
		{"searchingActiveBorderColor", theme.Semantic.Yellow, false},
		{"optionsTextColor", theme.UI.Accent, false},
		{"selectedLineBgColor", adapter.SelectionBg, false},
		{"inactiveViewSelectedLineBgColor", adapter.Surface0, false},
		{"cherryPickedCommitFgColor", theme.Semantic.Green, false},
		{"cherryPickedCommitBgColor", adapter.Surface1, false},
		{"markedBaseCommitFgColor", theme.Semantic.Orange, false},
		{"markedBaseCommitBgColor", adapter.Surface1, false},
		{"unstagedChangesColor", theme.Semantic.Red, false},
		{"defaultFgColor", theme.UI.Foreground, false},
	}
}

// themeSectionLines is the `  theme:` mapping under `gui:`.
func themeSectionLines(theme catalog.Theme) []string {
	lines := []string{"  theme:"}
	for _, entry := range themeColors(theme) {
		lines = append(lines, "    "+entry.key+":", "      - '"+string(entry.color)+"'")
		if entry.bold {
			lines = append(lines, "      - bold")
		}
	}
	return lines
}

// RenderTheme returns the complete Lazygit theme fragment for one catalog theme.
func RenderTheme(theme catalog.Theme) string {
	lines := append([]string{"gui:"}, themeSectionLines(theme)...)
	lines = append(lines, "  authorColors:", "    '*': '"+string(theme.UI.AccentSecondary)+"'")
	return strings.Join(lines, "\n") + "\n"
}

// marked returns the start and end line of a managed block, found=false when it is absent, or
// errIncompleteMarkers when the markers are incomplete or out of order.
func marked(lines []string, startMarker, endMarker string) (start, end int, found bool, err error) {
	var starts, ends []int
	for i, line := range lines {
		switch pycompat.Strip(line) {
		case startMarker:
			starts = append(starts, i)
		case endMarker:
			ends = append(ends, i)
		}
	}
	if len(starts) == 0 && len(ends) == 0 {
		return 0, 0, false, nil
	}
	if len(starts) != 1 || len(ends) != 1 || ends[0] < starts[0] {
		return 0, 0, false, errIncompleteMarkers
	}
	return starts[0], ends[0], true, nil
}

// replaceMarked swaps a complete managed block for replacement and leaves lines unchanged when there is none.
// The caller has already inspected the markers as complete.
func replaceMarked(lines []string, startMarker, endMarker string, replacement []string) []string {
	start, end, found, err := marked(lines, startMarker, endMarker)
	if err != nil || !found {
		return lines
	}
	return slices.Concat(lines[:start], replacement, lines[end+1:])
}

func isWildcardAuthorLine(line string) bool {
	key, _, separator := strings.Cut(pycompat.Strip(line), ":")
	if !separator {
		return false
	}
	switch pycompat.Strip(key) {
	case "*", "'*'", `"*"`:
		return true
	}
	return false
}

func normalizedKey(line string) string {
	key, _, _ := strings.Cut(line, " #")
	return pycompat.RStrip(key)
}

// lineIsTopLevel reports whether a non-empty YAML line starts at document level.
func lineIsTopLevel(line string) bool {
	return pycompat.Strip(line) != "" &&
		!strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, "#")
}

func topLevelSection(lines []string, name string) (start, end int, found bool) {
	start = slices.IndexFunc(lines, func(line string) bool { return normalizedKey(line) == name+":" })
	if start < 0 {
		return 0, 0, false
	}
	for i := start + 1; i < len(lines); i++ {
		if lineIsTopLevel(lines[i]) {
			return start, i, true
		}
	}
	return start, len(lines), true
}

func guiSubsectionBounds(lines []string, name string) (start, end int, found bool) {
	guiStart, guiEnd, found := topLevelSection(lines, "gui")
	if !found {
		return 0, 0, false
	}
	section := -1
	for i := guiStart + 1; i < guiEnd; i++ {
		if normalizedKey(lines[i]) == "  "+name+":" {
			section = i
			break
		}
	}
	if section < 0 {
		return 0, 0, false
	}
	for i := section + 1; i < guiEnd; i++ {
		if strings.HasPrefix(lines[i], "  ") && !strings.HasPrefix(lines[i], "    ") {
			return section, i, true
		}
	}
	return section, guiEnd, true
}

func managedThemeLines(theme catalog.Theme) []string {
	lines := []string{"  " + managedThemeStart, "  # sf2-themes: " + theme.Metadata.SelectableID()}
	lines = append(lines, themeSectionLines(theme)...)
	return append(lines, "  "+managedThemeEnd)
}

func managedAuthorLines(theme catalog.Theme) []string {
	return []string{
		"  " + managedAuthorStart,
		"  # sf2-themes: " + theme.Metadata.SelectableID(),
		"  authorColors:",
		"    '*': '" + string(theme.UI.AccentSecondary) + "'",
		"  " + managedAuthorEnd,
	}
}

func managedAuthorEntryLines(theme catalog.Theme) []string {
	const indent = "    "
	return []string{
		indent + managedAuthorStart,
		indent + "# sf2-themes: " + theme.Metadata.SelectableID(),
		indent + "'*': '" + string(theme.UI.AccentSecondary) + "'",
		indent + managedAuthorEnd,
	}
}

func containsMarker(lines []string, marker string) bool {
	return slices.ContainsFunc(lines, func(line string) bool { return pycompat.Strip(line) == marker })
}

// MergeConfig merges the selected theme into config.yml, replacing any unmarked gui.theme.
func MergeConfig(existing string, theme catalog.Theme) string {
	lines := pycompat.SplitLines(existing)
	lines = replaceMarked(lines, managedThemeStart, managedThemeEnd, managedThemeLines(theme))
	markerAt := slices.IndexFunc(lines, func(line string) bool { return pycompat.Strip(line) == managedAuthorStart })
	hadManagedAuthor := markerAt >= 0
	if hadManagedAuthor {
		marker := lines[markerAt]
		indent := utf8.RuneCountInString(marker) - utf8.RuneCountInString(pycompat.LStrip(marker))
		replacement := managedAuthorLines(theme)
		if indent >= 4 {
			replacement = managedAuthorEntryLines(theme)
		}
		lines = replaceMarked(lines, managedAuthorStart, managedAuthorEnd, replacement)
	}

	if start, end, found := guiSubsectionBounds(lines, "theme"); found {
		guiStart, guiEnd, guiFound := topLevelSection(lines, "gui")
		hasManagedTheme := guiFound && containsMarker(lines[guiStart:guiEnd], managedThemeStart)
		if !hasManagedTheme {
			lines = slices.Concat(lines[:start], managedThemeLines(theme), lines[end:])
		}
	} else if _, guiEnd, guiFound := topLevelSection(lines, "gui"); guiFound {
		lines = slices.Insert(lines, guiEnd, managedThemeLines(theme)...)
	} else {
		lines = append(append(lines, "", "gui:"), managedThemeLines(theme)...)
	}

	if !hadManagedAuthor {
		if start, end, found := guiSubsectionBounds(lines, "authorColors"); !found {
			_, guiEnd, _ := topLevelSection(lines, "gui")
			lines = slices.Insert(lines, guiEnd, managedAuthorLines(theme)...)
		} else if !containsMarker(lines[start:end], managedAuthorStart) {
			wildcard := -1
			for i := start + 1; i < end; i++ {
				if isWildcardAuthorLine(lines[i]) {
					wildcard = i
					break
				}
			}
			if wildcard < 0 {
				lines = slices.Insert(lines, end, managedAuthorEntryLines(theme)...)
			} else {
				lines = slices.Concat(lines[:wildcard], managedAuthorEntryLines(theme), lines[wildcard+1:])
			}
		}
	}
	return pycompat.RStrip(strings.Join(lines, "\n")) + "\n"
}

// Inspect classifies gui.theme in config.yml: a complete managed block is managed, an unmarked gui.theme is
// foreign, and incomplete or out-of-order markers are broken.
func Inspect(themes []catalog.Theme, configDir string) (plan.Inspection, error) {
	dir, err := Dir(configDir)
	if err != nil {
		return plan.Inspection{}, err
	}
	path := pycompat.Path(dir, configFileName)
	existing, err := plan.ReadExisting(path)
	if err != nil {
		return plan.Inspection{}, err
	}
	lines := pycompat.SplitLines(existing)
	start, end, themeFound, themeErr := marked(lines, managedThemeStart, managedThemeEnd)
	_, _, _, authorErr := marked(lines, managedAuthorStart, managedAuthorEnd)
	if themeErr != nil || authorErr != nil {
		return plan.Inspection{Ownership: plan.Broken, Detail: path + " has " + errIncompleteMarkers.Error()}, nil
	}
	if themeFound {
		id, found := plan.RecordedID(strings.Join(lines[start:end], "\n"), idComment)
		return plan.Recorded(plan.Managed, id, found, themes), nil
	}
	if _, _, found := guiSubsectionBounds(lines, "theme"); found {
		return plan.Inspection{Ownership: plan.Foreign}, nil
	}
	return plan.Inspection{Ownership: plan.Absent}, nil
}

// Plan writes every catalog fragment and, when slot is set, the selection in config.yml.
func Plan(theme catalog.Theme, themes []catalog.Theme, configDir string, slot bool) ([]plan.Step, error) {
	dir, err := Dir(configDir)
	if err != nil {
		return nil, err
	}
	steps := make([]plan.Step, 0, len(themes)+1)
	for _, entry := range themes {
		path := pycompat.Path(dir, themeDirName, entry.Metadata.SelectableID()+themeSuffix)
		steps = append(steps, plan.Write(path, RenderTheme(entry)))
	}
	if slot {
		path := pycompat.Path(dir, configFileName)
		existing, err := plan.ReadExisting(path)
		if err != nil {
			return nil, err
		}
		steps = append(steps, plan.Write(path, MergeConfig(existing, theme)))
	}
	return steps, nil
}
