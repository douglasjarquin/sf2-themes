package wezterm

import (
	"errors"
	"regexp"
	"slices"
	"strings"

	"github.com/douglasjarquin/sf2-themes/v2/internal/fsys"
	"github.com/douglasjarquin/sf2-themes/v2/internal/plan"
	"github.com/douglasjarquin/sf2-themes/v2/internal/pycompat"
)

// wezterm.lua ownership is exact recognition of what sf2 generates; anything else is the user's. Guessing is a
// failure, and sf2 refuses rather than edit code it did not write.

const (
	watchLine  = "wezterm.add_to_config_reload_watch_list(sf2_current)"
	schemeLine = "local sf2_scheme = dofile(sf2_current)"
)

var (
	termThemeAssign = `sf2_env.TERM_THEME = (type(sf2_scheme) == "string" and sf2_scheme:find("-light", 1, true)) and "light" or "dark"`
	sf2SchemeValue  = regexp.MustCompile(`^"(?:sf2-[^"]*|street-fighter-2|Street Fighter II - [^"]*|street-fighter-ii-[^"]*)"$`)
	selectorLine    = regexp.MustCompile(`^local sf2_current = "((?:[^"\\]|\\.)*)"$`)
	builderUse      = regexp.MustCompile(`^(` + pycompat.Word + `+)\.color_scheme = (sf2_scheme|dofile\(sf2_current\))$`)
	indentPattern   = regexp.MustCompile(`^[ \t]*`)
	// pointerName is every name sf2 has given a selection pointer.
	pointerName = regexp.MustCompile(`^wezterm-current(-[0-9a-f]{12})?\.lua$`)
)

// luaPathLiteral escapes a filesystem path for a Lua double-quoted string.
func luaPathLiteral(path string) string {
	return strings.ReplaceAll(strings.ReplaceAll(path, `\`, `\\`), `"`, `\"`)
}

// integrationLines is the block sf2 generates, without line breaks.
func integrationLines(pointer, builder string) []string {
	return []string{
		`local sf2_current = "` + luaPathLiteral(pointer) + `"`,
		watchLine,
		schemeLine,
		builder + ".color_scheme = sf2_scheme",
		"local sf2_env = " + builder + ".set_environment_variables or {}",
		termThemeAssign,
		builder + ".set_environment_variables = sf2_env",
	}
}

// integrationSnippet returns the exact Lua block a human should paste.
//
// TERM_THEME follows the selected sf2 scheme so Cursor Agent and similar TUIs do not guess light chrome against a
// dark scheme (or the reverse).
func integrationSnippet(pointer, builder string) string {
	return strings.Join(integrationLines(pointer, builder), "\n") + "\n"
}

// starterConfig returns a complete starter wezterm.lua that selects the managed scheme.
func starterConfig(pointer string) string {
	return `local wezterm = require("wezterm")` + "\nlocal config = wezterm.config_builder()\n" +
		integrationSnippet(pointer, "config") + "return config\n"
}

// unescapeLua decodes a Lua short string body that uses only the escapes luaPathLiteral writes.
func unescapeLua(body string) (string, bool) {
	var b strings.Builder
	for i := 0; i < len(body); i++ {
		if body[i] != '\\' {
			b.WriteByte(body[i])
			continue
		}
		i++
		if i == len(body) || (body[i] != '\\' && body[i] != '"') {
			return "", false
		}
		b.WriteByte(body[i])
	}
	return b.String(), true
}

// blockKind is a form of sf2 integration.
type blockKind int

const (
	// fullBlock is the integration sf2 writes today.
	fullBlock blockKind = iota
	// unguardedBlock is the integration releases before TERM_THEME wrote: the selector, the watch line, and
	// `<builder>.color_scheme = dofile(sf2_current)`.
	unguardedBlock
	// markerBlock is the first three lines of fullBlock, followed by the user's own use of sf2_scheme: the snippet sf2
	// prints, pasted and adapted by hand.
	markerBlock
)

// block is one recognized integration: its kind, the pointer it loads, the byte range of its lines, the range of the
// pointer's string literal, its indent, and the builder it assigns.
type block struct {
	kind         blockKind
	pointer      string
	start, end   int
	literalStart int
	literalEnd   int
	indent       string
	builder      string
}

// line is one source line: the range of its content and the end including its line break.
type line struct{ start, end, next int }

func lines(source string) []line {
	var all []line
	for start := 0; start < len(source); {
		end := lineEnd(source, start)
		next := end
		if strings.HasPrefix(source[end:], "\r\n") {
			next += 2
		} else if next < len(source) {
			next++
		}
		all = append(all, line{start, end, next})
		start = next
	}
	return all
}

// recognize returns every sf2 integration at the top level of a completely lexed file: its lines exactly as sf2
// generates them under one indent, starting at a token boundary outside any function, block, or bracket.
func recognize(s scan) []block {
	var found []block
	all := lines(s.source)
	text := func(i int, indent string) (string, bool) {
		if i >= len(all) {
			return "", false
		}
		content := s.source[all[i].start:all[i].end]
		return strings.TrimPrefix(content, indent), strings.HasPrefix(content, indent)
	}
	for i := range all {
		indent := indentPattern.FindString(s.source[all[i].start:all[i].end])
		first, _ := text(i, indent)
		match := selectorLine.FindStringSubmatchIndex(first)
		if match == nil {
			continue
		}
		body := first[match[2]:match[3]]
		pointer, ok := unescapeLua(body)
		if !ok || luaPathLiteral(pointer) != body {
			continue
		}
		if watch, ok := text(i+1, indent); !ok || watch != watchLine {
			continue
		}
		b := block{pointer: pointer, start: all[i].start, indent: indent,
			literalStart: all[i].start + len(indent) + match[2] - 1, literalEnd: all[i].start + len(indent) + match[3] + 1}
		third, _ := text(i+2, indent)
		last := i + 2
		switch use := builderUse.FindStringSubmatch(third); {
		case use != nil && use[2] == "dofile(sf2_current)":
			b.kind, b.builder = unguardedBlock, use[1]
		case third == schemeLine:
			b.kind = markerBlock
			if use, _ := text(i+3, indent); builderUse.MatchString(use) {
				b.builder = builderUse.FindStringSubmatch(use)[1]
				full := integrationLines(pointer, b.builder)[3:]
				matches := true
				for j, want := range full {
					if got, ok := text(i+3+j, indent); !ok || got != want {
						matches = false
					}
				}
				if matches {
					b.kind, last = fullBlock, i+2+len(full)
				}
			}
		default:
			continue
		}
		b.end = all[last].next
		at := s.codeAt(b.start)
		if !s.boundary(b.start) || !s.boundary(b.end) || at == len(s.code) || s.code[at].depth != 0 || s.code[at].nest != 0 {
			continue
		}
		found = append(found, b)
	}
	return found
}

// assignment is one `color_scheme =` in code: the index of its `color_scheme` name and, when sf2 can remove it, the
// byte range of the whole statement.
type assignment struct {
	at         int
	removable  bool
	start, end int
	sf2        bool
	usesScheme bool
}

// assignments returns every color_scheme assignment in code outside skip.
func assignments(s scan, skip *block, builder string) []assignment {
	var found []assignment
	for i, t := range s.code {
		if skip != nil && t.start >= skip.start && t.start < skip.end {
			continue
		}
		field := s.is(i, nameToken, "color_scheme") && s.is(i+1, otherToken, "=")
		indexed := s.is(i, stringToken, "") && strings.Trim(t.text(s.source), `"'`) == "color_scheme" &&
			s.is(i+1, otherToken, "]") && s.is(i+2, otherToken, "=")
		if !field && !indexed {
			continue
		}
		a := assignment{at: i}
		if field {
			a.start, a.end, a.removable = removableStatement(s, i, builder)
			value := s.code[i+2 : min(i+3, len(s.code))]
			if a.removable && len(value) == 1 && s.codeAt(a.end) == i+3 {
				a.sf2 = value[0].kind == stringToken && sf2SchemeValue.MatchString(value[0].text(s.source))
				a.usesScheme = value[0].kind == nameToken && value[0].text(s.source) == "sf2_scheme"
			} else if s.is(i+2, nameToken, "sf2_scheme") && (s.is(i+3, otherToken, ",") || s.is(i+3, otherToken, "}")) {
				a.usesScheme = true
			}
		}
		found = append(found, a)
	}
	return found
}

// removableStatement reports whether the color_scheme name at code index i is the whole top-level statement
// `<builder>.color_scheme = <value>` that starts its line and ends on it, and returns that statement's range. Its
// value balances and ends the expression, and the next code begins a new statement, so removing the range leaves
// valid Lua and every neighbouring token in place.
func removableStatement(s scan, i int, builder string) (int, int, bool) {
	if i < 2 || !s.is(i-1, otherToken, ".") || !s.is(i-2, nameToken, builder) {
		return 0, 0, false
	}
	first := s.code[i-2]
	lineStart, lineStop := lineOf(s.source, first.start)
	if first.depth != 0 || first.nest != 0 || (i > 2 && s.code[i-3].end > lineStart) {
		return 0, 0, false
	}
	last, open := i+1, 0
	for j := i + 2; j < len(s.code) && s.code[j].start < lineStop; j++ {
		if s.code[j].end > lineStop {
			return 0, 0, false
		}
		switch s.code[j].text(s.source) {
		case "(", "[", "{":
			open++
		case ")", "]", "}":
			open--
		}
		last = j
	}
	end := s.code[last]
	if last == i+1 || open != 0 || !endsExpression(s, end) {
		return 0, 0, false
	}
	if last+1 < len(s.code) && continuesExpression(s, s.code[last+1]) {
		return 0, 0, false
	}
	return first.start, end.end, true
}

var reserved = map[string]bool{
	"and": true, "break": true, "do": true, "else": true, "elseif": true, "end": true, "for": true, "function": true,
	"goto": true, "if": true, "in": true, "local": true, "not": true, "or": true, "repeat": true, "return": true,
	"then": true, "until": true, "while": true,
}

func endsExpression(s scan, t token) bool {
	text := t.text(s.source)
	switch t.kind {
	case nameToken:
		return !reserved[text]
	case stringToken, longStringToken, numberToken:
		return true
	}
	return text == ")" || text == "]" || text == "}"
}

func continuesExpression(s scan, t token) bool {
	text := t.text(s.source)
	switch t.kind {
	case nameToken:
		return text == "and" || text == "or"
	case stringToken, longStringToken:
		return true
	case otherToken:
		return text != ";" && text != "::"
	}
	return false
}

// builderName returns the name of a top-level `local <name> = wezterm.config_builder()` that the chunk ends by
// returning, and the start of that `return` line.
func builderName(s scan) (string, int, bool) {
	n := len(s.code)
	if n < 2 || !s.is(n-2, nameToken, "return") || !s.is(n-1, nameToken, "") || s.code[n-2].depth != 0 {
		return "", 0, false
	}
	name := s.code[n-1].text(s.source)
	returnLine, _ := lineOf(s.source, s.code[n-2].start)
	if i := s.codeAt(returnLine); i != n-2 {
		return "", 0, false
	}
	for i := range s.code {
		if s.is(i, nameToken, "local") && s.is(i+1, nameToken, name) && s.is(i+2, otherToken, "=") &&
			s.is(i+3, nameToken, "wezterm") && s.is(i+4, otherToken, ".") && s.is(i+5, nameToken, "config_builder") &&
			s.is(i+6, otherToken, "(") && s.is(i+7, otherToken, ")") && s.code[i].depth == 0 && s.code[i].nest == 0 {
			return name, returnLine, true
		}
	}
	return "", 0, false
}

// mentionsSF2 reports whether code outside skip names an sf2 integration variable or a pointer path.
func mentionsSF2(s scan, skip *block) bool {
	for _, t := range s.code {
		if skip != nil && t.start >= skip.start && t.start < skip.end {
			continue
		}
		text := t.text(s.source)
		if (t.kind == nameToken && (text == "sf2_current" || text == "sf2_scheme")) ||
			(t.kind == stringToken && strings.Contains(text, "wezterm-current")) {
			return true
		}
	}
	return false
}

// usesMarker reports whether top-level code after a marker block reads sf2_scheme and nothing assigns it again.
func usesMarker(s scan, b block) bool {
	used := false
	for i := s.codeAt(b.end); i < len(s.code); i++ {
		if !s.is(i, nameToken, "sf2_scheme") {
			continue
		}
		if s.is(i+1, otherToken, "=") || s.is(i-1, nameToken, "local") {
			return false
		}
		used = used || s.code[i].depth == 0
	}
	return used
}

// analysis is what sf2 knows about one wezterm.lua.
type analysis struct {
	scan        scan
	ownership   plan.Ownership
	block       *block
	assignments []assignment
	builder     string
	returnLine  int
	// selector is the pointer the configuration loads, or "" when sf2 cannot tell.
	selector string
}

// analyze classifies existing for the target whose own pointer is pointer.
//
//	recognized                                     ownership
//	nothing at all                                 absent
//	file that does not lex completely              unknown shape
//	more than one integration                      unknown shape
//	integration loading a path that is no pointer  unknown shape
//	full or unguarded block, other color_scheme    foreign
//	full block                                     managed
//	unguarded block                                legacy
//	marker whose sf2_scheme top-level code uses    managed (unless another color_scheme)
//	builder config with only sf2 literal schemes   legacy
//	builder config with another color_scheme       foreign
//	builder config without any                     absent (unknown shape if it mentions sf2's names)
//	anything else                                  unknown shape
func analyze(existing, pointer string) analysis {
	a := analysis{scan: lexLua(existing), ownership: plan.UnknownShape}
	if pycompat.Strip(existing) == "" {
		a.ownership, a.selector = plan.Absent, pointer
		return a
	}
	if !a.scan.complete {
		return a
	}
	a.builder, a.returnLine, _ = builderName(a.scan)
	blocks := recognize(a.scan)
	if len(blocks) > 1 {
		return a
	}
	if len(blocks) == 1 {
		b := blocks[0]
		a.block = &b
		if b.builder != "" {
			a.builder = b.builder
		}
		if !fsys.SamePath(b.pointer, pointer) && !pointerName.MatchString(pycompat.Name(b.pointer)) {
			return a
		}
		a.selector = b.pointer
		a.assignments = assignments(a.scan, &b, a.builder)
		others := slices.ContainsFunc(a.assignments, func(x assignment) bool { return !x.usesScheme })
		switch {
		case b.kind == markerBlock && (!usesMarker(a.scan, b) || others):
			a.selector = ""
		case others:
			a.ownership = plan.Foreign
		case b.kind == unguardedBlock:
			a.ownership = plan.Legacy
		default:
			a.ownership = plan.Managed
		}
		return a
	}
	a.selector = pointer
	if a.builder == "" {
		a.selector = ""
		return a
	}
	a.assignments = assignments(a.scan, nil, a.builder)
	switch {
	case slices.ContainsFunc(a.assignments, func(x assignment) bool { return !x.sf2 }):
		a.ownership = plan.Foreign
	case len(a.assignments) > 0:
		a.ownership = plan.Legacy
	case mentionsSF2(a.scan, nil):
		a.selector = ""
	default:
		a.ownership = plan.Absent
	}
	return a
}

// classify reports who owns the color scheme in wezterm.lua.
func classify(existing, pointer string) plan.Ownership { return analyze(existing, pointer).ownership }

// assignedPointer returns the pointer the recognized integration loads.
func assignedPointer(existing, pointer string) (string, bool) {
	a := analyze(existing, pointer)
	return a.selector, a.block != nil && a.selector != ""
}

// edit is a replacement of source[start:end].
type edit struct {
	start, end  int
	replacement string
}

func applyEdits(source string, edits []edit) string {
	slices.SortFunc(edits, func(a, b edit) int { return b.start - a.start })
	for _, e := range edits {
		source = source[:e.start] + e.replacement + source[e.end:]
	}
	return source
}

// removal drops a statement's range, and the whole line when nothing else is on it.
func removal(source string, start, end int) edit {
	lineStart, lineStop := lineOf(source, start)
	if strings.TrimSpace(source[lineStart:start]) == "" && strings.TrimSpace(source[end:lineStop]) == "" {
		next := lineStop
		if strings.HasPrefix(source[next:], "\r\n") {
			next += 2
		} else if next < len(source) {
			next++
		}
		return edit{lineStart, next, ""}
	}
	return edit{start, end, ""}
}

var errCannotIntegrate = errors.New("sf2-themes cannot edit this WezTerm config safely")

// integrate returns wezterm.lua selecting pointer: a starter config for an empty file, the target's pointer in a
// recognized integration, an unguarded block upgraded to the full one, and otherwise the full block inserted before
// the final `return <builder>` with every removable color_scheme statement removed. It edits nothing else, and
// refuses when the result is not a config sf2 then recognizes as managed.
func integrate(existing, pointer string) (string, error) {
	a := analyze(existing, pointer)
	if a.ownership == plan.Absent && pycompat.Strip(existing) == "" {
		return starterConfig(pointer), nil
	}
	if a.ownership == plan.UnknownShape {
		return "", errCannotIntegrate
	}
	var edits []edit
	for _, x := range a.assignments {
		if x.usesScheme && a.block != nil && a.block.kind == markerBlock {
			continue
		}
		if !x.removable {
			return "", errCannotIntegrate
		}
		edits = append(edits, removal(existing, x.start, x.end))
	}
	if b := a.block; b != nil {
		edits = append(edits, edit{b.literalStart, b.literalEnd, `"` + luaPathLiteral(pointer) + `"`})
		if b.kind == unguardedBlock {
			third := lines(existing[b.start:b.end])[2]
			var upgraded []string
			for _, text := range integrationLines(pointer, b.builder)[2:] {
				upgraded = append(upgraded, b.indent+text)
			}
			edits = append(edits, edit{b.start + third.start, b.start + third.end, strings.Join(upgraded, "\n")})
		}
	}
	updated := applyEdits(existing, edits)
	if a.block == nil {
		name, returnLine, ok := builderName(lexLua(updated))
		if !ok {
			return "", errCannotIntegrate
		}
		updated = pycompat.RStrip(updated[:returnLine]) + "\n" + integrationSnippet(pointer, name) + updated[returnLine:]
	}
	if check := analyze(updated, pointer); check.ownership != plan.Managed || !fsys.SamePath(check.selector, pointer) {
		return "", errCannotIntegrate
	}
	return updated, nil
}
