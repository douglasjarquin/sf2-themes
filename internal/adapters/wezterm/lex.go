package wezterm

import (
	"fmt"
	"regexp"
	"strings"
)

// tokenKind is the lexical class of one Lua token.
type tokenKind int

const (
	nameToken tokenKind = iota
	keywordToken
	stringToken
	longStringToken
	commentToken
	numberToken
	symbolToken
)

// token is one Lua token and its byte range in the source.
type token struct {
	kind       tokenKind
	start, end int
}

func (t token) text(source string) string { return source[t.start:t.end] }

var keywords = map[string]bool{
	"and": true, "break": true, "do": true, "else": true, "elseif": true, "end": true, "false": true, "for": true,
	"function": true, "goto": true, "if": true, "in": true, "local": true, "nil": true, "not": true, "or": true,
	"repeat": true, "return": true, "then": true, "true": true, "until": true, "while": true,
}

// symbols are Lua 5.4's operators and punctuation, longest first.
var symbols = []string{
	"...", "..", "==", "~=", "<=", ">=", "//", "::", "<<", ">>",
	"+", "-", "*", "/", "%", "^", "#", "&", "~", "|", "<", ">", "=", "(", ")", "{", "}", "[", "]", ";", ":", ",", ".",
}

var (
	decimalNumeral = regexp.MustCompile(`^([0-9]+\.?[0-9]*|\.[0-9]+)([eE][+-]?[0-9]+)?$`)
	hexNumeral     = regexp.MustCompile(`^0[xX]([0-9a-fA-F]+\.?[0-9a-fA-F]*|\.[0-9a-fA-F]+)([pP][+-]?[0-9]+)?$`)
)

// lexLua splits source into Lua 5.4 tokens. Anything Lua's own lexer would reject, or that sf2 cannot be sure it
// reads as Lua does, is an error: an unterminated string or comment, an invalid escape or numeral, or a character Lua
// does not accept outside strings and comments.
func lexLua(source string) ([]token, error) {
	var tokens []token
	i := 0
	if strings.HasPrefix(source, "#") {
		// Lua skips a first line starting with '#'.
		i = lineEnd(source, 0)
	}
	for i < len(source) {
		c := source[i]
		start := i
		var kind tokenKind
		switch {
		case strings.IndexByte(" \t\r\n\f\v", c) >= 0:
			i++
			continue
		case strings.HasPrefix(source[i:], "--"):
			kind = commentToken
			if end, closed, isLong := longBracketEnd(source, i+2); isLong {
				if !closed {
					return nil, fmt.Errorf("unfinished long comment at byte %d", start)
				}
				i = end
			} else {
				i = lineEnd(source, i)
			}
		case c == '[' && isLongOpen(source, i):
			kind = longStringToken
			end, closed, _ := longBracketEnd(source, i)
			if !closed {
				return nil, fmt.Errorf("unfinished long string at byte %d", start)
			}
			i = end
		case c == '"' || c == '\'':
			kind = stringToken
			end, err := shortStringEnd(source, i)
			if err != nil {
				return nil, err
			}
			i = end
		case isNameStart(c):
			for i < len(source) && isNamePart(source[i]) {
				i++
			}
			kind = nameToken
			if keywords[source[start:i]] {
				kind = keywordToken
			}
		case isDigit(c) || (c == '.' && i+1 < len(source) && isDigit(source[i+1])):
			kind = numberToken
			for i < len(source) {
				d := source[i]
				exponent := (d == '+' || d == '-') && strings.IndexByte("eEpP", source[i-1]) >= 0
				if !isNamePart(d) && d != '.' && !exponent {
					break
				}
				i++
			}
			numeral := source[start:i]
			if !decimalNumeral.MatchString(numeral) && !hexNumeral.MatchString(numeral) {
				return nil, fmt.Errorf("malformed number %q at byte %d", numeral, start)
			}
		default:
			kind = symbolToken
			matched := ""
			for _, symbol := range symbols {
				if strings.HasPrefix(source[i:], symbol) {
					matched = symbol
					break
				}
			}
			if matched == "" {
				return nil, fmt.Errorf("unexpected character %q at byte %d", c, start)
			}
			i += len(matched)
		}
		tokens = append(tokens, token{kind: kind, start: start, end: i})
	}
	return tokens, nil
}

func isLongOpen(source string, i int) bool {
	_, _, isLong := longBracketEnd(source, i)
	return isLong
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

// shortStringEnd returns the end of the quoted string opening at i, validating every escape as Lua 5.4 does.
func shortStringEnd(source string, i int) (int, error) {
	start, quote := i, source[i]
	for i++; i < len(source); i++ {
		switch source[i] {
		case quote:
			return i + 1, nil
		case '\n', '\r':
			return 0, fmt.Errorf("unfinished string at byte %d", start)
		case '\\':
			i++
			if i >= len(source) {
				return 0, fmt.Errorf("unfinished string at byte %d", start)
			}
			switch e := source[i]; {
			case strings.IndexByte("abfnrtv\\\"'", e) >= 0:
			case e == '\n' || e == '\r':
				if i+1 < len(source) && (source[i+1] == '\n' || source[i+1] == '\r') && source[i+1] != e {
					i++
				}
			case e == 'z':
				for i+1 < len(source) && strings.IndexByte(" \t\r\n\f\v", source[i+1]) >= 0 {
					i++
				}
			case e == 'x':
				if i+2 >= len(source) || !isHex(source[i+1]) || !isHex(source[i+2]) {
					return 0, fmt.Errorf("invalid \\x escape at byte %d", i)
				}
				i += 2
			case e == 'u':
				end := strings.IndexByte(source[i:], '}')
				if i+1 >= len(source) || source[i+1] != '{' || end < 3 || !allHex(source[i+2:i+end]) || end-2 > 8 {
					return 0, fmt.Errorf("invalid \\u escape at byte %d", i)
				}
				i += end
			case isDigit(e):
				value, n := 0, 0
				for n < 3 && i < len(source) && isDigit(source[i]) {
					value = value*10 + int(source[i]-'0')
					i++
					n++
				}
				i--
				if value > 255 {
					return 0, fmt.Errorf("decimal escape too large at byte %d", i)
				}
			default:
				return 0, fmt.Errorf("invalid escape at byte %d", i)
			}
		}
	}
	return 0, fmt.Errorf("unfinished string at byte %d", start)
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

func isHex(c byte) bool { return isDigit(c) || (c|0x20 >= 'a' && c|0x20 <= 'f') }

func allHex(s string) bool {
	for i := 0; i < len(s); i++ {
		if !isHex(s[i]) {
			return false
		}
	}
	return s != ""
}
