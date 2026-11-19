package wezterm

import (
	"regexp"
	"strings"

	"github.com/douglasjarquin/sf2-themes/v2/internal/fsys"
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

// code is existing with comments and long-bracket strings blanked, so commented-out or quoted integration never
// counts. Offsets and line breaks are kept, so a line of code is the same line of existing.
func code(existing string) string { return blankInactive(existing) }

// lineBreakBytes are the ASCII bytes Python's splitlines breaks on; blanking keeps them so lines stay aligned.
const lineBreakBytes = "\n\r\v\f\x1c\x1d\x1e"

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

// assignment is the one active `local sf2_current = "<path>"` of an integration: the decoded path and the byte range
// of its string literal.
type assignment struct {
	path       string
	start, end int
}

// selectorAssignment returns the integration's pointer assignment. Only code counts: comments and strings never do.
// A config with no assignment, more than one assignment to sf2_current, a value sf2 cannot decode, or no
// `dofile(sf2_current)` that loads it has none.
func selectorAssignment(existing string) (assignment, bool) {
	tokens := codeTokens(lex(existing))
	is := func(i int, kind tokenKind, text string) bool {
		return i >= 0 && i < len(tokens) && tokens[i].kind == kind && (text == "" || tokens[i].text(existing) == text)
	}
	var found []assignment
	loaded := false
	for i := range tokens {
		if !is(i, nameToken, "sf2_current") {
			continue
		}
		if is(i-2, nameToken, "dofile") && is(i-1, otherToken, "(") && is(i+1, otherToken, ")") {
			loaded = true
		}
		if !is(i+1, otherToken, "=") {
			continue
		}
		literal := tokens[min(i+2, len(tokens)-1)]
		if !is(i-1, nameToken, "local") || !is(i+2, stringToken, "") || literal.end-literal.start < 2 ||
			literal.text(existing)[0] != existing[literal.end-1] {
			return assignment{}, false
		}
		// A literal that is only part of an expression is not the path.
		for _, operator := range []string{".", ":", "[", "("} {
			if is(i+3, otherToken, operator) {
				return assignment{}, false
			}
		}
		path, ok := unescapeLua(existing[literal.start+1 : literal.end-1])
		if !ok {
			return assignment{}, false
		}
		found = append(found, assignment{path: path, start: literal.start, end: literal.end})
	}
	if len(found) != 1 || !loaded {
		return assignment{}, false
	}
	return found[0], true
}

// assignedPointer returns the pointer path the active integration loads.
func assignedPointer(existing string) (string, bool) {
	named, ok := selectorAssignment(existing)
	return named.path, ok
}

// namesAny reports whether path is the same location as any of paths.
func namesAny(path string, paths []string) bool {
	for _, candidate := range paths {
		if fsys.SamePath(path, candidate) {
			return true
		}
	}
	return false
}

// retarget moves an integration from a former pointer to the target's own pointer by rewriting the string literal of
// the one active sf2_current assignment that names it; every other byte, including comments, is kept.
func retarget(existing string, former []string, pointer string) string {
	named, ok := selectorAssignment(existing)
	if !ok || !namesAny(named.path, former) {
		return existing
	}
	return existing[:named.start] + `"` + luaPathLiteral(pointer) + `"` + existing[named.end:]
}

// referencesPointer reports whether the code, not a comment or long string, refers to an sf2 pointer.
func referencesPointer(existing string) bool {
	for _, t := range codeTokens(lex(existing)) {
		if t.kind == stringToken && strings.Contains(t.text(existing), "wezterm-current") {
			return true
		}
	}
	return false
}

// upgradeTermThemeGuard inserts TERM_THEME wiring after the managed color_scheme assignment when possible.
func upgradeTermThemeGuard(existing string) (string, bool) {
	active := code(existing)
	builder := builderPattern.FindStringSubmatch(active)
	if builder == nil {
		return "", false
	}
	name := builder[1]
	lines := pycompat.SplitLinesKeepEnds(existing)
	schemeIdx := -1
	for i, line := range pycompat.SplitLinesKeepEnds(active) {
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

// stripSchemeLines drops the lines of code whose color_scheme assignment is one of kinds; an assignment in a comment
// or long string is not code and stays.
func stripSchemeLines(existing string, kinds map[string]bool) string {
	active := pycompat.SplitLines(code(existing))
	var kept []string
	for i, line := range pycompat.SplitLines(existing) {
		if !kinds[schemeKind(active[i])] {
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
	for _, line := range pycompat.SplitLines(code(existing)) {
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

// classify reports who owns the color scheme in wezterm.lua. former are the pointers earlier releases used for this
// target, which an integration may still load until sf2 moves it to pointer.
//
// Pointer integration is recognised before the config shape, so a pasted snippet in a config sf2-themes cannot
// edit still counts as managed.
func classify(existing, pointer string, former []string) plan.Ownership {
	if pycompat.Strip(existing) == "" {
		return plan.Absent
	}
	code := code(existing)
	if referencesPointer(existing) {
		// Only an integration that loads this target's pointer, or a former one sf2 can migrate, is sf2's.
		if named, ok := assignedPointer(existing); !ok || !namesAny(named, append([]string{pointer}, former...)) {
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

// integrate returns wezterm.lua with the managed pointer integration, moving an integration that loads a former
// pointer to pointer.
//
// The caller has already classified the config and approved replacing any color-scheme assignment; a shape
// classify rejects is returned unchanged.
func integrate(existing, pointer string, former []string) string {
	if pycompat.Strip(existing) == "" {
		return starterConfig(pointer)
	}
	if classify(existing, pointer, former) == plan.UnknownShape {
		return existing
	}
	existing = retarget(existing, former, pointer)
	code := code(existing)
	if referencesPointer(existing) {
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
	returned := returnNamePattern.FindStringIndex(blankInactive(cleaned))
	if !named || returned == nil {
		return existing
	}
	assignment := pycompat.RStrip(integrationSnippet(pointer, name))
	prefix := pycompat.RStrip(cleaned[:returned[0]])
	suffix := cleaned[returned[0]:]
	return prefix + "\n" + assignment + "\n" + suffix
}
