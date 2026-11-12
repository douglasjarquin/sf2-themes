package parity

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Golden text format, one golden per case:
//
//	step apply codex --config-dir {SANDBOX}/config/codex --verbose
//	exit 0
//	stdout
//	| created: {SANDBOX}/config/codex/themes/sf2-main.tmTheme
//	= 3f9a1c2b4d5e6f70
//	|   changes: 37 created
//	stderr
//	tree
//	dir 0755 config/codex
//	file 0644 8e0d55a1b2c3d4e5 config/codex/config.toml
//	link config/x -> {SANDBOX}/elsewhere/x
//
// A stream line "| text" is one output line, "\ text" a last line with no newline, and
// "= <hash>" a unified diff longer than the inline limit, kept in the blob store. A stream
// header "<stream> exempt" marks help and usage text. File contents always live in the
// blob store, named by the first 16 hex digits of their SHA-256. Because file contents and
// long diffs are stored once per distinct content, a theme edit changes one blob per
// written file and diff, not one per case.

// Placeholders the normalizer writes.
const (
	maskedText  = "<masked>"
	backupStamp = "<STAMP>"
)

// inlineLimit is the most lines a diff keeps inline before it moves to the blob store.
const inlineLimit = 30

var backupName = regexp.MustCompile(`\.bak\.(\d{8}T\d{6}Z)$`)

// StepResult is one normalized, masked step.
type StepResult struct {
	Args         []string
	Exit         int
	Stdout       string
	Stderr       string
	StdoutExempt bool
	StderrExempt bool
}

// Entry is one normalized path under the sandbox.
type Entry struct {
	Kind    string // "dir", "file", or "link"
	Path    string // slash-separated, relative to the sandbox root
	Mode    fs.FileMode
	Hash    string // file
	Content []byte // file, normalized
	Target  string // link, normalized
}

// Result is one normalized run of a case.
type Result struct {
	Steps []StepResult
	Tree  []Entry
}

// Hash names a blob.
func Hash(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:8])
}

// Normalize replaces the sandbox root with the placeholder.
func Normalize(raw []byte, root string) []byte {
	return []byte(strings.ReplaceAll(string(raw), root, Sandbox))
}

// Expand replaces the placeholder with the sandbox root.
func Expand(text, root string) string {
	return strings.ReplaceAll(text, Sandbox, root)
}

// usageErrorLine is the line of a usage error that is compared; the usage text above it is not.
var usageErrorLine = regexp.MustCompile(`^sf2-themes( [a-z-]+)?: error: `)

// ApplyMasks masks one normalized stream. It reports whether the stream is exempt, and
// returns the tail masks for this stream that matched no line.
func ApplyMasks(text string, stream Stream, masks []Mask) (masked string, exempt bool, unmatched []Mask) {
	out := strings.Split(text, "\n")
	for _, mask := range masks {
		if mask.Stream != stream {
			continue
		}
		if mask.Whole {
			return "", true, nil
		}
		if mask.ErrorLine {
			kept := out[:0:0]
			for _, line := range out {
				if usageErrorLine.MatchString(line) {
					kept = append(kept, line)
				}
			}
			if len(kept) == 0 {
				unmatched = append(unmatched, mask)
			}
			out = append(kept, "")
			continue
		}
		matched := false
		for i, line := range out {
			if at := strings.Index(line, mask.After); at >= 0 && (mask.After != "" || line != "") {
				out[i] = line[:at+len(mask.After)] + maskedText
				matched = true
			}
		}
		if !matched {
			unmatched = append(unmatched, mask)
		}
	}
	return strings.Join(out, "\n"), false, unmatched
}

// Snapshot records every path under root. File contents and link targets are normalized,
// backup timestamps are masked, and when one file has several backups only the newest is
// kept, so backups compare as a set however the run's seconds fell.
func Snapshot(root string) ([]Entry, error) {
	var entries []Entry
	newest := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		entry := Entry{Path: rel, Mode: info.Mode().Perm()}
		switch {
		case info.Mode()&fs.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			entry.Kind, entry.Mode, entry.Target = "link", 0, string(Normalize([]byte(target), root))
		case info.IsDir():
			entry.Kind = "dir"
		case info.Mode().IsRegular():
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			entry.Kind, entry.Content = "file", Normalize(content, root)
			entry.Hash = Hash(entry.Content)
			if match := backupName.FindStringSubmatchIndex(rel); match != nil {
				stamp := rel[match[2]:match[3]]
				entry.Path = rel[:match[2]] + backupStamp
				if previous, seen := newest[entry.Path]; seen && previous > stamp {
					return nil
				}
				newest[entry.Path] = stamp
				entries = dropPath(entries, entry.Path)
			}
		default:
			return fmt.Errorf("unexpected file type at %s: %v", rel, info.Mode())
		}
		entries = append(entries, entry)
		return nil
	})
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return entries, err
}

func dropPath(entries []Entry, path string) []Entry {
	kept := entries[:0]
	for _, entry := range entries {
		if entry.Path != path {
			kept = append(kept, entry)
		}
	}
	return kept
}

// Render writes a result in the golden format. store puts a blob and returns its hash.
func Render(r Result, store func([]byte) string) string {
	var b strings.Builder
	for _, s := range r.Steps {
		fmt.Fprintf(&b, "step %s\nexit %d\n", formatArgs(s.Args), s.Exit)
		renderStream(&b, Stdout, s.Stdout, s.StdoutExempt, store)
		renderStream(&b, Stderr, s.Stderr, s.StderrExempt, store)
	}
	b.WriteString("tree\n")
	for _, e := range r.Tree {
		switch e.Kind {
		case "dir":
			fmt.Fprintf(&b, "dir %04o %s\n", e.Mode, e.Path)
		case "file":
			fmt.Fprintf(&b, "file %04o %s %s\n", e.Mode, store(e.Content), e.Path)
		case "link":
			fmt.Fprintf(&b, "link %s -> %s\n", e.Path, e.Target)
		}
	}
	return b.String()
}

func renderStream(b *strings.Builder, stream Stream, text string, exempt bool, store func([]byte) string) {
	if exempt {
		fmt.Fprintf(b, "%s exempt\n", stream)
		return
	}
	fmt.Fprintf(b, "%s\n", stream)
	lines := strings.SplitAfter(text, "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	for i := 0; i < len(lines); {
		if end := diffEnd(lines, i); end-i > inlineLimit {
			fmt.Fprintf(b, "= %s\n", store([]byte(strings.Join(lines[i:end], ""))))
			i = end
			continue
		}
		if line, eol := strings.CutSuffix(lines[i], "\n"); eol {
			fmt.Fprintf(b, "| %s\n", line)
		} else {
			fmt.Fprintf(b, "\\ %s\n", line)
		}
		i++
	}
}

// diffEnd returns where a unified diff that starts at lines[start] ends, or -1 when no
// well-formed diff starts there. Each line keeps its newline.
func diffEnd(lines []string, start int) int {
	if start+1 >= len(lines) || !strings.HasPrefix(lines[start], "--- ") || !strings.HasPrefix(lines[start+1], "+++ ") {
		return -1
	}
	i, hunks := start+2, 0
	for i < len(lines) && strings.HasPrefix(lines[i], "@@ -") {
		removed, added, ok := hunkCounts(lines[i])
		if !ok {
			return -1
		}
		for i++; removed > 0 || added > 0; i++ {
			if i >= len(lines) || lines[i] == "" {
				return -1
			}
			switch lines[i][0] {
			case ' ':
				removed, added = removed-1, added-1
			case '-':
				removed--
			case '+':
				added--
			default:
				return -1
			}
			if removed < 0 || added < 0 {
				return -1
			}
		}
		hunks++
	}
	if hunks == 0 {
		return -1
	}
	return i
}

var hunkHeader = regexp.MustCompile(`^@@ -\d+(?:,(\d+))? \+\d+(?:,(\d+))? @@\n$`)

func hunkCounts(line string) (removed, added int, ok bool) {
	match := hunkHeader.FindStringSubmatch(line)
	if match == nil {
		return 0, 0, false
	}
	count := func(text string) int {
		if text == "" {
			return 1
		}
		n, _ := strconv.Atoi(text)
		return n
	}
	return count(match[1]), count(match[2]), true
}

func formatArgs(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		if arg == "" || strings.ContainsAny(arg, " \t\"'\\") {
			arg = strconv.Quote(arg)
		}
		quoted[i] = arg
	}
	return strings.Join(quoted, " ")
}

// ParseGolden reads a golden's tree section and every blob hash the golden refers to.
// File entries carry their hash; the caller loads contents from the blob store.
func ParseGolden(golden string) (tree []Entry, blobs []string, err error) {
	inTree := false
	for _, line := range strings.Split(strings.TrimSuffix(golden, "\n"), "\n") {
		if inTree {
			entry, err := parseEntry(line)
			if err != nil {
				return nil, nil, err
			}
			if entry.Kind == "file" {
				blobs = append(blobs, entry.Hash)
			}
			tree = append(tree, entry)
			continue
		}
		head, rest, _ := strings.Cut(line, " ")
		switch head {
		case "tree":
			inTree = true
		case "step", "exit", string(Stdout), string(Stderr), "|", "\\":
		case "=":
			blobs = append(blobs, rest)
		default:
			return nil, nil, fmt.Errorf("unexpected golden line %q", line)
		}
	}
	if !inTree {
		return nil, nil, fmt.Errorf("golden has no tree section")
	}
	return tree, blobs, nil
}

func parseEntry(line string) (Entry, error) {
	kind, rest, _ := strings.Cut(line, " ")
	switch kind {
	case "dir", "file":
		fields := strings.SplitN(rest, " ", map[string]int{"dir": 2, "file": 3}[kind])
		if len(fields) < 2 {
			return Entry{}, fmt.Errorf("bad tree entry %q", line)
		}
		mode, err := strconv.ParseUint(fields[0], 8, 32)
		if err != nil {
			return Entry{}, fmt.Errorf("bad mode in %q", line)
		}
		entry := Entry{Kind: kind, Mode: fs.FileMode(mode), Path: fields[len(fields)-1]}
		if kind == "file" {
			entry.Hash = fields[1]
		}
		return entry, nil
	case "link":
		path, target, ok := strings.Cut(rest, " -> ")
		if !ok {
			return Entry{}, fmt.Errorf("bad link in %q", line)
		}
		return Entry{Kind: "link", Path: path, Target: target}, nil
	}
	return Entry{}, fmt.Errorf("unknown tree entry %q", line)
}
