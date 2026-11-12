package lazygit

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
	"github.com/douglasjarquin/sf2-themes/v2/internal/plan"
)

// Every expected value here was recorded from the reference adapter, src/sf2_theme/adapters/lazygit.py, and
// testdata/lazygit-main.yml is a copy of tests/snapshots/lazygit-main.yml.

func load(t *testing.T, id string) (catalog.Theme, []catalog.Theme) {
	t.Helper()
	themes, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	theme, err := catalog.GetTheme(id, themes)
	if err != nil {
		t.Fatal(err)
	}
	return theme, themes
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestRenderThemeMatchesTheSnapshot(t *testing.T) {
	main, _ := load(t, "main")
	if got, want := RenderTheme(main), readFile(t, "testdata/lazygit-main.yml"); got != want {
		t.Errorf("RenderTheme(main) = %q, want %q", got, want)
	}
}

const managedMainTheme = "  # >>> sf2-themes managed theme\n  # sf2-themes: sf2-main\n  theme:\n" +
	"    activeBorderColor:\n      - '#ad8705'\n      - bold\n    inactiveBorderColor:\n      - '#2a3243'\n" +
	"    searchingActiveBorderColor:\n      - '#a58324'\n    optionsTextColor:\n      - '#ad8705'\n" +
	"    selectedLineBgColor:\n      - '#453714'\n    inactiveViewSelectedLineBgColor:\n      - '#1d2433'\n" +
	"    cherryPickedCommitFgColor:\n      - '#4e975b'\n    cherryPickedCommitBgColor:\n      - '#2a3243'\n" +
	"    markedBaseCommitFgColor:\n      - '#c1753c'\n    markedBaseCommitBgColor:\n      - '#2a3243'\n" +
	"    unstagedChangesColor:\n      - '#c86e67'\n    defaultFgColor:\n      - '#cad1de'\n" +
	"  # <<< sf2-themes managed theme\n"

func TestMergeConfig(t *testing.T) {
	main, _ := load(t, "main")
	for _, tc := range []struct{ name, existing, want string }{
		{
			"adopted theme with a commented gui and a quoted wildcard author",
			"gui:  # interface\n  theme:\n    activeBorderColor:\n      - red\n  authorColors:\n    'bob': '#0000ff'\n" +
				"    \"*\": '#abc'\nos:\n  edit: vim\n",
			"gui:  # interface\n" + managedMainTheme + "  authorColors:\n    'bob': '#0000ff'\n" +
				"    # >>> sf2-themes managed author colors\n    # sf2-themes: sf2-main\n    '*': '#cf6a63'\n" +
				"    # <<< sf2-themes managed author colors\nos:\n  edit: vim\n",
		},
		{
			// A new config starts with a blank line, as the reference writes it.
			"empty config",
			"",
			"\ngui:\n" + managedMainTheme + "  # >>> sf2-themes managed author colors\n  # sf2-themes: sf2-main\n" +
				"  authorColors:\n    '*': '#cf6a63'\n  # <<< sf2-themes managed author colors\n",
		},
	} {
		if got := MergeConfig(tc.existing, main); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestInspect(t *testing.T) {
	main, themes := load(t, "main")
	for _, tc := range []struct {
		content string
		want    plan.Ownership
		detail  string
	}{
		{"gui:\n  theme:\n    activeBorderColor:\n      - red\n", plan.Foreign, ""},
		{"gui:  \n  theme:   # custom\n    activeBorderColor: ['#ffffff']\n", plan.Foreign, ""},
		{"gui:\n  sidePanelWidth: 0.3\n", plan.Absent, ""},
		{MergeConfig("", main), plan.Managed, ""},
		{"gui:\n  # <<< sf2-themes managed theme\n  # >>> sf2-themes managed theme\n", plan.Broken, " has incomplete sf2-themes markers"},
	} {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yml")
		if err := os.WriteFile(path, []byte(tc.content), 0o644); err != nil {
			t.Fatal(err)
		}
		got, err := Inspect(themes, dir)
		if err != nil {
			t.Fatal(err)
		}
		wantDetail := ""
		if tc.detail != "" {
			wantDetail = path + tc.detail
		}
		if got.Ownership != tc.want || got.Detail != wantDetail {
			t.Errorf("%q: %s %q, want %s %q", tc.content, got.Ownership, got.Detail, tc.want, wantDetail)
		}
		if tc.want == plan.Managed && (got.Current == nil || got.Current.Metadata.ID != "main") {
			t.Errorf("%q: current %+v, want main", tc.content, got.Current)
		}
	}
}

// The default directory is the one OS-dependent path: Application Support on macOS, XDG elsewhere.
func TestDefaultDirectoryFollowsThePlatform(t *testing.T) {
	t.Setenv("HOME", "/home/u")
	t.Setenv("LAZYGIT_CONFIG_DIR", "")
	cases := map[string]string{"/x/cfg": "/x/cfg/lazygit", "": "/home/u/.config/lazygit"}
	if runtime.GOOS == "darwin" {
		cases = map[string]string{"/x/cfg": "/home/u/Library/Application Support/lazygit"}
	}
	for xdg, want := range cases {
		t.Setenv("XDG_CONFIG_HOME", xdg)
		got, err := Dir("")
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("XDG_CONFIG_HOME=%q: %q, want %q", xdg, got, want)
		}
	}
	t.Setenv("LAZYGIT_CONFIG_DIR", "~/lg")
	if got, err := Dir(""); err != nil || got != "/home/u/lg" {
		t.Errorf("LAZYGIT_CONFIG_DIR=~/lg: %q, %v", got, err)
	}
}

func TestPlanWritesEveryFragmentAndTheSelectionOnlyWithTheSlot(t *testing.T) {
	main, themes := load(t, "main")
	dir := t.TempDir()
	for slot, want := range map[bool]int{false: len(themes), true: len(themes) + 1} {
		steps, err := Plan(main, themes, dir, slot)
		if err != nil {
			t.Fatal(err)
		}
		if len(steps) != want {
			t.Fatalf("slot=%v: %d steps, want %d", slot, len(steps), want)
		}
		if !strings.HasSuffix(steps[0].Path, "/themes/sf2-main.yml") {
			t.Errorf("first step %q", steps[0].Path)
		}
	}
}
