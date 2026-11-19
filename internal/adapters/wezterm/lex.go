package wezterm

import "strings"

// tokenKind is the lexical class of one Lua token.
type tokenKind int

const (
	nameToken tokenKind = iota
	stringToken
	longStringToken
	commentToken
	otherToken
)

// token is one Lua token and its byte range in the source.
type token struct {
	kind       tokenKind
	start, end int
}

func (t token) text(source string) string { return source[t.start:t.end] }

// lex splits Lua source into tokens: names, short strings, long-bracket strings, short and long-bracket comments,
// and everything else one operator or number at a time. Whitespace is dropped. An unterminated string or comment runs
// to the end of its line (short) or of the source (long), so nothing inside it is ever read as code.
func lex(source string) []token {
	var tokens []token
	for i := 0; i < len(source); {
		c := source[i]
		start := i
		switch {
		case c == ' ' || c == '\t' || c == '\r' || c == '\n' || c == '\f' || c == '\v':
			i++
			continue
		case strings.HasPrefix(source[i:], "--"):
			if end, ok := longBracketEnd(source, i+2); ok {
				i = end
			} else {
				i = lineEnd(source, i)
			}
			tokens = append(tokens, token{commentToken, start, i})
			continue
		case c == '[':
			if end, ok := longBracketEnd(source, i); ok {
				i = end
				tokens = append(tokens, token{longStringToken, start, i})
				continue
			}
			i++
		case c == '"' || c == '\'':
			i = shortStringEnd(source, i)
			tokens = append(tokens, token{stringToken, start, i})
			continue
		case isNameStart(c):
			for i < len(source) && isNamePart(source[i]) {
				i++
			}
			tokens = append(tokens, token{nameToken, start, i})
			continue
		case isDigit(c):
			for i < len(source) && (isNamePart(source[i]) || source[i] == '.') {
				i++
			}
		case i+1 < len(source) && strings.Contains("=~<>", string(c)) && source[i+1] == '=':
			i += 2
		default:
			i++
		}
		tokens = append(tokens, token{otherToken, start, i})
	}
	return tokens
}

// longBracketEnd returns the end of the long bracket `[==[ ... ]==]` opening at i, which runs to the end of the
// source when unterminated, and false when no long bracket opens there.
func longBracketEnd(source string, i int) (int, bool) {
	if i >= len(source) || source[i] != '[' {
		return 0, false
	}
	level := 0
	for i+1+level < len(source) && source[i+1+level] == '=' {
		level++
	}
	if i+1+level >= len(source) || source[i+1+level] != '[' {
		return 0, false
	}
	closing := "]" + strings.Repeat("=", level) + "]"
	body := i + 2 + level
	if end := strings.Index(source[body:], closing); end >= 0 {
		return body + end + len(closing), true
	}
	return len(source), true
}

// shortStringEnd returns the end of the quoted string opening at i: past its closing quote, or at the end of the line
// when it is unterminated. A backslash escapes the next byte, including a newline.
func shortStringEnd(source string, i int) int {
	quote := source[i]
	for i++; i < len(source); i++ {
		switch source[i] {
		case '\\':
			i++
		case quote:
			return i + 1
		case '\n':
			return i
		}
	}
	return len(source)
}

func lineEnd(source string, i int) int {
	if end := strings.IndexByte(source[i:], '\n'); end >= 0 {
		return i + end
	}
	return len(source)
}

func isNameStart(c byte) bool { return c == '_' || (c|0x20 >= 'a' && c|0x20 <= 'z') }

func isNamePart(c byte) bool { return isNameStart(c) || isDigit(c) }

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// codeTokens drops comments, the tokens that never execute.
func codeTokens(tokens []token) []token {
	var kept []token
	for _, t := range tokens {
		if t.kind != commentToken {
			kept = append(kept, t)
		}
	}
	return kept
}

// blankInactive returns source with every comment and long-bracket string replaced by spaces, keeping line breaks,
// so line-oriented matching sees only code and short strings, at the same offsets and line numbers.
func blankInactive(source string) string {
	blanked := []byte(source)
	for _, t := range lex(source) {
		if t.kind != commentToken && t.kind != longStringToken {
			continue
		}
		for i := t.start; i < t.end; i++ {
			// Non-ASCII bytes and line breaks stay, so the blanked source splits into the same lines.
			if blanked[i] < 0x80 && strings.IndexByte(lineBreakBytes, blanked[i]) < 0 {
				blanked[i] = ' '
			}
		}
	}
	return string(blanked)
}
