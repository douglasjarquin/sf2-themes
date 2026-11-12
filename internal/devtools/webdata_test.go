package devtools

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	themesDir        = "../../themes"
	committedWebData = "../../web/src/data/generated-theme-data.json"
)

// copyCatalog copies the catalog into a temporary directory so a test can edit one theme.
func copyCatalog(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.CopyFS(dir, os.DirFS(themesDir)); err != nil {
		t.Fatal(err)
	}
	return dir
}

func editFile(t *testing.T, path, old, replacement string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(old)) {
		t.Fatalf("%s does not contain %q", path, old)
	}
	if err := os.WriteFile(path, bytes.Replace(data, []byte(old), []byte(replacement), 1), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runWebData(source, output string, check bool) (code int, stdout, stderr string) {
	var out, errOut bytes.Buffer
	code = WebData(source, output, check, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestWebDataCheckVerifiesCommittedFile(t *testing.T) {
	code, stdout, stderr := runWebData(themesDir, committedWebData, true)
	if code != 0 || stderr != "" {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	// Recorded from the Python generator run with the same relative --output.
	if want := "verified generated theme data: ../../web/src/data/generated-theme-data.json\n"; stdout != want {
		t.Fatalf("stdout %q, want %q", stdout, want)
	}
}

func TestWebDataCheckReportsStaleFile(t *testing.T) {
	stale := filepath.Join(t.TempDir(), "stale.json")
	if err := os.WriteFile(stale, []byte("{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, output := range []string{stale, filepath.Join(filepath.Dir(stale), "missing.json")} {
		code, stdout, stderr := runWebData(themesDir, output, true)
		if code != 1 || stdout != "" {
			t.Fatalf("%s: exit %d, stdout %q", output, code, stdout)
		}
		if want := "stale generated theme data: " + output + "\n"; stderr != want {
			t.Fatalf("stderr %q, want %q", stderr, want)
		}
	}
}

func TestWebDataFromEditedSource(t *testing.T) {
	source := copyCatalog(t)
	for _, name := range []string{"main.toml", "main-light.toml"} {
		editFile(t, filepath.Join(source, name), "Family baseline", "Edited baseline")
	}
	output := filepath.Join(t.TempDir(), "nested", "theme-data.json")
	code, stdout, stderr := runWebData(source, output, false)
	if code != 0 || stderr != "" {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	if want := "wrote " + output + "\n"; stdout != want {
		t.Fatalf("stdout %q, want %q", stdout, want)
	}
	got, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	committed, err := os.ReadFile(committedWebData)
	if err != nil {
		t.Fatal(err)
	}
	// The Python generator, given this edited source, writes the committed file with exactly the two
	// stage values changed.
	want := strings.ReplaceAll(string(committed), `"stage": "Arcade · Family baseline"`, `"stage": "Arcade · Edited baseline"`)
	if string(got) != want {
		t.Fatal("generated JSON does not match the committed file with the edited stage")
	}
}

func TestWebDataCatalogErrorIsBare(t *testing.T) {
	source := copyCatalog(t)
	editFile(t, filepath.Join(source, "main.toml"), `red = "#c86e67"`, `red = "gold"`)
	output := filepath.Join(t.TempDir(), "theme-data.json")
	code, stdout, stderr := runWebData(source, output, false)
	if code != 1 || stdout != "" {
		t.Fatalf("exit %d, stdout %q", code, stdout)
	}
	// Recorded from the Python generator with the same edit.
	if want := "color must be #RRGGBB, got 'gold'\n"; stderr != want {
		t.Fatalf("stderr %q, want %q", stderr, want)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatal("output was written despite the catalog error")
	}
}
