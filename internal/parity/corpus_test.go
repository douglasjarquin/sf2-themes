package parity

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
	"github.com/douglasjarquin/sf2-themes/v2/internal/pycompat"
)

// The parser corpus holds the Go TOML and JSON readers to the verdicts Python 3.11's
// tomllib and json record, so a user config is broken for both implementations or neither.
const (
	corpusDir       = "testdata/corpus"
	verdictsFile    = "testdata/corpus/verdicts.txt"
	divergencesFile = "testdata/corpus/divergences.txt"
)

// corpusReaders report whether the Go reader the adapters use accepts a config file's raw
// bytes, after the same UTF-8 decoding and newline translation the adapters apply. A nil
// reader is not built yet; pointing an entry at the real reader enables its half of the corpus.
var corpusReaders = map[string]func([]byte) bool{
	"toml": func(data []byte) bool {
		text, err := pycompat.DecodeText(data)
		if err != nil {
			return false
		}
		_, err = catalog.DecodeTOML(text)
		return err == nil
	},
	"json": func(data []byte) bool {
		text, err := pycompat.DecodeText(data)
		if err != nil {
			return false
		}
		_, err = pycompat.Loads(text)
		return err == nil
	},
}

// verdictScript records what the reference adapters' read does with each input: read the
// file as UTF-8 text with universal newlines, then tomllib.loads or json.loads it.
// Any other exception propagates and fails the recording.
const verdictScript = `
import json, sys, tomllib
from pathlib import Path

root = Path(sys.argv[1])
for rel in sys.stdin.read().splitlines():
    load = tomllib.loads if rel.startswith("toml/") else json.loads
    try:
        load((root / rel).read_text(encoding="utf-8"))
    except (UnicodeDecodeError, tomllib.TOMLDecodeError, json.JSONDecodeError):
        print("reject", rel)
    else:
        print("accept", rel)
`

// corpusInputs lists every corpus input, slash-separated and relative to the corpus directory.
func corpusInputs(t *testing.T) []string {
	t.Helper()
	var inputs []string
	err := filepath.WalkDir(corpusDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(corpusDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "toml/") && strings.HasSuffix(rel, ".toml") || strings.HasPrefix(rel, "json/") && strings.HasSuffix(rel, ".json") {
			inputs = append(inputs, rel)
		}
		return nil
	})
	must(t, err)
	sort.Strings(inputs)
	return inputs
}

// readVerdicts parses "accept <input>" or "reject <input>" lines.
func readVerdicts(text string) (map[string]bool, error) {
	verdicts := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSuffix(text, "\n"), "\n") {
		verdict, input, ok := strings.Cut(line, " ")
		if !ok || verdict != "accept" && verdict != "reject" {
			return nil, fmt.Errorf("bad verdict line %q", line)
		}
		verdicts[input] = verdict == "accept"
	}
	return verdicts, nil
}

// readDivergences parses "<input>: <config shape it affects>" lines; # starts a comment.
func readDivergences(text string) (map[string]string, error) {
	divergences := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		input, shape, ok := strings.Cut(line, ": ")
		if !ok || shape == "" {
			return nil, fmt.Errorf("divergence %q names no config shape", line)
		}
		divergences[input] = shape
	}
	return divergences, nil
}

// corpusMismatches lists the inputs of one format where reader disagrees with the reference
// verdict and the disagreement is not a listed divergence, and the listed divergences
// reader no longer has, so the list stays exactly the divergences that remain.
func corpusMismatches(format string, verdicts map[string]bool, divergences map[string]string, reader func([]byte) bool, read func(string) ([]byte, error)) ([]string, error) {
	var mismatches []string
	for input, accepted := range verdicts {
		if !strings.HasPrefix(input, format+"/") {
			continue
		}
		data, err := read(input)
		if err != nil {
			return nil, err
		}
		_, listed := divergences[input]
		switch got := reader(data); {
		case got != accepted && !listed:
			mismatches = append(mismatches, fmt.Sprintf("%s: reference accepts=%v, Go accepts=%v", input, accepted, got))
		case got == accepted && listed:
			mismatches = append(mismatches, fmt.Sprintf("%s: listed as a divergence, but the Go reader agrees; remove it from the list", input))
		}
	}
	sort.Strings(mismatches)
	return mismatches, nil
}

func loadCorpus(t *testing.T) (map[string]bool, map[string]string) {
	t.Helper()
	text, err := os.ReadFile(verdictsFile)
	must(t, err)
	verdicts, err := readVerdicts(string(text))
	must(t, err)
	text, err = os.ReadFile(divergencesFile)
	must(t, err)
	divergences, err := readDivergences(string(text))
	must(t, err)
	return verdicts, divergences
}

func readCorpusInput(input string) ([]byte, error) {
	return os.ReadFile(filepath.Join(corpusDir, filepath.FromSlash(input)))
}

// referenceVerdicts runs the verdict script on Python 3.11 over every input.
func referenceVerdicts(t *testing.T, inputs []string) string {
	t.Helper()
	python, err := python311()
	must(t, err)
	// -I: the inputs are downloaded data; nothing from the working directory is importable.
	command := exec.Command(python, "-I", "-c", verdictScript, corpusDir)
	command.Stdin = strings.NewReader(strings.Join(inputs, "\n"))
	var stderr bytes.Buffer
	command.Stderr = &stderr
	out, err := command.Output()
	if err != nil {
		t.Fatalf("verdict script: %v\n%s", err, stderr.String())
	}
	return string(out)
}

// TestCorpusVerdictsMatchTheReference rewrites the verdicts in record mode and re-checks them
// against Python 3.11 in reference mode.
func TestCorpusVerdictsMatchTheReference(t *testing.T) {
	if !referenceMode() {
		t.Skip("reference verdicts are checked with " + envReference + " set")
	}
	got := referenceVerdicts(t, corpusInputs(t))
	if recording() {
		must(t, os.WriteFile(verdictsFile, []byte(got), 0o644))
		return
	}
	want, err := os.ReadFile(verdictsFile)
	must(t, err)
	if string(want) != got {
		t.Errorf("Python 3.11 verdicts differ from %s:\n%s", verdictsFile, describeDifference(string(want), got))
	}
}

// TestCorpusIsComplete needs no reader: every input has one verdict and every listed
// divergence names an input.
func TestCorpusIsComplete(t *testing.T) {
	if recording() {
		t.Skip("record mode rewrites the verdicts")
	}
	verdicts, divergences := loadCorpus(t)
	inputs := corpusInputs(t)
	for _, input := range inputs {
		if _, ok := verdicts[input]; !ok {
			t.Errorf("%s has no recorded verdict", input)
		}
		delete(verdicts, input)
	}
	for input := range verdicts {
		t.Errorf("verdict for missing input %s", input)
	}
	all := map[string]bool{}
	for _, input := range inputs {
		all[input] = true
	}
	for input := range divergences {
		if !all[input] {
			t.Errorf("divergence names missing input %s", input)
		}
	}
}

// TestCorpusGoReadersAgree holds each Go reader to the reference verdicts.
func TestCorpusGoReadersAgree(t *testing.T) {
	verdicts, divergences := loadCorpus(t)
	for _, format := range []string{"toml", "json"} {
		t.Run(format, func(t *testing.T) {
			reader := corpusReaders[format]
			if reader == nil {
				t.Skipf("the Go %s reader is not built yet", strings.ToUpper(format))
			}
			mismatches, err := corpusMismatches(format, verdicts, divergences, reader, readCorpusInput)
			must(t, err)
			if len(mismatches) > 0 {
				t.Errorf("the Go %s reader and %s disagree:\n%s", format, divergencesFile, strings.Join(mismatches, "\n"))
			}
		})
	}
}

// A reader whose verdict flips on one input fails the corpus unless that input is a listed divergence.
func TestCorpusFlagsAFlippedVerdict(t *testing.T) {
	verdicts := map[string]bool{"toml/a.toml": true, "toml/b.toml": false, "json/c.json": true}
	contents := map[string][]byte{"toml/a.toml": []byte("accept"), "toml/b.toml": []byte("reject"), "json/c.json": []byte("accept")}
	read := func(input string) ([]byte, error) { return contents[input], nil }
	agreeing := func(data []byte) bool { return string(data) == "accept" }
	flipped := func(data []byte) bool { return true }

	if got, err := corpusMismatches("toml", verdicts, nil, agreeing, read); err != nil || len(got) != 0 {
		t.Fatalf("an agreeing reader reported %v, %v", got, err)
	}
	got, err := corpusMismatches("toml", verdicts, nil, flipped, read)
	must(t, err)
	if want := []string{"toml/b.toml: reference accepts=false, Go accepts=true"}; strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("flipped verdict reported %v, want %v", got, want)
	}
	listed := map[string]string{"toml/b.toml": "a config that reads only under TOML 1.1"}
	if got, err := corpusMismatches("toml", verdicts, listed, flipped, read); err != nil || len(got) != 0 {
		t.Errorf("a listed divergence still failed: %v, %v", got, err)
	}
	got, err = corpusMismatches("toml", verdicts, listed, agreeing, read)
	must(t, err)
	if want := "toml/b.toml: listed as a divergence, but the Go reader agrees; remove it from the list"; strings.Join(got, "\n") != want {
		t.Errorf("a stale divergence reported %v, want %q", got, want)
	}
}
