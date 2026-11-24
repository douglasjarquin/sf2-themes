package wezterm

import (
	"errors"
	"fmt"
)

// chunk is a wezterm.lua that parsed completely as Lua 5.4: its tokens, and the byte range of each statement at the
// top level of the chunk. returned is the name the chunk ends by returning, when its last statement is
// `return <name>`.
type chunk struct {
	source     string
	tokens     []token
	statements []span
	returned   string
	// bindings are the names the chunk binds, in source order, at any depth: locals, parameters, loop variables,
	// `function <name>` statements, and assignments to a bare name.
	bindings []binding
}

// binding is one name bound at byte offset at.
type binding struct {
	name string
	at   int
}

// maxLocals is MAXVARS in Lua 5.4's lparser.c: the most locals one function may have active at once.
const maxLocals = 200

// span is a byte range of the source.
type span struct{ start, end int }

// parseLua verifies that source is a syntactically valid Lua 5.4 chunk. It checks syntax, the compile errors Lua
// reports for misplaced `break` and `...`, and the limit of 200 locals declared at the top level of the chunk, the
// statements sf2 adds its own locals beside. Other compile-time limits (locals active inside nested blocks or
// functions, upvalues, nesting depth, constants) are not checked. Constructs whose compile-time rules sf2 does not
// check, goto and labels and the <const> and <close> attributes, are refused as unverifiable rather than guessed.
func parseLua(source string) (chunk, error) {
	tokens, err := lexLua(source)
	if err != nil {
		return chunk{}, err
	}
	p := &parser{source: source}
	for _, t := range tokens {
		if t.kind != commentToken {
			p.code = append(p.code, t)
		}
	}
	c := chunk{source: source, tokens: tokens}
	p.topLevel = &c.statements
	if err := p.block(true, false); err != nil {
		return chunk{}, err
	}
	if p.i != len(p.code) {
		return chunk{}, p.fail("unexpected " + p.peekText())
	}
	if p.locals > maxLocals {
		return chunk{}, fmt.Errorf("it declares %d locals at its top level; Lua allows %d", p.locals, maxLocals)
	}
	c.returned = p.returned
	c.bindings = p.bindings
	return c, nil
}

type parser struct {
	source   string
	code     []token
	i        int
	topLevel *[]span
	returned string
	// locals counts the locals the main chunk declares at its top level.
	locals int
	// bindings collects every name the chunk binds; see chunk.
	bindings []binding
}

// bind records the name token just consumed as bound.
func (p *parser) bind() {
	t := p.code[p.i-1]
	p.bindings = append(p.bindings, binding{name: t.text(p.source), at: t.start})
}

// expectBound consumes a name and records it as bound.
func (p *parser) expectBound() error {
	if err := p.expect(nameToken, ""); err != nil {
		return err
	}
	p.bind()
	return nil
}

var errUnverifiable = errors.New("uses Lua sf2-themes does not verify")

func (p *parser) fail(message string) error {
	at := len(p.source)
	if p.i < len(p.code) {
		at = p.code[p.i].start
	}
	return fmt.Errorf("%s at byte %d", message, at)
}

func (p *parser) peekText() string {
	if p.i >= len(p.code) {
		return "end of file"
	}
	return fmt.Sprintf("%q", p.code[p.i].text(p.source))
}

func (p *parser) is(kind tokenKind, text string) bool {
	return p.i < len(p.code) && p.code[p.i].kind == kind && (text == "" || p.code[p.i].text(p.source) == text)
}

func (p *parser) sym(text string) bool { return p.is(symbolToken, text) }
func (p *parser) kw(text string) bool  { return p.is(keywordToken, text) }

func (p *parser) accept(kind tokenKind, text string) bool {
	if p.is(kind, text) {
		p.i++
		return true
	}
	return false
}

func (p *parser) expect(kind tokenKind, text string) error {
	if !p.accept(kind, text) {
		want := text
		if want == "" {
			want = "a name"
		}
		return p.fail("expected " + want + " near " + p.peekText())
	}
	return nil
}

func (p *parser) blockEnds() bool {
	return p.i >= len(p.code) || p.kw("end") || p.kw("else") || p.kw("elseif") || p.kw("until")
}

// block parses statements up to the end of the block. top records each statement's range for the main chunk.
func (p *parser) block(top, inLoop bool) error {
	return p.blockIn(top, inLoop, true)
}

func (p *parser) blockIn(top, inLoop, vararg bool) error {
	for !p.blockEnds() {
		start := p.code[p.i].start
		if p.kw("return") {
			p.i++
			if !p.blockEnds() && !p.sym(";") {
				first := p.i
				if err := p.explist(inLoop, vararg); err != nil {
					return err
				}
				if top && p.i == first+1 && p.code[first].kind == nameToken {
					p.returned = p.code[first].text(p.source)
				}
			}
			p.accept(symbolToken, ";")
			if top {
				*p.topLevel = append(*p.topLevel, span{start, p.code[p.i-1].end})
			}
			if !p.blockEnds() {
				return p.fail("'return' must end its block")
			}
			return nil
		}
		first := p.i
		if err := p.statement(inLoop, vararg); err != nil {
			return err
		}
		if top {
			p.locals += p.declared(first)
			*p.topLevel = append(*p.topLevel, span{start, p.code[p.i-1].end})
			p.returned = ""
		}
	}
	return nil
}

// declared is the number of locals the statement starting at token first declares: one for `local function`, one
// per name for `local a, b`, and none for any other statement.
func (p *parser) declared(first int) int {
	if !p.code[first].isKeyword(p.source, "local") {
		return 0
	}
	if p.code[first+1].isKeyword(p.source, "function") {
		return 1
	}
	n := 0
	for i := first + 1; i < p.i && p.code[i].kind == nameToken; i += 2 {
		n++
		if i+1 >= p.i || p.code[i+1].kind != symbolToken || p.code[i+1].text(p.source) != "," {
			break
		}
	}
	return n
}

func (p *parser) statement(inLoop, vararg bool) error {
	switch {
	case p.accept(symbolToken, ";"):
		return nil
	case p.sym("::") || p.kw("goto"):
		return fmt.Errorf("goto or a label %w", errUnverifiable)
	case p.accept(keywordToken, "break"):
		if !inLoop {
			return p.fail("break outside a loop")
		}
		return nil
	case p.accept(keywordToken, "do"):
		return p.closeBlock(inLoop, vararg, "end")
	case p.accept(keywordToken, "while"):
		if err := p.expr(inLoop, vararg); err != nil {
			return err
		}
		if err := p.expect(keywordToken, "do"); err != nil {
			return err
		}
		return p.closeBlock(true, vararg, "end")
	case p.accept(keywordToken, "repeat"):
		if err := p.blockIn(false, true, vararg); err != nil {
			return err
		}
		if err := p.expect(keywordToken, "until"); err != nil {
			return err
		}
		return p.expr(inLoop, vararg)
	case p.accept(keywordToken, "if"):
		for {
			if err := p.expr(inLoop, vararg); err != nil {
				return err
			}
			if err := p.expect(keywordToken, "then"); err != nil {
				return err
			}
			if err := p.blockIn(false, inLoop, vararg); err != nil {
				return err
			}
			if !p.accept(keywordToken, "elseif") {
				break
			}
		}
		if p.accept(keywordToken, "else") {
			if err := p.blockIn(false, inLoop, vararg); err != nil {
				return err
			}
		}
		return p.expect(keywordToken, "end")
	case p.accept(keywordToken, "for"):
		return p.forStatement(vararg)
	case p.accept(keywordToken, "function"):
		if err := p.expect(nameToken, ""); err != nil {
			return err
		}
		if !p.sym(".") && !p.sym(":") {
			p.bind()
		}
		for p.accept(symbolToken, ".") {
			if err := p.expect(nameToken, ""); err != nil {
				return err
			}
		}
		if p.accept(symbolToken, ":") {
			if err := p.expect(nameToken, ""); err != nil {
				return err
			}
		}
		return p.funcBody()
	case p.accept(keywordToken, "local"):
		if p.accept(keywordToken, "function") {
			if err := p.expectBound(); err != nil {
				return err
			}
			return p.funcBody()
		}
		for {
			if err := p.expectBound(); err != nil {
				return err
			}
			if p.sym("<") {
				return fmt.Errorf("a local attribute %w", errUnverifiable)
			}
			if !p.accept(symbolToken, ",") {
				break
			}
		}
		if p.accept(symbolToken, "=") {
			return p.explist(inLoop, vararg)
		}
		return nil
	}
	return p.exprStatement(inLoop, vararg)
}

func (p *parser) closeBlock(inLoop, vararg bool, closer string) error {
	if err := p.blockIn(false, inLoop, vararg); err != nil {
		return err
	}
	return p.expect(keywordToken, closer)
}

func (p *parser) forStatement(vararg bool) error {
	if err := p.expectBound(); err != nil {
		return err
	}
	if p.accept(symbolToken, "=") {
		for n := 0; ; n++ {
			if err := p.expr(false, vararg); err != nil {
				return err
			}
			if n == 2 || !p.accept(symbolToken, ",") {
				if n == 0 {
					return p.fail("expected ','")
				}
				break
			}
		}
	} else {
		for p.accept(symbolToken, ",") {
			if err := p.expectBound(); err != nil {
				return err
			}
		}
		if err := p.expect(keywordToken, "in"); err != nil {
			return err
		}
		if err := p.explist(false, vararg); err != nil {
			return err
		}
	}
	if err := p.expect(keywordToken, "do"); err != nil {
		return err
	}
	return p.closeBlock(true, vararg, "end")
}

// exprStatement is an assignment or a function call; any other expression is not a statement.
func (p *parser) exprStatement(inLoop, vararg bool) error {
	first := p.i
	call, assignable, err := p.suffixedExpr(inLoop, vararg)
	if err != nil {
		return err
	}
	targets := []int{first}
	last := []int{p.i}
	if !p.sym("=") && !p.sym(",") {
		if !call {
			return p.fail("syntax error: expected a call or an assignment")
		}
		return nil
	}
	for {
		if !assignable {
			return p.fail("syntax error: cannot assign to this expression")
		}
		if !p.accept(symbolToken, ",") {
			break
		}
		targets = append(targets, p.i)
		if _, assignable, err = p.suffixedExpr(inLoop, vararg); err != nil {
			return err
		}
		last = append(last, p.i)
	}
	if err := p.expect(symbolToken, "="); err != nil {
		return err
	}
	// A target that is one name rebinds that name; a field or index target leaves it as it was.
	for i, start := range targets {
		if last[i] == start+1 && p.code[start].kind == nameToken {
			t := p.code[start]
			p.bindings = append(p.bindings, binding{name: t.text(p.source), at: t.start})
		}
	}
	return p.explist(inLoop, vararg)
}

// suffixedExpr parses a primary expression and its suffixes and reports whether it ends in a call and whether it
// can be assigned to.
func (p *parser) suffixedExpr(inLoop, vararg bool) (call, assignable bool, err error) {
	switch {
	case p.accept(nameToken, ""):
		assignable = true
	case p.accept(symbolToken, "("):
		if err := p.expr(inLoop, vararg); err != nil {
			return false, false, err
		}
		if err := p.expect(symbolToken, ")"); err != nil {
			return false, false, err
		}
	default:
		return false, false, p.fail("unexpected " + p.peekText())
	}
	for {
		switch {
		case p.accept(symbolToken, "."):
			if err := p.expect(nameToken, ""); err != nil {
				return false, false, err
			}
			call, assignable = false, true
		case p.accept(symbolToken, "["):
			if err := p.expr(inLoop, vararg); err != nil {
				return false, false, err
			}
			if err := p.expect(symbolToken, "]"); err != nil {
				return false, false, err
			}
			call, assignable = false, true
		case p.accept(symbolToken, ":"):
			if err := p.expect(nameToken, ""); err != nil {
				return false, false, err
			}
			if err := p.args(inLoop, vararg); err != nil {
				return false, false, err
			}
			call, assignable = true, false
		case p.sym("(") || p.sym("{") || p.is(stringToken, "") || p.is(longStringToken, ""):
			if err := p.args(inLoop, vararg); err != nil {
				return false, false, err
			}
			call, assignable = true, false
		default:
			return call, assignable, nil
		}
	}
}

func (p *parser) args(inLoop, vararg bool) error {
	switch {
	case p.accept(stringToken, "") || p.accept(longStringToken, ""):
		return nil
	case p.sym("{"):
		return p.table(inLoop, vararg)
	}
	if err := p.expect(symbolToken, "("); err != nil {
		return err
	}
	if !p.sym(")") {
		if err := p.explist(inLoop, vararg); err != nil {
			return err
		}
	}
	return p.expect(symbolToken, ")")
}

func (p *parser) table(inLoop, vararg bool) error {
	if err := p.expect(symbolToken, "{"); err != nil {
		return err
	}
	for !p.sym("}") {
		switch {
		case p.accept(symbolToken, "["):
			if err := p.expr(inLoop, vararg); err != nil {
				return err
			}
			if err := p.expect(symbolToken, "]"); err != nil {
				return err
			}
			if err := p.expect(symbolToken, "="); err != nil {
				return err
			}
		case p.is(nameToken, "") && p.i+1 < len(p.code) && p.code[p.i+1].kind == symbolToken &&
			p.code[p.i+1].text(p.source) == "=":
			p.i += 2
		}
		if err := p.expr(inLoop, vararg); err != nil {
			return err
		}
		if !p.accept(symbolToken, ",") && !p.accept(symbolToken, ";") {
			break
		}
	}
	return p.expect(symbolToken, "}")
}

func (p *parser) funcBody() error {
	if err := p.expect(symbolToken, "("); err != nil {
		return err
	}
	vararg := false
	if !p.sym(")") {
		for {
			if p.accept(symbolToken, "...") {
				vararg = true
				break
			}
			if err := p.expectBound(); err != nil {
				return err
			}
			if !p.accept(symbolToken, ",") {
				break
			}
		}
	}
	if err := p.expect(symbolToken, ")"); err != nil {
		return err
	}
	if err := p.blockIn(false, false, vararg); err != nil {
		return err
	}
	return p.expect(keywordToken, "end")
}

func (p *parser) explist(inLoop, vararg bool) error {
	for {
		if err := p.expr(inLoop, vararg); err != nil {
			return err
		}
		if !p.accept(symbolToken, ",") {
			return nil
		}
	}
}

var binaryOperators = map[string]bool{
	"+": true, "-": true, "*": true, "/": true, "//": true, "%": true, "^": true, "..": true, "==": true, "~=": true,
	"<": true, "<=": true, ">": true, ">=": true, "and": true, "or": true, "&": true, "|": true, "~": true,
	"<<": true, ">>": true,
}

func (p *parser) expr(inLoop, vararg bool) error {
	for {
		for p.accept(keywordToken, "not") || p.accept(symbolToken, "-") || p.accept(symbolToken, "#") ||
			p.accept(symbolToken, "~") {
		}
		if err := p.simpleExpr(inLoop, vararg); err != nil {
			return err
		}
		if p.i >= len(p.code) {
			return nil
		}
		t := p.code[p.i]
		if (t.kind != symbolToken && t.kind != keywordToken) || !binaryOperators[t.text(p.source)] {
			return nil
		}
		p.i++
	}
}

func (p *parser) simpleExpr(inLoop, vararg bool) error {
	switch {
	case p.accept(keywordToken, "nil") || p.accept(keywordToken, "true") || p.accept(keywordToken, "false") ||
		p.accept(numberToken, "") || p.accept(stringToken, "") || p.accept(longStringToken, ""):
		return nil
	case p.sym("..."):
		if !vararg {
			return p.fail("cannot use '...' outside a vararg function")
		}
		p.i++
		return nil
	case p.accept(keywordToken, "function"):
		return p.funcBody()
	case p.sym("{"):
		return p.table(inLoop, vararg)
	}
	_, _, err := p.suffixedExpr(inLoop, vararg)
	return err
}
