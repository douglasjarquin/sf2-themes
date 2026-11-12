package pycompat

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestUnifiedDiff(t *testing.T) {
	for _, tc := range diffCases {
		if got := UnifiedDiff(tc.before, tc.after, tc.from, "/tmp/config/settings.json"); got != tc.want {
			t.Errorf("%s:\ngot:\n%s\nwant:\n%s", tc.name, got, tc.want)
		}
	}
}

func TestUnifiedDiffOfNewFileHasDevNullHeaderAndNoTimestamps(t *testing.T) {
	got := UnifiedDiff(nil, []string{"a\n"}, "/dev/null", "/x/settings.json")
	if want := "--- /dev/null\n+++ /x/settings.json\n@@ -0,0 +1 @@\n+a\n"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// Random edits recorded from Python, a tenth of them on 200 or more lines so the autojunk rule applies.
func TestUnifiedDiffMatchesRecordedFuzz(t *testing.T) {
	data, err := os.ReadFile("testdata/diff_fuzz.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct{ A, B, Diff string }
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for i, tc := range cases {
		if got := UnifiedDiff(SplitLinesKeepEnds(tc.A), SplitLinesKeepEnds(tc.B), "old", "new"); got != tc.Diff {
			t.Errorf("case %d (%d lines):\ngot:\n%s\nwant:\n%s", i, strings.Count(tc.A, "\n"), got, tc.Diff)
		}
	}
}
