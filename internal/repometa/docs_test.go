package repometa

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// skippedDocDirs hold generated or third-party Markdown this repository does not author.
var skippedDocDirs = map[string]bool{".git": true, "node_modules": true, "dist": true, ".astro": true}

// markdownFiles lists every Markdown file the repository authors, relative to root.
func markdownFiles(t *testing.T) []string {
	t.Helper()
	var files []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && skippedDocDirs[entry.Name()] {
			return filepath.SkipDir
		}
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// untaggedFences returns the 1-based line numbers of opening code fences that carry no language tag.
// A fence opens with three or more backticks or tildes indented at most three spaces, and closes with a
// run of the same character at least as long and nothing after it (CommonMark).
func untaggedFences(markdown string) []int {
	var untagged []int
	var open string
	for i, line := range strings.Split(markdown, "\n") {
		trimmed := strings.TrimLeft(line, " ")
		if len(line)-len(trimmed) > 3 {
			continue
		}
		var marker string
		for _, char := range []string{"`", "~"} {
			run := len(trimmed) - len(strings.TrimLeft(trimmed, char))
			if run >= 3 {
				marker = strings.Repeat(char, run)
			}
		}
		if marker == "" {
			continue
		}
		info := strings.TrimSpace(trimmed[len(marker):])
		switch {
		case open == "":
			open = marker
			if info == "" {
				untagged = append(untagged, i+1)
			}
		case marker[0] == open[0] && len(marker) >= len(open) && info == "":
			open = ""
		}
	}
	return untagged
}

func TestUntaggedFencesFindsOnlyBareOpeningFences(t *testing.T) {
	doc := "```sh\nls\n```\n\n```\nbare\n```\n\n````md\n```\nnested\n```\n````\n   ~~~\nindented\n~~~\n    ```\nindented code, not a fence\n"
	got := untaggedFences(doc)
	if len(got) != 2 || got[0] != 5 || got[1] != 14 {
		t.Fatalf("untaggedFences = %v, want [5 14]", got)
	}
}

// Agents lift code from docs verbatim, so every fenced block names its language (AgentReady AR-READ-05).
func TestMarkdownCodeFencesAreLanguageTagged(t *testing.T) {
	files := markdownFiles(t)
	if len(files) == 0 {
		t.Fatal("no Markdown files found")
	}
	for _, rel := range []string{"README.md", "docs/sf2-themes/install.md"} {
		found := false
		for _, file := range files {
			found = found || file == rel
		}
		if !found {
			t.Fatalf("%s was not scanned", rel)
		}
	}
	for _, rel := range files {
		for _, line := range untaggedFences(readFile(t, rel)) {
			t.Errorf("%s:%d: code fence has no language tag; use ```text for plain output", rel, line)
		}
	}
}
