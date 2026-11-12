// Package starship owns the marked palette block in starship.toml, so prompt styles track the active SF2 theme, and
// the matching zsh-syntax-highlighting snippet.
package starship

import (
	"errors"
	"os"
	"regexp"
	"strings"

	"github.com/douglasjarquin/sf2-themes/v2/internal/adapters"
	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
	"github.com/douglasjarquin/sf2-themes/v2/internal/plan"
	"github.com/douglasjarquin/sf2-themes/v2/internal/pycompat"
)

// App is the Starship adapter.
var App = adapters.App{
	Label:      "Starship",
	Inspect:    Inspect,
	Plan:       Plan,
	StderrHint: "Source the zsh highlight snippet after zsh-syntax-highlighting:\n  " + sourceHint + "\n",
}

const (
	managedStart = "# >>> sf2-themes managed theme"
	managedEnd   = "# <<< sf2-themes managed theme"
)

var (
	idComment       = regexp.MustCompile(`^# sf2-themes:` + pycompat.Space + `+(` + pycompat.NotSpace + `+)` + pycompat.Space + `*$`)
	tableHeader     = regexp.MustCompile(`^` + pycompat.Space + `*\[`)
	topLevelPalette = regexp.MustCompile(`^` + pycompat.Space + `*palette` + pycompat.Space + `*=`)
	multiLineQuotes = []string{`"""`, `'''`}
)

// errPaletteNotTopLevel is raised when the merged document would not load the managed palette.
var errPaletteNotTopLevel = errors.New("Starship config would not load the managed palette as a top-level key") //nolint:staticcheck // the reference message starts with the app name

// paletteColor is one named Starship color and the SF2 color it resolves to.
type paletteColor struct {
	key   string
	color func(catalog.Theme) catalog.HexColor
}

// Starship resolves named styles like `blue` through `palette`, not the terminal ANSI table.
var paletteKeys = []paletteColor{
	{"black", func(t catalog.Theme) catalog.HexColor { return t.UI.Background }},
	{"red", func(t catalog.Theme) catalog.HexColor { return t.Semantic.Red }},
	{"green", func(t catalog.Theme) catalog.HexColor { return t.Semantic.Green }},
	{"yellow", func(t catalog.Theme) catalog.HexColor { return t.Semantic.Yellow }},
	{"blue", func(t catalog.Theme) catalog.HexColor { return t.UI.Accent }},
	{"purple", func(t catalog.Theme) catalog.HexColor { return t.UI.AccentSecondary }},
	{"cyan", func(t catalog.Theme) catalog.HexColor { return t.Semantic.Cyan }},
	{"white", func(t catalog.Theme) catalog.HexColor { return t.UI.Foreground }},
	{"bright-black", func(t catalog.Theme) catalog.HexColor { return t.UI.Muted }},
	{"bright-red", func(t catalog.Theme) catalog.HexColor { return t.Semantic.Red }},
	{"bright-green", func(t catalog.Theme) catalog.HexColor { return t.Semantic.Green }},
	{"bright-yellow", func(t catalog.Theme) catalog.HexColor { return t.Semantic.Yellow }},
	{"bright-blue", func(t catalog.Theme) catalog.HexColor { return t.UI.Accent }},
	{"bright-purple", func(t catalog.Theme) catalog.HexColor { return t.UI.AccentSecondary }},
	{"bright-cyan", func(t catalog.Theme) catalog.HexColor { return t.Semantic.Cyan }},
	{"bright-white", func(t catalog.Theme) catalog.HexColor { return t.UI.Foreground }},
}

func configRoot() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return pycompat.ExpandUser(xdg)
	}
	home, err := pycompat.Home()
	return pycompat.Path(home, ".config"), err
}

func configPath(configDir string) (string, error) {
	if configDir != "" {
		return pycompat.Path(configDir, "starship.toml"), nil
	}
	if configured := os.Getenv("STARSHIP_CONFIG"); configured != "" {
		return pycompat.ExpandUser(configured)
	}
	root, err := configRoot()
	return pycompat.Path(root, "starship.toml"), err
}

func renderBlock(theme catalog.Theme) string {
	lines := []string{
		managedStart,
		"# sf2-themes: " + theme.Metadata.SelectableID(),
		`palette = "sf2"`,
		"",
		"[palettes.sf2]",
	}
	for _, entry := range paletteKeys {
		lines = append(lines, entry.key+` = "`+string(entry.color(theme))+`"`)
	}
	lines = append(lines, managedEnd, "")
	return strings.Join(lines, "\n")
}

// withoutBlock returns the config with the managed block removed, or false when its markers are incomplete.
func withoutBlock(existing string) (string, bool) {
	if !strings.Contains(existing, managedStart) && !strings.Contains(existing, managedEnd) {
		return existing, true
	}
	start := strings.Index(existing, managedStart)
	end := strings.Index(existing, managedEnd)
	if start == -1 || end == -1 || end < start {
		return "", false
	}
	after := end + len(managedEnd)
	if after < len(existing) && existing[after] == '\n' {
		after++
	}
	return existing[:start] + existing[after:], true
}

// openString returns the multi-line string delimiter still open after line, given the one open before it ("" for
// none).
func openString(line, delimiter string) string {
	position := 0
	for {
		if delimiter == "" {
			first := -1
			for _, quote := range multiLineQuotes {
				index := strings.Index(line[position:], quote)
				if index != -1 && (first == -1 || position+index < first) {
					first, delimiter = position+index, quote
				}
			}
			if first == -1 {
				return ""
			}
			position = first
		} else {
			index := strings.Index(line[position:], delimiter)
			if index == -1 {
				return delimiter
			}
			position += index
			delimiter = ""
		}
		position += 3
	}
}

// topLevelEnd returns where the top-level key region ends, keeping a comment that heads the first table with it.
//
// Lines inside a multi-line string, such as a preset's `format`, are never table headers.
func topLevelEnd(lines []string) int {
	end := len(lines)
	delimiter := ""
	for index, line := range lines {
		if delimiter == "" && tableHeader.MatchString(line) {
			end = index
			break
		}
		delimiter = openString(line, delimiter)
	}
	for end > 0 && strings.HasPrefix(pycompat.LStrip(lines[end-1]), "#") {
		end--
	}
	return end
}

// mergeTheme places the managed palette block in the top-level key region.
//
// Starship reads `palette` only as a top-level key, so the block goes before the first table. Any top-level
// `palette` outside the block is replaced. The merged document is parsed before it is returned.
func mergeTheme(existing string, theme catalog.Theme) (string, error) {
	source := existing
	if remaining, complete := withoutBlock(existing); complete {
		source = remaining
	}
	lines := pycompat.SplitLines(source)
	split := topLevelEnd(lines)
	var head []string
	for _, line := range lines[:split] {
		if !topLevelPalette.MatchString(line) {
			head = append(head, line)
		}
	}
	parts := []string{
		strings.TrimRight(strings.Join(head, "\n"), "\n"),
		strings.TrimRight(renderBlock(theme), "\n"),
		strings.Trim(strings.Join(lines[split:], "\n"), "\n"),
	}
	var kept []string
	for _, part := range parts {
		if part != "" {
			kept = append(kept, part)
		}
	}
	merged := strings.Join(kept, "\n\n") + "\n"
	doc, err := catalog.DecodeTOML(merged)
	if err != nil {
		return "", err
	}
	if palette, ok := doc["palette"].(string); !ok || palette != "sf2" {
		return "", errPaletteNotTopLevel
	}
	return merged, nil
}

// Inspect classifies the palette setting of starship.toml.
func Inspect(themes []catalog.Theme, configDir string) (plan.Inspection, error) {
	path, err := configPath(configDir)
	if err != nil {
		return plan.Inspection{}, err
	}
	existing, err := plan.ReadExisting(path)
	if err != nil {
		return plan.Inspection{}, err
	}
	outside, complete := withoutBlock(existing)
	if !complete {
		return plan.Inspection{Ownership: plan.Broken, Detail: path + " has incomplete sf2-themes markers"}, nil
	}
	if _, err := catalog.DecodeTOML(existing); err != nil {
		return plan.Inspection{Ownership: plan.Broken, Detail: path + " is not valid TOML: " + err.Error()}, nil
	}
	doc, err := catalog.DecodeTOML(outside)
	if err != nil {
		return plan.Inspection{Ownership: plan.Broken, Detail: path + " is not valid TOML: " + err.Error()}, nil
	}
	if _, userPalette := doc["palette"]; userPalette {
		return plan.Inspection{Ownership: plan.Foreign}, nil
	}
	if outside != existing {
		id, found := plan.RecordedID(existing, idComment)
		return plan.Recorded(plan.Managed, id, found, themes), nil
	}
	return plan.Inspection{Ownership: plan.Absent}, nil
}

// Plan returns the managed palette block and the matching zsh highlight snippet, or nothing when slot keeps the
// setting.
func Plan(theme catalog.Theme, _ []catalog.Theme, configDir string, slot bool) ([]plan.Step, error) {
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
	merged, err := mergeTheme(existing, theme)
	if err != nil {
		return nil, err
	}
	snippet, err := planZshSyntax(theme)
	if err != nil {
		return nil, err
	}
	return []plan.Step{plan.Write(path, merged), snippet}, nil
}
