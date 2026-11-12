package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func run3(args ...string) (code int, stdout, stderr string) {
	var out, errOut bytes.Buffer
	code = run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

// The test runs in cmd/sf2-dev, so this also covers resolving the default paths from below the module root.
func TestWebDataCheckDefaultsToModuleRoot(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	if root, err = filepath.EvalSymlinks(root); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := run3("web-data", "--check")
	if code != 0 || stderr != "" {
		t.Fatalf("exit %d, stderr %q", code, stderr)
	}
	if want := "verified generated theme data: " + root + "/web/src/data/generated-theme-data.json\n"; stdout != want {
		t.Fatalf("stdout %q, want %q", stdout, want)
	}
}

func TestUsageErrorsExitTwo(t *testing.T) {
	// The error text after the program name is recorded from the Python generator's argparse.
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"web-data", "--bogus"}, "error: unrecognized arguments: --bogus\n"},
		{[]string{"web-data", "extra"}, "error: unrecognized arguments: extra\n"},
		{[]string{"web-data", "--source"}, "error: argument --source: expected one argument\n"},
	}
	for _, c := range cases {
		code, stdout, stderr := run3(c.args...)
		if code != 2 || stdout != "" {
			t.Fatalf("%v: exit %d, stdout %q", c.args, code, stdout)
		}
		if !strings.HasPrefix(stderr, "usage: sf2-dev web-data ") || !strings.HasSuffix(stderr, "sf2-dev web-data: "+c.want) {
			t.Fatalf("%v: stderr %q", c.args, stderr)
		}
	}
	if code, _, _ := run3("nope"); code != 2 {
		t.Fatalf("unknown command exit %d, want 2", code)
	}
}
