// Package herdrapp installs SF2 themes for the Herdr desktop app and selects a dark and light pair through
// `theme.dark` and `theme.light` in settings.json.
package herdrapp

import (
	"errors"
	"os"

	"github.com/douglasjarquin/sf2-themes/v2/internal/adapters"
	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
	"github.com/douglasjarquin/sf2-themes/v2/internal/plan"
	"github.com/douglasjarquin/sf2-themes/v2/internal/pycompat"
)

// App is the Herdr desktop app adapter.
var App = adapters.App{
	Label:   "Herdr app",
	Inspect: Inspect,
	Plan:    Plan,
	Target:  target,
	Pair:    true,
	Reload:  "settings apply live; appearance, layout themes, and the theme picker can override the pair",
}

// home is --config-dir, then `$XDG_CONFIG_HOME/herdr-app`, then ~/.config/herdr-app.
// An XDG_CONFIG_HOME that is set but empty is used as is, so the directory is the relative `herdr-app`, as the
// reference does.
func home(configDir string) (string, error) {
	if configDir != "" {
		return configDir, nil
	}
	root, ok := os.LookupEnv("XDG_CONFIG_HOME")
	if !ok {
		userHome, err := pycompat.Home()
		if err != nil {
			return "", err
		}
		root = pycompat.Path(userHome, ".config")
	}
	expanded, err := pycompat.ExpandUser(root)
	if err != nil {
		return "", err
	}
	return pycompat.Path(expanded, "herdr-app"), nil
}

func target(configDir string) (string, error) {
	dir, err := home(configDir)
	return pycompat.Path(dir, "settings.json"), err
}

// RenderTheme returns the Herdr app theme JSON for one catalog theme.
func RenderTheme(theme catalog.Theme) string {
	ui := theme.UI
	ansi := make([]any, 0, 16)
	for _, color := range append(theme.AnsiNormal.Colors(), theme.AnsiBright.Colors()...) {
		ansi = append(ansi, string(color))
	}
	colors := pycompat.NewObject()
	colors.Set("canvas", string(ui.Background))
	colors.Set("chrome", string(ui.Surface))
	colors.Set("surface", string(ui.Overlay))
	colors.Set("border", string(ui.Border))
	colors.Set("accent", string(ui.Accent))
	document := pycompat.NewObject()
	document.Set("background", string(ui.Background))
	document.Set("foreground", string(ui.Foreground))
	document.Set("selection", string(ui.SelectionBackground))
	document.Set("cursor", string(ui.Cursor))
	document.Set("ansi", ansi)
	document.Set("colors", colors)
	return pycompat.Dumps(document) + "\n"
}

// errBroken is a settings.json that cannot be read as Herdr app settings; its text is the inspection's Detail.
type errBroken struct{ message string }

func (e errBroken) Error() string { return e.message }

// readSettings returns the settings document, an empty one when the file is missing or blank, or errBroken.
func readSettings(path string) (*pycompat.Object, error) {
	existing, err := plan.ReadExisting(path)
	if err != nil {
		return nil, err
	}
	if pycompat.Strip(existing) == "" {
		return pycompat.NewObject(), nil
	}
	loaded, err := pycompat.Loads(existing)
	if err != nil {
		return nil, errBroken{path + " is not valid JSON: " + err.Error()}
	}
	data, ok := loaded.(*pycompat.Object)
	if !ok {
		return nil, errBroken{path + " must contain a JSON object"}
	}
	if selection, present := data.Get("theme"); present {
		switch selection.(type) {
		case string, *pycompat.Object:
		default:
			return nil, errBroken{path + " theme must be a name or object"}
		}
	}
	return data, nil
}

// Inspect classifies the `theme` selection in settings.json. Only a dark and light pair that matches one catalog
// pair is managed.
func Inspect(themes []catalog.Theme, configDir string) (plan.Inspection, error) {
	dir, err := home(configDir)
	if err != nil {
		return plan.Inspection{}, err
	}
	data, err := readSettings(pycompat.Path(dir, "settings.json"))
	var broken errBroken
	if errors.As(err, &broken) {
		return plan.Inspection{Ownership: plan.Broken, Detail: broken.message}, nil
	}
	if err != nil {
		return plan.Inspection{}, err
	}
	raw, present := data.Get("theme")
	if !present {
		return plan.Inspection{Ownership: plan.Absent}, nil
	}
	if selection, ok := raw.(*pycompat.Object); ok {
		darkRaw, _ := selection.Get("dark")
		lightRaw, _ := selection.Get("light")
		darkID, darkOK := darkRaw.(string)
		lightID, lightOK := lightRaw.(string)
		if darkOK && lightOK {
			installed, err := catalog.InstalledTheme(darkID, themes)
			if err != nil {
				return plan.Inspection{Ownership: plan.Foreign}, nil
			}
			dark, light, err := catalog.ThemePair(installed, themes)
			if err != nil {
				return plan.Inspection{Ownership: plan.Foreign}, nil
			}
			if darkID == dark.Metadata.SelectableID() && lightID == light.Metadata.SelectableID() {
				return plan.Inspection{Ownership: plan.Managed, Current: &dark}, nil
			}
		}
	}
	return plan.Inspection{Ownership: plan.Foreign}, nil
}

// Plan writes every catalog theme file and, when slot is set, selects the dark and light pair for theme while
// keeping every other key of an existing `theme` object in place.
func Plan(theme catalog.Theme, themes []catalog.Theme, configDir string, slot bool) ([]plan.Step, error) {
	dir, err := home(configDir)
	if err != nil {
		return nil, err
	}
	steps := make([]plan.Step, 0, len(themes)+1)
	for _, candidate := range themes {
		path := pycompat.Path(dir, "themes", candidate.Metadata.SelectableID()+".json")
		steps = append(steps, plan.Write(path, RenderTheme(candidate)))
	}
	if slot {
		path := pycompat.Path(dir, "settings.json")
		data, err := readSettings(path)
		if err != nil {
			return nil, err
		}
		dark, light, err := catalog.ThemePair(theme, themes)
		if err != nil {
			return nil, err
		}
		merged := pycompat.NewObject()
		if raw, _ := data.Get("theme"); raw != nil {
			if selection, ok := raw.(*pycompat.Object); ok {
				for _, key := range selection.Keys() {
					value, _ := selection.Get(key)
					merged.Set(key, value)
				}
			}
		}
		merged.Set("dark", dark.Metadata.SelectableID())
		merged.Set("light", light.Metadata.SelectableID())
		data.Set("theme", merged)
		steps = append(steps, plan.Write(path, pycompat.Dumps(data)+"\n"))
	}
	return steps, nil
}
