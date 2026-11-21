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

// newSHA is the output of `printf 'new\n' | shasum -a 256`.
const newSHA = "7aa7a5359173d05b63cfd682e3c38487f3cb4f7f1d60659fe59fab1505977d4c"

func TestDryRunReportsEveryOutcomeWithDiffsAndTouchesNothing(t *testing.T) {
	root := t.TempDir()
	t.Chdir(root)
	if err := os.Mkdir("r", 0o755); err != nil {
		t.Fatal(err)
	}
	writeText(t, "r/changed.toml", "keep\nold\n", 0o644)
	writeText(t, "r/unchanged.toml", "same\n", 0o644)
	results, err := Commit([]plan.Step{
		plan.Write("r/new/created.toml", "new\n"),
		plan.Write("r/changed.toml", "keep\nnew\n"),
		plan.Write("r/unchanged.toml", "same\n"),
	}, Options{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	want := []Result{
		{Requested: "r/new/created.toml", Path: "r/new/created.toml", Action: WouldCreate, Diff: "--- /dev/null\n+++ r/new/created.toml\n@@ -0,0 +1 @@\n+new\n", Fingerprint: newSHA},
		{Requested: "r/changed.toml", Path: "r/changed.toml", Action: WouldUpdate, Diff: "--- r/changed.toml\n+++ r/changed.toml\n@@ -1,2 +1,2 @@\n keep\n-old\n+new\n", Fingerprint: "eb073290fa115c022d5fb4d5c11f71219c3f0d3a6af409fba606c6067ce292a6"},
		{Requested: "r/unchanged.toml", Path: "r/unchanged.toml", Action: Unchanged, Fingerprint: "a6328afc76e9db71da297ebff4b0d3e7a7eb3b01d917c05a6573fef121b6ecb6"},
	}
	if !slices.Equal(results, want) {
		t.Errorf("results\n%#v\nwant\n%#v", results, want)
	}
	if got := entries(t, "r"); !slices.Equal(got, []string{"changed.toml", "unchanged.toml"}) {
		t.Errorf("entries %v", got)
	}
	if readText(t, "r/changed.toml") != "keep\nold\n" {
		t.Error("dry-run changed a file")
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
	results, err := Commit([]plan.Step{plan.Write(path, "a\nc\n")}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if readText(t, path) != "a\nc\n" || mode(t, path) != 0o600 {
		t.Errorf("content %q, mode %o", readText(t, path), mode(t, path))
	}
	backup := filepath.Join(dir, "crlf.toml.bak.20260102T020405Z")
	if len(results) != 1 || results[0].Backup != backup {
		t.Errorf("results %v, want backup %s", results, backup)
	}
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
	if !slices.Equal(results, []Result{{Requested: path, Path: path, Action: Unchanged, Fingerprint: newSHA}}) {
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

// replaceState stages and publishes content at path.
func replaceState(path, content string) error {
	staged, err := Stage(path)
	if err != nil {
		return err
	}
	return staged.Publish(content, 0o644)
}

func TestPublishWritesStateWithoutABackupAndKeepsIdenticalFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state", "manifest.json")
	if err := replaceState(path, "one\n"); err != nil {
		t.Fatal(err)
	}
	if readText(t, path) != "one\n" || mode(t, path) != 0o644 {
		t.Errorf("content %q, mode %o", readText(t, path), mode(t, path))
	}
	if err := os.Chmod(path, 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := replaceState(path, "one\n"); err != nil {
		t.Fatal(err)
	}
	if after, err := os.Stat(path); err != nil || !os.SameFile(before, after) {
		t.Errorf("identical content replaced the file: %v", err)
	}
	if err := replaceState(path, "two\n"); err != nil {
		t.Fatal(err)
	}
	if readText(t, path) != "two\n" || mode(t, path) != 0o600 {
		t.Errorf("content %q, mode %o", readText(t, path), mode(t, path))
	}
	if got := entries(t, filepath.Join(dir, "state")); !slices.Equal(got, []string{"manifest.json"}) {
		t.Errorf("entries %v", got)
	}
}

func TestStageRefusesASymlink(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real.json")
	writeText(t, real, "keep\n", 0o644)
	link := filepath.Join(dir, "manifest.json")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	if err := replaceState(link, "x\n"); err == nil || !strings.Contains(err.Error(), "refusing to replace symlink") {
		t.Errorf("err %v", err)
	}
	if readText(t, real) != "keep\n" {
		t.Error("wrote through the symlink")
	}
}

func TestAnUnchangedFileIsFingerprintedByItsBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "crlf.toml")
	writeText(t, path, "new\r\n", 0o644)
	results, err := Commit([]plan.Step{plan.Write(path, "new\n")}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	// The output of `printf 'new\r\n' | shasum -a 256`.
	const crlf = "9f5d1ba48d2fd5c3aaa39bad078e9a929f1d292511f8febc308def37c3840ab2"
	if len(results) != 1 || results[0].Action != Unchanged || results[0].Fingerprint != crlf {
		t.Errorf("results %+v", results)
	}
}

func TestPreflightRefusesWithoutWriting(t *testing.T) {
	dir := t.TempDir()
	writeText(t, filepath.Join(dir, "real.toml"), "x\n", 0o644)
	if err := os.Symlink(filepath.Join(dir, "real.toml"), filepath.Join(dir, "link.toml")); err != nil {
		t.Fatal(err)
	}
	_, err := Preflight([]plan.Step{plan.Write(filepath.Join(dir, "new.toml"), "n\n"), plan.Write(filepath.Join(dir, "link.toml"), "y\n")}, Options{})
	if err == nil || !strings.Contains(err.Error(), "refusing to replace symlink") {
		t.Errorf("err %v", err)
	}
	if got := entries(t, dir); !slices.Equal(got, []string{"link.toml", "real.toml"}) {
		t.Errorf("entries %v", got)
	}
}

// failRename makes the nth publishing rename of a commit fail, as a full disk or a vanished directory would after
// earlier writes succeeded.
func failRename(t *testing.T, n int) {
	t.Helper()
	calls := 0
	rename = func(from, to string) error {
		calls++
		if calls == n {
			return &os.LinkError{Op: "rename", Old: from, New: to, Err: os.ErrPermission}
		}
		return os.Rename(from, to)
	}
	t.Cleanup(func() { rename = os.Rename })
}

// listing records every path under root with its mode and content.
func listing(t *testing.T, root string) []string {
	t.Helper()
	var entries []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		content := ""
		if info.Mode().IsRegular() {
			content = readText(t, path)
		}
		entries = append(entries, path+" "+info.Mode().String()+" "+content)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func TestACommitThatFailsPartWayUndoesItsWrites(t *testing.T) {
	root := t.TempDir()
	writeText(t, root+"/changed.toml", "old\r\n", 0o640)
	writeText(t, root+"/same.toml", "same\n", 0o644)
	steps := []plan.Step{
		plan.Write(root+"/changed.toml", "new\n"),
		plan.Write(root+"/same.toml", "same\n"),
		plan.Write(root+"/made/deep/created.toml", "created\n"),
		plan.Write(root+"/last.toml", "last\n"),
	}
	before := listing(t, root)
	failRename(t, 3)
	results, err := Commit(steps, Options{})
	if err == nil {
		t.Fatal("the failing rename was not reported")
	}
	if len(results) != 0 {
		t.Errorf("undone changes still reported: %v", results)
	}
	if after := listing(t, root); !slices.Equal(after, before) {
		t.Errorf("disk changed:\n%s\nwant\n%s", strings.Join(after, "\n"), strings.Join(before, "\n"))
	}
}

func TestRollingBackACompleteCommitRestoresEveryFile(t *testing.T) {
	root := t.TempDir()
	writeText(t, root+"/changed.toml", "old\n", 0o600)
	before := listing(t, root)
	transaction, err := Apply([]plan.Step{
		plan.Write(root+"/changed.toml", "new\n"),
		plan.Write(root+"/made/created.toml", "created\n"),
	}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if readText(t, root+"/changed.toml") != "new\n" {
		t.Fatal("not written")
	}
	if err := transaction.Rollback(); err != nil {
		t.Fatal(err)
	}
	if after := listing(t, root); !slices.Equal(after, before) {
		t.Errorf("disk changed:\n%s\nwant\n%s", strings.Join(after, "\n"), strings.Join(before, "\n"))
	}
	if len(transaction.Results) != 0 {
		t.Errorf("results after a rollback: %v", transaction.Results)
	}
}

func TestPreflightRefusesAWriteThatWouldFail(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(root+"/dir.toml", 0o755); err != nil {
		t.Fatal(err)
	}
	writeText(t, root+"/file", "not a directory\n", 0o644)
	cases := map[string]string{
		root + "/dir.toml":         "is not a regular file",
		root + "/file/inside.toml": "is not a directory",
	}
	if os.Geteuid() != 0 {
		if err := os.Mkdir(root+"/locked", 0o555); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = os.Chmod(root+"/locked", 0o755) })
		cases[root+"/locked/new/inside.toml"] = "permission denied"
	}
	for path, want := range cases {
		before := listing(t, root)
		_, err := Commit([]plan.Step{plan.Write(root+"/first.toml", "x\n"), plan.Write(path, "x\n")}, Options{})
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v, want %q", path, err, want)
		}
		if after := listing(t, root); !slices.Equal(after, before) {
			t.Errorf("%s: a refused commit wrote", path)
		}
	}
}

func TestAFileAlreadyHoldingItsContentNeedsNoWriteAccess(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes a read-only directory")
	}
	root := t.TempDir()
	writeText(t, root+"/same.toml", "same\n", 0o644)
	if err := os.Chmod(root, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(root, 0o755) })
	results, err := Commit([]plan.Step{plan.Write(root+"/same.toml", "same\n")}, Options{})
	if err != nil || len(results) != 1 || results[0].Action != Unchanged {
		t.Errorf("results %v, err %v", results, err)
	}
}

func TestABackupNeverReplacesAnotherFile(t *testing.T) {
	root := t.TempDir()
	stamp := time.Date(2026, 11, 21, 6, 0, 0, 0, time.UTC)
	now = func() time.Time { return stamp }
	t.Cleanup(func() { now = time.Now })
	writeText(t, root+"/a.toml", "first\n", 0o644)
	taken := root + "/a.toml.bak.20261121T060000Z"
	writeText(t, taken, "an earlier backup\n", 0o644)
	results, err := Commit([]plan.Step{plan.Write(root+"/a.toml", "second\n")}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if readText(t, taken) != "an earlier backup\n" {
		t.Error("the earlier backup was replaced")
	}
	if results[0].Backup != taken+".1" || readText(t, taken+".1") != "first\n" {
		t.Errorf("backup %s", results[0].Backup)
	}
}

func TestResolveRefusesANameLookedUpInAFile(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	writeText(t, root+"/file", "not a directory\n", 0o644)
	writeText(t, root+"/user.lua", "mine\n", 0o644)
	if err := os.Symlink(root+"/file/../user.lua", root+"/link.lua"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.ReadFile(root + "/link.lua"); err == nil {
		t.Fatal("the kernel opened the link; the case no longer tests anything")
	}
	if got, err := Resolve(root + "/link.lua"); err == nil || !strings.Contains(err.Error(), "is not a directory") {
		t.Errorf("Resolve = %q, %v", got, err)
	}
}
