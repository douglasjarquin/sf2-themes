package wezterm

import (
	"regexp"
	"strings"

	"github.com/douglasjarquin/sf2-themes/v2/internal/plan"
	"github.com/douglasjarquin/sf2-themes/v2/internal/pycompat"
)

// Conservative wezterm.lua integration. Guessing is a failure.

var (
	builderPattern     = regexp.MustCompile(`(?m)^local` + pycompat.Space + `+(` + pycompat.Word + `+)` + pycompat.Space + `*=` + pycompat.Space + `*wezterm\.config_builder\(\)` + pycompat.Space + `*$`)
	returnNamePattern  = regexp.MustCompile(`(?m)^return` + pycompat.Space + `+(` + pycompat.Word + `+)` + pycompat.Space + `*$`)
	colorSchemeLine    = regexp.MustCompile(`^` + pycompat.Space + `*(?:` + pycompat.Word + `+\.)?color_scheme` + pycompat.Space + `*=`)
	sf2SchemeValue     = regexp.MustCompile(`"(?:sf2-[^"]*|street-fighter-2|Street Fighter II - [^"]*|street-fighter-ii-[^"]*)"`)
	leadingIndent      = regexp.MustCompile(`^(` + pycompat.Space + `*)`)
	termThemeGuard     = "TERM_THEME"
	termThemeAssign    = `sf2_env.TERM_THEME = (type(sf2_scheme) == "string" and sf2_scheme:find("-light", 1, true)) and "light" or "dark"`
	schemeKindsToStrip = map[string]bool{"sf2": true, "other": true}
)

// luaPathLiteral escapes a filesystem path for a Lua double-quoted string.
func luaPathLiteral(path string) string {
	return strings.ReplaceAll(strings.ReplaceAll(path, `\`, `\\`), `"`, `\"`)
}

// integrationSnippet returns the exact Lua block a human should paste.
//
// TERM_THEME follows the selected sf2 scheme so Cursor Agent and similar TUIs do not guess light chrome against a
// dark scheme (or the reverse).
func integrationSnippet(pointer, builder string) string {
	return strings.Join([]string{
		`local sf2_current = "` + luaPathLiteral(pointer) + `"`,
		"wezterm.add_to_config_reload_watch_list(sf2_current)",
		"local sf2_scheme = dofile(sf2_current)",
		builder + ".color_scheme = sf2_scheme",
		"local sf2_env = " + builder + ".set_environment_variables or {}",
		termThemeAssign,
		builder + ".set_environment_variables = sf2_env",
		"",
	}, "\n")
}

// starterConfig returns a complete starter wezterm.lua that selects the managed scheme.
func starterConfig(pointer string) string {
	return strings.Join([]string{
		`local wezterm = require("wezterm")`,
		"local config = wezterm.config_builder()",
		pycompat.RStrip(integrationSnippet(pointer, "config")),
		"return config",
		"",
	}, "\n")
}

// code drops Lua comment lines so commented-out integration never counts.
func code(existing string) string {
	var kept []string
	for _, line := range pycompat.SplitLines(existing) {
		if !strings.HasPrefix(pycompat.LStrip(line), "--") {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}

func referencesPointer(code, pointer string) bool {
	return strings.Contains(code, "wezterm-current.lua") &&
		(strings.Contains(code, pointer) || strings.Contains(code, "dofile"))
}

// upgradeTermThemeGuard inserts TERM_THEME wiring after the managed color_scheme assignment when possible.
func upgradeTermThemeGuard(existing string) (string, bool) {
	builder := builderPattern.FindStringSubmatch(existing)
	if builder == nil {
		return "", false
	}
	name := builder[1]
	lines := pycompat.SplitLinesKeepEnds(existing)
	schemeIdx := -1
	for i, line := range lines {
		if strings.Contains(line, "dofile(sf2_current)") && strings.Contains(line, "color_scheme") {
			schemeIdx = i
			break
		}
	}
	if schemeIdx < 0 {
		return "", false
	}
	indent := leadingIndent.FindStringSubmatch(lines[schemeIdx])[1]
	replacement := []string{
		indent + "local sf2_scheme = dofile(sf2_current)\n",
		indent + name + ".color_scheme = sf2_scheme\n",
		indent + "local sf2_env = " + name + ".set_environment_variables or {}\n",
		indent + termThemeAssign + "\n",
		indent + name + ".set_environment_variables = sf2_env\n",
	}
	upgraded := append(append(append([]string{}, lines[:schemeIdx]...), replacement...), lines[schemeIdx+1:]...)
	return strings.Join(upgraded, ""), true
}

func schemeKind(line string) string {
	if !colorSchemeLine.MatchString(line) {
		return ""
	}
	if sf2SchemeValue.MatchString(line) {
		return "sf2"
	}
	return "other"
}

func stripSchemeLines(existing string, kinds map[string]bool) string {
	var kept []string
	for _, line := range pycompat.SplitLines(existing) {
		if !kinds[schemeKind(line)] {
			kept = append(kept, line)
		}
	}
	newline := ""
	if strings.HasSuffix(existing, "\n") {
		newline = "\n"
	}
	return strings.Join(kept, "\n") + newline
}

func hasSchemeKind(existing, kind string) bool {
	for _, line := range pycompat.SplitLines(existing) {
		if schemeKind(line) == kind {
			return true
		}
	}
	return false
}

func builderName(code string) (string, bool) {
	builder := builderPattern.FindStringSubmatch(code)
	returned := returnNamePattern.FindStringSubmatch(code)
	if builder == nil || returned == nil || builder[1] != returned[1] {
		return "", false
	}
	return builder[1], true
}

// classify reports who owns the color scheme in wezterm.lua.
//
// Pointer integration is recognised before the config shape, so a pasted snippet in a config sf2-themes cannot
// edit still counts as managed.
func classify(existing, pointer string) plan.Ownership {
	if pycompat.Strip(existing) == "" {
		return plan.Absent
	}
	code := code(existing)
	if referencesPointer(code, pointer) {
		if strings.Contains(code, termThemeGuard) {
			return plan.Managed
		}
		if _, ok := upgradeTermThemeGuard(existing); ok {
			return plan.Legacy
		}
		return plan.UnknownShape
	}
	if _, ok := builderName(code); !ok {
		return plan.UnknownShape
	}
	if hasSchemeKind(existing, "other") {
		return plan.Foreign
	}
	if hasSchemeKind(existing, "sf2") {
		return plan.Legacy
	}
	return plan.Absent
}

// integrate returns wezterm.lua with the managed pointer integration.
//
// The caller has already classified the config and approved replacing any color-scheme assignment; a shape
// classify rejects is returned unchanged.
func integrate(existing, pointer string) string {
	if pycompat.Strip(existing) == "" {
		return starterConfig(pointer)
	}
	code := code(existing)
	if referencesPointer(code, pointer) {
		if strings.Contains(code, termThemeGuard) {
			return existing
		}
		if upgraded, ok := upgradeTermThemeGuard(existing); ok {
			return upgraded
		}
		return existing
	}
	name, named := builderName(code)
	cleaned := stripSchemeLines(existing, schemeKindsToStrip)
	returned := returnNamePattern.FindStringIndex(cleaned)
	if !named || returned == nil {
		return existing
	}
	assignment := pycompat.RStrip(integrationSnippet(pointer, name))
	prefix := pycompat.RStrip(cleaned[:returned[0]])
	suffix := cleaned[returned[0]:]
	return prefix + "\n" + assignment + "\n" + suffix
}
