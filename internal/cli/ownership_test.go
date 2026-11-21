package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/douglasjarquin/sf2-themes/v2/internal/fsys"
	"github.com/douglasjarquin/sf2-themes/v2/internal/plan"
	"github.com/douglasjarquin/sf2-themes/v2/internal/store"
)

// Regression cases from the independent reviews of PRs #85 and #86, each driving the real WezTerm adapter through the CLI in
// temporary XDG directories. Expected values come from the files on disk, never from the code under test.

// sandbox isolates HOME and every XDG and WezTerm location for one test and returns its root.
func sandbox(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", root+"/home")
	t.Setenv("XDG_CONFIG_HOME", root+"/config")
	t.Setenv("XDG_STATE_HOME", root+"/state")
	t.Setenv("WEZTERM_CONFIG_DIR", "")
	t.Setenv("WEZTERM_CONFIG_FILE", "")
	return root
}

const seedLua = "local wezterm = require(\"wezterm\")\nlocal config = wezterm.config_builder()\n" +
	"config.font_size = 12\nreturn config\n"

func put(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func diskSHA(t *testing.T, path string) string {
	t.Helper()
	sum := sha256.Sum256([]byte(read(t, path)))
	return hex.EncodeToString(sum[:])
}

func mustRun(t *testing.T, want int, args ...string) string {
	t.Helper()
	stdout, stderr, status := run(args...)
	if status != want {
		t.Fatalf("%v: exit %d, want %d\nstdout %s\nstderr %s", args, status, want, stdout, stderr)
	}
	return strings.TrimSpace(stdout)
}

func entryFor(t *testing.T, root, target string) store.Target {
	t.Helper()
	manifest, err := store.Load(root + "/state/sf2/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	entry := manifest.Find("wezterm", target)
	if entry == nil {
		t.Fatalf("no entry for %s in %+v", target, manifest)
	}
	return *entry
}

// assertOnDisk checks that every recorded fingerprint is the SHA-256 of the bytes now at its path.
func assertOnDisk(t *testing.T, entry store.Target) {
	t.Helper()
	files := append([]store.File{}, entry.Files...)
	if entry.Setting != nil {
		files = append(files, *entry.Setting)
	}
	for _, file := range files {
		if got := diskSHA(t, file.Path); got != file.SHA256 {
			t.Errorf("%s: recorded %s, on disk %s", file.Path, file.SHA256, got)
		}
	}
}

// Finding 1: a refused run records nothing.
func TestARefusedApplyLeavesTheManifestUntouched(t *testing.T) {
	root := sandbox(t)
	target := root + "/target"
	put(t, target+"/wezterm.lua", seedLua)
	mustRun(t, 0, "apply", "wezterm", "--theme", "guile", "--config-dir", target, "--adopt")
	manifest := read(t, root+"/state/sf2/manifest.json")
	victim := target + "/colors/sf2-balrog.toml"
	if err := os.Rename(victim, root+"/relocated.toml"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(root+"/relocated.toml", victim); err != nil {
		t.Fatal(err)
	}
	mustRun(t, 1, "apply", "wezterm", "--theme", "ken", "--config-dir", target)
	if got := read(t, root+"/state/sf2/manifest.json"); got != manifest {
		t.Errorf("refused run changed the manifest:\n%s", got)
	}
}

// Finding 1: a commit that fails part way keeps every earlier record, updates what it wrote, and records the
// selection still in effect.
func TestAFailedApplyLeavesEverythingAsItWas(t *testing.T) {
	for name, block := range map[string]func(t *testing.T, target string){
		// A directory where a late scheme goes fails that write after the earlier ones would have succeeded.
		"directory in place of a scheme": func(t *testing.T, target string) {
			if err := os.RemoveAll(target + "/colors/sf2-zangief.toml"); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(target+"/colors/sf2-zangief.toml", 0o755); err != nil {
				t.Fatal(err)
			}
		},
		"read-only colors directory": func(t *testing.T, target string) {
			if os.Geteuid() == 0 {
				t.Skip("root writes a read-only directory")
			}
			if err := os.MkdirAll(target+"/colors", 0o755); err != nil {
				t.Fatal(err)
			}
			// A missing scheme makes the run create a file there.
			if err := os.RemoveAll(target + "/colors/sf2-akuma.toml"); err != nil {
				t.Fatal(err)
			}
			chmod(t, target+"/colors", 0o555)
		},
		"immutable wezterm.lua": func(t *testing.T, target string) {
			setImmutable(t, target+"/wezterm.lua")
		},
	} {
		for _, first := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/first run %v", name, first), func(t *testing.T) {
				root := sandbox(t)
				target := root + "/target"
				put(t, target+"/wezterm.lua", seedLua)
				if !first {
					mustRun(t, 0, "apply", "wezterm", "--theme", "guile", "--config-dir", target, "--adopt")
					// The next apply changes the config, so a failure there would strand the run part way.
					put(t, target+"/wezterm.lua", seedLua)
				}
				block(t, target)
				before := snapshot(t, root)
				_, stderr, status := run("apply", "wezterm", "--theme", "vega", "--config-dir", target, "--adopt")
				if status != 1 {
					t.Fatalf("exit %d\n%s", status, stderr)
				}
				if after := snapshot(t, root); after != before {
					t.Errorf("a failed apply changed the disk:\n%s", lineDiff(before, after))
				}
			})
		}
	}
}

// chmod sets path's mode for one test and restores a writable mode after it, so the temporary directory can go.
func chmod(t *testing.T, path string, mode os.FileMode) {
	t.Helper()
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o755) })
}

// lineDiff lists the lines of two snapshots that differ.
func lineDiff(before, after string) string {
	was := map[string]bool{}
	for _, line := range strings.Split(before, "\n") {
		was[line] = true
	}
	is := map[string]bool{}
	var b strings.Builder
	for _, line := range strings.Split(after, "\n") {
		is[line] = true
		if !was[line] {
			b.WriteString("+ " + line + "\n")
		}
	}
	for _, line := range strings.Split(before, "\n") {
		if !is[line] {
			b.WriteString("- " + line + "\n")
		}
	}
	return b.String()
}

// Finding 2: concurrent commands for distinct targets all keep their records.
func TestConcurrentAppliesKeepEveryRecord(t *testing.T) {
	root := sandbox(t)
	const count = 12
	var group sync.WaitGroup
	statuses := make([]int, count)
	for i := range count {
		group.Add(1)
		go func() {
			defer group.Done()
			_, _, statuses[i] = run("apply", "wezterm", "--theme", "ryu", "--config-dir", fmt.Sprintf("%s/t%02d", root, i))
		}()
	}
	group.Wait()
	manifest, err := store.Load(root + "/state/sf2/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	for i, status := range statuses {
		if status != 0 {
			t.Errorf("apply %d exited %d", i, status)
		}
	}
	if len(manifest.Targets) != count {
		t.Errorf("%d targets recorded, want %d", len(manifest.Targets), count)
	}
}

// Finding 3: unusable state refuses before the application is changed, and a retry still keeps the original.
func TestUnusableStateRefusesBeforeTheAppIsChanged(t *testing.T) {
	for name, block := range map[string]func(t *testing.T, root string){
		"manifest symlink": func(t *testing.T, root string) {
			put(t, root+"/real-manifest", `{"version":1,"targets":[]}`)
			if err := os.MkdirAll(root+"/state/sf2", 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(root+"/real-manifest", root+"/state/sf2/manifest.json"); err != nil {
				t.Fatal(err)
			}
		},
		"backups file": func(t *testing.T, root string) { put(t, root+"/state/sf2/backups", "blocks\n") },
	} {
		t.Run(name, func(t *testing.T) {
			root := sandbox(t)
			target := root + "/target"
			put(t, target+"/wezterm.lua", seedLua)
			block(t, root)
			mustRun(t, 1, "apply", "wezterm", "--theme", "ryu", "--config-dir", target, "--adopt")
			if got := read(t, target+"/wezterm.lua"); got != seedLua {
				t.Errorf("wezterm.lua changed:\n%s", got)
			}
			if _, err := os.Stat(target + "/colors"); !os.IsNotExist(err) {
				t.Errorf("colors written: %v", err)
			}
			if err := os.RemoveAll(root + "/state"); err != nil {
				t.Fatal(err)
			}
			mustRun(t, 0, "apply", "wezterm", "--theme", "ryu", "--config-dir", target, "--adopt")
			if entry := entryFor(t, root, target+"/wezterm.lua"); entry.Prior == "" || read(t, entry.Prior) != seedLua {
				t.Errorf("prior %q", entry.Prior)
			}
		})
	}
}

// Finding 4: trailing data after the manifest refuses the run instead of being dropped.
func TestTrailingManifestDataRefusesTheRun(t *testing.T) {
	root := sandbox(t)
	content := `{"version":1,"targets":[]}TRAILING`
	put(t, root+"/state/sf2/manifest.json", content)
	mustRun(t, 1, "apply", "wezterm", "--theme", "ryu", "--config-dir", root+"/target")
	if got := read(t, root+"/state/sf2/manifest.json"); got != content {
		t.Errorf("manifest rewritten: %s", got)
	}
	if _, err := os.Stat(root + "/target"); !os.IsNotExist(err) {
		t.Errorf("target written: %v", err)
	}
}

// legacyIntegration is wezterm.lua as earlier releases wrote it, loading the shared pointer through quote.
func legacyIntegration(legacy, quote string) string {
	return "local wezterm = require(\"wezterm\")\nlocal config = wezterm.config_builder()\nconfig.font_size = 12\n" +
		"local sf2_current = " + quote + legacy + quote + "\nwezterm.add_to_config_reload_watch_list(sf2_current)\n" +
		"local sf2_scheme = dofile(sf2_current)\nconfig.color_scheme = sf2_scheme\n" +
		"local sf2_env = config.set_environment_variables or {}\n" +
		"sf2_env.TERM_THEME = (type(sf2_scheme) == \"string\" and sf2_scheme:find(\"-light\", 1, true)) and \"light\" or \"dark\"\n" +
		"config.set_environment_variables = sf2_env\nreturn config\n"
}

// Finding 5: no target's pointer is the legacy shared file, even a --config-dir equal to XDG_CONFIG_HOME.
func TestAConfigDirAtXDGConfigHomeNeverWritesTheLegacyPointer(t *testing.T) {
	root := sandbox(t)
	legacy := root + "/config/sf2-theme/wezterm-current.lua"
	put(t, legacy, "-- sf2-themes: sf2-ryu\nreturn \"sf2-ryu\"\n")
	put(t, root+"/other/wezterm.lua", legacyIntegration(legacy, `"`))
	mustRun(t, 0, "apply", "wezterm", "--theme", "ken", "--config-dir", root+"/config")
	if got := read(t, legacy); got != "-- sf2-themes: sf2-ryu\nreturn \"sf2-ryu\"\n" {
		t.Errorf("legacy pointer rewritten:\n%s", got)
	}
	if got := mustRun(t, 0, "current", "wezterm", "--config-dir", root+"/other"); got != "sf2-ryu" {
		t.Errorf("unmigrated target now selects %s", got)
	}
	if got := mustRun(t, 0, "current", "wezterm", "--config-dir", root+"/config"); got != "sf2-ken" {
		t.Errorf("config-dir target selects %s", got)
	}
}

// Finding 5: two WEZTERM_CONFIG_FILE targets in one WezTerm directory keep independent selections.
func TestTwoWeztermConfigFilesKeepIndependentSelections(t *testing.T) {
	root := sandbox(t)
	for _, theme := range []string{"ryu", "ken"} {
		t.Setenv("WEZTERM_CONFIG_FILE", root+"/"+theme+".lua")
		mustRun(t, 0, "apply", "wezterm", "--theme", theme)
	}
	for _, theme := range []string{"ryu", "ken"} {
		t.Setenv("WEZTERM_CONFIG_FILE", root+"/"+theme+".lua")
		if got := mustRun(t, 0, "current", "wezterm"); got != "sf2-"+theme {
			t.Errorf("%s.lua selects %s", theme, got)
		}
	}
}

// Finding 6, narrowed in round 5: a comment naming the legacy pointer is not ownership and is never rewritten; a
// line with code and a comment is more than sf2 verifies, so the config is left to the user.
func TestACommentNamingTheLegacyPointerIsNotMigrated(t *testing.T) {
	root := sandbox(t)
	legacy := root + "/config/sf2-theme/wezterm-current.lua"
	put(t, legacy, "-- sf2-themes: sf2-ryu\nreturn \"sf2-ryu\"\n")
	inline := `config.font_size = 12 -- old path was "` + legacy + `"`
	lua := strings.Replace(seedLua, "config.font_size = 12", inline, 1)
	put(t, root+"/a/wezterm.lua", lua)
	stdout, _, status := run("setup", "wezterm", "--config-dir", root+"/a")
	if status != 1 || strings.Contains(stdout, "sf2-ryu") {
		t.Errorf("setup exit %d adopted another target's selection:\n%s", status, stdout)
	}
	if read(t, root+"/a/wezterm.lua") != lua {
		t.Error("wezterm.lua rewritten")
	}
	mustRun(t, 1, "current", "wezterm", "--config-dir", root+"/a")
}

// Finding 6, narrowed in round 5: a single-quoted integration was never generated by sf2, so it is the user's own
// scheme. Apply refuses, with or without --adopt, writing nothing and giving the manual steps, including removing the
// older loader.
func TestASingleQuotedIntegrationIsTheUsersOwnScheme(t *testing.T) {
	root := sandbox(t)
	legacy := root + "/config/sf2-theme/wezterm-current.lua"
	put(t, legacy, "-- sf2-themes: sf2-ryu\nreturn \"sf2-ryu\"\n")
	target := root + "/t"
	put(t, target+"/wezterm.lua", legacyIntegration(legacy, "'"))
	before := snapshot(t, root)
	for _, extra := range [][]string{nil, {"--adopt"}} {
		_, stderr, status := run(append([]string{"apply", "wezterm", "--theme", "ken", "--config-dir", target}, extra...)...)
		if status != 1 || !strings.Contains(stderr, "older sf2-themes loader") || !strings.Contains(stderr, "local sf2_current = \""+ownPointer(target+"/wezterm.lua")+"\"") {
			t.Errorf("%v: exit %d\n%s", extra, status, stderr)
		}
		if after := snapshot(t, root); after != before {
			t.Errorf("%v: refusal wrote", extra)
		}
	}
}

// snapshot lists every path under root with its mode, mtime, and content, so a test can tell nothing was written.
func snapshot(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		content := ""
		if info.Mode().IsRegular() {
			content = diskSHA(t, path)
		}
		fmt.Fprintf(&b, "%s %v %d %s\n", path, info.Mode(), info.ModTime().UnixNano(), content)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return b.String()
}

// Finding 7: the kept original is the exact bytes, and fingerprints are the bytes on disk.
func TestPriorAndFingerprintsAreTheBytesOnDisk(t *testing.T) {
	root := sandbox(t)
	target := root + "/t"
	original := strings.ReplaceAll(seedLua, "\n", "\r\n")
	put(t, target+"/wezterm.lua", original)
	mustRun(t, 0, "apply", "wezterm", "--theme", "ryu", "--config-dir", target, "--adopt")
	entry := entryFor(t, root, target+"/wezterm.lua")
	if got := read(t, entry.Prior); got != original {
		t.Errorf("prior %q, want %q", got, original)
	}
	lua := target + "/wezterm.lua"
	if adopted := read(t, lua); strings.Count(adopted, "\r\n") != strings.Count(adopted, "\n") {
		t.Errorf("adoption changed the file's CRLF line breaks:\n%q", adopted)
	}
	if out := mustRun(t, 0, "apply", "wezterm", "--theme", "ryu", "--config-dir", target); !strings.Contains(out, "no changes") {
		t.Errorf("reapply wrote:\n%s", out)
	}
	assertOnDisk(t, entryFor(t, root, target+"/wezterm.lua"))
}

// ownPointer is the selection pointer of the wezterm.lua at lua: beside the file the filesystem opens, named after
// its canonical path.
func ownPointer(lua string) string {
	resolved, err := fsys.Resolve(lua)
	if err != nil {
		panic(err)
	}
	return filepath.Dir(resolved) + "/sf2-theme/wezterm-current-" + fsys.Fingerprint(resolved)[:12] + ".lua"
}

// livePointer is the one selection pointer a --config-dir target holds.
func livePointer(t *testing.T, target string) string {
	t.Helper()
	found, _ := filepath.Glob(target + "/sf2-theme/wezterm-current*.lua")
	if len(found) != 1 {
		t.Fatalf("pointers in %s: %v", target, found)
	}
	return found[0]
}

// pointerFor is the selection pointer content apply writes for theme, taken from a scratch target.
func pointerFor(t *testing.T, root, theme string) string {
	t.Helper()
	scratch := root + "/scratch-" + theme
	mustRun(t, 0, "apply", "wezterm", "--theme", theme, "--config-dir", scratch)
	return read(t, livePointer(t, scratch))
}

// rendezvous makes the next n commands that write signal on the returned channel once they have decided without
// the lock and are about to wait for it.
func rendezvous(t *testing.T, n int) chan struct{} {
	arrived := make(chan struct{}, n)
	previous := awaitingLock
	awaitingLock = func() { arrived <- struct{}{} }
	t.Cleanup(func() { awaitingLock = previous })
	return arrived
}

// PR #86 finding 1: a command that waited for the lock decides again from the configuration it finds under it, so a
// queued setup never undoes an apply that committed first.
func TestAQueuedSetupKeepsTheSelectionCommittedWhileItWaited(t *testing.T) {
	root := sandbox(t)
	target := root + "/target"
	ken := pointerFor(t, root, "ken")
	mustRun(t, 0, "apply", "wezterm", "--theme", "ryu", "--config-dir", target)
	unlock, err := fsys.Lock(root + "/state/sf2/manifest.lock")
	if err != nil {
		t.Fatal(err)
	}
	arrived := rendezvous(t, 1)
	done := make(chan string)
	go func() {
		stdout, stderr, status := run("setup", "wezterm", "--config-dir", target)
		done <- fmt.Sprintf("exit %d\n%s%s", status, stdout, stderr)
	}()
	// Setup has read Ryu and now waits for the lock; another command commits Ken meanwhile.
	<-arrived
	put(t, ownPointer(target+"/wezterm.lua"), ken)
	unlock()
	if out := <-done; !strings.HasPrefix(out, "exit 0\n") || !strings.Contains(out, "kept sf2-ken") {
		t.Errorf("setup:\n%s", out)
	}
	if got := mustRun(t, 0, "current", "wezterm", "--config-dir", target); got != "sf2-ken" {
		t.Errorf("current %s, want the committed sf2-ken", got)
	}
	if got := entryFor(t, root, target+"/wezterm.lua").Selection; got != "sf2-ken" {
		t.Errorf("recorded %s", got)
	}

	// The same holds for real commands queued behind the lock in either order.
	unlock, err = fsys.Lock(root + "/state/sf2/manifest.lock")
	if err != nil {
		t.Fatal(err)
	}
	arrived = rendezvous(t, 2)
	statuses := make(chan int, 2)
	for _, args := range [][]string{
		{"apply", "wezterm", "--theme", "guile", "--config-dir", target},
		{"setup", "wezterm", "--config-dir", target},
	} {
		go func() {
			_, _, status := run(args...)
			statuses <- status
		}()
	}
	<-arrived
	<-arrived
	unlock()
	for range 2 {
		if status := <-statuses; status != 0 {
			t.Errorf("exit %d", status)
		}
	}
	if got := mustRun(t, 0, "current", "wezterm", "--config-dir", target); got != "sf2-guile" {
		t.Errorf("current %s, want the explicit apply's sf2-guile", got)
	}
}

// PR #86 finding 2: a state directory sf2 cannot write refuses before the application changes, even when the lock
// and an earlier record already exist.
func TestAReadOnlyStateDirectoryRefusesBeforeTheAppIsChanged(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root writes a read-only directory")
	}
	root := sandbox(t)
	target := root + "/target"
	mustRun(t, 0, "apply", "wezterm", "--theme", "ryu", "--config-dir", target)
	manifest := read(t, root+"/state/sf2/manifest.json")
	lua := read(t, target+"/wezterm.lua")
	pointer := read(t, ownPointer(target+"/wezterm.lua"))
	if err := os.Chmod(root+"/state/sf2", 0o500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(root+"/state/sf2", 0o700) })
	mustRun(t, 1, "apply", "wezterm", "--theme", "ken", "--config-dir", target)
	if read(t, ownPointer(target+"/wezterm.lua")) != pointer || read(t, target+"/wezterm.lua") != lua {
		t.Error("the application changed before the state refusal")
	}
	if got := mustRun(t, 0, "current", "wezterm", "--config-dir", target); got != "sf2-ryu" {
		t.Errorf("current %s", got)
	}
	if got := read(t, root+"/state/sf2/manifest.json"); got != manifest {
		t.Errorf("manifest changed:\n%s", got)
	}
}

// PR #86 finding 3, narrowed in round 5: a `..` spelling is refused outright; a symlinked spelling is the directory it
// opens, and neither writes the legacy shared pointer.
func TestAnEquivalentSpellingOfXDGConfigHomeNeverWritesTheLegacyPointer(t *testing.T) {
	root := sandbox(t)
	legacy := root + "/config/sf2-theme/wezterm-current.lua"
	const ryu = "-- sf2-themes: sf2-ryu\nreturn \"sf2-ryu\"\n"
	put(t, legacy, ryu)
	put(t, root+"/other/wezterm.lua", legacyIntegration(legacy, `"`))
	before := snapshot(t, root)
	mustRun(t, 1, "apply", "wezterm", "--theme", "ken", "--config-dir", root+"/config/../config")
	if after := snapshot(t, root); after != before {
		t.Error("the `..` spelling wrote")
	}
	if err := os.Symlink(root+"/config", root+"/alias"); err != nil {
		t.Fatal(err)
	}
	mustRun(t, 0, "apply", "wezterm", "--theme", "ken", "--config-dir", root+"/alias")
	if got := read(t, legacy); got != ryu {
		t.Errorf("legacy pointer rewritten:\n%s", got)
	}
	if got := mustRun(t, 0, "current", "wezterm", "--config-dir", root+"/other"); got != "sf2-ryu" {
		t.Errorf("unmigrated target now selects %s", got)
	}
	if got := mustRun(t, 0, "current", "wezterm", "--config-dir", root+"/config"); got != "sf2-ken" {
		t.Errorf("the real directory selects %s", got)
	}
}

// PR #86 finding 4: a WEZTERM_CONFIG_FILE target set up by phase A, whose wezterm.lua loads the WezTerm directory's
// shared pointer, keeps its selection and moves to its own pointer without changing the shared one.
func TestAPhaseAAlternateConfigUpgradesWithItsSelection(t *testing.T) {
	root := sandbox(t)
	profile := root + "/profile.lua"
	shared := root + "/config/wezterm/sf2-theme/wezterm-current.lua"
	ryu := pointerFor(t, root, "ryu")
	// Phase A's output for `WEZTERM_CONFIG_FILE=profile.lua apply wezterm --theme ryu`.
	put(t, shared, ryu)
	put(t, profile, strings.Replace(legacyIntegration(shared, `"`), "config.font_size = 12\n", "", 1))
	t.Setenv("WEZTERM_CONFIG_FILE", profile)
	if got := mustRun(t, 0, "current", "wezterm"); got != "sf2-ryu" {
		t.Errorf("before the upgrade: %s", got)
	}
	if out := mustRun(t, 0, "setup", "wezterm"); !strings.Contains(out, "kept sf2-ryu") {
		t.Errorf("setup:\n%s", out)
	}
	lua := read(t, profile)
	if strings.Contains(lua, shared) || !strings.Contains(lua, `local sf2_current = "`+ownPointer(profile)+`"`) {
		t.Errorf("not moved to its own pointer:\n%s", lua)
	}
	if got := mustRun(t, 0, "current", "wezterm"); got != "sf2-ryu" {
		t.Errorf("after the upgrade: %s", got)
	}
	mustRun(t, 0, "apply", "wezterm", "--theme", "ken")
	if got := mustRun(t, 0, "current", "wezterm"); got != "sf2-ken" {
		t.Errorf("after apply: %s", got)
	}
	if got := read(t, shared); got != ryu {
		t.Errorf("the shared pointer another target may load changed:\n%s", got)
	}
}

// PR #86 finding 5: --config-dir X and WEZTERM_CONFIG_FILE=X/wezterm.lua are one target with one selection.
func TestBothSpellingsOfOneWeztermConfigShareItsSelection(t *testing.T) {
	root := sandbox(t)
	target := root + "/target"
	mustRun(t, 0, "apply", "wezterm", "--theme", "ryu", "--config-dir", target)
	t.Setenv("WEZTERM_CONFIG_FILE", target+"/wezterm.lua")
	if got := mustRun(t, 0, "current", "wezterm"); got != "sf2-ryu" {
		t.Errorf("through WEZTERM_CONFIG_FILE: %s", got)
	}
	mustRun(t, 0, "apply", "wezterm", "--theme", "ken")
	if out := mustRun(t, 0, "setup", "wezterm"); !strings.Contains(out, "kept sf2-ken") {
		t.Errorf("setup through WEZTERM_CONFIG_FILE:\n%s", out)
	}
	t.Setenv("WEZTERM_CONFIG_FILE", "")
	if got := mustRun(t, 0, "current", "wezterm", "--config-dir", target); got != "sf2-ken" {
		t.Errorf("through --config-dir: %s", got)
	}
	if out := mustRun(t, 0, "setup", "wezterm", "--config-dir", target); !strings.Contains(out, "kept sf2-ken") {
		t.Errorf("setup through --config-dir:\n%s", out)
	}
	own := filepath.Base(ownPointer(target + "/wezterm.lua"))
	for _, dir := range []string{target + "/sf2-theme", root + "/config/wezterm/sf2-theme"} {
		entries, _ := os.ReadDir(dir)
		for _, entry := range entries {
			if !strings.HasPrefix(entry.Name(), own) {
				t.Errorf("unused pointer %s/%s", dir, entry.Name())
			}
		}
	}
}

// PR #86 finding 6: an integration inside a Lua long comment is not active and is never edited; a long comment is
// more than sf2 verifies, so the config is left to the user with the steps to finish by hand.
func TestAnIntegrationInALongCommentIsNotTheSelection(t *testing.T) {
	root := sandbox(t)
	legacy := root + "/config/sf2-theme/wezterm-current.lua"
	put(t, legacy, "-- sf2-themes: sf2-ryu\nreturn \"sf2-ryu\"\n")
	put(t, root+"/b/wezterm.lua", legacyIntegration(legacy, `"`))
	comment := "--[=[\nlocal sf2_current = \"" + legacy + "\"\nlocal sf2_scheme = dofile(sf2_current)\n" +
		"config.color_scheme = sf2_scheme\nlocal TERM_THEME = \"untouched\"\n]=]\n"
	target := root + "/a"
	lua := strings.Replace(seedLua, "return config", comment+"return config", 1)
	put(t, target+"/wezterm.lua", lua)
	mustRun(t, 1, "current", "wezterm", "--config-dir", target)
	_, stderr, status := run("setup", "wezterm", "--config-dir", target)
	if status != 1 || !strings.Contains(stderr, "local sf2_current = \""+ownPointer(target+"/wezterm.lua")+"\"") {
		t.Errorf("setup exit %d\n%s", status, stderr)
	}
	if read(t, target+"/wezterm.lua") != lua {
		t.Error("wezterm.lua edited")
	}
	if got := mustRun(t, 0, "current", "wezterm", "--config-dir", root+"/b"); got != "sf2-ryu" {
		t.Errorf("other target %s", got)
	}
}

// phaseAProfile writes what the phase A release left for `WEZTERM_CONFIG_FILE=profile.lua apply wezterm --theme ryu`:
// profile.lua loading the WezTerm directory's shared pointer, that pointer, and a manifest entry claiming both.
func phaseAProfile(t *testing.T, root string) (profile, shared, ryu string) {
	t.Helper()
	profile = root + "/profile.lua"
	shared = root + "/config/wezterm/sf2-theme/wezterm-current.lua"
	ryu = pointerFor(t, root, "ryu")
	put(t, shared, ryu)
	put(t, profile, strings.Replace(legacyIntegration(shared, `"`), "config.font_size = 12\n", "", 1))
	session, err := store.Open("wezterm")
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	session.Manifest.Record(store.Target{App: "wezterm", Target: profile, Selection: "sf2-ryu",
		Setting: &store.File{Path: profile, SHA256: diskSHA(t, profile)},
		Files:   []store.File{{Path: shared, SHA256: diskSHA(t, shared)}}})
	if err := session.Save(); err != nil {
		t.Fatal(err)
	}
	return profile, shared, ryu
}

// Round 3 finding 1 and round 4 findings 1 and 6: a target is the file the filesystem opens, and any other spelling of
// it, `..`, letter case, or Unicode normalization, is refused before anything is written.
func TestSelectorIdentityFollowsTheFilesystem(t *testing.T) {
	root := sandbox(t)
	legacy := root + "/config/sf2-theme/wezterm-current.lua"
	const ryu = "-- sf2-themes: sf2-ryu\nreturn \"sf2-ryu\"\n"
	put(t, legacy, ryu)
	put(t, root+"/other/wezterm.lua", legacyIntegration(legacy, `"`))
	// A path through a missing directory and back out reaches a user's file inspection never saw.
	user := strings.Replace(seedLua, "return config", "config.color_scheme = \"Mine\"\nconfig.font_size = 63\nreturn config", 1)
	put(t, root+"/profile.lua", user)
	before := snapshot(t, root)
	for _, spelling := range []string{root + "/absent/../profile.lua", root + "/other/../profile.lua"} {
		t.Setenv("WEZTERM_CONFIG_FILE", spelling)
		mustRun(t, 1, "apply", "wezterm", "--theme", "ryu")
	}
	if after := snapshot(t, root); after != before {
		t.Error("a `..` spelling wrote")
	}

	// A followed file symlink is the file it opens.
	mustRun(t, 0, "apply", "wezterm", "--theme", "ryu", "--config-dir", root+"/a")
	if err := os.MkdirAll(root+"/b", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(root+"/a/wezterm.lua", root+"/b/wezterm.lua"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WEZTERM_CONFIG_FILE", root+"/b/wezterm.lua")
	if got := mustRun(t, 0, "current", "wezterm"); got != "sf2-ryu" {
		t.Errorf("through the symlink: %s", got)
	}
	mustRun(t, 0, "apply", "wezterm", "--theme", "ken", "--follow-symlinks")
	t.Setenv("WEZTERM_CONFIG_FILE", "")
	if got := mustRun(t, 0, "current", "wezterm", "--config-dir", root+"/a"); got != "sf2-ken" {
		t.Errorf("the linked file selects %s", got)
	}
	manifest, err := store.Load(root + "/state/sf2/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if entry := manifest.Find("wezterm", root+"/b/wezterm.lua"); entry != nil {
		t.Errorf("the symlink spelling got its own record: %+v", entry)
	}

	// Letter case and Unicode normalization a volume treats as one file are refused, where it does.
	put(t, root+"/caf\u00e9/wezterm.lua", seedLua)
	before = snapshot(t, root)
	for _, alias := range []string{root + "/CONFIG", root + "/A", root + "/cafe\u0301"} {
		if _, err := os.Stat(alias); err != nil {
			continue
		}
		mustRun(t, 1, "apply", "wezterm", "--theme", "ken", "--config-dir", alias)
		t.Setenv("WEZTERM_CONFIG_FILE", alias+"/WEZTERM.LUA")
		mustRun(t, 1, "apply", "wezterm", "--theme", "ken")
		t.Setenv("WEZTERM_CONFIG_FILE", "")
	}
	if after := snapshot(t, root); after != before {
		t.Error("an alternative spelling wrote")
	}
	if got := read(t, legacy); got != ryu {
		t.Errorf("legacy pointer rewritten:\n%s", got)
	}
}

// Round 3 finding 2: updating the default target never changes the shared pointer a phase A profile still loads,
// and the profile then moves off it with its own selection and stops claiming it.
func TestTheDefaultTargetNeverOverwritesAPointerAnotherTargetLoads(t *testing.T) {
	root := sandbox(t)
	profile, shared, ryu := phaseAProfile(t, root)
	mustRun(t, 0, "apply", "wezterm", "--theme", "ken")
	if got := read(t, shared); got != ryu {
		t.Errorf("the default target rewrote the shared pointer:\n%s", got)
	}
	t.Setenv("WEZTERM_CONFIG_FILE", profile)
	if got := mustRun(t, 0, "current", "wezterm"); got != "sf2-ryu" {
		t.Errorf("profile selects %s", got)
	}
	if out := mustRun(t, 0, "setup", "wezterm"); !strings.Contains(out, "kept sf2-ryu") {
		t.Errorf("setup:\n%s", out)
	}
	if got := mustRun(t, 0, "current", "wezterm"); got != "sf2-ryu" {
		t.Errorf("after migration: %s", got)
	}
	manifest, err := store.Load(root + "/state/sf2/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	owners := map[string][]string{}
	for _, entry := range manifest.Targets {
		for _, file := range entry.Files {
			if strings.Contains(file.Path, "/sf2-theme/") {
				owners[file.Path] = append(owners[file.Path], entry.Target)
			}
		}
	}
	for path, targets := range owners {
		if len(targets) != 1 || path == shared {
			t.Errorf("%s claimed by %v", path, targets)
		}
	}
	if entry := entryFor(t, root, profile); entry.Setting == nil {
		t.Error("the profile's setting record was lost")
	}
}

// Round 3 medium: WEZTERM_CONFIG_DIR does not move a profile's selection.
func TestTheWeztermDirectoryDoesNotMoveASelection(t *testing.T) {
	root := sandbox(t)
	t.Setenv("WEZTERM_CONFIG_FILE", root+"/profile.lua")
	t.Setenv("WEZTERM_CONFIG_DIR", root+"/a")
	mustRun(t, 0, "apply", "wezterm", "--theme", "ryu")
	t.Setenv("WEZTERM_CONFIG_DIR", root+"/b")
	if got := mustRun(t, 0, "current", "wezterm"); got != "sf2-ryu" {
		t.Errorf("current %s", got)
	}
	mustRun(t, 0, "apply", "wezterm", "--theme", "ken")
	t.Setenv("WEZTERM_CONFIG_DIR", root+"/a")
	if got := mustRun(t, 0, "current", "wezterm"); got != "sf2-ken" {
		t.Errorf("current %s", got)
	}
}

// Round 3 finding 3: a manifest that cannot be replaced refuses before the application changes.
func TestAnImmutableManifestRefusesBeforeTheAppIsChanged(t *testing.T) {
	chflags, err := exec.LookPath("chflags")
	if err != nil {
		t.Skip("no chflags here")
	}
	root := sandbox(t)
	target := root + "/target"
	mustRun(t, 0, "apply", "wezterm", "--theme", "ryu", "--config-dir", target)
	manifest := root + "/state/sf2/manifest.json"
	if out, err := exec.Command(chflags, "uchg", manifest).CombinedOutput(); err != nil {
		t.Skipf("chflags uchg: %v %s", err, out)
	}
	t.Cleanup(func() { _ = exec.Command(chflags, "nouchg", manifest).Run() })
	before := snapshot(t, target)
	mustRun(t, 1, "apply", "wezterm", "--theme", "ken", "--config-dir", target)
	if after := snapshot(t, target); after != before {
		t.Error("the application changed before the refusal")
	}
	if stranded, _ := filepath.Glob(root + "/state/sf2/.manifest.json.*"); len(stranded) > 0 {
		t.Errorf("stranded %v", stranded)
	}
	if err := exec.Command(chflags, "nouchg", manifest).Run(); err != nil {
		t.Fatal(err)
	}
	mustRun(t, 0, "apply", "wezterm", "--theme", "ken", "--config-dir", target)
	if got := entryFor(t, root, target+"/wezterm.lua").Selection; got != "sf2-ken" {
		t.Errorf("recorded %s", got)
	}
}

// Round 3 mediums: after an interrupted command, the next run cleans up its temporary manifest and records what is
// on disk.
func TestTheNextRunRecoversFromAnInterruptedOne(t *testing.T) {
	root := sandbox(t)
	target := root + "/target"
	mustRun(t, 0, "apply", "wezterm", "--theme", "ryu", "--config-dir", target)
	// The run that died had changed the pointer to Ken and staged, but not published, its manifest.
	put(t, livePointer(t, target), pointerFor(t, root, "ken"))
	put(t, root+"/state/sf2/.manifest.json.123456", "partial")
	put(t, root+"/state/sf2/.manifest.json.123456.probe", "partial")
	if out := mustRun(t, 0, "setup", "wezterm", "--config-dir", target); !strings.Contains(out, "kept sf2-ken") {
		t.Errorf("setup:\n%s", out)
	}
	entry := entryFor(t, root, target+"/wezterm.lua")
	if entry.Selection != "sf2-ken" {
		t.Errorf("recorded %s", entry.Selection)
	}
	assertOnDisk(t, entry)
	if stranded, _ := filepath.Glob(root + "/state/sf2/.manifest.json.*"); len(stranded) > 0 {
		t.Errorf("stranded %v", stranded)
	}
}

// Round 3 low: a run that changes nothing writes nothing, not even the state directory, and neither does a refusal
// decided under the lock.
func TestARunThatChangesNothingWritesNoState(t *testing.T) {
	root := sandbox(t)
	target := root + "/target"
	mustRun(t, 0, "apply", "wezterm", "--theme", "ryu", "--config-dir", target)
	before := snapshot(t, root)
	if out := mustRun(t, 0, "apply", "wezterm", "--theme", "ryu", "--config-dir", target); !strings.Contains(out, "no changes") {
		t.Errorf("repeat apply:\n%s", out)
	}
	if after := snapshot(t, root); after != before {
		t.Errorf("repeat apply wrote:\n%s\nbefore:\n%s", after, before)
	}

	unlock, err := fsys.Lock(root + "/state/sf2/manifest.lock")
	if err != nil {
		t.Fatal(err)
	}
	arrived := rendezvous(t, 1)
	done := make(chan int)
	go func() {
		_, _, status := run("apply", "wezterm", "--theme", "ken", "--config-dir", target)
		done <- status
	}()
	<-arrived
	// While it waits, the user switches to a scheme of their own.
	lua := target + "/wezterm.lua"
	put(t, lua, strings.Replace(seedLua, "return config", "config.color_scheme = \"Builtin\"\nreturn config", 1))
	before = snapshot(t, root)
	unlock()
	if status := <-done; status != 1 {
		t.Errorf("exit %d", status)
	}
	if after := snapshot(t, root); after != before {
		t.Errorf("the refusal wrote:\n%s\nbefore:\n%s", after, before)
	}
}

// Round 3 finding 4: a dormant function holding sf2-like lines is not an integration, so apply refuses without
// writing, with or without --adopt, and never edits it.
func TestADormantFunctionIsNotAnIntegration(t *testing.T) {
	root := sandbox(t)
	legacy := root + "/config/sf2-theme/wezterm-current.lua"
	put(t, legacy, "-- sf2-themes: sf2-ryu\nreturn \"sf2-ryu\"\n")
	target := root + "/target"
	put(t, target+"/wezterm.lua", strings.Replace(seedLua, "return config", "config.color_scheme = \"Builtin User Scheme\"\n"+
		"local function inactive() local sf2_current = \""+legacy+"\"; return dofile(sf2_current) end\n"+
		"local TERM_THEME = \"untouched\"\nreturn config", 1))
	mustRun(t, 1, "current", "wezterm", "--config-dir", target)
	before := snapshot(t, root)
	for _, extra := range [][]string{nil, {"--adopt"}} {
		_, stderr, status := run(append([]string{"apply", "wezterm", "--theme", "ken", "--config-dir", target}, extra...)...)
		if status != 1 || !strings.Contains(stderr, "older sf2-themes loader") {
			t.Errorf("%v: exit %d\n%s", extra, status, stderr)
		}
	}
	if after := snapshot(t, root); after != before {
		t.Error("the refusal wrote")
	}
}

// Round 3 finding 5 and round 4 finding 2: --adopt inserts sf2's block only into a file it verifies line by line, and
// otherwise refuses with nothing written, so no statement of the user's is ever removed.
func TestAdoptionKeepsALongCommentCloserAndSelectsTheTheme(t *testing.T) {
	root := sandbox(t)
	target := root + "/target"
	put(t, target+"/wezterm.lua", seedLua)
	mustRun(t, 0, "apply", "wezterm", "--theme", "ryu", "--config-dir", target, "--adopt")
	lua := read(t, target+"/wezterm.lua")
	if !strings.HasPrefix(lua, strings.TrimSuffix(seedLua, "return config\n")) || !strings.Contains(lua, `local sf2_current = "`+ownPointer(target+"/wezterm.lua")+`"`) {
		t.Errorf("wezterm.lua:\n%s", lua)
	}
	if got := mustRun(t, 0, "current", "wezterm", "--config-dir", target); got != "sf2-ryu" {
		t.Errorf("current %s", got)
	}
	for name, block := range map[string]string{
		"closer then two statements": "--[=[\nkeep this note\n]=] config.color_scheme = \"Builtin User Scheme\"; config.font_size = 21",
		"user scheme":                "config.color_scheme = \"Builtin User Scheme\"",
		"nested scheme":              "if wezterm.gui then\n  config.color_scheme = \"A\"\nend",
		"invalid Lua":                "local invalid =",
		"unterminated string":        "config.font = \"Monaspace",
		"unterminated comment":       "--[[ never closed",
	} {
		dir := root + "/" + strings.ReplaceAll(name, " ", "-")
		put(t, dir+"/wezterm.lua", strings.Replace(seedLua, "return config", block+"\nreturn config", 1))
		before := snapshot(t, root)
		_, stderr, status := run("apply", "wezterm", "--theme", "ryu", "--config-dir", dir, "--adopt")
		if status != 1 {
			t.Errorf("%s: exit %d", name, status)
		}
		if name != "invalid Lua" && !strings.HasPrefix(name, "unterminated") && !strings.Contains(stderr, "local sf2_current = \""+ownPointer(dir+"/wezterm.lua")+"\"") {
			t.Errorf("%s: no manual steps:\n%s", name, stderr)
		}
		if after := snapshot(t, root); after != before {
			t.Errorf("%s: the refused adoption wrote", name)
		}
	}
}

// Round 4 finding 3: the three generated lines read back as a field are not an integration, so sf2 never rewrites
// them or reports a selection.
func TestAMarkerReadAsAFieldIsNotAnIntegration(t *testing.T) {
	root := sandbox(t)
	target := root + "/target"
	lua := strings.Replace(seedLua, "return config", "local sf2_current = \""+ownPointer(target+"/wezterm.lua")+"\"\n"+
		"wezterm.add_to_config_reload_watch_list(sf2_current)\nlocal sf2_scheme = dofile(sf2_current)\n"+
		"local unused = other.sf2_scheme\nreturn config", 1)
	put(t, target+"/wezterm.lua", lua)
	mustRun(t, 1, "apply", "wezterm", "--theme", "ken", "--config-dir", target)
	if read(t, target+"/wezterm.lua") != lua {
		t.Error("wezterm.lua edited")
	}
	mustRun(t, 1, "current", "wezterm", "--config-dir", target)
}

// Round 4 finding 4: Lua sf2 cannot verify is refused before anything is written, with or without --adopt.
func TestMalformedLuaWritesNothing(t *testing.T) {
	for name, block := range map[string]string{
		"incomplete local":     "local invalid =",
		"unterminated string":  "config.font = \"Monaspace",
		"unterminated comment": "--[[ never closed",
	} {
		t.Run(name, func(t *testing.T) {
			root := sandbox(t)
			target := root + "/target"
			put(t, target+"/wezterm.lua", strings.Replace(seedLua, "return config", block+"\nreturn config", 1))
			before := snapshot(t, root)
			for _, args := range [][]string{
				{"setup", "wezterm", "--config-dir", target},
				{"apply", "wezterm", "--theme", "ryu", "--config-dir", target},
				{"apply", "wezterm", "--theme", "ryu", "--config-dir", target, "--adopt"},
			} {
				mustRun(t, 1, args...)
			}
			if after := snapshot(t, root); after != before {
				t.Errorf("wrote:\n%s", after)
			}
		})
	}
}

// Round 4 medium: a run after an interrupted migration, whose Lua already loads its own pointer, still stops
// claiming the shared pointer it moved off.
func TestARetryAfterAnInterruptedMigrationDropsTheOldClaim(t *testing.T) {
	root := sandbox(t)
	profile, shared, ryu := phaseAProfile(t, root)
	// The interrupted run had already moved the Lua and written its own pointer, but not the manifest.
	own := ownPointer(profile)
	put(t, own, ryu)
	put(t, profile, strings.Replace(read(t, profile), shared, own, 1))
	if entry := entryFor(t, root, profile); len(entry.Files) != 1 || entry.Files[0].Path != shared {
		t.Fatalf("seeded claim %+v", entry.Files)
	}
	t.Setenv("WEZTERM_CONFIG_FILE", profile)
	if out := mustRun(t, 0, "setup", "wezterm"); !strings.Contains(out, "kept sf2-ryu") {
		t.Errorf("setup:\n%s", out)
	}
	for _, file := range entryFor(t, root, profile).Files {
		if file.Path == shared {
			t.Errorf("still claims %s", shared)
		}
	}
}

// Round 4 medium: a path Lua could not carry verbatim is refused before a pointer or config is written.
func TestAControlCharacterInTheConfigPathWritesNothing(t *testing.T) {
	root := sandbox(t)
	dir := root + "/line\nbreak"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WEZTERM_CONFIG_FILE", dir+"/profile.lua")
	before := snapshot(t, root)
	mustRun(t, 1, "apply", "wezterm", "--theme", "ryu")
	if after := snapshot(t, root); after != before {
		t.Error("wrote")
	}
}

// Round 6, finding 1: a followed symlink is held to the rules the path itself is. A link through a missing directory
// and back out with `..` opens nothing, and a link to another spelling of an existing file names it a second way;
// neither may reach a file and overwrite it.
func TestAFollowedSymlinkCannotReachAnotherFile(t *testing.T) {
	root := sandbox(t)
	foreign := strings.Replace(seedLua, "return config", "config.color_scheme = \"User Scheme\"\nreturn config", 1)
	put(t, root+"/dots/profile.lua", foreign)
	if err := os.Symlink("missing/../profile.lua", root+"/dots/alias.lua"); err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, root)
	t.Setenv("WEZTERM_CONFIG_FILE", root+"/dots/alias.lua")
	_, stderr, status := run("apply", "wezterm", "--theme", "ken", "--follow-symlinks")
	if status != 1 || !strings.Contains(stderr, "`..`") || snapshot(t, root) != before {
		t.Errorf("a link through a missing directory: exit %d, stderr %q, wrote:\n%s", status, stderr, snapshot(t, root))
	}

	put(t, root+"/case/wezterm.lua", "")
	t.Setenv("WEZTERM_CONFIG_FILE", root+"/case/wezterm.lua")
	mustRun(t, 0, "apply", "wezterm", "--theme", "ryu")
	if _, err := os.Stat(root + "/case/WEZTERM.LUA"); err != nil {
		t.Skip("the volume is case-sensitive, so another case names another file")
	}
	if err := os.Symlink("WEZTERM.LUA", root+"/case/alias.lua"); err != nil {
		t.Fatal(err)
	}
	before = snapshot(t, root)
	t.Setenv("WEZTERM_CONFIG_FILE", root+"/case/alias.lua")
	_, stderr, status = run("apply", "wezterm", "--theme", "ken", "--follow-symlinks")
	if status != 1 || !strings.Contains(stderr, "not spelled as it is on disk") || snapshot(t, root) != before {
		t.Errorf("a link to another spelling: exit %d, stderr %q, wrote:\n%s", status, stderr, snapshot(t, root))
	}
}

// Round 6, finding 2: every refusal writes nothing, a valid shape sf2 cannot edit included: no scheme files, pointer,
// backup, manifest, or state.
func TestARefusedValidShapeWritesNothing(t *testing.T) {
	for _, verb := range []string{"apply", "setup"} {
		for _, adopt := range []bool{false, true} {
			root := sandbox(t)
			target := root + "/t"
			put(t, target+"/wezterm.lua", "return { font_size = 12 }\n")
			before := snapshot(t, root)
			args := []string{verb, "wezterm", "--config-dir", target}
			if adopt {
				args = append(args, "--adopt")
			}
			stdout, stderr, status := run(args...)
			if status != 1 || stdout != "" || !strings.Contains(stderr, "Add this integration") {
				t.Errorf("%v: exit %d\nstdout %q\nstderr %q", args, status, stdout, stderr)
			}
			if after := snapshot(t, root); after != before {
				t.Errorf("%v wrote:\n%s", args, after)
			}
		}
	}
}

// Round 6, finding 3: an adopted file is its own bytes with sf2's block inserted before its return, nothing else
// changed: line breaks, blank lines, trailing white space, comments, and a missing final line break all stay.
func TestAdoptionKeepsEveryByteOfTheFile(t *testing.T) {
	for name, nl := range map[string]string{"LF": "\n", "CRLF": "\r\n"} {
		root := sandbox(t)
		target := root + "/t"
		original := strings.Join([]string{
			`local wezterm = require("wezterm")`, "local config = wezterm.config_builder()", "",
			"-- maintain spacing  ", "config.font_size = 12\t", "", "", "return config",
		}, nl)
		put(t, target+"/wezterm.lua", original)
		mustRun(t, 0, "apply", "wezterm", "--theme", "ryu", "--adopt", "--config-dir", target)
		at := strings.Index(original, "return config")
		block := strings.Join([]string{
			`local sf2_current = "` + ownPointer(target+"/wezterm.lua") + `"`,
			"wezterm.add_to_config_reload_watch_list(sf2_current)",
			"local sf2_scheme = dofile(sf2_current)",
			"config.color_scheme = sf2_scheme",
			"local sf2_env = config.set_environment_variables or {}",
			`sf2_env.TERM_THEME = (type(sf2_scheme) == "string" and sf2_scheme:find("-light", 1, true)) and "light" or "dark"`,
			"config.set_environment_variables = sf2_env",
		}, nl) + nl
		if got, want := read(t, target+"/wezterm.lua"), original[:at]+block+original[at:]; got != want {
			t.Errorf("%s: adopted\n%q\nwant\n%q", name, got, want)
		}
		if got := mustRun(t, 0, "current", "wezterm", "--config-dir", target); got != "sf2-ryu" {
			t.Errorf("%s: current %q", name, got)
		}
		if out := mustRun(t, 0, "apply", "wezterm", "--theme", "ryu", "--config-dir", target); !strings.Contains(out, "no changes") {
			t.Errorf("%s: reapply wrote:\n%s", name, out)
		}
	}

	// Lines that end differently cannot take lines that match them, so sf2 prints the steps and writes nothing.
	root := sandbox(t)
	target := root + "/t"
	put(t, target+"/wezterm.lua", strings.Replace(seedLua, "\n", "\r\n", 1))
	before := snapshot(t, root)
	_, stderr, status := run("apply", "wezterm", "--theme", "ryu", "--adopt", "--config-dir", target)
	if status != 1 || !strings.Contains(stderr, "do not all end alike") || !strings.Contains(stderr, "Add this integration") {
		t.Errorf("mixed line breaks: exit %d, stderr %q", status, stderr)
	}
	if snapshot(t, root) != before {
		t.Error("mixed line breaks: adoption wrote")
	}
}

// A replaced sf2 form keeps the file's CRLF line breaks, the ones between its own lines included.
func TestALegacyFormInACRLFFileKeepsItsLineBreaks(t *testing.T) {
	root := sandbox(t)
	target := root + "/t"
	original := strings.ReplaceAll(strings.Replace(seedLua, "return config", "config.color_scheme = \"street-fighter-2\"\nreturn config", 1), "\n", "\r\n")
	put(t, target+"/wezterm.lua", original)
	mustRun(t, 0, "apply", "wezterm", "--theme", "ryu", "--config-dir", target)
	got := read(t, target+"/wezterm.lua")
	if strings.Count(got, "\r\n") != strings.Count(got, "\n") || !strings.Contains(got, "config.color_scheme = sf2_scheme\r\n") {
		t.Errorf("replaced form changed the line breaks:\n%q", got)
	}
	if out := mustRun(t, 0, "apply", "wezterm", "--theme", "ryu", "--config-dir", target); !strings.Contains(out, "no changes") {
		t.Errorf("reapply wrote:\n%s", out)
	}
}

// Only Lua's own white space makes a config blank; a file of other characters is not Lua sf2 may replace.
func TestAFileOfNonLuaSpaceIsNotBlank(t *testing.T) {
	root := sandbox(t)
	target := root + "/t"
	put(t, target+"/wezterm.lua", " ")
	before := snapshot(t, root)
	if _, _, status := run("apply", "wezterm", "--theme", "ryu", "--config-dir", target); status != 1 || snapshot(t, root) != before {
		t.Errorf("a no-break space config: exit %d, wrote:\n%s", status, snapshot(t, root))
	}
}

// Round 7 finding 1: sf2 never deletes or replaces a file it did not write, whatever its name.
func TestAFileSf2DidNotWriteIsNeverDeletedOrReplaced(t *testing.T) {
	cases := []struct {
		app, file string
		refused   bool
	}{
		// Names earlier releases used, which sf2 once deleted by name.
		{"wezterm", "colors/street-fighter-ii-ryu.toml", false},
		{"nvim", "colors/street-fighter-ii-ryu.lua", false},
		{"codex", "themes/ryu.tmTheme", false},
		// Names sf2 writes today, holding the user's own bytes.
		{"wezterm", "colors/sf2-ryu.toml", true},
		{"nvim", "colors/sf2-ryu.lua", true},
		{"nvim", "sf2-theme/current.lua", true},
		{"codex", "themes/sf2-ryu.tmTheme", true},
	}
	for _, c := range cases {
		t.Run(c.app+"/"+c.file, func(t *testing.T) {
			root := sandbox(t)
			target := root + "/target"
			const mine = "-- my own file, not sf2's\n"
			put(t, target+"/"+c.file, mine)
			before := snapshot(t, root)
			want := 0
			if c.refused {
				want = 1
			}
			_, stderr, status := run("apply", c.app, "--theme", "ken", "--config-dir", target)
			if status != want {
				t.Fatalf("exit %d, want %d\n%s", status, want, stderr)
			}
			if got := read(t, target+"/"+c.file); got != mine {
				t.Errorf("the user's file now holds %q", got)
			}
			if backups, _ := filepath.Glob(target + "/" + c.file + ".bak.*"); len(backups) > 0 {
				t.Errorf("backed up and replaced: %v", backups)
			}
			if c.refused {
				if !strings.Contains(stderr, "refusing to replace "+target+"/"+c.file) {
					t.Errorf("stderr %q", stderr)
				}
				if after := snapshot(t, root); after != before {
					t.Errorf("a refused run wrote:\n%s", lineDiff(before, after))
				}
			}
		})
	}
}

// A file sf2 wrote and the user changed afterwards is the user's: sf2 refuses to replace it.
func TestAnEditedFileSf2WroteIsTheUsers(t *testing.T) {
	root := sandbox(t)
	target := root + "/target"
	mustRun(t, 0, "apply", "wezterm", "--theme", "ryu", "--config-dir", target)
	scheme := target + "/colors/sf2-ken.toml"
	put(t, scheme, read(t, scheme)+"# my tweak\n")
	before := snapshot(t, root)
	_, stderr, status := run("apply", "wezterm", "--theme", "ken", "--config-dir", target)
	if status != 1 || !strings.Contains(stderr, "refusing to replace "+scheme) {
		t.Fatalf("exit %d\n%s", status, stderr)
	}
	if after := snapshot(t, root); after != before {
		t.Errorf("a refused run wrote:\n%s", lineDiff(before, after))
	}
}

// What releases before the manifest wrote is exactly what sf2 writes for some selection, so it is replaced as sf2's.
func TestAPointerAnEarlierReleaseWroteIsReplaced(t *testing.T) {
	root := sandbox(t)
	target := root + "/target"
	mustRun(t, 0, "apply", "nvim", "--theme", "ryu", "--config-dir", target)
	if err := os.Remove(root + "/state/sf2/manifest.json"); err != nil {
		t.Fatal(err)
	}
	mustRun(t, 0, "apply", "nvim", "--theme", "ken", "--config-dir", target)
	if got := mustRun(t, 0, "current", "nvim", "--config-dir", target); got != "sf2-ken" {
		t.Errorf("current %q", got)
	}
}

// Round 7 finding 2: when sf2 cannot record a commit, it undoes the commit and the state it kept for it.
func TestARecordThatCannotBeSavedUndoesTheRun(t *testing.T) {
	root := sandbox(t)
	target := root + "/target/wezterm.lua"
	put(t, target, seedLua)
	owned, err := openOwnership("wezterm", target)
	if err != nil {
		t.Fatal(err)
	}
	// The state directory and its lock exist once a run holds the lock; everything after is undone.
	before := snapshot(t, root)
	steps := []plan.Step{plan.Write(target, "return {}\n"), plan.Write(root+"/target/colors/sf2-ryu.toml", "scheme\n")}
	run := prepared{steps: steps, destinations: []string{target, root + "/target/colors/sf2-ryu.toml"}, target: target}
	if err := owned.prepareWrites(run, fsys.Options{}); err != nil {
		t.Fatal(err)
	}
	if owned.kept == "" {
		t.Fatal("the run kept no copy of the setting")
	}
	transaction, err := fsys.Apply(steps, fsys.Options{})
	if err != nil {
		t.Fatal(err)
	}
	err = owned.finish(transaction, nil, func(bool) error { return errors.New("the manifest could not be saved") })
	owned.session.Close()
	if err == nil || !strings.Contains(err.Error(), "could not be saved") {
		t.Fatalf("err %v", err)
	}
	if after := snapshot(t, root); after != before {
		t.Errorf("the undone run left changes:\n%s", lineDiff(before, after))
	}
}

// A pointer sf2 recorded that names a theme the catalog no longer has is sf2's, so setup resets it.
func TestARecordedPointerToAMissingThemeIsReset(t *testing.T) {
	root := sandbox(t)
	target := root + "/target"
	mustRun(t, 0, "apply", "nvim", "--theme", "ryu", "--config-dir", target)
	pointer := target + "/sf2-theme/current.lua"
	gone := strings.ReplaceAll(read(t, pointer), "sf2-ryu", "sf2-retired")
	put(t, pointer, gone)
	path := root + "/state/sf2/manifest.json"
	manifest, err := store.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	for i := range manifest.Targets[0].Files {
		if manifest.Targets[0].Files[i].Path == pointer {
			manifest.Targets[0].Files[i].SHA256 = fsys.Fingerprint(gone)
		}
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	put(t, path, string(data)+"\n")
	if out := mustRun(t, 0, "setup", "nvim", "--config-dir", target); !strings.Contains(out, "selection unreadable, reset to sf2-main") {
		t.Errorf("stdout %q", out)
	}
}

// Round 8 item 1: a builder that shadows a name sf2's block uses is never given the block.
func TestABuilderThatShadowsTheWeztermModuleTakesTheManualPath(t *testing.T) {
	shadowing := "local wezterm = require(\"wezterm\")\nlocal wezterm = wezterm.config_builder()\n" +
		"wezterm.font_size = 12\nreturn wezterm\n"
	for name, c := range map[string]struct {
		lua  string
		args []string
	}{
		"adopt":          {shadowing, []string{"--adopt"}},
		"adopt sf2_env":  {strings.NewReplacer("local config", "local sf2_env", "config.font", "sf2_env.font", "return config", "return sf2_env").Replace(seedLua), []string{"--adopt"}},
		"literal form":   {strings.Replace(shadowing, "return wezterm", "wezterm.color_scheme = \"street-fighter-2\"\nreturn wezterm", 1), nil},
		"apply no adopt": {shadowing, nil},
	} {
		t.Run(name, func(t *testing.T) {
			root := sandbox(t)
			target := root + "/target"
			put(t, target+"/wezterm.lua", c.lua)
			before := snapshot(t, root)
			_, stderr, status := run(append([]string{"apply", "wezterm", "--theme", "ryu", "--config-dir", target}, c.args...)...)
			if status != 1 {
				t.Fatalf("exit %d\n%s", status, stderr)
			}
			if name != "apply no adopt" && !strings.Contains(stderr, "Rename the builder first") {
				t.Errorf("stderr %q", stderr)
			}
			if after := snapshot(t, root); after != before {
				t.Errorf("a refused run wrote:\n%s", lineDiff(before, after))
			}
		})
	}
}

// Round 8 item 2: sf2 never writes a wezterm.lua past Lua's limit of 200 locals in its main chunk.
func TestTheLocalLimitIsKept(t *testing.T) {
	for count, want := range map[int]int{195: 0, 196: 1, 197: 1, 198: 1, 199: 1} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			root := sandbox(t)
			target := root + "/target"
			var locals []string
			for i := range count {
				locals = append(locals, fmt.Sprintf("local value%d", i))
			}
			// The two locals of the builder plus count; the full form adds three more.
			lua := strings.Replace(seedLua, "return config", strings.Join(locals, "\n")+
				"\nconfig.color_scheme = \"street-fighter-2\"\nreturn config", 1)
			put(t, target+"/wezterm.lua", lua)
			before := snapshot(t, root)
			_, stderr, status := run("apply", "wezterm", "--theme", "ryu", "--config-dir", target)
			if status != want {
				t.Fatalf("exit %d, want %d\n%s", status, want, stderr)
			}
			if want == 1 {
				if !strings.Contains(stderr, "locals") {
					t.Errorf("stderr %q", stderr)
				}
				if after := snapshot(t, root); after != before {
					t.Errorf("a refused run wrote:\n%s", lineDiff(before, after))
				}
			}
		})
	}
}

// Round 8 item 3: state sf2 could not record in refuses the run before the lock, so nothing at all is written.
func TestUnusableStateRefusesWithoutCreatingState(t *testing.T) {
	for name, block := range map[string]func(t *testing.T, root string){
		"immutable manifest": func(t *testing.T, root string) {
			setImmutable(t, root+"/state/sf2/manifest.json")
		},
		"symlinked manifest": func(t *testing.T, root string) {
			manifest := root + "/state/sf2/manifest.json"
			put(t, root+"/elsewhere.json", read(t, manifest))
			if err := os.Remove(manifest); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(root+"/elsewhere.json", manifest); err != nil {
				t.Fatal(err)
			}
		},
		"symlinked lock": func(t *testing.T, root string) {
			lock := root + "/state/sf2/manifest.lock"
			if err := os.Remove(lock); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(root+"/elsewhere.lock", lock); err != nil {
				t.Fatal(err)
			}
		},
	} {
		t.Run(name, func(t *testing.T) {
			root := sandbox(t)
			target := root + "/target"
			mustRun(t, 0, "apply", "wezterm", "--theme", "ryu", "--config-dir", target)
			block(t, root)
			before := snapshot(t, root)
			mustRun(t, 1, "apply", "wezterm", "--theme", "ken", "--config-dir", target)
			if after := snapshot(t, root); after != before {
				t.Errorf("a refused run wrote:\n%s", lineDiff(before, after))
			}
		})
	}
}

// Round 8 item 6: every file sf2 cannot prove it wrote is named in one refusal.
func TestEveryUnprovenFileIsNamedAtOnce(t *testing.T) {
	root := sandbox(t)
	target := root + "/target"
	mustRun(t, 0, "apply", "wezterm", "--theme", "ryu", "--config-dir", target)
	edited := []string{target + "/colors/sf2-akuma.toml", target + "/colors/sf2-ken.toml", target + "/colors/sf2-vega.toml"}
	for _, path := range edited {
		put(t, path, read(t, path)+"# mine\n")
	}
	_, stderr, status := run("apply", "wezterm", "--theme", "ken", "--config-dir", target)
	if status != 1 || !strings.Contains(stderr, "refusing to replace 3 files") {
		t.Fatalf("exit %d\n%s", status, stderr)
	}
	for _, path := range edited {
		if !strings.Contains(stderr, "\n  "+path) {
			t.Errorf("%s not named:\n%s", path, stderr)
		}
	}
}
