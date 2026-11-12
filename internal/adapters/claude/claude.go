// Package claude installs SF2 custom themes for Claude Code and selects one through the `theme` key in
// settings.json.
package claude

import (
	"errors"
	"os"
	"strings"

	"github.com/douglasjarquin/sf2-themes/v2/internal/adapters"
	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
	"github.com/douglasjarquin/sf2-themes/v2/internal/plan"
	"github.com/douglasjarquin/sf2-themes/v2/internal/pycompat"
)

// App is the Claude Code adapter.
var App = adapters.App{
	Label:   "Claude Code",
	Inspect: Inspect,
	Plan:    Plan,
	Target:  target,
	Reload:  "reselect with /theme in a running Claude Code session",
}

const (
	themeDirName       = "themes"
	settingsFileName   = "settings.json"
	customThemePrefix  = "custom:"
	managedThemePrefix = customThemePrefix + catalog.SelectablePrefix
)

// errNotObject is a settings document that is valid JSON but not an object, which Inspect reports as broken.
var errNotObject = errors.New("settings.json must contain a JSON object")

// overrideTokens maps Claude Code theme tokens onto SF2 roles, in the order they are written.
var overrideTokens = []struct {
	key   string
	color func(catalog.Theme) catalog.HexColor
}{
	{"claude", func(t catalog.Theme) catalog.HexColor { return t.UI.Accent }},
	{"error", func(t catalog.Theme) catalog.HexColor { return t.Semantic.Red }},
	{"success", func(t catalog.Theme) catalog.HexColor { return t.Semantic.Green }},
	{"warning", func(t catalog.Theme) catalog.HexColor { return t.Semantic.Yellow }},
	{"text", func(t catalog.Theme) catalog.HexColor { return t.UI.Foreground }},
	{"permission", func(t catalog.Theme) catalog.HexColor { return t.Semantic.Orange }},
	{"bashBorder", func(t catalog.Theme) catalog.HexColor { return t.UI.Border }},
	{"diffAdded", func(t catalog.Theme) catalog.HexColor { return t.Semantic.Green }},
	{"diffRemoved", func(t catalog.Theme) catalog.HexColor { return t.Semantic.Red }},
	{"suggestion", func(t catalog.Theme) catalog.HexColor { return t.UI.AccentSecondary }},
}

// home is --config-dir, then CLAUDE_CONFIG_DIR when set and not empty, then ~/.claude.
func home(configDir string) (string, error) {
	if configDir != "" {
		return configDir, nil
	}
	if configured := os.Getenv("CLAUDE_CONFIG_DIR"); configured != "" {
		return pycompat.ExpandUser(configured)
	}
	userHome, err := pycompat.Home()
	if err != nil {
		return "", err
	}
	return pycompat.Path(userHome, ".claude"), nil
}

func target(configDir string) (string, error) {
	dir, err := home(configDir)
	return pycompat.Path(dir, settingsFileName), err
}

// ThemeKey is the settings.json `theme` value that selects theme.
func ThemeKey(theme catalog.Theme) string {
	return customThemePrefix + theme.Metadata.SelectableID()
}

// RenderTheme returns the Claude Code custom-theme JSON for one catalog theme.
func RenderTheme(theme catalog.Theme) string {
	overrides := pycompat.NewObject()
	for _, token := range overrideTokens {
		overrides.Set(token.key, string(token.color(theme)))
	}
	document := pycompat.NewObject()
	document.Set("name", "SF2 "+theme.Metadata.DisplayName)
	document.Set("base", string(theme.Metadata.Variant))
	document.Set("overrides", overrides)
	return pycompat.Dumps(document) + "\n"
}

// MergeSettings sets the `theme` key in the settings document and keeps every other key where it was.
// The existing text must already have inspected as a JSON object or as empty.
func MergeSettings(existing string, theme catalog.Theme) (string, error) {
	data := pycompat.NewObject()
	if pycompat.Strip(existing) != "" {
		loaded, err := pycompat.Loads(existing)
		if err != nil {
			return "", err
		}
		object, ok := loaded.(*pycompat.Object)
		if !ok {
			return "", errNotObject
		}
		data = object
	}
	data.Set("theme", ThemeKey(theme))
	return pycompat.Dumps(data) + "\n", nil
}

// Inspect classifies the `theme` key in settings.json.
func Inspect(themes []catalog.Theme, configDir string) (plan.Inspection, error) {
	dir, err := home(configDir)
	if err != nil {
		return plan.Inspection{}, err
	}
	path := pycompat.Path(dir, settingsFileName)
	existing, err := plan.ReadExisting(path)
	if err != nil {
		return plan.Inspection{}, err
	}
	if pycompat.Strip(existing) == "" {
		return plan.Inspection{Ownership: plan.Absent}, nil
	}
	loaded, err := pycompat.Loads(existing)
	if err != nil {
		return plan.Inspection{Ownership: plan.Broken, Detail: path + " is not valid JSON: " + err.Error()}, nil
	}
	data, ok := loaded.(*pycompat.Object)
	if !ok {
		return plan.Inspection{Ownership: plan.Broken, Detail: path + " must contain a JSON object"}, nil
	}
	configured, present := data.Get("theme")
	if !present {
		return plan.Inspection{Ownership: plan.Absent}, nil
	}
	if name, isString := configured.(string); isString && strings.HasPrefix(name, managedThemePrefix) {
		return plan.Recorded(plan.Managed, strings.TrimPrefix(name, customThemePrefix), true, themes), nil
	}
	return plan.Inspection{Ownership: plan.Foreign}, nil
}

// Plan writes every catalog theme file and, when slot is set, selects theme in settings.json.
func Plan(theme catalog.Theme, themes []catalog.Theme, configDir string, slot bool) ([]plan.Step, error) {
	dir, err := home(configDir)
	if err != nil {
		return nil, err
	}
	steps := make([]plan.Step, 0, len(themes)+1)
	for _, entry := range themes {
		path := pycompat.Path(dir, themeDirName, entry.Metadata.SelectableID()+".json")
		steps = append(steps, plan.Write(path, RenderTheme(entry)))
	}
	if slot {
		path := pycompat.Path(dir, settingsFileName)
		existing, err := plan.ReadExisting(path)
		if err != nil {
			return nil, err
		}
		merged, err := MergeSettings(existing, theme)
		if err != nil {
			return nil, err
		}
		steps = append(steps, plan.Write(path, merged))
	}
	return steps, nil
}
