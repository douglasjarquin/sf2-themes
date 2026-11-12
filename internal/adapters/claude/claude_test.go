package claude_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/douglasjarquin/sf2-themes/v2/internal/adapters/claude"
	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
	"github.com/douglasjarquin/sf2-themes/v2/internal/cli"
	"github.com/douglasjarquin/sf2-themes/v2/internal/plan"
)

// Every expected value here was recorded from the reference adapter, src/sf2_theme/adapters/claude.py.

func load(t *testing.T, id string) (catalog.Theme, []catalog.Theme) {
	t.Helper()
	themes, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	theme, err := catalog.GetTheme(id, themes)
	if err != nil {
		t.Fatal(err)
	}
	return theme, themes
}

func TestRenderThemeMatchesTheReference(t *testing.T) {
	ryu, _ := load(t, "ryu")
	want, err := os.ReadFile("testdata/sf2-ryu.json")
	if err != nil {
		t.Fatal(err)
	}
	if got := claude.RenderTheme(ryu); got != string(want) {
		t.Errorf("RenderTheme(ryu) = %q, want %q", got, want)
	}
}

func TestMergeKeepsKeyOrderAndAddsThemeLast(t *testing.T) {
	ryu, _ := load(t, "ryu")
	got, err := claude.MergeSettings(`{"zeta": 1, "alpha": {"b": 2, "a": [1, 2.5, null]}, "mid": "x"}`, ryu)
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"zeta\": 1,\n  \"alpha\": {\n    \"b\": 2,\n    \"a\": [\n      1,\n      2.5,\n      null\n    ]\n  },\n" +
		"  \"mid\": \"x\",\n  \"theme\": \"custom:sf2-ryu\"\n}\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMergeWritesNonASCIIAsEscapes(t *testing.T) {
	ryu, _ := load(t, "ryu")
	got, err := claude.MergeSettings("{\"name\": \"café \U0001F600\", \"theme\": \"dark\"}", ryu)
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"name\": \"caf\\u00e9 \\ud83d\\ude00\",\n  \"theme\": \"custom:sf2-ryu\"\n}\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestInspectReadsAManagedCustomTheme(t *testing.T) {
	_, themes := load(t, "main")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), []byte(`{"theme": "custom:sf2-ryu-light"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := claude.Inspect(themes, dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.Ownership != plan.Managed || got.Current == nil || got.Current.Metadata.ID != "ryu-light" {
		t.Errorf("got %+v, want managed ryu-light", got)
	}
}

func TestSettingsThatAreNotJSONExitOneWithTheFixedPrefix(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(path, []byte("{\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for command, prefix := range map[string]string{
		"current": "error: " + path + " is not valid JSON: ",
		"apply":   "error: Claude Code config cannot be changed safely: " + path + " is not valid JSON: ",
	} {
		var stdout, stderr bytes.Buffer
		status := cli.Main([]string{command, "claude", "--config-dir", dir}, &stdout, &stderr)
		if status != 1 || stdout.Len() != 0 || !strings.HasPrefix(stderr.String(), prefix) {
			t.Errorf("%s: status %d, stdout %q, stderr %q, want prefix %q", command, status, stdout.String(), stderr.String(), prefix)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "themes")); !os.IsNotExist(err) {
		t.Errorf("themes directory was written for a broken config: %v", err)
	}
}
