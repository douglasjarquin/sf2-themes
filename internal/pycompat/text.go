// Package pycompat reproduces the Python behaviors whose results reach the bytes the CLI writes or prints:
// text-mode reads, line splitting and whitespace stripping, Python 3 `re` character classes, pathlib paths,
// `repr()` of a string, the `json` module, and `difflib.unified_diff`.
package pycompat

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Python 3 `re` character classes for str patterns, written for Go RE2, whose own `\s` and `\w` are ASCII only.
// Use them where a Python pattern says `\s`, `\S`, or `\w`, for example `^local` + Space + `+(` + Word + `+)`.
const (
	// Space is `\s`: exactly the characters str.isspace() accepts.
	Space = `[` + spaceSet + `]`
	// NotSpace is `\S`.
	NotSpace = `[^` + spaceSet + `]`
	// Word is `\w`: letters, numbers, and the underscore.
	Word = `[\p{L}\p{N}_]`
)

const spaceSet = `\t-\r\x1c-\x20\x{85}\x{a0}\x{1680}\x{2000}-\x{200a}\x{2028}\x{2029}\x{202f}\x{205f}\x{3000}`

// ErrInvalidUTF8 is wrapped by ReadText when the file is not UTF-8, where Python raises UnicodeDecodeError.
var ErrInvalidUTF8 = errors.New("not valid UTF-8")

// ReadText is Path.read_text(encoding="utf-8"): the file decoded as strict UTF-8,
// with `\r\n` and a lone `\r` both read as `\n`.
func ReadText(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if !utf8.Valid(data) {
		return "", fmt.Errorf("%s: %w", path, ErrInvalidUTF8)
	}
	return strings.ReplaceAll(strings.ReplaceAll(string(data), "\r\n", "\n"), "\r", "\n"), nil
}

// IsSpace is str.isspace() for one character, the set str.strip() removes and `\s` matches.
func IsSpace(r rune) bool {
	switch r {
	case 0x85, 0xa0, 0x1680, 0x2028, 0x2029, 0x202f, 0x205f, 0x3000:
		return true
	}
	return (r >= '\t' && r <= '\r') || (r >= 0x1c && r <= 0x20) || (r >= 0x2000 && r <= 0x200a)
}

// IsWord reports whether `\w` matches the character.
func IsWord(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsNumber(r)
}

// Strip is str.strip() with no argument.
func Strip(s string) string {
	return strings.TrimFunc(s, IsSpace)
}

// LStrip is str.lstrip() with no argument.
func LStrip(s string) string {
	return strings.TrimLeftFunc(s, IsSpace)
}

// RStrip is str.rstrip() with no argument.
func RStrip(s string) string {
	return strings.TrimRightFunc(s, IsSpace)
}

// SplitLines is str.splitlines(): it breaks on every Python line boundary, not only `\n`, and drops the boundaries.
func SplitLines(s string) []string {
	return splitLines(s, false)
}

// SplitLinesKeepEnds is str.splitlines(keepends=True): each line keeps the boundary that ended it.
func SplitLinesKeepEnds(s string) []string {
	return splitLines(s, true)
}

func splitLines(s string, keepEnds bool) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if !isLineBreak(r) {
			i += size
			continue
		}
		end := i
		i += size
		if r == '\r' && i < len(s) && s[i] == '\n' {
			i++
		}
		if keepEnds {
			end = i
		}
		lines = append(lines, s[start:end])
		start = i
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func isLineBreak(r rune) bool {
	switch r {
	case '\n', '\v', '\f', '\r', 0x1c, 0x1d, 0x1e, 0x85, 0x2028, 0x2029:
		return true
	}
	return false
}
