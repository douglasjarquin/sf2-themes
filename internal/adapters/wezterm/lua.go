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

// pointerAssignment is the `local sf2_current = <string>` line sf2's integration starts with, optionally followed by a
// comment. Group 1 is a double-quoted body and group 2 a single-quoted one.
var pointerAssignment = regexp.MustCompile(`^` + pycompat.Space + `*local` + pycompat.Space + `+sf2_current` +
	pycompat.Space + `*=` + pycompat.Space + `*(?:"((?:[^"\\]|\\.)*)"|'((?:[^'\\]|\\.)*)')` + pycompat.Space + `*(?:--.*)?$`)

// unescapeLua decodes a Lua short string body that uses only the escapes luaPathLiteral writes, plus \'.
func unescapeLua(body string) (string, bool) {
	var b strings.Builder
	for i := 0; i < len(body); i++ {
		if body[i] != '\\' {
			b.WriteByte(body[i])
			continue
		}
		i++
		if i == len(body) || !strings.ContainsRune(`\"'`, rune(body[i])) {
			return "", false
		}
		b.WriteByte(body[i])
	}
	return b.String(), true
}

// assignedPointer returns the pointer path the one sf2_current assignment on a code line names. Comments never
// count, and a config with no such assignment, more than one, or a string sf2 cannot decode names none.
func assignedPointer(existing string) (string, bool) {
	var named []string
	for _, line := range pycompat.SplitLines(code(existing)) {
		match := pointerAssignment.FindStringSubmatchIndex(line)
		if match == nil {
			continue
		}
		var body string
		if match[2] >= 0 {
			body = line[match[2]:match[3]]
		} else {
			body = line[match[4]:match[5]]
		}
		path, ok := unescapeLua(body)
		if !ok {
			return "", false
		}
		named = append(named, path)
	}
	if len(named) != 1 {
		return "", false
	}
	return named[0], true
}

// retarget moves an integration from the legacy shared pointer to the target's own pointer by rewriting the one
// sf2_current assignment that names it; every other line, including comments, is kept byte for byte.
func retarget(existing, legacy, pointer string) string {
	if named, ok := assignedPointer(existing); !ok || named != legacy {
		return existing
	}
	lines := pycompat.SplitLinesKeepEnds(existing)
	for i, line := range lines {
		body := strings.TrimRight(line, "\r\n")
		if strings.HasPrefix(pycompat.LStrip(body), "--") {
			continue
		}
		match := pointerAssignment.FindStringSubmatchIndex(body)
		if match == nil {
			continue
		}
		start, end := match[4]-1, match[5]+1
		if match[2] >= 0 {
			start, end = match[2]-1, match[3]+1
		}
		lines[i] = body[:start] + `"` + luaPathLiteral(pointer) + `"` + body[end:] + line[len(body):]
	}
	return strings.Join(lines, "")
}

func referencesPointer(code, pointer string) bool {
	return strings.Contains(code, "wezterm-current") &&
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
func classify(existing, pointer, legacy string) plan.Ownership {
	if pycompat.Strip(existing) == "" {
		return plan.Absent
	}
	code := code(existing)
	if referencesPointer(code, pointer) {
		// Only an integration that loads this target's pointer, or the legacy shared one sf2 can migrate, is sf2's.
		if named, ok := assignedPointer(existing); !ok || (named != pointer && named != legacy) {
			return plan.UnknownShape
		}
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
func integrate(existing, pointer, legacy string) string {
	if pycompat.Strip(existing) == "" {
		return starterConfig(pointer)
	}
	if classify(existing, pointer, legacy) == plan.UnknownShape {
		return existing
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
