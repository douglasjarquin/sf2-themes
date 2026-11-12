package catalog

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/douglasjarquin/sf2-themes/v2/internal/pycompat"
)

func requireTable(raw any, name string) (map[string]any, error) {
	if table, ok := raw.(map[string]any); ok {
		return table, nil
	}
	return nil, fmt.Errorf("%s must be a table", name)
}

func requireStr(raw any, present bool, name string) (string, error) {
	if !present {
		return "", fmt.Errorf("missing %s", name)
	}
	if text, ok := raw.(string); ok {
		return text, nil
	}
	return "", fmt.Errorf("%s must be a string", name)
}

func requireNonemptyStr(raw any, present bool, name string) (string, error) {
	text, err := requireStr(raw, present, name)
	if err != nil {
		return "", err
	}
	if pycompat.Strip(text) == "" {
		return "", fmt.Errorf("%s must not be empty", name)
	}
	return text, nil
}

func requireInt(raw any, name string) (int64, error) {
	if number, ok := raw.(int64); ok {
		return number, nil
	}
	return 0, fmt.Errorf("%s must be an integer", name)
}

func requireStrList(raw any, present bool, name string) ([]string, error) {
	if !present {
		return nil, fmt.Errorf("missing %s", name)
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%s must be an array of strings", name)
	}
	values := make([]string, 0, len(items))
	for index, item := range items {
		text, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("%s[%d] must be a string", name, index)
		}
		values = append(values, text)
	}
	return values, nil
}

func unknownKeys(table map[string]any, allowed []string) []string {
	return sortedKeys(table, func(key string) bool { return !slices.Contains(allowed, key) })
}

func hexFields(table map[string]any, fields []string, prefix string) (map[string]HexColor, error) {
	if unknown := unknownKeys(table, fields); len(unknown) > 0 {
		return nil, fmt.Errorf("unknown %s keys: %s", prefix, strings.Join(unknown, ", "))
	}
	colors := make(map[string]HexColor, len(fields))
	for _, name := range fields {
		raw, present := table[name]
		text, err := requireStr(raw, present, prefix+"."+name)
		if err != nil {
			return nil, err
		}
		color, err := ParseHex(text)
		if err != nil {
			return nil, err
		}
		colors[name] = color
	}
	return colors, nil
}

func parseKind(raw, source string) (ThemeKind, error) {
	switch kind := ThemeKind(raw); kind {
	case KindMain, KindCharacter:
		return kind, nil
	}
	return "", fmt.Errorf("%s: kind must be main or character", source)
}

func parseEra(raw, source string) (IntroducedIn, error) {
	switch era := IntroducedIn(raw); era {
	case IntroducedMain, IntroducedWorldWarrior, IntroducedChampionEdition, IntroducedSuper, IntroducedSuperTurbo:
		return era, nil
	}
	return "", fmt.Errorf("%s: unknown introduced_in %s", source, pycompat.Repr(raw))
}

func parseVariant(raw, source string) (ThemeVariant, error) {
	switch variant := ThemeVariant(raw); variant {
	case VariantDark, VariantLight:
		return variant, nil
	}
	return "", fmt.Errorf("%s: variant must be dark or light", source)
}

var (
	topLevelKeys = []string{"schema_version", "meta", "ui", "semantic", "ansi"}
	metaKeys     = []string{
		"id", "display_name", "kind", "introduced_in", "character", "aliases", "name", "variant", "family", "stage",
	}
	ansiKeys = []string{"normal", "bright"}
)

// ParseTheme parses a complete theme table decoded by DecodeTOML. source names the file in error messages.
func ParseTheme(raw any, source string) (Theme, error) {
	table, err := requireTable(raw, source)
	if err != nil {
		return Theme{}, err
	}
	if extra := unknownKeys(table, topLevelKeys); len(extra) > 0 {
		return Theme{}, fmt.Errorf("%s: unknown top-level keys: %s", source, strings.Join(extra, ", "))
	}
	version, err := requireInt(table["schema_version"], source+".schema_version")
	if err != nil {
		return Theme{}, err
	}
	if version != 1 {
		return Theme{}, fmt.Errorf("%s: unsupported schema_version %d", source, version)
	}
	meta, err := requireTable(table["meta"], source+".meta")
	if err != nil {
		return Theme{}, err
	}
	str := func(key string) (string, error) {
		value, present := meta[key]
		return requireStr(value, present, source+".meta."+key)
	}
	nonempty := func(key string) (string, error) {
		value, present := meta[key]
		return requireNonemptyStr(value, present, source+".meta."+key)
	}

	kindRaw, err := str("kind")
	if err != nil {
		return Theme{}, err
	}
	kind, err := parseKind(kindRaw, source)
	if err != nil {
		return Theme{}, err
	}
	var character *string
	characterRaw, hasCharacter := meta["character"]
	name, isString := characterRaw.(string)
	switch {
	case !hasCharacter && kind == KindMain:
	case hasCharacter && isString && kind == KindCharacter:
		character = &name
	default:
		return Theme{}, fmt.Errorf("%s: character is required for character themes and forbidden for main", source)
	}
	if extra := unknownKeys(meta, metaKeys); len(extra) > 0 {
		return Theme{}, fmt.Errorf("%s: unknown meta keys: %s", source, strings.Join(extra, ", "))
	}

	metadata := ThemeMetadata{Kind: kind, Character: character}
	if metadata.ID, err = str("id"); err != nil {
		return Theme{}, err
	}
	if metadata.DisplayName, err = str("display_name"); err != nil {
		return Theme{}, err
	}
	eraRaw, err := str("introduced_in")
	if err != nil {
		return Theme{}, err
	}
	if metadata.IntroducedIn, err = parseEra(eraRaw, source); err != nil {
		return Theme{}, err
	}
	aliasesRaw, hasAliases := meta["aliases"]
	if metadata.Aliases, err = requireStrList(aliasesRaw, hasAliases, source+".meta.aliases"); err != nil {
		return Theme{}, err
	}
	if metadata.Name, err = nonempty("name"); err != nil {
		return Theme{}, err
	}
	variantRaw, err := str("variant")
	if err != nil {
		return Theme{}, err
	}
	if metadata.Variant, err = parseVariant(variantRaw, source); err != nil {
		return Theme{}, err
	}
	if metadata.Family, err = nonempty("family"); err != nil {
		return Theme{}, err
	}
	if metadata.Stage, err = nonempty("stage"); err != nil {
		return Theme{}, err
	}

	colors := func(raw any, name string, fields []string) (map[string]HexColor, error) {
		section, err := requireTable(raw, name)
		if err != nil {
			return nil, err
		}
		return hexFields(section, fields, name)
	}
	ui, err := colors(table["ui"], source+".ui", UIColorFields)
	if err != nil {
		return Theme{}, err
	}
	semantic, err := colors(table["semantic"], source+".semantic", SemanticFields)
	if err != nil {
		return Theme{}, err
	}
	ansi, err := requireTable(table["ansi"], source+".ansi")
	if err != nil {
		return Theme{}, err
	}
	if extra := unknownKeys(ansi, ansiKeys); len(extra) > 0 {
		return Theme{}, fmt.Errorf("%s: unknown ansi keys: %s", source, strings.Join(extra, ", "))
	}
	normal, err := colors(ansi["normal"], source+".ansi.normal", ANSIOrder)
	if err != nil {
		return Theme{}, err
	}
	bright, err := colors(ansi["bright"], source+".ansi.bright", ANSIOrder)
	if err != nil {
		return Theme{}, err
	}
	return Theme{
		Metadata:   metadata,
		UI:         uiColorsFrom(ui),
		Semantic:   semanticColorsFrom(semantic),
		AnsiNormal: ansiColorsFrom(normal),
		AnsiBright: ansiColorsFrom(bright),
	}, nil
}

// errorLines joins messages the way the reference raises several findings as one error.
func errorLines(messages []string) error {
	return errors.New(strings.Join(messages, "\n"))
}
