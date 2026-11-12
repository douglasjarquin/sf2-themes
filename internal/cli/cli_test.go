package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func run(args ...string) (stdout, stderr string, status int) {
	var out, errOut bytes.Buffer
	status = Main(args, &out, &errOut)
	return out.String(), errOut.String(), status
}

// The testdata/*.stdout goldens were recorded from `python3.11 ./sf2-themes <args>` at the parity reference.
func TestReadOnlyCommandsMatchTheReferenceOutput(t *testing.T) {
	cases := []struct {
		golden string
		args   []string
	}{
		{"apps", []string{"apps"}},
		{"themes", []string{"themes"}},
		{"show-claw", []string{"show", "claw"}},
		{"show-main-light", []string{"show", "Street Fighter II - Main Light"}},
		{"validate-all", []string{"validate", "--all"}},
		{"validate-all", []string{"validate"}},
		{"validate-names", []string{"validate", "ryu", "claw", "main-light"}},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			want, err := os.ReadFile(filepath.Join("testdata", tc.golden+".stdout"))
			if err != nil {
				t.Fatal(err)
			}
			stdout, stderr, status := run(tc.args...)
			if stdout != string(want) || stderr != "" || status != 0 {
				t.Errorf("status %d, stderr %q, stdout differs from %s: %v", status, stderr, tc.golden, stdout != string(want))
			}
		})
	}
}

func TestUnknownThemeIsAnErrorLine(t *testing.T) {
	for _, args := range [][]string{{"show", "nope"}, {"validate", "ryu", "nope"}} {
		stdout, stderr, status := run(args...)
		if stdout != "" || stderr != "error: unknown theme: nope\n" || status != 1 {
			t.Errorf("%v: status %d, stdout %q, stderr %q", args, status, stdout, stderr)
		}
	}
}

// Usage errors keep the reference's error line and exit status 2; only the usage line above it is rewritten.
// The error lines were recorded from the reference, without `install`, which this CLI no longer has.
func TestUsageErrorsMatchTheReferenceErrorLine(t *testing.T) {
	const commands = "(choose from 'apps', 'themes', 'show', 'validate', 'setup', 'apply', 'current')"
	const apps = "(choose from 'wezterm', 'herdr', 'herdr-app', 'nvim', 'codex', 'starship', 'lazygit', 'claude', 'slack')"
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"bogus"}, "sf2-themes: error: argument COMMAND: invalid choice: 'bogus' " + commands},
		{[]string{"install", "wezterm"}, "sf2-themes: error: argument COMMAND: invalid choice: 'install' " + commands},
		{[]string{"--bogus"}, "sf2-themes: error: unrecognized arguments: --bogus"},
		{[]string{"apps", "extra"}, "sf2-themes: error: unrecognized arguments: extra"},
		{[]string{"show", "a", "b"}, "sf2-themes: error: unrecognized arguments: b"},
		{[]string{"show"}, "sf2-themes show: error: the following arguments are required: theme"},
		{[]string{"apply", "--bogus"}, "sf2-themes apply: error: the following arguments are required: app"},
		{[]string{"apply", "bogusapp"}, "sf2-themes apply: error: argument app: invalid choice: 'bogusapp' " + apps},
		{[]string{"apply", "wezterm", "--theme"}, "sf2-themes apply: error: argument --theme: expected one argument"},
		{[]string{"apply", "wezterm", "--theme", "--dry-run"}, "sf2-themes apply: error: argument --theme: expected one argument"},
		{[]string{"apply", "wezterm", "--dry-run=1"}, "sf2-themes apply: error: argument --dry-run: ignored explicit argument '1'"},
		{[]string{"apply", "wezterm", "--verbose=x"}, "sf2-themes apply: error: argument --verbose/-v: ignored explicit argument 'x'"},
		{[]string{"--help=1"}, "sf2-themes: error: argument -h/--help: ignored explicit argument '1'"},
		{[]string{"-V=1"}, "sf2-themes: error: argument --version/-V: ignored explicit argument '1'"},
		{[]string{"validate", "--all=1"}, "sf2-themes validate: error: argument --all: ignored explicit argument '1'"},
		{[]string{"apps", "--version"}, "sf2-themes: error: unrecognized arguments: --version"},
		{[]string{"current"}, "sf2-themes current: error: the following arguments are required: app"},
		{[]string{"apply", "-", "wezterm"}, "sf2-themes apply: error: argument app: invalid choice: '-' " + apps},
		{[]string{"--"}, "sf2-themes: error: unrecognized arguments: --"},
		{[]string{"--", "apps"}, "sf2-themes: error: argument COMMAND: invalid choice: '--' " + commands},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			stdout, stderr, status := run(tc.args...)
			lines := strings.Split(strings.TrimSuffix(stderr, "\n"), "\n")
			if status != 2 || stdout != "" || len(lines) != 2 || !strings.HasPrefix(lines[0], "usage: ") || lines[1] != tc.want {
				t.Errorf("status %d, stdout %q, stderr %q; want exit 2 and %q", status, stdout, stderr, tc.want)
			}
		})
	}
}

func TestAbbreviatedFlagIsUnknown(t *testing.T) {
	_, stderr, status := run("apply", "wezterm", "--dry")
	if status != 2 || !strings.HasSuffix(stderr, "sf2-themes: error: unrecognized arguments: --dry\n") {
		t.Errorf("status %d, stderr %q", status, stderr)
	}
}

func TestNoArgumentsPrintsHelpAndSucceeds(t *testing.T) {
	stdout, stderr, status := run()
	if status != 0 || stderr != "" || stdout != topHelp {
		t.Errorf("status %d, stderr %q, stdout %q", status, stderr, stdout)
	}
	if strings.Contains(stdout, "install") && strings.Contains(stdout, "deprecated") {
		t.Error("help still mentions the removed install command")
	}
}

func TestHelpAndVersionActWhereverTheyAppear(t *testing.T) {
	// The reference prints help or the version as soon as it reaches the flag, ignoring unknown options before it.
	for _, args := range [][]string{{"-h"}, {"--bogus", "--help"}} {
		if stdout, _, status := run(args...); status != 0 || stdout != topHelp {
			t.Errorf("%v: status %d", args, status)
		}
	}
	for _, args := range [][]string{{"--version"}, {"-V", "apps"}, {"--bogus", "--version"}} {
		if stdout, _, status := run(args...); status != 0 || stdout != "dev\n" {
			t.Errorf("%v: status %d, stdout %q", args, status, stdout)
		}
	}
	stdout, _, status := run("apply", "-h", "bogus")
	if status != 0 || !strings.HasPrefix(stdout, "usage: sf2-themes apply ") {
		t.Errorf("apply -h: status %d, stdout %q", status, stdout)
	}
}

// Command help keeps the reference's argparse layout as printed at a width wide enough not to wrap.
func TestCommandHelpMatchesTheUnwrappedReference(t *testing.T) {
	want := `usage: sf2-themes validate [-h] [--all] [THEME ...]

positional arguments:
  THEME       catalog ids (default: the whole catalog)

options:
  -h, --help  show this help message and exit
  --all       validate every catalog theme
`
	if stdout, _, _ := run("validate", "--help"); stdout != want {
		t.Errorf("validate --help = %q", stdout)
	}
	want = `usage: sf2-themes current [-h] [--config-dir PATH] {wezterm,herdr,herdr-app,nvim,codex,starship,lazygit,claude,slack}

positional arguments:
  {wezterm,herdr,herdr-app,nvim,codex,starship,lazygit,claude,slack}
                        application to read

options:
  -h, --help            show this help message and exit
  --config-dir PATH     read under PATH instead of the app's config location
`
	if stdout, _, _ := run("current", "-h"); stdout != want {
		t.Errorf("current -h = %q", stdout)
	}
}

func TestEveryFileWritingAppHasAnAdapter(t *testing.T) {
	for _, name := range appNames {
		if _, ok := apps[name]; !ok && name != "slack" {
			t.Errorf("%s has no registered adapter", name)
		}
	}
}
