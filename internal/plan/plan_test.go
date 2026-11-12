package plan

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
	"github.com/douglasjarquin/sf2-themes/v2/internal/pycompat"
)

// Expected values were recorded from sf2_theme.plan under Python 3.11.

func TestRecordedIDMatchesTheStrippedLineAtItsStart(t *testing.T) {
	pattern := regexp.MustCompile(`--` + pycompat.Space + `*sf2-themes:` + pycompat.Space + `*(` + pycompat.NotSpace + `+)`)
	cases := []struct {
		text  string
		id    string
		found bool
	}{
		{"x\n  -- sf2-themes: sf2-ryu \r\n-- sf2-themes: second\n", "sf2-ryu", true},
		{"a -- sf2-themes: x\n", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		if id, found := RecordedID(tc.text, pattern); id != tc.id || found != tc.found {
			t.Errorf("RecordedID(%q) = %q, %v; want %q, %v", tc.text, id, found, tc.id, tc.found)
		}
	}
	optional := regexp.MustCompile(`theme(?:=(` + pycompat.Word + `+))?`)
	if id, found := RecordedID("theme\n", optional); found {
		t.Errorf("a group that did not take part recorded %q", id)
	}
}

func TestRecordedResolvesAnInstalledIdentityOrIsUnreadable(t *testing.T) {
	themes, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	got := Recorded(Managed, "sf2-ken-light", true, themes)
	if got.Ownership != Managed || got.Current == nil || got.Current.Metadata.ID != "ken-light" || got.Unreadable {
		t.Errorf("resolved: %+v", got)
	}
	for _, found := range []bool{true, false} {
		got := Recorded(Legacy, "nope", found, themes)
		if got.Ownership != Legacy || got.Current != nil || !got.Unreadable {
			t.Errorf("unresolved (found %v): %+v", found, got)
		}
	}
}

func TestReadExistingReadsTextOrNothing(t *testing.T) {
	dir := t.TempDir()
	if text, err := ReadExisting(filepath.Join(dir, "missing.toml")); text != "" || err != nil {
		t.Errorf("missing file: %q, %v", text, err)
	}
	path := filepath.Join(dir, "crlf.toml")
	if err := os.WriteFile(path, []byte("a\r\nb\rc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if text, err := ReadExisting(path); text != "a\nb\nc\n" || err != nil {
		t.Errorf("crlf file: %q, %v", text, err)
	}
	if _, err := ReadExisting(dir); err == nil {
		t.Error("reading a directory succeeded")
	}
}
