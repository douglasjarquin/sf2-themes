package pycompat

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPath(t *testing.T) {
	for _, tc := range pathCases {
		if got := Path(tc.parts...); got != tc.want {
			t.Errorf("Path(%q) = %q, want %q", tc.parts, got, tc.want)
		}
	}
}

func TestNameParentWithName(t *testing.T) {
	for _, tc := range namePartsCases {
		if got := Name(tc.path); got != tc.name {
			t.Errorf("Name(%q) = %q, want %q", tc.path, got, tc.name)
		}
		if got := Parent(tc.path); got != tc.parent {
			t.Errorf("Parent(%q) = %q, want %q", tc.path, got, tc.parent)
		}
		if tc.withName == "" {
			continue
		}
		if got := WithName(tc.path, "n.bak"); got != tc.withName {
			t.Errorf("WithName(%q, n.bak) = %q, want %q", tc.path, got, tc.withName)
		}
	}
}

func TestExpandUser(t *testing.T) {
	for _, tc := range expandUserCases {
		t.Setenv("HOME", tc.home)
		got, err := ExpandUser(tc.path)
		if tc.wantErr {
			if !errors.Is(err, ErrNoHome) {
				t.Errorf("HOME=%q ExpandUser(%q) = %q, %v, want ErrNoHome", tc.home, tc.path, got, err)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Errorf("HOME=%q ExpandUser(%q) = %q, %v, want %q", tc.home, tc.path, got, err, tc.want)
		}
	}
}

func TestHome(t *testing.T) {
	t.Setenv("HOME", "/home/u/")
	if got, err := Home(); err != nil || got != "/home/u" {
		t.Errorf("Home() = %q, %v", got, err)
	}
}

// resolveLayout builds the tree the recorded cases were resolved in; "@" in a case is the resolved temporary root.
func resolveLayout(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(root+"/real", 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(root+"/real/file", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	for link, target := range map[string]string{
		"link":     "real",
		"abs":      root + "/real",
		"loop1":    "loop2",
		"loop2":    "loop1",
		"dangling": "missing/target",
		"chain":    "link/file",
		"uplink":   "real/..",
	} {
		if err := os.Symlink(target, root+"/"+link); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestResolve(t *testing.T) {
	root := resolveLayout(t)
	t.Chdir(root)
	for _, tc := range resolveCases {
		want := root + strings.TrimPrefix(tc.want, "@")
		if got := Resolve(strings.ReplaceAll(tc.path, "@", root)); got != want {
			t.Errorf("Resolve(%q) = %q, want %q", tc.path, got, want)
		}
	}
}
