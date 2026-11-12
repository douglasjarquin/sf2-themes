package lazygit_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/douglasjarquin/sf2-themes/v2/internal/cli"
)

// The refusal was recorded from the reference CLI, which writes nothing.
func TestUnmarkedThemeRefusesWithoutAdopt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yml")
	original := "gui:\n  theme:\n    activeBorderColor:\n      - red\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	status := cli.Main([]string{"apply", "lazygit", "--config-dir", dir}, &stdout, &stderr)
	want := "error: Lazygit already uses a theme sf2-themes does not manage; pass --adopt to replace it, " +
		"or run `setup lazygit` without --theme to keep it and install the theme files\n"
	if status != 1 || stdout.Len() != 0 || stderr.String() != want {
		t.Errorf("status %d, stdout %q, stderr %q", status, stdout.String(), stderr.String())
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); len(entries) != 1 || string(got) != original {
		t.Errorf("the refusal changed the config directory: %d entries, config %q", len(entries), got)
	}
}
