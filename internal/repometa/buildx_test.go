package repometa

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const requireBuildx = "scripts/ci/require-buildx.sh"

// minBuildxVersion reads the floor that require-buildx.sh enforces.
func minBuildxVersion(t *testing.T) string {
	t.Helper()
	m := regexp.MustCompile(`(?m)^MIN_BUILDX_VERSION="([0-9]+\.[0-9]+\.[0-9]+)"$`).FindStringSubmatch(readFile(t, requireBuildx))
	if m == nil {
		t.Fatalf("%s sets no MIN_BUILDX_VERSION", requireBuildx)
	}
	return m[1]
}

// versionAtLeast reports whether the dotted version have is at least want.
func versionAtLeast(t *testing.T, have, want string) bool {
	t.Helper()
	parse := func(v string) [3]int {
		var out [3]int
		parts := strings.Split(strings.TrimPrefix(v, "v"), ".")
		if len(parts) != 3 {
			t.Fatalf("version %q is not major.minor.patch", v)
		}
		for i, p := range parts {
			n, err := strconv.Atoi(p)
			if err != nil {
				t.Fatalf("version %q: %v", v, err)
			}
			out[i] = n
		}
		return out
	}
	h, w := parse(have), parse(want)
	for i := range h {
		if h[i] != w[i] {
			return h[i] > w[i]
		}
	}
	return true
}

func TestRequireBuildxEnforcesTheMinimum(t *testing.T) {
	if minBuildxVersion(t) != "0.38.1" {
		t.Fatalf("MIN_BUILDX_VERSION = %s, want 0.38.1", minBuildxVersion(t))
	}
	script, err := filepath.Abs(filepath.Join(root, requireBuildx))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		output string
		status int
		pass   bool
	}{
		{"github.com/docker/buildx v0.38.1 0123456789abcdef", 0, true},
		{"github.com/docker/buildx v0.39.0 360040a78d6113dc84e5d4dcdf0f438e1be7a41f", 0, true},
		{"github.com/docker/buildx v1.0.0 0123456789abcdef", 0, true},
		{"github.com/docker/buildx v0.38.10 0123456789abcdef", 0, true},
		{"github.com/docker/buildx v0.38.1-desktop.1 0123456789abcdef", 0, true},
		{"github.com/docker/buildx 0.40.0+ds1 0ubuntu1", 0, true},
		{"github.com/docker/buildx v0.38.0 0123456789abcdef", 0, false},
		{"github.com/docker/buildx v0.37.2 2d379c0c3f22da0d2759d132a0ec81ca949098f0", 0, false},
		{"github.com/docker/buildx v0.9.99 0123456789abcdef", 0, false},
		{"github.com/docker/buildx dev 0123456789abcdef", 0, false},
		{"docker: 'buildx' is not a docker command.", 1, false},
	} {
		t.Run(tc.output, func(t *testing.T) {
			bin := t.TempDir()
			if err := os.WriteFile(filepath.Join(bin, "output"), []byte(tc.output+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			fake := "#!/bin/sh\n" +
				`[ "$1 $2" = "buildx version" ] || { echo "unexpected: $*" >&2; exit 64; }` + "\n" +
				`cat "$(dirname "$0")/output"` + "\nexit " + strconv.Itoa(tc.status) + "\n"
			if err := os.WriteFile(filepath.Join(bin, "docker"), []byte(fake), 0o755); err != nil {
				t.Fatal(err)
			}
			command := exec.Command("bash", script)
			command.Env = []string{"PATH=" + bin + string(os.PathListSeparator) + os.Getenv("PATH")}
			stdout, err := command.Output()
			if passed := err == nil; passed != tc.pass {
				t.Fatalf("passed = %v, want %v (err %v)", passed, tc.pass, err)
			}
			// Callers print the image tag on stdout, so the gate writes only to stderr.
			if len(stdout) != 0 {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			stderr := ""
			if exitErr, ok := err.(*exec.ExitError); ok {
				stderr = string(exitErr.Stderr)
			}
			if !tc.pass && !strings.Contains(stderr, tc.output) {
				t.Errorf("stderr %q does not show the docker buildx version output", stderr)
			}
		})
	}
}

func TestPublishImagesPinsBuildxAndChecksItBeforeBuilding(t *testing.T) {
	steps := job(t, loadWorkflow(t, "publish-images.yml"), "publish").Steps
	setup, check, firstBuild := -1, -1, -1
	for i, s := range steps {
		switch {
		case strings.HasPrefix(s.Uses, "docker/setup-buildx-action@"):
			setup = i
			pinned := s.With["version"]
			if !regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+$`).MatchString(pinned) {
				t.Errorf("setup-buildx-action version = %q, want an exact vX.Y.Z release", pinned)
			} else if !versionAtLeast(t, pinned, minBuildxVersion(t)) {
				t.Errorf("setup-buildx-action version %s is older than %s", pinned, minBuildxVersion(t))
			}
		case s.Run == requireBuildx:
			check = i
		case strings.HasPrefix(s.Uses, "docker/build-push-action@") && firstBuild < 0:
			firstBuild = i
		}
	}
	if setup < 0 || check < 0 || firstBuild < 0 {
		t.Fatalf("setup-buildx = %d, %s = %d, first build-push = %d; want all present", setup, requireBuildx, check, firstBuild)
	}
	if setup >= check || check >= firstBuild {
		t.Errorf("want setup-buildx (%d) < %s (%d) < first build-push (%d)", setup, requireBuildx, check, firstBuild)
	}
}

func TestEveryScriptedImageBuildRequiresBuildxFirst(t *testing.T) {
	for _, rel := range []string{"scripts/ci/ensure-toolchain-image.sh", "scripts/ci/ensure-dev-image.sh"} {
		text := readFile(t, rel)
		build := strings.Index(text, "docker buildx build")
		if build < 0 {
			t.Fatalf("%s no longer runs docker buildx build; update this contract", rel)
		}
		gate := strings.Index(text, "\n"+requireBuildx+"\n")
		if gate < 0 || gate > build {
			t.Errorf("%s must run %s before docker buildx build", rel, requireBuildx)
		}
	}
}
