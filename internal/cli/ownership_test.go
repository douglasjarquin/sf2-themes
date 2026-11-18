package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/douglasjarquin/sf2-themes/v2/internal/store"
)

// Regression cases from the independent review of PR #85, each driving the real WezTerm adapter through the CLI in
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

// Finding 6: a single-quoted legacy integration reports the selection it loads, migrates, and keeps it.
func TestASingleQuotedLegacyIntegrationMigratesWithItsSelection(t *testing.T) {
	root := sandbox(t)
	legacy := root + "/config/sf2-theme/wezterm-current.lua"
	put(t, legacy, "-- sf2-themes: sf2-ryu\nreturn \"sf2-ryu\"\n")
	target := root + "/t"
	put(t, target+"/wezterm.lua", legacyIntegration(legacy, "'"))
	if got := mustRun(t, 0, "current", "wezterm", "--config-dir", target); got != "sf2-ryu" {
		t.Errorf("before migration: %s", got)
	}
	if out := mustRun(t, 0, "setup", "wezterm", "--config-dir", target); !strings.Contains(out, "kept sf2-ryu") {
		t.Errorf("setup:\n%s", out)
	}
	lua := read(t, target+"/wezterm.lua")
	if strings.Contains(lua, legacy) || !strings.Contains(lua, `local sf2_current = "`+target+`/sf2-theme/wezterm-current.lua"`) {
		t.Errorf("not migrated:\n%s", lua)
	}
	if got := mustRun(t, 0, "current", "wezterm", "--config-dir", target); got != "sf2-ryu" {
		t.Errorf("after migration: %s", got)
	}
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
