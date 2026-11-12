package parity

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// Record mode is chosen by environment, not by a go test flag, because
// `go test ./...` rejects a flag the other packages do not define.
//
// envRecord, set to 1, rewrites the goldens from the Go binary built from this checkout.
// Record only after reviewing the behavior change the new goldens will pin.
const envRecord = "SF2_PARITY_RECORD"

const (
	goldenDir = "testdata/golden"
	blobDir   = "testdata/blobs"
)

var (
	subjectOnce sync.Once
	subjectBin  string
	subjectErr  error
	buildDir    string
)

func recording() bool { return os.Getenv(envRecord) == "1" }

func TestMain(m *testing.M) {
	flag.Parse()
	// Every case runs under umask 022 so new files and directories get the modes a user sees.
	syscall.Umask(0o022)
	code := m.Run()
	if buildDir != "" {
		_ = os.RemoveAll(buildDir)
	}
	if code == 0 && recording() && flag.Lookup("test.run").Value.String() == "" {
		if err := pruneGoldens(); err != nil {
			fmt.Fprintln(os.Stderr, "prune goldens:", err)
			code = 1
		}
	}
	os.Exit(code)
}

// underTest builds the sf2-themes binary once and returns its path.
func underTest(t *testing.T) string {
	t.Helper()
	subjectOnce.Do(func() {
		dir, err := os.MkdirTemp("", "sf2-parity-bin-")
		if err != nil {
			subjectErr = err
			return
		}
		buildDir = dir
		bin := filepath.Join(dir, "sf2-themes")
		build := exec.Command("go", "build", "-o", bin, "./cmd/sf2-themes")
		build.Dir = filepath.Join("..", "..")
		if out, err := build.CombinedOutput(); err != nil {
			subjectErr = fmt.Errorf("go build ./cmd/sf2-themes: %w\n%s", err, out)
			return
		}
		subjectBin = bin
	})
	if subjectErr != nil {
		t.Fatal(subjectErr)
	}
	return subjectBin
}

// skipReason returns why a case does not run on this operating system, or "".
func skipReason(c Case) string {
	if c.PerOS && runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		return "default locations are recorded for darwin and linux only"
	}
	return ""
}

func goldenPath(c Case) string {
	name := c.Name
	if c.PerOS {
		name += "." + runtime.GOOS
	}
	return filepath.Join(goldenDir, filepath.FromSlash(name)+".golden")
}

func casesByName() map[string]Case {
	byName := map[string]Case{}
	for _, c := range Cases() {
		byName[c.Name] = c
	}
	return byName
}

// blobs is the golden blob store. In record mode it writes through to testdata.
type blobs struct {
	mu     sync.Mutex
	actual map[string][]byte
}

var store = &blobs{actual: map[string][]byte{}}

func (b *blobs) put(content []byte) string {
	hash := Hash(content)
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, seen := b.actual[hash]; seen {
		return hash
	}
	b.actual[hash] = content
	if recording() {
		path := filepath.Join(blobDir, hash)
		if existing, err := os.ReadFile(path); err != nil || !bytes.Equal(existing, content) {
			if err := os.MkdirAll(blobDir, 0o755); err != nil {
				panic(err)
			}
			if err := os.WriteFile(path, content, 0o644); err != nil {
				panic(err)
			}
		}
	}
	return hash
}

func (b *blobs) get(hash string) ([]byte, bool) {
	b.mu.Lock()
	content, ok := b.actual[hash]
	b.mu.Unlock()
	return content, ok
}

func readBlob(hash string) ([]byte, error) {
	return os.ReadFile(filepath.Join(blobDir, hash))
}

// newSandbox creates an empty sandbox root resolved through symlinks.
func newSandbox(t *testing.T, dir string) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "home"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

func seedSandbox(t *testing.T, c Case, root string) {
	t.Helper()
	if c.SeedFrom != "" {
		source, ok := casesByName()[c.SeedFrom]
		if !ok {
			t.Fatalf("seed case %q does not exist", c.SeedFrom)
		}
		golden, err := os.ReadFile(goldenPath(source))
		if err != nil {
			t.Fatalf("seed from %s: %v", c.SeedFrom, err)
		}
		tree, _, err := ParseGolden(string(golden))
		if err != nil {
			t.Fatalf("seed from %s: %v", c.SeedFrom, err)
		}
		for _, entry := range tree {
			path := filepath.Join(root, filepath.FromSlash(entry.Path))
			switch entry.Kind {
			case "dir":
				must(t, os.MkdirAll(path, entry.Mode))
				must(t, os.Chmod(path, entry.Mode))
			case "file":
				content, err := readBlob(entry.Hash)
				must(t, err)
				must(t, os.MkdirAll(filepath.Dir(path), 0o755))
				must(t, os.WriteFile(path, []byte(Expand(string(content), root)), entry.Mode))
				must(t, os.Chmod(path, entry.Mode))
			case "link":
				must(t, os.MkdirAll(filepath.Dir(path), 0o755))
				must(t, os.Symlink(Expand(entry.Target, root), path))
			}
		}
	}
	for _, file := range c.Seed {
		path := filepath.Join(root, filepath.FromSlash(file.Path))
		must(t, os.MkdirAll(filepath.Dir(path), 0o755))
		if file.Link != "" {
			must(t, os.Symlink(Expand(file.Link, root), path))
			continue
		}
		mode := fs.FileMode(0o644)
		if file.Mode != 0 {
			mode = fs.FileMode(file.Mode)
		}
		must(t, os.WriteFile(path, []byte(Expand(file.Content, root)), mode))
		must(t, os.Chmod(path, mode))
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// environment is the cleared environment a step runs in. Every adapter variable
// (XDG_CONFIG_HOME, WEZTERM_CONFIG_DIR, WEZTERM_CONFIG_FILE, STARSHIP_CONFIG,
// NVIM_CONFIG_DIR, LAZYGIT_CONFIG_DIR, HERDR_CONFIG_PATH, CODEX_HOME,
// CLAUDE_CONFIG_DIR, SF2_THEME_DIR) is unset unless the case sets it.
func environment(c Case, root string) []string {
	env := []string{
		"HOME=" + filepath.Join(root, "home"),
		"PATH=/usr/bin:/bin",
		"TZ=UTC",
		"LANG=C.UTF-8",
		"LC_ALL=C.UTF-8",
		"COLUMNS=80",
	}
	keys := make([]string, 0, len(c.Env))
	for key := range c.Env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		env = append(env, key+"="+Expand(c.Env[key], root))
	}
	return env
}

type rawStep struct {
	exit           int
	stdout, stderr []byte
}

func runSteps(t *testing.T, bin string, c Case, root string, todo []Step) []rawStep {
	t.Helper()
	env := environment(c, root)
	var steps []rawStep
	for _, s := range todo {
		args := make([]string, len(s.Args))
		for i, arg := range s.Args {
			args[i] = Expand(arg, root)
		}
		command := exec.Command(bin, args...)
		command.Dir = root
		command.Env = env
		var stdout, stderr bytes.Buffer
		command.Stdout, command.Stderr = &stdout, &stderr
		err := command.Run()
		exit := 0
		var exitErr *exec.ExitError
		switch {
		case errors.As(err, &exitErr) && exitErr.ExitCode() >= 0:
			exit = exitErr.ExitCode()
		case err != nil:
			t.Fatalf("run %v: %v", s.Args, err)
		}
		steps = append(steps, rawStep{exit: exit, stdout: stdout.Bytes(), stderr: stderr.Bytes()})
	}
	return steps
}

// runCase runs a case in a fresh sandbox under dir and returns its normalized result,
// the tail masks that matched nothing, and the tree, without backups, as it stood before the last step.
func runCase(t *testing.T, bin string, c Case, dir string) (result Result, unmatched []Mask, beforeLast []Entry) {
	t.Helper()
	root := newSandbox(t, dir)
	seedSandbox(t, c, root)
	before, err := Snapshot(root)
	must(t, err)
	for i, s := range c.Steps {
		beforeLast = before
		raw := runSteps(t, bin, c, root, c.Steps[i:i+1])[0]
		after, err := Snapshot(root)
		must(t, err)
		stdout, stdoutExempt, missOut := ApplyMasks(string(Normalize(raw.stdout, root)), Stdout, s.Masks)
		stderr, stderrExempt, missErr := ApplyMasks(string(Normalize(raw.stderr, root)), Stderr, s.Masks)
		unmatched = append(append(unmatched, missOut...), missErr...)
		result.Steps = append(result.Steps, StepResult{
			Args: s.Args, Exit: raw.exit,
			Stdout: stdout, Stderr: stderr,
			StdoutExempt: stdoutExempt, StderrExempt: stderrExempt,
			Backups: Backups(before, after),
		})
		before = after
	}
	return Result{Steps: result.Steps, Tree: WithoutBackups(before)}, unmatched, WithoutBackups(beforeLast)
}

func TestParity(t *testing.T) {
	var direct, seeded []Case
	for _, c := range Cases() {
		if c.SeedFrom != "" {
			seeded = append(seeded, c)
		} else {
			direct = append(direct, c)
		}
	}
	// Seeded cases read another case's golden, so they run after every golden is recorded.
	for _, phase := range []struct {
		name  string
		cases []Case
	}{{"direct", direct}, {"seeded", seeded}} {
		t.Run(phase.name, func(t *testing.T) {
			for _, c := range phase.cases {
				t.Run(c.Name, func(t *testing.T) {
					t.Parallel()
					if reason := skipReason(c); reason != "" {
						t.Skip(reason)
					}
					checkCase(t, underTest(t), c)
				})
			}
		})
	}
}

func checkCase(t *testing.T, bin string, c Case) {
	var before map[string]string
	if c.GuardRealHome {
		before = realHomeSnapshot(t)
	}
	result, unmatched, beforeLast := runCase(t, bin, c, t.TempDir())
	if c.GuardRealHome {
		if after := realHomeSnapshot(t); !equalMaps(before, after) {
			t.Errorf("the real home directory changed: before %v, after %v", before, after)
		}
	}
	if c.NoChanges {
		checkNoChanges(t, result, beforeLast)
	}
	text := Render(result, store.put)
	path := goldenPath(c)
	if recording() {
		if len(unmatched) > 0 {
			t.Fatalf("masks matched no output: %+v", unmatched)
		}
		must(t, os.MkdirAll(filepath.Dir(path), 0o755))
		must(t, os.WriteFile(path, []byte(text), 0o644))
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no golden for %s (record with `mise run parity:record` on %s): %v", c.Name, runtime.GOOS, err)
	}
	if string(want) != text {
		t.Errorf("%s differs from its golden %s:\n%s", c.Name, path, describeDifference(string(want), text))
	}
}

// checkNoChanges requires the last step to report no changes, write no backup, and leave the tree as it was.
func checkNoChanges(t *testing.T, result Result, before []Entry) {
	t.Helper()
	last := result.Steps[len(result.Steps)-1]
	if !strings.Contains(last.Stdout, "  changes: no changes\n") {
		t.Errorf("last step did not report no changes:\n%s", last.Stdout)
	}
	if len(last.Backups) > 0 {
		t.Errorf("last step wrote backups: %s", Render(Result{Steps: []StepResult{{Backups: last.Backups}}}, Hash))
	}
	if was, now := Render(Result{Tree: before}, Hash), Render(Result{Tree: result.Tree}, Hash); was != now {
		t.Errorf("last step changed the tree:\n%s", describeDifference(was, now))
	}
}

// realHomeSnapshot hashes the real home directory's config files the CLI could write by default.
func realHomeSnapshot(t *testing.T) map[string]string {
	t.Helper()
	home, err := os.UserHomeDir()
	must(t, err)
	exact := []string{
		".config/wezterm/wezterm.lua", ".wezterm.lua", ".config/sf2-theme/wezterm-current.lua",
		".config/sf2-theme/zsh-syntax-highlighting.zsh", ".config/starship.toml", ".config/herdr/config.toml",
		".config/herdr-app/settings.json", ".codex/config.toml", ".claude/settings.json",
		"Library/Application Support/lazygit/config.yml", ".config/lazygit/config.yml",
	}
	globs := []string{
		".config/wezterm/colors/sf2-*.toml", ".config/nvim/colors/sf2-*.lua", ".config/nvim/sf2-theme/*",
		".config/nvim/plugin/sf2-theme.lua", ".config/herdr-app/themes/sf2-*.json", ".codex/themes/sf2-*.tmTheme",
		".claude/themes/sf2-*.json", "Library/Application Support/lazygit/themes/sf2-*.yml", ".config/lazygit/themes/sf2-*.yml",
	}
	paths := []string{}
	for _, rel := range exact {
		paths = append(paths, filepath.Join(home, rel))
	}
	for _, pattern := range globs {
		matches, err := filepath.Glob(filepath.Join(home, pattern))
		must(t, err)
		paths = append(paths, matches...)
	}
	snapshot := map[string]string{}
	for _, path := range paths {
		if content, err := os.ReadFile(path); err == nil {
			snapshot[path] = Hash(content)
		}
	}
	return snapshot
}

func equalMaps(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if b[key] != value {
			return false
		}
	}
	return true
}

// describeDifference shows the first differing line of a golden and, when it names a blob,
// the first differing line inside that blob.
func describeDifference(want, got string) string {
	wantLines, gotLines := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := 0; i < len(wantLines) || i < len(gotLines); i++ {
		w, g := lineAt(wantLines, i), lineAt(gotLines, i)
		if w == g {
			continue
		}
		var b strings.Builder
		from := max(0, i-3)
		for j := from; j < i; j++ {
			fmt.Fprintf(&b, "  %s\n", wantLines[j])
		}
		fmt.Fprintf(&b, "- %s\n+ %s\n", w, g)
		wantHash, gotHash := blobIn(w), blobIn(g)
		if wantHash != "" && gotHash != "" {
			wantBlob, _ := readBlob(wantHash)
			gotBlob, _ := store.get(gotHash)
			fmt.Fprintf(&b, "inside the blob:\n%s", firstLineDifference(string(wantBlob), string(gotBlob)))
		}
		return b.String()
	}
	return "(no line differs)"
}

func lineAt(lines []string, i int) string {
	if i < len(lines) {
		return lines[i]
	}
	return "<end of golden>"
}

func blobIn(line string) string {
	fields := strings.Fields(line)
	switch {
	case len(fields) >= 4 && fields[0] == "file":
		return fields[2]
	case len(fields) == 2 && fields[0] == "=":
		return fields[1]
	}
	return ""
}

func firstLineDifference(want, got string) string {
	wantLines, gotLines := strings.Split(want, "\n"), strings.Split(got, "\n")
	for i := 0; i < len(wantLines) || i < len(gotLines); i++ {
		if w, g := lineAt(wantLines, i), lineAt(gotLines, i); w != g {
			return fmt.Sprintf("line %d\n- %s\n+ %s\n", i+1, w, g)
		}
	}
	return "(no line differs)\n"
}

// pruneGoldens removes goldens no case records and blobs no golden refers to. It runs
// only after a full record.
func pruneGoldens() error {
	allowed := map[string]bool{}
	for _, c := range Cases() {
		name := filepath.Join(goldenDir, filepath.FromSlash(c.Name))
		if c.PerOS {
			allowed[name+".darwin.golden"] = true
			allowed[name+".linux.golden"] = true
		} else {
			allowed[name+".golden"] = true
		}
	}
	referenced := map[string]bool{}
	err := filepath.WalkDir(goldenDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !allowed[path] {
			return os.Remove(path)
		}
		golden, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, refs, err := ParseGolden(string(golden))
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		for _, ref := range refs {
			referenced[ref] = true
		}
		return nil
	})
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(blobDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !referenced[entry.Name()] {
			if err := os.Remove(filepath.Join(blobDir, entry.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}

// TestGoldenStoreMatchesTheCaseTable needs no binary: every golden belongs to a case,
// every blob a golden names exists, and no blob is orphaned.
func TestGoldenStoreMatchesTheCaseTable(t *testing.T) {
	if recording() {
		t.Skip("record mode rewrites the store and prunes it afterwards")
	}
	byName := map[string]Case{}
	for _, c := range Cases() {
		if _, dup := byName[c.Name]; dup {
			t.Errorf("duplicate case name %s", c.Name)
		}
		byName[c.Name] = c
		if len(c.Steps) == 0 {
			t.Errorf("%s has no steps", c.Name)
		}
		if len(c.Groups) == 0 {
			t.Errorf("%s belongs to no group", c.Name)
		}
		if c.SeedFrom != "" {
			if source, ok := byName[c.SeedFrom]; !ok || source.SeedFrom != "" {
				t.Errorf("%s seeds from %s, which must be an earlier recorded case without a seed of its own", c.Name, c.SeedFrom)
			}
		}
	}
	referenced := map[string]bool{}
	goldens := map[string]bool{}
	err := filepath.WalkDir(goldenDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		goldens[path] = true
		golden, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, refs, err := ParseGolden(string(golden))
		if err != nil {
			t.Errorf("%s: %v", path, err)
			return nil
		}
		for _, ref := range refs {
			referenced[ref] = true
			if _, err := os.Stat(filepath.Join(blobDir, ref)); err != nil {
				t.Errorf("%s names missing blob %s", path, ref)
			}
		}
		return nil
	})
	must(t, err)
	for _, c := range Cases() {
		if c.PerOS {
			for _, goos := range []string{"darwin", "linux"} {
				path := filepath.Join(goldenDir, filepath.FromSlash(c.Name)+"."+goos+".golden")
				if !goldens[path] {
					t.Errorf("no %s golden for %s", goos, c.Name)
				}
				delete(goldens, path)
			}
			continue
		}
		path := goldenPath(c)
		if !goldens[path] {
			t.Errorf("no golden for %s", c.Name)
		}
		delete(goldens, path)
	}
	for path := range goldens {
		t.Errorf("golden %s belongs to no case", path)
	}
	entries, err := os.ReadDir(blobDir)
	must(t, err)
	for _, entry := range entries {
		if !referenced[entry.Name()] {
			t.Errorf("blob %s is referred to by no golden", entry.Name())
		}
	}
}

// A sandbox reached through a symlink, as macOS temp directories are, records the same golden.
func TestSymlinkedSandboxRecordsTheSameGolden(t *testing.T) {
	c := casesByName()["wezterm/absent/apply"]
	if reason := skipReason(c); reason != "" {
		t.Skip(reason)
	}
	bin := underTest(t)
	base := t.TempDir()
	target := filepath.Join(base, "target")
	link := filepath.Join(base, "link")
	must(t, os.Mkdir(target, 0o755))
	must(t, os.Symlink(target, link))
	result, _, _ := runCase(t, bin, c, link)
	want, err := os.ReadFile(goldenPath(c))
	must(t, err)
	if got := Render(result, store.put); got != string(want) {
		t.Errorf("a symlinked sandbox changed the golden:\n%s", describeDifference(string(want), got))
	}
}

// A tail mask hides only the text after the fixed prefix; a changed prefix still fails.
func TestTailMaskStillComparesItsPrefix(t *testing.T) {
	mask := []Mask{Tail(" is not valid TOML: ")}
	golden, _, _ := ApplyMasks("error: {SANDBOX}/config/starship/starship.toml is not valid TOML: Invalid value (at line 1, column 11)\n", Stderr, mask)
	if want := "error: {SANDBOX}/config/starship/starship.toml is not valid TOML: <masked>\n"; golden != want {
		t.Fatalf("masked golden = %q, want %q", golden, want)
	}
	for _, tc := range []struct {
		name   string
		actual string
		same   bool
	}{
		{"another decoder's message", "error: {SANDBOX}/config/starship/starship.toml is not valid TOML: toml: line 1: expected value\n", true},
		{"changed wording before the marker", "error: {SANDBOX}/config/starship/starship.toml is invalid TOML: Invalid value\n", false},
		{"changed path before the marker", "error: {SANDBOX}/config/starship.toml is not valid TOML: Invalid value\n", false},
		{"missing error prefix", "{SANDBOX}/config/starship/starship.toml is not valid TOML: Invalid value\n", false},
		{"extra line", "error: {SANDBOX}/config/starship/starship.toml is not valid TOML: x\nmore\n", false},
	} {
		got, _, _ := ApplyMasks(tc.actual, Stderr, mask)
		if (got == golden) != tc.same {
			t.Errorf("%s: masked %q against golden %q, want same=%v", tc.name, got, golden, tc.same)
		}
	}
	// The mask leaves stdout alone.
	if out, _, unmatched := ApplyMasks("is not valid TOML: x\n", Stdout, mask); out != "is not valid TOML: x\n" || unmatched != nil {
		t.Errorf("stdout was masked: %q %v", out, unmatched)
	}
}

// Each step records the backups it wrote, new or over an earlier one, with the stamp masked:
// a second backup records the same whether it fell in the first one's second, replacing it,
// or in a later second, beside it; an extra backup in a step is recorded.
func TestBackupsAreRecordedPerStep(t *testing.T) {
	root := t.TempDir()
	write := func(name, content string, at time.Time) {
		path := filepath.Join(root, name)
		must(t, os.WriteFile(path, []byte(content), 0o644))
		must(t, os.Chtimes(path, at, at))
	}
	snapshot := func() []Entry {
		entries, err := Snapshot(root)
		must(t, err)
		return entries
	}
	render := func(backups []Entry) string {
		return Render(Result{Steps: []StepResult{{Args: []string{"apply"}, Backups: backups}}}, Hash)
	}
	first, second := time.Unix(1_800_000_000, 0), time.Unix(1_800_000_000, 500)
	backupLine := func(content string) string {
		return "backup 0644 " + Hash([]byte(content)) + " config.toml.bak.<STAMP>\n"
	}
	stepLines := "step apply\nexit 0\nstdout\nstderr\n"

	write("config.toml", "original\n", first)
	start := snapshot()
	write("config.toml", "second\n", first)
	write("config.toml.bak.20261112T101500Z", "original\n", first)
	afterOne := snapshot()
	if got, want := render(Backups(start, afterOne)), stepLines+backupLine("original\n")+"tree\n"; got != want {
		t.Errorf("first step recorded\n%s\nwant\n%s", got, want)
	}
	if tree := Render(Result{Tree: WithoutBackups(afterOne)}, Hash); strings.Contains(tree, ".bak.") {
		t.Errorf("the tree holds a backup:\n%s", tree)
	}

	// The second step backs up "second", replacing the first backup in the same second.
	write("config.toml.bak.20261112T101500Z", "second\n", second)
	sameSecond := render(Backups(afterOne, snapshot()))
	// The same step in a later second leaves the first backup beside its own.
	write("config.toml.bak.20261112T101500Z", "original\n", first)
	write("config.toml.bak.20261112T101501Z", "second\n", second)
	laterSecond := render(Backups(afterOne, snapshot()))
	if want := stepLines + backupLine("second\n") + "tree\n"; sameSecond != want || laterSecond != want {
		t.Errorf("the second step recorded\nsame second:\n%s\nlater second:\n%s\nwant:\n%s", sameSecond, laterSecond, want)
	}

	// The same backup content written again in the same second still counts as written.
	afterTwo := snapshot()
	write("config.toml.bak.20261112T101501Z", "second\n", second.Add(time.Millisecond))
	if got, want := render(Backups(afterTwo, snapshot())), stepLines+backupLine("second\n")+"tree\n"; got != want {
		t.Errorf("a rewritten backup recorded\n%s\nwant\n%s", got, want)
	}

	// A step that also writes an extra, older backup records it.
	afterTwo = snapshot()
	write("config.toml.bak.20000101T000000Z", "extra\n", second)
	if got, want := render(Backups(afterTwo, snapshot())), stepLines+backupLine("extra\n")+"tree\n"; got != want {
		t.Errorf("an extra backup recorded\n%s\nwant\n%s", got, want)
	}
}
