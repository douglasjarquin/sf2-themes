package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/douglasjarquin/sf2-themes/v2/internal/fsys"
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
	mustRun(t, 0, "apply", "wezterm", "--theme", "guile", "--config-dir", target)
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
func TestAPartialFailureKeepsEarlierRecordsAndTheSelectionInEffect(t *testing.T) {
	root := sandbox(t)
	target := root + "/target"
	put(t, target+"/wezterm.lua", seedLua)
	mustRun(t, 0, "apply", "wezterm", "--theme", "guile", "--config-dir", target)
	before := entryFor(t, root, target+"/wezterm.lua")
	put(t, target+"/colors/sf2-akuma-light.toml", "needs repair\n")
	if err := os.Remove(target + "/colors/sf2-balrog.toml"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(target+"/colors/sf2-balrog.toml", 0o755); err != nil {
		t.Fatal(err)
	}
	mustRun(t, 1, "apply", "wezterm", "--theme", "vega", "--config-dir", target)
	after := entryFor(t, root, target+"/wezterm.lua")
	if len(after.Files) != len(before.Files) || after.Setting == nil || *after.Setting != *before.Setting {
		t.Errorf("records lost: %d -> %d files, setting %v -> %v", len(before.Files), len(after.Files), before.Setting, after.Setting)
	}
	if after.Selection != "sf2-guile" || mustRun(t, 0, "current", "wezterm", "--config-dir", target) != "sf2-guile" {
		t.Errorf("selection %q", after.Selection)
	}
	repaired := target + "/colors/sf2-akuma-light.toml"
	for _, file := range after.Files {
		if file.Path == repaired && file.SHA256 != diskSHA(t, repaired) {
			t.Errorf("repaired file recorded %s", file.SHA256)
		}
	}
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
			mustRun(t, 1, "apply", "wezterm", "--theme", "ryu", "--config-dir", target)
			if got := read(t, target+"/wezterm.lua"); got != seedLua {
				t.Errorf("wezterm.lua changed:\n%s", got)
			}
			if _, err := os.Stat(target + "/colors"); !os.IsNotExist(err) {
				t.Errorf("colors written: %v", err)
			}
			if err := os.RemoveAll(root + "/state"); err != nil {
				t.Fatal(err)
			}
			mustRun(t, 0, "apply", "wezterm", "--theme", "ryu", "--config-dir", target)
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

// Finding 6: a comment naming the legacy pointer is not ownership and is never rewritten.
func TestACommentNamingTheLegacyPointerIsNotMigrated(t *testing.T) {
	root := sandbox(t)
	legacy := root + "/config/sf2-theme/wezterm-current.lua"
	put(t, legacy, "-- sf2-themes: sf2-ryu\nreturn \"sf2-ryu\"\n")
	inline := `config.font_size = 12 -- old path was "` + legacy + `"`
	put(t, root+"/a/wezterm.lua", strings.Replace(seedLua, "config.font_size = 12", inline, 1))
	out := mustRun(t, 0, "setup", "wezterm", "--config-dir", root+"/a")
	if strings.Contains(out, "sf2-ryu") {
		t.Errorf("setup adopted another target's selection:\n%s", out)
	}
	if !strings.Contains(read(t, root+"/a/wezterm.lua"), inline) {
		t.Error("comment rewritten")
	}
	if got := mustRun(t, 0, "current", "wezterm", "--config-dir", root+"/a"); got != "sf2-main" {
		t.Errorf("current %s", got)
	}
}

// Finding 6, changed in round 4: a single-quoted integration was never generated by sf2, so it is the user's own
// scheme. Apply refuses without writing anything and points to --adopt, which then installs sf2's integration.
func TestASingleQuotedIntegrationIsTheUsersOwnScheme(t *testing.T) {
	root := sandbox(t)
	legacy := root + "/config/sf2-theme/wezterm-current.lua"
	put(t, legacy, "-- sf2-themes: sf2-ryu\nreturn \"sf2-ryu\"\n")
	target := root + "/t"
	put(t, target+"/wezterm.lua", legacyIntegration(legacy, "'"))
	before := snapshot(t, root)
	_, stderr, status := run("apply", "wezterm", "--theme", "ken", "--config-dir", target)
	if status != 1 || !strings.Contains(stderr, "--adopt") {
		t.Errorf("exit %d\n%s", status, stderr)
	}
	if after := snapshot(t, root); after != before {
		t.Errorf("refusal wrote:\n%s\nbefore:\n%s", after, before)
	}
	mustRun(t, 0, "apply", "wezterm", "--theme", "ken", "--config-dir", target, "--adopt")
	if got := mustRun(t, 0, "current", "wezterm", "--config-dir", target); got != "sf2-ken" {
		t.Errorf("after --adopt: %s", got)
	}
	if got := read(t, legacy); got != "-- sf2-themes: sf2-ryu\nreturn \"sf2-ryu\"\n" {
		t.Errorf("legacy pointer changed: %s", got)
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
	mustRun(t, 0, "apply", "wezterm", "--theme", "ryu", "--config-dir", target)
	entry := entryFor(t, root, target+"/wezterm.lua")
	if got := read(t, entry.Prior); got != original {
		t.Errorf("prior %q, want %q", got, original)
	}
	lua := target + "/wezterm.lua"
	put(t, lua, strings.ReplaceAll(read(t, lua), "\n", "\r\n"))
	if out := mustRun(t, 0, "apply", "wezterm", "--theme", "ryu", "--config-dir", target); !strings.Contains(out, "no changes") {
		t.Errorf("reapply wrote:\n%s", out)
	}
	assertOnDisk(t, entryFor(t, root, target+"/wezterm.lua"))
}

// ownPointer is the selection pointer of the wezterm.lua at lua: beside the file the filesystem opens, named after
// its canonical path.
func ownPointer(lua string) string {
	resolved := fsys.Canonical(lua)
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

// PR #86 finding 3: no spelling of XDG_CONFIG_HOME, dot-dot or symlinked, writes the legacy shared pointer.
func TestAnEquivalentSpellingOfXDGConfigHomeNeverWritesTheLegacyPointer(t *testing.T) {
	root := sandbox(t)
	legacy := root + "/config/sf2-theme/wezterm-current.lua"
	const ryu = "-- sf2-themes: sf2-ryu\nreturn \"sf2-ryu\"\n"
	put(t, legacy, ryu)
	put(t, root+"/other/wezterm.lua", legacyIntegration(legacy, `"`))
	if err := os.Symlink(root+"/config", root+"/alias"); err != nil {
		t.Fatal(err)
	}
	for _, spelling := range []string{root + "/config/../config", root + "/alias"} {
		mustRun(t, 0, "apply", "wezterm", "--theme", "ken", "--config-dir", spelling)
		if got := read(t, legacy); got != ryu {
			t.Errorf("%s: legacy pointer rewritten:\n%s", spelling, got)
		}
		if got := mustRun(t, 0, "current", "wezterm", "--config-dir", root+"/other"); got != "sf2-ryu" {
			t.Errorf("%s: unmigrated target now selects %s", spelling, got)
		}
		if got := mustRun(t, 0, "current", "wezterm", "--config-dir", spelling); got != "sf2-ken" {
			t.Errorf("%s: selects %s", spelling, got)
		}
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

// PR #86 finding 6: an integration inside a Lua long comment is not active and is never edited.
func TestAnIntegrationInALongCommentIsNotTheSelection(t *testing.T) {
	root := sandbox(t)
	legacy := root + "/config/sf2-theme/wezterm-current.lua"
	put(t, legacy, "-- sf2-themes: sf2-ryu\nreturn \"sf2-ryu\"\n")
	put(t, root+"/b/wezterm.lua", legacyIntegration(legacy, `"`))
	comment := "--[=[\nlocal sf2_current = \"" + legacy + "\"\nlocal sf2_scheme = dofile(sf2_current)\n" +
		"config.color_scheme = sf2_scheme\nlocal TERM_THEME = \"untouched\"\n]=]\n"
	target := root + "/a"
	put(t, target+"/wezterm.lua", strings.Replace(seedLua, "return config", comment+"return config", 1))
	mustRun(t, 1, "current", "wezterm", "--config-dir", target)
	if out := mustRun(t, 0, "setup", "wezterm", "--config-dir", target); !strings.Contains(out, "no selection found, selected sf2-main") {
		t.Errorf("setup:\n%s", out)
	}
	lua := read(t, target+"/wezterm.lua")
	if !strings.Contains(lua, comment) || !strings.Contains(lua, `local sf2_current = "`+ownPointer(target+"/wezterm.lua")+`"`) {
		t.Errorf("wezterm.lua:\n%s", lua)
	}
	if got := mustRun(t, 0, "current", "wezterm", "--config-dir", target); got != "sf2-main" {
		t.Errorf("current %s", got)
	}
	if got := mustRun(t, 0, "current", "wezterm", "--config-dir", root+"/b"); got != "sf2-ryu" {
		t.Errorf("other target %s", got)
	}
}

// caseInsensitive reports whether root's volume opens a name in any letter case.
func caseInsensitive(t *testing.T, root string) bool {
	t.Helper()
	put(t, root+"/probe-case", "")
	_, err := os.Stat(root + "/PROBE-CASE")
	return err == nil
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

// Round 3 finding 1: selector identity is the file the filesystem opens, for equivalent and for distinct spellings.
func TestSelectorIdentityFollowsTheFilesystem(t *testing.T) {
	root := sandbox(t)
	legacy := root + "/config/sf2-theme/wezterm-current.lua"
	const ryu = "-- sf2-themes: sf2-ryu\nreturn \"sf2-ryu\"\n"
	put(t, legacy, ryu)
	put(t, root+"/other/wezterm.lua", legacyIntegration(legacy, `"`))

	// `link/..` opens the link target's parent, a different file from the one beside the link.
	put(t, root+"/real/nested/.keep", "")
	if err := os.MkdirAll(root+"/root", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(root+"/real/nested", root+"/root/link"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WEZTERM_CONFIG_FILE", root+"/root/link/../profile.lua")
	mustRun(t, 0, "apply", "wezterm", "--theme", "ryu")
	t.Setenv("WEZTERM_CONFIG_FILE", root+"/root/profile.lua")
	mustRun(t, 0, "apply", "wezterm", "--theme", "ken")
	t.Setenv("WEZTERM_CONFIG_FILE", root+"/root/link/../profile.lua")
	if got := mustRun(t, 0, "current", "wezterm"); got != "sf2-ryu" {
		t.Errorf("link/.. target now selects %s", got)
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

	if !caseInsensitive(t, root) {
		t.Log("case-sensitive volume: letter-case spellings are distinct files here")
		return
	}
	// Another letter case of XDG_CONFIG_HOME is that directory, so the legacy shared pointer stays untouched.
	mustRun(t, 0, "apply", "wezterm", "--theme", "ken", "--config-dir", root+"/CONFIG")
	if got := read(t, legacy); got != ryu {
		t.Errorf("legacy pointer rewritten:\n%s", got)
	}
	if got := mustRun(t, 0, "current", "wezterm", "--config-dir", root+"/other"); got != "sf2-ryu" {
		t.Errorf("unmigrated target now selects %s", got)
	}
	// Another letter case of the same wezterm.lua shares its selection.
	t.Setenv("WEZTERM_CONFIG_FILE", root+"/a/WEZTERM.LUA")
	if got := mustRun(t, 0, "current", "wezterm"); got != "sf2-ken" {
		t.Errorf("WEZTERM.LUA selects %s", got)
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
// writing and never edits it.
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
	_, stderr, status := run("apply", "wezterm", "--theme", "ken", "--config-dir", target)
	if status != 1 || !strings.Contains(stderr, "--adopt") {
		t.Errorf("exit %d\n%s", status, stderr)
	}
	if after := snapshot(t, root); after != before {
		t.Error("the refusal wrote")
	}
}

// Round 3 finding 5: adoption removes only the user's assignment, keeping a long comment's closer on its line, and
// the result selects the theme.
func TestAdoptionKeepsALongCommentCloserAndSelectsTheTheme(t *testing.T) {
	root := sandbox(t)
	target := root + "/target"
	put(t, target+"/wezterm.lua", strings.Replace(seedLua, "return config",
		"--[=[\nkeep this note\n]=] config.color_scheme = \"Builtin User Scheme\"\nreturn config", 1))
	mustRun(t, 0, "apply", "wezterm", "--theme", "ryu", "--config-dir", target, "--adopt")
	lua := read(t, target+"/wezterm.lua")
	if !strings.Contains(lua, "--[=[\nkeep this note\n]=]\n") || strings.Contains(lua, "Builtin User Scheme") ||
		!strings.Contains(lua, `local sf2_current = "`+ownPointer(target+"/wezterm.lua")+`"`) {
		t.Errorf("wezterm.lua:\n%s", lua)
	}
	if got := mustRun(t, 0, "current", "wezterm", "--config-dir", target); got != "sf2-ryu" {
		t.Errorf("current %s", got)
	}
	// A setting sf2 cannot edit safely is refused with nothing written, even with --adopt.
	put(t, root+"/nested/wezterm.lua", strings.Replace(seedLua, "return config",
		"if wezterm.gui then\n  config.color_scheme = \"A\"\nend\nreturn config", 1))
	before := snapshot(t, root)
	mustRun(t, 1, "apply", "wezterm", "--theme", "ryu", "--config-dir", root+"/nested", "--adopt")
	if after := snapshot(t, root); after != before {
		t.Error("the refused adoption wrote")
	}
}
