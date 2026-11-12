package catalog

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/douglasjarquin/sf2-themes/v2/internal/pycompat"
	"github.com/douglasjarquin/sf2-themes/v2/themes"
)

// Parse loads every embedded theme without contrast or uniqueness enforcement.
func Parse() ([]Theme, error) {
	return ParseFS(themes.FS)
}

// Load loads the embedded catalog and rejects themes that fail error-severity checks.
func Load() ([]Theme, error) {
	return LoadFS(themes.FS)
}

// ParseFS reads main.toml, then main-light.toml when present, then characters/*.toml in sorted order,
// and parses each without contrast or uniqueness enforcement.
// Error messages name each file by its path inside fsys, for example characters/ryu.toml.
func ParseFS(fsys fs.FS) ([]Theme, error) {
	names := []string{"main.toml"}
	if _, err := fs.Stat(fsys, "main-light.toml"); err == nil {
		names = append(names, "main-light.toml")
	}
	characters, err := fs.Glob(fsys, "characters/*.toml")
	if err != nil {
		return nil, err
	}
	names = append(names, characters...)
	parsed := make([]Theme, 0, len(names))
	for _, name := range names {
		data, err := fs.ReadFile(fsys, name)
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("missing theme file: %s", name)
		}
		if err != nil {
			return nil, err
		}
		text, err := pycompat.DecodeText(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		raw, err := DecodeTOML(text)
		if err != nil {
			return nil, err
		}
		theme, err := ParseTheme(raw, name)
		if err != nil {
			return nil, err
		}
		parsed = append(parsed, theme)
	}
	return parsed, nil
}

// LoadFS is ParseFS followed by the catalog checks and every theme's error-severity checks.
func LoadFS(fsys fs.FS) ([]Theme, error) {
	parsed, err := ParseFS(fsys)
	if err != nil {
		return nil, err
	}
	if messages := errorMessages(ValidateCatalog(parsed)); len(messages) > 0 {
		return nil, errorLines(messages)
	}
	for _, theme := range parsed {
		if err := RequireValid(theme); err != nil {
			return nil, err
		}
	}
	return parsed, nil
}

// GetTheme resolves a theme by id, alias, or display name, ignoring case.
func GetTheme(name string, catalog []Theme) (Theme, error) {
	lowered := strings.ToLower(name)
	for _, theme := range catalog {
		meta := theme.Metadata
		if lowered == strings.ToLower(meta.ID) || lowered == strings.ToLower(meta.DisplayName) {
			return theme, nil
		}
		for _, alias := range meta.Aliases {
			if lowered == strings.ToLower(alias) {
				return theme, nil
			}
		}
	}
	return Theme{}, fmt.Errorf("unknown theme: %s", name)
}

// ThemePair returns the dark and light siblings for a selected catalog theme.
func ThemePair(theme Theme, catalog []Theme) (dark, light Theme, err error) {
	darkID := strings.TrimSuffix(theme.Metadata.ID, lightSuffix)
	if dark, err = GetTheme(darkID, catalog); err != nil {
		return Theme{}, Theme{}, err
	}
	if light, err = GetTheme(darkID+lightSuffix, catalog); err != nil {
		return Theme{}, Theme{}, err
	}
	return dark, light, nil
}

// InstalledTheme resolves a catalog id or an adapter-installed sf2- identity.
func InstalledTheme(name string, catalog []Theme) (Theme, error) {
	return GetTheme(strings.TrimPrefix(name, SelectablePrefix), catalog)
}

// DefaultTheme returns the main theme.
func DefaultTheme(catalog []Theme) (Theme, error) {
	return GetTheme(mainID, catalog)
}

// ShowTheme renders canonical TOML for a resolved theme.
func ShowTheme(theme Theme) string {
	return ThemeToTOML(theme)
}

// CatalogIssues returns printable `error: ...` and `warning: ...` lines for the whole catalog.
func CatalogIssues(catalog []Theme) []string {
	var lines []string
	for _, issue := range ValidateCatalog(catalog) {
		lines = append(lines, string(issue.Severity)+": "+issue.Message)
	}
	for _, theme := range catalog {
		for _, issue := range ValidateTheme(theme) {
			lines = append(lines, string(issue.Severity)+": "+issue.Message)
		}
	}
	return lines
}
