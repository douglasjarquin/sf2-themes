package pycompat

import (
	"testing"
	"unicode"
)

func TestRepr(t *testing.T) {
	for _, tc := range reprCases {
		if got := Repr(tc.in); got != tc.want {
			t.Errorf("Repr(%q) = %s, want %s", tc.in, got, tc.want)
		}
	}
}

// Repr keeps a character as itself exactly when Python's str.isprintable() does, across the characters
// Python 3.11 has assigned.
func TestReprKeepsWhatPythonPrints(t *testing.T) {
	for r := rune(0); r <= 0x10ffff; r++ {
		if (r >= 0xd800 && r <= 0xdfff) || !inRanges(pythonAssigned, r) || r == '\'' || r == '\\' {
			continue
		}
		want := inRanges(pythonPrintable, r)
		if got := unicode.IsPrint(r); got != want {
			t.Errorf("U+%04X printable: got %v, want %v", r, got, want)
		}
		if kept := Repr(string(r)) == "'"+string(r)+"'"; kept != want {
			t.Errorf("Repr(U+%04X) keeps the character: got %v, want %v", r, kept, want)
		}
	}
}
