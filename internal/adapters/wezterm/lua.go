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

// wezterm.lua ownership is exact recognition of what sf2 generated, in a file that parses completely. sf2 writes a
// new file, replaces one of its own forms as a whole, or, with --adopt, inserts its block into a file whose every
// line it can verify. It never edits a line the user wrote; anything else is left to the user, with exact steps.

const (
	watchLine  = "wezterm.add_to_config_reload_watch_list(sf2_current)"
	schemeLine = "local sf2_scheme = dofile(sf2_current)"
)

var (
	termThemeAssign = `sf2_env.TERM_THEME = (type(sf2_scheme) == "string" and sf2_scheme:find("-light", 1, true)) and "light" or "dark"`
	selectorLine    = regexp.MustCompile(`^local sf2_current = "((?:[^"\\]|\\.)*)"$`)
	unguardedUse    = regexp.MustCompile(`^(` + pycompat.Word + `+)\.color_scheme = dofile\(sf2_current\)$`)
	guardedUse      = regexp.MustCompile(`^(` + pycompat.Word + `+)\.color_scheme = sf2_scheme$`)
	// literalScheme is the line the first releases wrote: a builder's color_scheme set to an sf2 scheme name.
	literalScheme  = regexp.MustCompile(`^(` + pycompat.Word + `+)\.color_scheme = "(?:sf2-[a-z0-9-]+|street-fighter-2|Street Fighter II - [A-Za-z0-9 .-]+|street-fighter-ii-[a-z0-9-]+)"$`)
	requireWezterm = regexp.MustCompile(`^local wezterm = require\(?["']wezterm["']\)?$`)
	indentPattern  = regexp.MustCompile(`^[ \t]*`)
	// pointerName is every name sf2 has given a selection pointer.
	pointerName = regexp.MustCompile(`^wezterm-current(-[0-9a-f]{12})?\.lua$`)
)

// luaPathLiteral escapes a filesystem path for a Lua double-quoted string. Callers refuse paths with control
// characters, so backslash and quote are the only characters that need escaping.
func luaPathLiteral(path string) string {
	return strings.ReplaceAll(strings.ReplaceAll(path, `\`, `\\`), `"`, `\"`)
}

// integrationLines is the block sf2 writes today, without line breaks.
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

// formKind is one integration sf2 has generated.
type formKind int

const (
	// fullForm is the block sf2 writes today.
	fullForm formKind = iota
	// unguardedForm is the block releases before TERM_THEME wrote: the selector, the watch line, and
	// `<builder>.color_scheme = dofile(sf2_current)`.
	unguardedForm
	// literalForm is the single line the first releases wrote: `<builder>.color_scheme = "<sf2 scheme>"`.
	literalForm
)

// form is one recognized integration: its kind, the pointer it loads (none for literalForm), its builder, its indent,
// and the byte range of its lines without the last line break.
type form struct {
	kind       formKind
	pointer    string
	builder    string
	indent     string
	start, end int
}

// line is the byte range of one line's content.
type line struct{ start, end int }

func lines(source string) []line {
	var all []line
	for start := 0; start < len(source); {
		end := lineEnd(source, start)
		all = append(all, line{start, end})
		start = end
		if strings.HasPrefix(source[start:], "\r\n") {
			start += 2
		} else if start < len(source) {
			start++
		}
	}
	return all
}

// recognize returns each generated form in c: exact generated lines under one indent that are whole statements at
// the top level of the chunk.
func recognize(c chunk) []form {
	all := lines(c.source)
	text := func(i int, indent string) string {
		if i >= len(all) {
			return ""
		}
		content := c.source[all[i].start:all[i].end]
		if !strings.HasPrefix(content, indent) {
			return ""
		}
		return content[len(indent):]
	}
	var found []form
	for i := range all {
		indent := indentPattern.FindString(c.source[all[i].start:all[i].end])
		first := text(i, indent)
		f := form{indent: indent, start: all[i].start}
		last := i
		if use := literalScheme.FindStringSubmatch(first); use != nil {
			f.kind, f.builder = literalForm, use[1]
		} else if match := selectorLine.FindStringSubmatch(first); match != nil {
			pointer, ok := unescapeLua(match[1])
			if !ok || luaPathLiteral(pointer) != match[1] || text(i+1, indent) != watchLine {
				continue
			}
			f.pointer = pointer
			if use := unguardedUse.FindStringSubmatch(text(i+2, indent)); use != nil {
				f.kind, f.builder, last = unguardedForm, use[1], i+2
			} else if use := guardedUse.FindStringSubmatch(text(i+3, indent)); use != nil && text(i+2, indent) == schemeLine {
				f.kind, f.builder, last = fullForm, use[1], i+6
				for j, want := range integrationLines(pointer, f.builder) {
					if text(i+j, indent) != want {
						f.builder = ""
					}
				}
				if f.builder == "" {
					continue
				}
			} else {
				continue
			}
		} else {
			continue
		}
		f.end = all[last].end
		if c.wholeStatements(f.start+len(f.indent), f.end) {
			found = append(found, f)
		}
	}
	return found
}

// wholeStatements reports whether [start, end) is exactly a run of consecutive top-level statements.
func (c chunk) wholeStatements(start, end int) bool {
	for i, s := range c.statements {
		if s.start != start {
			continue
		}
		for _, next := range c.statements[i:] {
			if next.end == end {
				return true
			}
			if next.end > end {
				return false
			}
		}
	}
	return false
}

// topLevelBefore reports whether a top-level statement ending before offset matches pattern.
func (c chunk) topLevelBefore(offset int, pattern *regexp.Regexp) bool {
	for _, s := range c.statements {
		if s.end <= offset && pattern.MatchString(c.source[s.start:s.end]) {
			return true
		}
	}
	return false
}

// mentions reports whether code outside [start, end) names color_scheme, or names an sf2 integration variable or
// pointer: something that decides the scheme besides sf2, or an older sf2 loader.
func (c chunk) mentions(start, end int) (scheme, loader bool) {
	for _, t := range c.tokens {
		if t.kind == commentToken || (t.start >= start && t.start < end) {
			continue
		}
		text := t.text(c.source)
		switch t.kind {
		case nameToken:
			scheme = scheme || text == "color_scheme"
			loader = loader || text == "sf2_current" || text == "sf2_scheme" || text == "sf2_env"
		case stringToken, longStringToken:
			scheme = scheme || strings.Contains(text, "color_scheme")
			loader = loader || strings.Contains(text, "wezterm-current")
		}
	}
	return scheme, loader
}

// tinyLine is every line a file sf2 adopts may hold, apart from the builder and its return.
var tinyLine = []*regexp.Regexp{
	regexp.MustCompile(`^$`),
	regexp.MustCompile(`^--([^\[].*)?$`),
	requireWezterm,
}

var (
	tinyBuilder = regexp.MustCompile(`^local (` + pycompat.Word + `+) = wezterm\.config_builder\(\)$`)
	tinyValue   = `(?:-?[0-9]+(?:\.[0-9]+)?|true|false|"[^"\\]*"|'[^'\\]*'|wezterm\.[a-z_]+\((?:"[^"\\]*"|-?[0-9]+(?:\.[0-9]+)?)?\))`
)

// adoptable returns the builder of a file sf2 can verify line by line and the start of its final `return` line:
// blank lines, line comments, `local wezterm = require("wezterm")`, one `local <builder> =
// wezterm.config_builder()`, single-line `<builder>.<setting> = <literal>` assignments other than color_scheme, and
// a final `return <builder>`.
func adoptable(source string) (string, int, bool) {
	builder, returnAt := "", -1
	var setting, final *regexp.Regexp
	for _, l := range lines(source) {
		text := strings.TrimSpace(source[l.start:l.end])
		switch {
		case returnAt >= 0 && text != "" && !strings.HasPrefix(text, "--"):
			return "", 0, false
		case builder == "" && tinyBuilder.MatchString(text):
			builder = tinyBuilder.FindStringSubmatch(text)[1]
			quoted := regexp.QuoteMeta(builder)
			setting = regexp.MustCompile(`^` + quoted + `\.(` + pycompat.Word + `+) = ` + tinyValue + `$`)
			final = regexp.MustCompile(`^return ` + quoted + `$`)
		case final != nil && final.MatchString(text):
			returnAt = l.start
		case setting != nil && setting.MatchString(text):
			if setting.FindStringSubmatch(text)[1] == "color_scheme" {
				return "", 0, false
			}
		case strings.HasPrefix(text, "--[") || !matchesAny(tinyLine, text):
			return "", 0, false
		}
	}
	return builder, returnAt, builder != "" && returnAt >= 0
}

func matchesAny(patterns []*regexp.Regexp, text string) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(text) {
			return true
		}
	}
	return false
}

// analysis is what sf2 knows about one wezterm.lua.
type analysis struct {
	ownership plan.Ownership
	form      *form
	// selector is the pointer the configuration loads, or "" when it loads none sf2 can vouch for.
	selector string
	// builder and returnAt place an adopted block, when the file is adoptable.
	builder  string
	returnAt int
	// manual is the reason sf2 will not write the setting itself, and what the user does instead.
	manual string
}

// analyze classifies existing for the target whose own pointer is pointer.
//
//	missing or blank file                                   absent: sf2 creates it
//	not a Lua chunk sf2 can verify                          broken: refused before any write
//	more than one sf2 form                                  unknown shape
//	full form, at the top level, nothing else sets scheme   managed (moved off a former pointer as a whole)
//	unguarded or literal form, likewise                     legacy: replaced by the full form as a whole
//	an sf2 form beside another color_scheme                 foreign: manual
//	no sf2 form, adoptable                                  foreign: --adopt inserts the full form
//	no sf2 form, any color_scheme                           foreign: manual
//	anything else                                           unknown shape: manual
func analyze(existing, pointer, lua string) analysis {
	if strings.Trim(existing, luaSpace) == "" {
		return analysis{ownership: plan.Absent, selector: pointer}
	}
	c, err := parseLua(existing)
	if err != nil {
		return analysis{ownership: plan.Broken, manual: lua + " is not Lua sf2-themes can verify: " + err.Error()}
	}
	forms := recognize(c)
	if len(forms) > 1 {
		return analysis{ownership: plan.UnknownShape, manual: manualSteps(lua, pointer, "config", false, false,
			"It holds more than one sf2-themes integration; keep one.")}
	}
	if len(forms) == 1 {
		f := forms[0]
		scheme, loader := c.mentions(f.start, f.end)
		declares := regexp.MustCompile(`^local ` + regexp.QuoteMeta(f.builder) + `\b`)
		a := analysis{form: &f, selector: f.pointer}
		switch {
		case c.rebinds(f.start, f.builder) != "":
			a.ownership, a.selector = plan.UnknownShape, ""
			a.manual = manualSteps(lua, pointer, f.builder, false, false, shadowed(c.rebinds(f.start, f.builder)))
		case f.kind != literalForm && !fsys.SamePath(f.pointer, pointer) && !pointerName.MatchString(pycompat.Name(f.pointer)):
			a.ownership, a.selector = plan.UnknownShape, ""
			a.manual = manualSteps(lua, pointer, f.builder, false, true, "Its integration loads a file that is no sf2-themes pointer.")
		case c.returned != f.builder || !c.topLevelBefore(f.start, requireWezterm) || !c.topLevelBefore(f.start, declares):
			a.ownership, a.selector = plan.UnknownShape, ""
			a.manual = manualSteps(lua, pointer, f.builder, false, true, "")
		case scheme || loader:
			a.ownership, a.selector = plan.Foreign, ""
			a.manual = manualSteps(lua, pointer, f.builder, scheme, loader, "")
		case f.kind == fullForm:
			a.ownership = plan.Managed
		default:
			a.ownership = plan.Legacy
			if f.kind == literalForm {
				a.selector = pointer
			}
		}
		return a
	}
	scheme, loader := c.mentions(0, 0)
	if builder, returnAt, ok := adoptable(existing); ok && c.rebinds(returnAt, builder) != "" {
		why := shadowed(c.rebinds(returnAt, builder))
		return analysis{ownership: plan.UnknownShape, manual: manualSteps(lua, pointer, builder, false, false, why)}
	} else if ok && !loader {
		return analysis{ownership: plan.Foreign, builder: builder, returnAt: returnAt}
	}
	a := analysis{ownership: plan.UnknownShape, manual: manualSteps(lua, pointer, "config", scheme, loader, "")}
	if scheme {
		a.ownership = plan.Foreign
	}
	return a
}

// blockFree and blockLocals are the names sf2's block reads from its surroundings and the locals it declares, read
// from the block's own text, so the check below cannot drift from what the block does. builderName stands for the
// builder there.
var blockFree, blockLocals = blockNames()

const builderName = "sf2_builder"

func blockNames() (free, locals []string) {
	source := strings.Join(integrationLines("", builderName), "\n")
	tokens, err := lexLua(source)
	if err != nil {
		panic(err)
	}
	var code []token
	for _, t := range tokens {
		if t.kind != commentToken {
			code = append(code, t)
		}
	}
	for i, t := range code {
		if t.kind != nameToken {
			continue
		}
		name := t.text(source)
		switch {
		case i > 0 && code[i-1].isKeyword(source, "local"):
			locals = append(locals, name)
		case i > 0 && code[i-1].kind == symbolToken && (code[i-1].text(source) == "." || code[i-1].text(source) == ":"):
			// A field name, not a variable.
		case name != builderName && !slices.Contains(locals, name) && !slices.Contains(free, name):
			free = append(free, name)
		}
	}
	return free, locals
}

// rebinds returns a name sf2's block, using builder, would not reach with its standard meaning at offset at, or "".
// The block reads its free names from its surroundings, so none may be bound before at: not as a local (the builder
// is one), a parameter, or a loop variable, and not assigned. The one binding allowed is the top-level `local wezterm
// = require("wezterm")` that gives `wezterm` its meaning, and `require` itself must keep its own. The builder must
// also not be a name the block declares, or the block would set its own local instead of the builder.
func (c chunk) rebinds(at int, builder string) string {
	if slices.Contains(blockLocals, builder) {
		return builder
	}
	watched := append(slices.Clone(blockFree), "require")
	for _, b := range c.bindings {
		if b.at >= at || !slices.Contains(watched, b.name) {
			continue
		}
		if b.name == "wezterm" && c.requiresWezterm(b.at) {
			continue
		}
		return b.name
	}
	return ""
}

// requiresWezterm reports whether the binding at offset at is a top-level `local wezterm = require("wezterm")`.
func (c chunk) requiresWezterm(at int) bool {
	for _, s := range c.statements {
		if s.start <= at && at < s.end {
			return requireWezterm.MatchString(c.source[s.start:s.end])
		}
	}
	return false
}

// shadowed is why sf2 will not add its block where name no longer has the meaning the block relies on.
func shadowed(name string) string {
	return "The integration uses " + name + ", which this file binds before the integration would run, so the " +
		"integration would not work. Rename that " + name + " first."
}

// manualSteps is how the user adds sf2's integration by hand when sf2 will not edit their file.
func manualSteps(lua, pointer, builder string, scheme, loader bool, why string) string {
	steps := "Add this integration to " + lua + " at the top level, after " + builder + " is created and before the " +
		"file returns it:\n\n" + integrationSnippet(pointer, builder)
	if why != "" {
		steps = why + "\n" + steps
	}
	if scheme {
		steps += "\nRemove the other color_scheme assignments, so this one selects the theme."
	}
	if loader {
		steps += "\nRemove the older sf2-themes loader: the lines that set or use sf2_current, sf2_scheme, or sf2_env."
	}
	return steps
}

// mixedBreaks is why sf2 will not add lines to a file whose lines do not all end alike.
const mixedBreaks = "Its lines do not all end alike, so sf2-themes cannot add lines that match them."

// lineBreak returns the one line break every line of source ends with, "\n" when it has none, and false when its
// lines end differently or with a lone carriage return.
func lineBreak(source string) (string, bool) {
	found := map[string]bool{}
	for _, b := range lineBreaks(source) {
		found[b] = true
	}
	switch {
	case len(found) == 0 || (len(found) == 1 && found["\n"]):
		return "\n", true
	case len(found) == 1 && found["\r\n"]:
		return "\r\n", true
	}
	return "", false
}

// lineBreaks returns each line break in source, in order.
func lineBreaks(source string) []string {
	var breaks []string
	for i := 0; i < len(source); i++ {
		switch {
		case strings.HasPrefix(source[i:], "\r\n"):
			breaks = append(breaks, "\r\n")
			i++
		case source[i] == '\n' || source[i] == '\r':
			breaks = append(breaks, source[i:i+1])
		}
	}
	return breaks
}

// classify reports who owns the color scheme in wezterm.lua.
func classify(existing, pointer string) plan.Ownership {
	return analyze(existing, pointer, "wezterm.lua").ownership
}

// integrate returns wezterm.lua selecting pointer: a starter config for a blank file, sf2's own form replaced as a
// whole by the full form, or, for an adoptable file, the full form inserted before its return. It refuses anything
// else, and any result sf2 would not then recognize as managed.
func integrate(existing, pointer, lua string) (string, error) {
	a := analyze(existing, pointer, lua)
	var updated string
	switch {
	case a.ownership == plan.Absent:
		return starterConfig(pointer), nil
	case a.ownership == plan.Managed || a.ownership == plan.Legacy:
		f := a.form
		block := integrationLines(pointer, f.builder)
		// The full form keeps the line breaks it has; a shorter form grows by the file's own line break.
		breaks := lineBreaks(existing[f.start:f.end])
		if len(breaks) != len(block)-1 {
			nl, ok := lineBreak(existing)
			if !ok {
				return "", errors.New(manualSteps(lua, pointer, f.builder, false, true, mixedBreaks))
			}
			breaks = slices.Repeat([]string{nl}, len(block)-1)
		}
		var b strings.Builder
		for i, text := range block {
			if i > 0 {
				b.WriteString(breaks[i-1])
			}
			b.WriteString(f.indent + text)
		}
		updated = existing[:f.start] + b.String() + existing[f.end:]
	case a.ownership == plan.Foreign && a.manual == "":
		// The adopted file is its own bytes with sf2's block inserted whole before its return, ended as its lines are.
		nl, ok := lineBreak(existing)
		if !ok {
			return "", errors.New(manualSteps(lua, pointer, a.builder, false, false, mixedBreaks))
		}
		block := strings.Join(integrationLines(pointer, a.builder), nl) + nl
		updated = existing[:a.returnAt] + block + existing[a.returnAt:]
	default:
		return "", errors.New(a.manual)
	}
	if check := analyze(updated, pointer, lua); check.ownership != plan.Managed || !fsys.SamePath(check.selector, pointer) {
		why := "sf2-themes could not verify its change to " + lua + "; nothing was written."
		if check.ownership == plan.Broken {
			// Adding sf2's block would break the file, such as by passing Lua's limit on locals.
			why = "With sf2-themes's change, " + check.manual + ". Nothing was written."
		}
		return "", errors.New(why + "\n" + manualSteps(lua, pointer, "config", false, false, ""))
	}
	return updated, nil
}
