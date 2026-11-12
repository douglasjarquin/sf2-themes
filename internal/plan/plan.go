// Package plan describes a mutation in full before anything is written, and the inspection that decides it.
//
// Adapters return Steps; only fsys.Commit touches disk. Paths are strings in the form pycompat.Path prints.
package plan

import (
	"errors"
	"io/fs"
	"regexp"

	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
	"github.com/douglasjarquin/sf2-themes/v2/internal/pycompat"
)

// Step is one planned change: replace Path with Content, or, when Remove is set, delete Path when it exists.
type Step struct {
	Path    string
	Content string
	Remove  bool
}

// Write plans replacing path with content.
func Write(path, content string) Step {
	return Step{Path: path, Content: content}
}

// Remove plans deleting path when it exists.
func Remove(path string) Step {
	return Step{Path: path, Remove: true}
}

// Ownership is who owns the theme setting in an app's configuration.
type Ownership string

// The ownership classes an inspection reports.
const (
	Absent       Ownership = "absent"
	Managed      Ownership = "managed"
	Legacy       Ownership = "legacy"
	Foreign      Ownership = "foreign"
	UnknownShape Ownership = "unknown shape"
	Broken       Ownership = "broken"
)

// Inspection is what one read of an app's configuration found.
//
// Current is the sf2 theme the app selects when it can be read; Unreadable marks an sf2 selection that exists but
// does not resolve. Detail explains a broken configuration or carries the manual fix for an unknown shape.
type Inspection struct {
	Ownership  Ownership
	Current    *catalog.Theme
	Unreadable bool
	Detail     string
}

// ReadExisting returns a config file's text as Path.read_text reads it, or "" when the file does not exist yet.
// A file that is not UTF-8 is an error, which the CLI prints as one `error:` line.
func ReadExisting(path string) (string, error) {
	text, err := pycompat.ReadText(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	return text, err
}

// RecordedID returns group 1 of the first line of text that pattern matches at its start after stripping,
// as Python's pattern.match(line.strip()) over text.splitlines() does. The pattern needs no leading `^`.
func RecordedID(text string, pattern *regexp.Regexp) (string, bool) {
	for _, line := range pycompat.SplitLines(text) {
		stripped := pycompat.Strip(line)
		loc := pattern.FindStringSubmatchIndex(stripped)
		if loc == nil || loc[0] != 0 {
			continue
		}
		// A group that took no part in the match is None in Python, which records no id.
		if loc[2] < 0 {
			return "", false
		}
		return stripped[loc[2]:loc[3]], true
	}
	return "", false
}

// Recorded inspects an sf2-owned setting whose recorded theme id may not resolve.
// Pass found=false when no id was recorded at all.
func Recorded(ownership Ownership, themeID string, found bool, themes []catalog.Theme) Inspection {
	if found {
		if theme, err := catalog.InstalledTheme(themeID, themes); err == nil {
			return Inspection{Ownership: ownership, Current: &theme}
		}
	}
	return Inspection{Ownership: ownership, Unreadable: true}
}
