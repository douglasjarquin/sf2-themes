package fsys

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/douglasjarquin/sf2-themes/v2/internal/plan"
)

// Expected values in this file were recorded from sf2_theme.filesystem.commit under Python 3.11.

func writeText(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

func readText(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func mode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode().Perm()
}

func actions(results []Result) []Action {
	var out []Action
	for _, result := range results {
		out = append(out, result.Action)
	}
	return out
}

func entries(t *testing.T, dir string) []string {
	t.Helper()
	list, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range list {
		names = append(names, entry.Name())
	}
	return names
}

func TestASymlinkAnywhereInThePlanCommitsNothing(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real.toml")
	writeText(t, target, "old\n", 0o644)
	link := filepath.Join(dir, "c.toml")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	steps := []plan.Step{
		plan.Write(filepath.Join(dir, "a.toml"), "a\n"),
		plan.Write(filepath.Join(dir, "b.toml"), "b\n"),
		plan.Write(link, "c\n"),
	}
	_, err := Commit(steps, Options{})
	want := "refusing to replace symlink " + link + "; pass --follow-symlinks to write through"
	if err == nil || err.Error() != want {
		t.Fatalf("err %v, want %q", err, want)
	}
	if got := entries(t, dir); !slices.Equal(got, []string{"c.toml", "real.toml"}) {
		t.Errorf("entries %v", got)
	}
	if readText(t, target) != "old\n" {
		t.Error("symlink target changed")
	}
}

func TestFollowSymlinksWritesTheResolvedTargetAndKeepsTheLink(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "real.toml")
	writeText(t, target, "old\n", 0o644)
	link := filepath.Join(dir, "c.toml")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	results, err := Commit([]plan.Step{
		plan.Write(filepath.Join(dir, "a.toml"), "a\n"),
		plan.Write(link, "c\n"),
	}, Options{FollowSymlinks: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := actions(results); !slices.Equal(got, []Action{Created, Updated}) {
		t.Errorf("actions %v", got)
	}
	if results[1].Path != target {
		t.Errorf("reported path %q, want the resolved %q", results[1].Path, target)
	}
	if info, err := os.Lstat(link); err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Error("the symlink itself was replaced")
	}
	if readText(t, target) != "c\n" {
		t.Error("target not written")
	}
}

func TestASymlinkLoopIsRefusedWhenFollowed(t *testing.T) {
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	if err := os.Symlink(a, b); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(b, a); err != nil {
		t.Fatal(err)
	}
	_, err = Commit([]plan.Step{plan.Write(a, "x\n")}, Options{FollowSymlinks: true})
	if want := "Symlink loop from '" + a + "'"; err == nil || err.Error() != want {
		t.Errorf("err %v, want %q", err, want)
	}
}

func TestDryRunReportsEveryOutcomeWithDiffsAndTouchesNothing(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if err := os.Mkdir("r", 0o755); err != nil {
		t.Fatal(err)
	}
	writeText(t, "r/changed.toml", "keep\nold\n", 0o644)
	writeText(t, "r/unchanged.toml", "same\n", 0o644)
	writeText(t, "r/legacy.toml", "legacy\n", 0o644)
	results, err := Commit([]plan.Step{
		plan.Write("r/new/created.toml", "new\n"),
		plan.Write("r/changed.toml", "keep\nnew\n"),
		plan.Write("r/unchanged.toml", "same\n"),
		plan.Remove("r/legacy.toml"),
	}, Options{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	want := []Result{
		{Path: "r/new/created.toml", Action: WouldCreate, Diff: "--- /dev/null\n+++ r/new/created.toml\n@@ -0,0 +1 @@\n+new\n"},
		{Path: "r/changed.toml", Action: WouldUpdate, Diff: "--- r/changed.toml\n+++ r/changed.toml\n@@ -1,2 +1,2 @@\n keep\n-old\n+new\n"},
		{Path: "r/unchanged.toml", Action: Unchanged},
		{Path: "r/legacy.toml", Action: WouldRemove},
	}
	if !slices.Equal(results, want) {
		t.Errorf("results\n%#v\nwant\n%#v", results, want)
	}
	if got := entries(t, "r"); !slices.Equal(got, []string{"changed.toml", "legacy.toml", "unchanged.toml"}) {
		t.Errorf("entries %v", got)
	}
	if readText(t, "r/changed.toml") != "keep\nold\n" || readText(t, "r/legacy.toml") != "legacy\n" {
		t.Error("dry-run changed a file")
	}
}

func TestRemovingAMissingFileReportsNothing(t *testing.T) {
	results, err := Commit([]plan.Step{plan.Remove(filepath.Join(t.TempDir(), "gone.toml"))}, Options{})
	if err != nil || len(results) != 0 {
		t.Errorf("results %v, err %v", results, err)
	}
}

func TestCommitRemovesAnExistingFile(t *testing.T) {
	legacy := filepath.Join(t.TempDir(), "legacy.toml")
	writeText(t, legacy, "legacy\n", 0o644)
	results, err := Commit([]plan.Step{plan.Remove(legacy)}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(results, []Result{{Path: legacy, Action: Removed}}) {
		t.Errorf("results %v", results)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Error("file still exists")
	}
}

func TestANewFileIsMode0644(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "fresh.toml")
	results, err := Commit([]plan.Step{plan.Write(path, "x\n")}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Action != Created || mode(t, path) != 0o644 || readText(t, path) != "x\n" {
		t.Errorf("action %s, mode %o", results[0].Action, mode(t, path))
	}
}

func TestAReplacedFileKeepsItsModeAndTheBackupHoldsTheDecodedOriginal(t *testing.T) {
	now = func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.FixedZone("x", 3600)) }
	t.Cleanup(func() { now = time.Now })
	dir := t.TempDir()
	path := filepath.Join(dir, "crlf.toml")
	writeText(t, path, "a\r\nb\r\n", 0o600)
	if _, err := Commit([]plan.Step{plan.Write(path, "a\nc\n")}, Options{}); err != nil {
		t.Fatal(err)
	}
	if readText(t, path) != "a\nc\n" || mode(t, path) != 0o600 {
		t.Errorf("content %q, mode %o", readText(t, path), mode(t, path))
	}
	backup := filepath.Join(dir, "crlf.toml.bak.20260102T020405Z")
	if got := readText(t, backup); got != "a\nb\n" {
		t.Errorf("backup %q", got)
	}
	if mode(t, backup) != 0o600 {
		t.Errorf("backup mode %o", mode(t, backup))
	}
}

func TestAReadOnlyFileIsReplacedAndStaysReadOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ro.toml")
	writeText(t, path, "old\n", 0o444)
	if _, err := Commit([]plan.Step{plan.Write(path, "new\n")}, Options{}); err != nil {
		t.Fatal(err)
	}
	if readText(t, path) != "new\n" || mode(t, path) != 0o444 {
		t.Errorf("content %q, mode %o", readText(t, path), mode(t, path))
	}
}

func TestUnchangedContentWritesNothingAndCreatesNoBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	writeText(t, path, "old\n", 0o644)
	if _, err := Commit([]plan.Step{plan.Write(path, "new\n")}, Options{}); err != nil {
		t.Fatal(err)
	}
	results, err := Commit([]plan.Step{plan.Write(path, "new\n")}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(results, []Result{{Path: path, Action: Unchanged}}) {
		t.Errorf("results %v", results)
	}
	var backups []string
	for _, name := range entries(t, dir) {
		if strings.HasPrefix(name, "config.toml.bak.") {
			backups = append(backups, name)
		}
	}
	if len(backups) != 1 {
		t.Errorf("backups %v, want the one from the first change", backups)
	}
}

func TestAFailedReplaceLeavesNoTemporaryFile(t *testing.T) {
	dir := t.TempDir()
	destination := filepath.Join(dir, "dir.toml")
	if err := os.Mkdir(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	writeText(t, filepath.Join(destination, "inner"), "x", 0o644)
	if _, err := Commit([]plan.Step{plan.Write(destination, "x\n")}, Options{}); err == nil {
		t.Fatal("replacing a directory succeeded")
	}
	if got := entries(t, dir); !slices.Equal(got, []string{"dir.toml"}) {
		t.Errorf("entries %v", got)
	}
}

func TestAnExistingFileThatIsNotUTF8IsAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	writeText(t, path, "\xff\xfe theme\n", 0o644)
	_, err := Commit([]plan.Step{plan.Write(path, "x\n")}, Options{})
	if err == nil || strings.Contains(err.Error(), "\n") {
		t.Errorf("err %v", err)
	}
	if readText(t, path) != "\xff\xfe theme\n" {
		t.Error("file changed")
	}
}
