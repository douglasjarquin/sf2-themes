package wezterm

import "strings"

// tokenKind is the lexical class of one Lua token.
type tokenKind int

const (
	nameToken tokenKind = iota
	stringToken
	longStringToken
	commentToken
	numberToken
	otherToken
)

// token is one Lua token, its byte range in the source, and for code tokens the block and bracket nesting it starts
// at: depth counts open function/do/if/repeat blocks, nest open parentheses, brackets, and braces.
type token struct {
	kind        tokenKind
	start, end  int
	depth, nest int
}

func (t token) text(source string) string { return source[t.start:t.end] }

// scan is one Lua source split into tokens. complete is false when a string or comment is unterminated or the blocks
// and brackets do not balance; sf2 recognizes nothing in such a file.
type scan struct {
	source   string
	tokens   []token
	code     []token
	complete bool
}

var twoCharOperators = map[string]bool{
	"==": true, "~=": true, "<=": true, ">=": true, "//": true, "..": true, "::": true, "<<": true, ">>": true,
}

// lexLua tokenizes source as Lua 5.4 does, far enough to tell code from comments and strings and to know which code
// sits at the top level of the chunk.
func lexLua(source string) scan {
	s := scan{source: source, complete: true}
	for i := 0; i < len(source); {
		c := source[i]
		start := i
		kind := otherToken
		closed := true
		switch {
		case c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '\f' || c == '\v':
			i++
			continue
		case strings.HasPrefix(source[i:], "--"):
			kind = commentToken
			if end, ok, isLong := longBracketEnd(source, i+2); isLong {
				i, closed = end, ok
			} else {
				i = lineEnd(source, i)
			}
		case c == '[':
			if end, ok, isLong := longBracketEnd(source, i); isLong {
				kind, i, closed = longStringToken, end, ok
			} else {
				i++
			}
		case c == '"' || c == '\'':
			kind = stringToken
			i, closed = shortStringEnd(source, i)
		case isNameStart(c):
			kind = nameToken
			for i < len(source) && isNamePart(source[i]) {
				i++
			}
		case isDigit(c) || (c == '.' && i+1 < len(source) && isDigit(source[i+1])):
			kind = numberToken
			for i < len(source) && (isNamePart(source[i]) || source[i] == '.' ||
				((source[i] == '+' || source[i] == '-') && strings.ContainsRune("eEpP", rune(source[i-1])))) {
				i++
			}
		case strings.HasPrefix(source[i:], "..."):
			i += 3
		default:
			i++
			if i < len(source) && twoCharOperators[source[start:i+1]] {
				i++
			}
		}
		if !closed {
			s.complete = false
		}
		s.tokens = append(s.tokens, token{kind: kind, start: start, end: i})
	}
	depth, nest := 0, 0
	for _, t := range s.tokens {
		if t.kind == commentToken {
			continue
		}
		text := t.text(source)
		switch {
		case t.kind == nameToken && (text == "end" || text == "until"):
			depth--
		case t.kind == otherToken && (text == ")" || text == "]" || text == "}"):
			nest--
		}
		if depth < 0 || nest < 0 {
			s.complete = false
			depth, nest = max(depth, 0), max(nest, 0)
		}
		t.depth, t.nest = depth, nest
		s.code = append(s.code, t)
		switch {
		case t.kind == nameToken && (text == "function" || text == "do" || text == "if" || text == "repeat"):
			depth++
		case t.kind == otherToken && (text == "(" || text == "[" || text == "{"):
			nest++
		}
	}
	if depth != 0 || nest != 0 {
		s.complete = false
	}
	return s
}

// longBracketEnd returns the end of the long bracket `[==[ ... ]==]` opening at i, whether it is closed, and whether
// a long bracket opens there at all.
func longBracketEnd(source string, i int) (end int, closed, isLong bool) {
	if i >= len(source) || source[i] != '[' {
		return 0, false, false
	}
	level := 0
	for i+1+level < len(source) && source[i+1+level] == '=' {
		level++
	}
	if i+1+level >= len(source) || source[i+1+level] != '[' {
		return 0, false, false
	}
	closing := "]" + strings.Repeat("=", level) + "]"
	body := i + 2 + level
	if at := strings.Index(source[body:], closing); at >= 0 {
		return body + at + len(closing), true, true
	}
	return len(source), false, true
}

// shortStringEnd returns the end of the quoted string opening at i and whether it is closed. A backslash escapes the
// next character, including a line break, and `\z` skips the whitespace after it, line breaks included.
func shortStringEnd(source string, i int) (int, bool) {
	quote := source[i]
	for i++; i < len(source); i++ {
		switch source[i] {
		case '\\':
			i++
			if i < len(source) && source[i] == 'z' {
				for i+1 < len(source) && strings.IndexByte(" \t\r\n\f\v", source[i+1]) >= 0 {
					i++
				}
			} else if i+1 < len(source) && source[i] == '\r' && source[i+1] == '\n' {
				i++
			}
		case quote:
			return i + 1, true
		case '\n', '\r':
			return i, false
		}
	}
	return len(source), false
}

func lineEnd(source string, i int) int {
	if end := strings.IndexAny(source[i:], "\r\n"); end >= 0 {
		return i + end
	}
	return len(source)
}

func isNameStart(c byte) bool { return c == '_' || (c|0x20 >= 'a' && c|0x20 <= 'z') }

func isNamePart(c byte) bool { return isNameStart(c) || isDigit(c) }

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// boundary reports whether offset falls between tokens rather than inside one.
func (s scan) boundary(offset int) bool {
	for _, t := range s.tokens {
		if t.start < offset && offset < t.end {
			return false
		}
	}
	return true
}

// codeAt returns the index in s.code of the first code token starting at or after offset, or len(s.code).
func (s scan) codeAt(offset int) int {
	for i, t := range s.code {
		if t.start >= offset {
			return i
		}
	}
	return len(s.code)
}

// is reports whether code token i exists and has kind and, when text is not "", that text.
func (s scan) is(i int, kind tokenKind, text string) bool {
	return i >= 0 && i < len(s.code) && s.code[i].kind == kind && (text == "" || s.code[i].text(s.source) == text)
}

// lineOf returns the start of the line holding offset and the end of its content, before the line break.
func lineOf(source string, offset int) (int, int) {
	start := strings.LastIndexAny(source[:offset], "\r\n") + 1
	return start, lineEnd(source, offset)
}
