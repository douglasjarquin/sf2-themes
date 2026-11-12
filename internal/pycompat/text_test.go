package pycompat

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"testing"
)

func TestReadTextReadsUniversalNewlines(t *testing.T) {
	for _, tc := range readTextCases {
		path := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(path, []byte(tc.file), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := ReadText(path)
		if err != nil {
			t.Fatalf("ReadText(%q): %v", tc.file, err)
		}
		if got != tc.text {
			t.Errorf("ReadText(%q) = %q, want %q", tc.file, got, tc.text)
		}
		if lines := SplitLinesKeepEnds(got); !slices.Equal(lines, tc.lines) {
			t.Errorf("lines of %q = %q, want %q", tc.file, lines, tc.lines)
		}
	}
}

func TestReadTextReadsCRLFAndCRAsThreeLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(path, []byte("a\r\nb\rc"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadText(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "a\nb\nc" || !slices.Equal(SplitLines(got), []string{"a", "b", "c"}) {
		t.Errorf("ReadText = %q, want three lines separated by LF", got)
	}
}

func TestReadTextRejectsInvalidUTF8(t *testing.T) {
	for _, raw := range invalidUTF8Files {
		path := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadText(path); !errors.Is(err, ErrInvalidUTF8) {
			t.Errorf("ReadText(%q) error = %v, want ErrInvalidUTF8", raw, err)
		}
	}
}

func TestReadTextReportsMissingFile(t *testing.T) {
	if _, err := ReadText(filepath.Join(t.TempDir(), "missing")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("error = %v, want not exist", err)
	}
}

func TestSplitLines(t *testing.T) {
	for _, tc := range splitLinesCases {
		if got := SplitLines(tc.in); !slices.Equal(got, tc.lines) {
			t.Errorf("SplitLines(%q) = %q, want %q", tc.in, got, tc.lines)
		}
		if got := SplitLinesKeepEnds(tc.in); !slices.Equal(got, tc.keepEnds) {
			t.Errorf("SplitLinesKeepEnds(%q) = %q, want %q", tc.in, got, tc.keepEnds)
		}
	}
}

func TestSplitLinesBreaksOnEveryPythonBoundary(t *testing.T) {
	for r := rune(0); r <= 0x10ffff; r++ {
		if r >= 0xd800 && r <= 0xdfff {
			continue
		}
		want := inRanges(pythonLineBreak, r)
		if got := len(SplitLines("a"+string(r)+"b")) == 2; got != want {
			t.Errorf("U+%04X breaks a line: got %v, want %v", r, got, want)
		}
	}
}

func TestStrip(t *testing.T) {
	for _, tc := range stripCases {
		if got := Strip(tc.in); got != tc.strip {
			t.Errorf("Strip(%q) = %q, want %q", tc.in, got, tc.strip)
		}
		if got := LStrip(tc.in); got != tc.lstrip {
			t.Errorf("LStrip(%q) = %q, want %q", tc.in, got, tc.lstrip)
		}
		if got := RStrip(tc.in); got != tc.rstrip {
			t.Errorf("RStrip(%q) = %q, want %q", tc.in, got, tc.rstrip)
		}
	}
}

func TestSpaceMatchesPythonWhitespace(t *testing.T) {
	space := regexp.MustCompile(`^` + Space + `$`)
	notSpace := regexp.MustCompile(`^` + NotSpace + `$`)
	for r := rune(0); r <= 0x10ffff; r++ {
		if r >= 0xd800 && r <= 0xdfff {
			continue
		}
		want := inRanges(pythonSpace, r)
		if got := IsSpace(r); got != want {
			t.Errorf("IsSpace(U+%04X) = %v, want %v", r, got, want)
		}
		s := string(r)
		if got := space.MatchString(s); got != want {
			t.Errorf("Space matches U+%04X: got %v, want %v", r, got, want)
		}
		if got := notSpace.MatchString(s); got == want {
			t.Errorf("NotSpace matches U+%04X: got %v, want %v", r, got, !want)
		}
	}
}

// Go's Unicode tables are newer than Python 3.11's, so the comparison covers the characters Python 3.11 has assigned.
func TestWordMatchesPythonWordCharacters(t *testing.T) {
	word := regexp.MustCompile(`^` + Word + `$`)
	for r := rune(0); r <= 0x10ffff; r++ {
		if (r >= 0xd800 && r <= 0xdfff) || !inRanges(pythonAssigned, r) {
			continue
		}
		want := inRanges(pythonWord, r)
		if got := IsWord(r); got != want {
			t.Errorf("IsWord(U+%04X) = %v, want %v", r, got, want)
		}
		if got := word.MatchString(string(r)); got != want {
			t.Errorf("Word matches U+%04X: got %v, want %v", r, got, want)
		}
	}
}

// The adapters' patterns, written with the classes above, find what Python's patterns find.
func TestClassesKeepPythonPatternResults(t *testing.T) {
	patterns := map[string]*regexp.Regexp{
		"wezterm builder":      regexp.MustCompile(`(?m)^local` + Space + `+(` + Word + `+)` + Space + `*=` + Space + `*wezterm\.config_builder\(\)` + Space + `*$`),
		"wezterm return":       regexp.MustCompile(`(?m)^return` + Space + `+(` + Word + `+)` + Space + `*$`),
		"wezterm color scheme": regexp.MustCompile(`^` + Space + `*(?:` + Word + `+\.)?color_scheme` + Space + `*=`),
		"managed id comment":   regexp.MustCompile(`^# sf2-themes:` + Space + `+(` + NotSpace + `+)` + Space + `*$`),
	}
	for _, tc := range regexCases {
		re := patterns[tc.pattern]
		found := re.FindStringSubmatch(tc.text)
		if (found != nil) != tc.matched {
			t.Errorf("%s on %q: matched %v, want %v", tc.pattern, tc.text, found != nil, tc.matched)
			continue
		}
		if found == nil {
			continue
		}
		group := found[0]
		if len(found) > 1 {
			group = found[1]
		}
		if group != tc.group {
			t.Errorf("%s on %q: group %q, want %q", tc.pattern, tc.text, group, tc.group)
		}
	}
}

func TestWordMatchesNonASCIIWeztermBuilderName(t *testing.T) {
	builder := regexp.MustCompile(`(?m)^local` + Space + `+(` + Word + `+)` + Space + `*=` + Space + `*wezterm\.config_builder\(\)` + Space + `*$`)
	found := builder.FindStringSubmatch("local café = wezterm.config_builder()")
	if found == nil || found[1] != "café" {
		t.Errorf("found %q, want the name café", found)
	}
}

func inRanges(ranges [][2]rune, r rune) bool {
	_, found := slices.BinarySearchFunc(ranges, r, func(span [2]rune, r rune) int {
		switch {
		case span[1] < r:
			return -1
		case span[0] > r:
			return 1
		}
		return 0
	})
	return found
}
