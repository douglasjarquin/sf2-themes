// Package codex installs SF2 `.tmTheme` files for Codex and selects one through the `theme` assignment inside
// `[tui]` in config.toml.
package codex

import (
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/douglasjarquin/sf2-themes/v2/internal/adapters"
	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
	"github.com/douglasjarquin/sf2-themes/v2/internal/plan"
	"github.com/douglasjarquin/sf2-themes/v2/internal/pycompat"
)

// App is the Codex adapter.
var App = adapters.App{
	Label:   "Codex",
	Inspect: Inspect,
	Plan:    Plan,
	Target:  target,
	Reload:  "restart Codex, or reselect with /theme",
}

const (
	themeDirName   = "themes"
	configFileName = "config.toml"
	themeSuffix    = ".tmTheme"
)

var (
	themeAssignment = regexp.MustCompile(`^` + pycompat.Space + `*theme` + pycompat.Space + `*=`)
	// section matches a whole-line table header. A header followed by a comment does not match, so such a
	// `[tui]` table is not found and a second one is appended, as the reference does.
	section   = regexp.MustCompile(`^` + pycompat.Space + `*\[([^\]]+)\]` + pycompat.Space + `*$`)
	inlineTUI = regexp.MustCompile(`^` + pycompat.Space + `*tui` + pycompat.Space + `*[.=]`)
)

type colorRole struct {
	key   string
	color func(catalog.Theme, catalog.AdapterColors) catalog.HexColor
}

var globalColors = []colorRole{
	{"background", func(t catalog.Theme, _ catalog.AdapterColors) catalog.HexColor { return t.UI.Background }},
	{"foreground", func(t catalog.Theme, _ catalog.AdapterColors) catalog.HexColor { return t.UI.Foreground }},
	{"caret", func(_ catalog.Theme, a catalog.AdapterColors) catalog.HexColor { return a.CursorBg }},
	{"invisibles", func(_ catalog.Theme, a catalog.AdapterColors) catalog.HexColor { return a.Overlay0 }},
	{"lineHighlight", func(_ catalog.Theme, a catalog.AdapterColors) catalog.HexColor { return a.Surface0 }},
	{"selection", func(_ catalog.Theme, a catalog.AdapterColors) catalog.HexColor { return a.SelectionBg }},
	{"findHighlight", func(t catalog.Theme, _ catalog.AdapterColors) catalog.HexColor { return t.Semantic.Yellow }},
	{"findHighlightForeground", func(t catalog.Theme, _ catalog.AdapterColors) catalog.HexColor { return t.UI.Background }},
}

var syntaxColors = []colorRole{
	{"comment", func(_ catalog.Theme, a catalog.AdapterColors) catalog.HexColor { return a.Subtext }},
	{"string", func(t catalog.Theme, _ catalog.AdapterColors) catalog.HexColor { return t.Semantic.Green }},
	{"constant.numeric", func(t catalog.Theme, _ catalog.AdapterColors) catalog.HexColor { return t.Semantic.Orange }},
	{"constant.language", func(t catalog.Theme, _ catalog.AdapterColors) catalog.HexColor { return t.Semantic.Orange }},
	{"entity.name.function", func(t catalog.Theme, _ catalog.AdapterColors) catalog.HexColor { return t.Semantic.Blue }},
	{"entity.name.type", func(t catalog.Theme, _ catalog.AdapterColors) catalog.HexColor { return t.Semantic.Yellow }},
	{"keyword", func(t catalog.Theme, _ catalog.AdapterColors) catalog.HexColor { return t.Semantic.Magenta }},
	{"storage", func(t catalog.Theme, _ catalog.AdapterColors) catalog.HexColor { return t.Semantic.Magenta }},
	{"support", func(t catalog.Theme, _ catalog.AdapterColors) catalog.HexColor { return t.Semantic.Cyan }},
	{"variable", func(t catalog.Theme, _ catalog.AdapterColors) catalog.HexColor { return t.UI.Foreground }},
	{"invalid", func(t catalog.Theme, _ catalog.AdapterColors) catalog.HexColor { return t.Semantic.Red }},
	{"markup.inserted", func(t catalog.Theme, _ catalog.AdapterColors) catalog.HexColor { return t.Semantic.Green }},
	{"markup.deleted", func(t catalog.Theme, _ catalog.AdapterColors) catalog.HexColor { return t.Semantic.Red }},
}

// home is --config-dir, then CODEX_HOME when set and not empty, then ~/.codex.
func home(configDir string) (string, error) {
	if configDir != "" {
		return configDir, nil
	}
	if configured := os.Getenv("CODEX_HOME"); configured != "" {
		return pycompat.ExpandUser(configured)
	}
	userHome, err := pycompat.Home()
	if err != nil {
		return "", err
	}
	return pycompat.Path(userHome, ".codex"), nil
}

func target(configDir string) (string, error) {
	dir, err := home(configDir)
	return pycompat.Path(dir, configFileName), err
}

// xmlEscape is xml.sax.saxutils.escape: `&`, `<`, and `>` are escaped, and quotes are not.
var xmlEscape = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

func setting(key, value string) string {
	return "<key>" + key + "</key><string>" + xmlEscape.Replace(value) + "</string>"
}

// RenderTheme returns the `.tmTheme` property list for one catalog theme.
func RenderTheme(theme catalog.Theme) string {
	adapter := catalog.ProjectAdapterColors(theme.UI)
	lines := []string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">`,
		`<plist version="1.0">`,
		"<dict>",
		setting("name", theme.Metadata.SelectableID()),
		"<key>settings</key>",
		"<array>",
		"<dict>",
		"<key>settings</key>",
		"<dict>",
	}
	for _, role := range globalColors {
		lines = append(lines, setting(role.key, string(role.color(theme, adapter))))
	}
	lines = append(lines, "</dict>", "</dict>")
	for _, role := range syntaxColors {
		lines = append(lines,
			"<dict>",
			setting("scope", role.key),
			"<key>settings</key>",
			"<dict>",
			setting("foreground", string(role.color(theme, adapter))),
			"</dict>",
			"</dict>",
		)
	}
	lines = append(lines, "</array>", "</dict>", "</plist>", "")
	return strings.Join(lines, "\n")
}

// sectionBounds returns the header line of the first table named name and the line that ends it.
func sectionBounds(lines []string, name string) (start, end int, found bool) {
	start = -1
	for i, line := range lines {
		if match := section.FindStringSubmatch(line); match != nil && pycompat.Strip(match[1]) == name {
			start = i
			break
		}
	}
	if start < 0 {
		return 0, 0, false
	}
	for i := start + 1; i < len(lines); i++ {
		if section.MatchString(lines[i]) {
			return start, i, true
		}
	}
	return start, len(lines), true
}

func hasInlineTUI(existing string) bool {
	for _, line := range pycompat.SplitLines(existing) {
		if inlineTUI.MatchString(line) {
			return true
		}
	}
	return false
}

// MergeConfig sets `theme` inside `[tui]`, appending the table or the assignment when either is missing.
func MergeConfig(existing string, theme catalog.Theme) string {
	lines := pycompat.SplitLines(existing)
	assignment := `theme = "` + theme.Metadata.SelectableID() + `"`
	start, end, found := sectionBounds(lines, "tui")
	if !found {
		if prefix := pycompat.RStrip(existing); prefix != "" {
			return prefix + "\n\n[tui]\n" + assignment + "\n"
		}
		return "[tui]\n" + assignment + "\n"
	}
	for i := start + 1; i < end; i++ {
		if themeAssignment.MatchString(lines[i]) {
			lines[i] = assignment
			return pycompat.RStrip(strings.Join(lines, "\n")) + "\n"
		}
	}
	lines = slices.Insert(lines, end, assignment)
	return pycompat.RStrip(strings.Join(lines, "\n")) + "\n"
}

// Inspect classifies the `[tui]` theme in config.toml.
func Inspect(themes []catalog.Theme, configDir string) (plan.Inspection, error) {
	dir, err := home(configDir)
	if err != nil {
		return plan.Inspection{}, err
	}
	path := pycompat.Path(dir, configFileName)
	existing, err := plan.ReadExisting(path)
	if err != nil {
		return plan.Inspection{}, err
	}
	config, err := catalog.DecodeTOML(existing)
	if err != nil {
		return plan.Inspection{Ownership: plan.Broken, Detail: path + " is not valid TOML: " + err.Error()}, nil
	}
	if _, _, found := sectionBounds(pycompat.SplitLines(existing), "tui"); !found {
		if hasInlineTUI(existing) {
			return plan.Inspection{
				Ownership: plan.UnknownShape,
				Detail: "Convert the inline tui value in " + path +
					" into a [tui] table, for example:\n\n[tui]\ntheme = \"...\"",
			}, nil
		}
		return plan.Inspection{Ownership: plan.Absent}, nil
	}
	tui, _ := config["tui"].(map[string]any)
	configured, present := tui["theme"]
	if !present || configured == nil {
		return plan.Inspection{Ownership: plan.Absent}, nil
	}
	name, isString := configured.(string)
	if isString && strings.HasPrefix(name, catalog.SelectablePrefix) {
		return plan.Recorded(plan.Managed, name, true, themes), nil
	}
	if isString {
		for _, entry := range themes {
			if entry.Metadata.ID == name {
				return plan.Inspection{Ownership: plan.Legacy, Current: &entry}, nil
			}
		}
	}
	return plan.Inspection{Ownership: plan.Foreign}, nil
}

// Plan removes unprefixed legacy theme files, writes every catalog theme file, and, when slot is set, selects
// theme in config.toml.
func Plan(theme catalog.Theme, themes []catalog.Theme, configDir string, slot bool) ([]plan.Step, error) {
	dir, err := home(configDir)
	if err != nil {
		return nil, err
	}
	steps := make([]plan.Step, 0, 2*len(themes)+1)
	for _, entry := range themes {
		steps = append(steps, plan.Remove(pycompat.Path(dir, themeDirName, entry.Metadata.ID+themeSuffix)))
	}
	for _, entry := range themes {
		steps = append(steps, plan.Write(pycompat.Path(dir, themeDirName, entry.Metadata.SelectableID()+themeSuffix), RenderTheme(entry)))
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
