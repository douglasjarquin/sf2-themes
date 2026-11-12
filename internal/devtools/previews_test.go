package devtools

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreviewsMatchCommittedSVGs(t *testing.T) {
	const committed = "../../docs/previews"
	dir := filepath.Join(t.TempDir(), "previews")
	var stdout, stderr bytes.Buffer
	if code := Previews(themesDir, dir, &stdout, &stderr); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr.String())
	}
	want, err := os.ReadDir(committed)
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(want) != 36 || len(got) != len(want) {
		t.Fatalf("wrote %d files, committed %d, want 36", len(got), len(want))
	}
	for i, entry := range want {
		if got[i].Name() != entry.Name() {
			t.Fatalf("file %d is %s, want %s", i, got[i].Name(), entry.Name())
		}
		wantBytes, err := os.ReadFile(filepath.Join(committed, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		gotBytes, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(gotBytes, wantBytes) {
			t.Errorf("%s differs from the committed preview", entry.Name())
		}
	}
	lines := strings.Split(strings.TrimSuffix(stdout.String(), "\n"), "\n")
	if len(lines) != 36 || lines[0] != "wrote "+filepath.Join(dir, "main.svg") {
		t.Fatalf("unexpected progress lines: %q", stdout.String())
	}
}
