package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/douglasjarquin/sf2-themes/v2/internal/adapters"
	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
	"github.com/douglasjarquin/sf2-themes/v2/internal/plan"
	"github.com/douglasjarquin/sf2-themes/v2/internal/pycompat"
	"github.com/douglasjarquin/sf2-themes/v2/internal/store"
)

// stubInspection is the inspection a gate case makes the stub adapter report.
type stubInspection struct {
	Ownership  string `json:"ownership"`
	Current    string `json:"current"`
	Unreadable bool   `json:"unreadable"`
	Detail     string `json:"detail"`
}

// gateCase is one entry of testdata/gate.json. Its want block was recorded by running the reference's
// sf2_theme.cli.dispatch under Python 3.11 with APPS["codex"] and APPS["wezterm"] replaced by stubs that behave
// like stubApp below, in a fresh directory seeded with Seed, where {DIR} stands for that directory. The `target:`
// summary line and the fixed-variant suffix on unpaired selections were added to that recording by hand, and
// pair-unknown-refused was changed by hand to write nothing: the reference wrote a pair app's files before refusing.
// verbose-every-outcome and dry-run were changed by hand to keep legacy.txt: the reference deleted it by name.
// A backup's timestamp suffix is recorded as `.bak`.
type gateCase struct {
	Name       string            `json:"name"`
	Args       []string          `json:"args"`
	Inspection stubInspection    `json:"inspection"`
	Seed       map[string]string `json:"seed"`
	Want       struct {
		Exit   int               `json:"exit"`
		Stdout string            `json:"stdout"`
		Stderr string            `json:"stderr"`
		Files  map[string]string `json:"files"`
	} `json:"want"`
}

// stubApp inspects as told and plans one theme file, the setting file when the gate allows it, and a legacy
// removal.
func stubApp(inspection *stubInspection, base adapters.App) adapters.App {
	base.Inspect = func(themes []catalog.Theme, _ string) (plan.Inspection, error) {
		found := plan.Inspection{
			Ownership:  plan.Ownership(inspection.Ownership),
			Unreadable: inspection.Unreadable,
			Detail:     inspection.Detail,
		}
		if inspection.Current != "" {
			theme, err := catalog.GetTheme(inspection.Current, themes)
			if err != nil {
				return plan.Inspection{}, err
			}
			found.Current = &theme
		}
		return found, nil
	}
	base.Plan = func(theme catalog.Theme, _ []catalog.Theme, configDir string, slot bool) ([]plan.Step, error) {
		id := theme.Metadata.SelectableID()
		steps := []plan.Step{plan.Write(pycompat.Path(configDir, "themes", id+".txt"), "palette "+id+"\n")}
		if slot {
			steps = append(steps, plan.Write(pycompat.Path(configDir, "config.txt"), "keep = 1\ntheme = "+id+"\n"))
		}
		return steps, nil
	}
	base.Target = func(configDir string) (string, error) {
		return pycompat.Path(configDir, "config.txt"), nil
	}
	return base
}

// useApp installs app under name for one test and restores whatever was registered there before.
func useApp(t *testing.T, name string, app adapters.App) {
	t.Helper()
	previous, had := apps[name]
	apps[name] = app
	t.Cleanup(func() {
		if had {
			apps[name] = previous
		} else {
			delete(apps, name)
		}
	})
}

var backupSuffix = regexp.MustCompile(`\.bak\.\d{8}T\d{6}Z$`)

// tree reads every file under dir, keyed by slash path, with a backup's timestamp suffix recorded as `.bak`.
func tree(t *testing.T, dir string) map[string]string {
	t.Helper()
	files := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		files[backupSuffix.ReplaceAllString(filepath.ToSlash(rel), ".bak")] = string(data)
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	return files
}

func TestOwnershipGateSelectionAndSummariesMatchTheReference(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "gate.json"))
	if err != nil {
		t.Fatal(err)
	}
	var cases []gateCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			inspection := tc.Inspection
			useApp(t, "codex", stubApp(&inspection, adapters.App{
				Label: "Stub", Reload: "restart the stub", StderrHint: "hint line\n",
			}))
			useApp(t, "wezterm", stubApp(&inspection, adapters.App{
				Label: "Pair Stub", Pair: true, Pointer: true, Reload: "reload the pair",
			}))
			root, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			dir := filepath.Join(root, "cfg")
			for name, content := range tc.Seed {
				path := filepath.Join(dir, name)
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			args := make([]string, len(tc.Args))
			for i, arg := range tc.Args {
				args[i] = strings.ReplaceAll(arg, "{DIR}", dir)
			}
			stdout, stderr, status := run(args...)
			stdout = strings.ReplaceAll(stdout, dir, "{DIR}")
			stderr = strings.ReplaceAll(stderr, dir, "{DIR}")
			if status != tc.Want.Exit || stdout != tc.Want.Stdout || stderr != tc.Want.Stderr {
				t.Errorf("exit %d, want %d\nstdout %q\nwant   %q\nstderr %q\nwant   %q",
					status, tc.Want.Exit, stdout, tc.Want.Stdout, stderr, tc.Want.Stderr)
			}
			got := tree(t, dir)
			if len(got) != len(tc.Want.Files) {
				t.Errorf("files %v, want %v", got, tc.Want.Files)
			}
			for name, content := range tc.Want.Files {
				if got[name] != content {
					t.Errorf("%s = %q, want %q", name, got[name], content)
				}
			}
		})
	}
}

func TestBrokenOwnershipRefusesAdoptWithoutWriting(t *testing.T) {
	inspection := stubInspection{Ownership: "broken", Detail: "bad"}
	useApp(t, "codex", stubApp(&inspection, adapters.App{Label: "Stub"}))
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.txt"), []byte("# >>>\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, status := run("apply", "codex", "--adopt", "--config-dir", dir)
	if status != 1 || stdout != "" || stderr != "error: Stub config cannot be changed safely: bad\n" {
		t.Errorf("status %d, stdout %q, stderr %q", status, stdout, stderr)
	}
	if got := tree(t, dir); len(got) != 1 || got["config.txt"] != "# >>>\n" {
		t.Errorf("files %v", got)
	}
}

// A config that is not UTF-8 is one `error:` line and exit 1, where the reference prints a traceback.
func TestAConfigThatIsNotUTF8IsOneErrorLine(t *testing.T) {
	dir := t.TempDir()
	config := filepath.Join(dir, "config.txt")
	if err := os.WriteFile(config, []byte("\xff\xfe theme\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	app := adapters.App{
		Label: "Stub",
		Inspect: func([]catalog.Theme, string) (plan.Inspection, error) {
			if _, err := plan.ReadExisting(config); err != nil {
				return plan.Inspection{}, err
			}
			return plan.Inspection{Ownership: plan.Managed}, nil
		},
	}
	useApp(t, "codex", app)
	stdout, stderr, status := run("apply", "codex", "--config-dir", dir)
	if status != 1 || stdout != "" || !strings.HasPrefix(stderr, "error: ") || strings.Count(stderr, "\n") != 1 {
		t.Errorf("status %d, stdout %q, stderr %q", status, stdout, stderr)
	}
}

// The import lines were recorded from the reference's tests/test_slack.py.
func TestSlackPrintsTheImportLineAndInstructions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "slack")
	stdout, stderr, status := run("apply", "slack", "--theme", "ken-light", "--adopt", "--dry-run", "--config-dir", dir)
	wantErr := "ken-light\nSet Slack Color Mode to Light.\nTurn Darker sidebars off.\n" +
		"Paste the line into Preferences, Appearance, Custom theme, Import theme.\n"
	if status != 0 || stdout != "#faeeed,#af3c40,#2d7633,#a94a46\n" || stderr != wantErr {
		t.Errorf("status %d, stdout %q, stderr %q", status, stdout, stderr)
	}
	if stdout, _, _ := run("setup", "slack"); stdout != "#131927,#ad8705,#4e975b,#c86e67\n" {
		t.Errorf("setup slack stdout %q", stdout)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("slack wrote its config location")
	}
	stdout, stderr, status = run("current", "slack", "--config-dir", dir)
	if status != 1 || stdout != "" || stderr != "error: no Slack theme to read\n" {
		t.Errorf("current slack: status %d, stdout %q, stderr %q", status, stdout, stderr)
	}
}

func TestRegisterAcceptsOnlyFileWritingAppNames(t *testing.T) {
	for _, name := range []string{"slack", "vim"} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("register(%q) did not panic", name)
				}
			}()
			register(name, adapters.App{})
		}()
	}
}

func TestApplyRecordsOwnedFilesTheSettingAndItsFirstPrior(t *testing.T) {
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	inspection := stubInspection{Ownership: "managed"}
	useApp(t, "codex", stubApp(&inspection, adapters.App{Label: "Stub"}))
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.txt"), []byte("mine = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "legacy.txt"), []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(state, "sf2", "manifest.json")

	if _, stderr, status := run("apply", "codex", "--theme", "ryu", "--dry-run", "--config-dir", dir); status != 0 {
		t.Fatalf("dry run: %d %s", status, stderr)
	}
	if _, err := os.Stat(filepath.Join(state, "sf2")); !os.IsNotExist(err) {
		t.Fatalf("dry run wrote state: %v", err)
	}

	for _, id := range []string{"ryu", "ken"} {
		// The stub's inspection after the commit reports what the run selected, as a real adapter reads it back.
		inspection.Current = id
		if _, stderr, status := run("apply", "codex", "--theme", id, "--config-dir", dir); status != 0 {
			t.Fatalf("apply %s: %d %s", id, status, stderr)
		}
	}
	manifest, err := store.Load(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	config := pycompat.Path(dir, "config.txt")
	entry := manifest.Find("codex", config)
	if entry == nil || len(manifest.Targets) != 1 {
		t.Fatalf("manifest %+v", manifest)
	}
	// The prior is the backup holding the user's own config, not the one holding sf2's first selection.
	prior, err := os.ReadFile(entry.Prior)
	if err != nil || string(prior) != "mine = 1\n" {
		t.Errorf("prior %s: %q, %v", entry.Prior, prior, err)
	}
	ken := "keep = 1\ntheme = sf2-ken\n"
	if entry.Selection != "sf2-ken" || entry.Setting == nil || *entry.Setting != (store.File{Path: config, SHA256: sha(ken)}) {
		t.Errorf("entry %+v", entry)
	}
	// Files written by earlier runs stay owned while they exist; nothing removed them.
	want := []store.File{
		{Path: pycompat.Path(dir, "themes", "sf2-ken.txt"), SHA256: sha("palette sf2-ken\n")},
		{Path: pycompat.Path(dir, "themes", "sf2-ryu.txt"), SHA256: sha("palette sf2-ryu\n")},
	}
	if !slices.Equal(entry.Files, want) {
		t.Errorf("files %+v, want %+v", entry.Files, want)
	}
}

func TestAnUnreadableManifestRefusesBeforeWriting(t *testing.T) {
	state := t.TempDir()
	t.Setenv("XDG_STATE_HOME", state)
	if err := os.MkdirAll(filepath.Join(state, "sf2"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(state, "sf2", "manifest.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	inspection := stubInspection{Ownership: "absent"}
	useApp(t, "codex", stubApp(&inspection, adapters.App{Label: "Stub"}))
	dir := t.TempDir()
	stdout, stderr, status := run("apply", "codex", "--config-dir", dir)
	if status != 1 || stdout != "" || !strings.Contains(stderr, "manifest.json is not a valid sf2 manifest") {
		t.Errorf("status %d, stdout %q, stderr %q", status, stdout, stderr)
	}
	if got := tree(t, dir); len(got) != 0 {
		t.Errorf("wrote %v", got)
	}
}

func sha(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}
