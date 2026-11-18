package repometa

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

const requireBuildx = "scripts/ci/require-buildx.sh"

// fakeDocker is a docker CLI stub. It appends each call to $FAKE_DOCKER_TRACE, answers
// `image inspect` from $FAKE_DOCKER_LOCAL and `pull` from $FAKE_DOCKER_REMOTE (space-separated
// image tags), and prints $FAKE_DOCKER_STDERR to stderr and $FAKE_DOCKER_VERSION's contents to
// stdout for `buildx version`.
const fakeDocker = `#!/bin/sh
printf '%s\n' "$*" >> "$FAKE_DOCKER_TRACE"
has() { case " $1 " in *" $2 "*) return 0 ;; esac; return 1; }
case "$1 $2" in
  "image inspect") has "$FAKE_DOCKER_LOCAL" "$3" ;;
  "pull "*) has "$FAKE_DOCKER_REMOTE" "$2" ;;
  "buildx version")
    [ -z "$FAKE_DOCKER_STDERR" ] || printf '%s\n' "$FAKE_DOCKER_STDERR" >&2
    cat "$FAKE_DOCKER_VERSION"
    ;;
  "buildx build") exit 0 ;;
  *) echo "fake docker: unexpected call: $*" >&2; exit 64 ;;
esac
`

type dockerRun struct {
	stdout, stderr string
	trace          []string
	err            error
}

// fakeImages lists the image tags the docker stub has locally and can pull.
type fakeImages struct{ local, remote []string }

// runWithFakeDocker runs a repository script with the docker stub first on PATH.
func runWithFakeDocker(t *testing.T, script, version, stderr string, images fakeImages) dockerRun {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "docker"), []byte(fakeDocker), 0o755); err != nil {
		t.Fatal(err)
	}
	versionFile := filepath.Join(dir, "version")
	if err := os.WriteFile(versionFile, []byte(version), 0o644); err != nil {
		t.Fatal(err)
	}
	tracePath := filepath.Join(dir, "trace")
	path, err := filepath.Abs(filepath.Join(root, script))
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command("bash", path)
	command.Env = []string{
		"PATH=" + dir + string(os.PathListSeparator) + os.Getenv("PATH"),
		"FAKE_DOCKER_TRACE=" + tracePath,
		"FAKE_DOCKER_VERSION=" + versionFile,
		"FAKE_DOCKER_STDERR=" + stderr,
		"FAKE_DOCKER_LOCAL=" + strings.Join(images.local, " "),
		"FAKE_DOCKER_REMOTE=" + strings.Join(images.remote, " "),
	}
	var out, errOut strings.Builder
	command.Stdout, command.Stderr = &out, &errOut
	run := dockerRun{err: command.Run()}
	var exitErr *exec.ExitError
	if run.err != nil && !errors.As(run.err, &exitErr) {
		t.Fatalf("%s: %v", script, run.err)
	}
	run.stdout, run.stderr = out.String(), errOut.String()
	if data, err := os.ReadFile(tracePath); err == nil {
		run.trace = strings.Split(strings.TrimSpace(string(data)), "\n")
	}
	return run
}

// verbs reduces a docker trace to its subcommands, e.g. "image inspect" or "buildx build".
func verbs(trace []string) []string {
	out := make([]string, 0, len(trace))
	for _, call := range trace {
		fields := strings.Fields(call)
		switch {
		case len(fields) >= 2 && (fields[0] == "image" || fields[0] == "buildx"):
			out = append(out, fields[0]+" "+fields[1])
		case len(fields) >= 1:
			out = append(out, fields[0])
		}
	}
	return out
}

// imageTag runs the fingerprint script the ensure scripts use to name an image.
func imageTag(t *testing.T, target string) string {
	t.Helper()
	command := exec.Command("bash", "scripts/ci/dev-image-fingerprint.sh", "--target", target)
	command.Dir = root
	out, err := command.Output()
	if err != nil {
		t.Fatalf("fingerprint %s: %v", target, err)
	}
	return "ghcr.io/douglasjarquin/sf2-themes-" + target + ":sha-" + strings.TrimSpace(string(out))
}

func minBuildxVersion(t *testing.T) string {
	t.Helper()
	m := regexp.MustCompile(`(?m)^MIN_BUILDX_VERSION="([0-9]+\.[0-9]+\.[0-9]+)"$`).FindStringSubmatch(readFile(t, requireBuildx))
	if m == nil {
		t.Fatalf("%s sets no MIN_BUILDX_VERSION", requireBuildx)
	}
	return m[1]
}

func TestRequireBuildxEnforcesTheMinimum(t *testing.T) {
	if got := minBuildxVersion(t); got != "0.38.1" {
		t.Fatalf("MIN_BUILDX_VERSION = %s, want 0.38.1", got)
	}
	record := func(v string) string { return "github.com/docker/buildx " + v + " 0123456789abcdef" }
	for _, tc := range []struct {
		name, version, stderr string
		pass                  bool
	}{
		{"the minimum", record("v0.38.1"), "", true},
		{"the current release", record("v0.39.0"), "", true},
		{"a later major", record("v1.0.0"), "", true},
		{"a two-digit patch", record("v0.38.10"), "", true},
		{"a later minor with a smaller patch", record("v0.39.0"), "", true},
		{"a leading zero", record("v0.038.001"), "", true},
		{"a component past 64 bits", record("v0.99999999999999999999999.0"), "", true},
		{"no v prefix", record("0.40.0"), "", true},
		{"build metadata on the minimum", record("v0.38.1+ds1"), "", true},
		{"a Docker Desktop build of the minimum", record("v0.38.1-desktop.1"), "", true},
		{"a prerelease of a later release", record("v0.39.0-rc.1"), "", true},
		{"a Docker warning on stderr", record("v0.39.0"), "WARNING: Error loading config file: /root/.docker/config.json", true},
		{"one patch below", record("v0.38.0"), "", false},
		{"build metadata below the minimum", record("v0.38.0+ds1"), "", false},
		{"the previous minor", record("v0.37.2"), "", false},
		{"a patch that overflows into the minor", record("v0.37.1001"), "", false},
		{"a minor that overflows into the major", record("v0.9.999999"), "", false},
		{"a prerelease of the minimum", record("v0.38.1-rc.1"), "", false},
		{"a Desktop prerelease of the minimum", record("v0.38.1-desktop.1.rc1"), "", false},
		{"not a version", record("dev"), "", false},
		{"no buildx record", "docker: 'buildx' is not a docker command.", "", false},
		{"the version only on stderr", "", record("v0.39.0"), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run := runWithFakeDocker(t, requireBuildx, tc.version+"\n", tc.stderr, fakeImages{})
			if passed := run.err == nil; passed != tc.pass {
				t.Fatalf("passed = %v, want %v\nstderr: %s", passed, tc.pass, run.stderr)
			}
			// Callers print the image tag on stdout, so the gate writes only to stderr.
			if run.stdout != "" {
				t.Errorf("stdout = %q, want empty", run.stdout)
			}
			if tc.stderr != "" && !strings.Contains(run.stderr, tc.stderr) {
				t.Errorf("stderr %q dropped Docker's diagnostic", run.stderr)
			}
			if strings.HasPrefix(tc.version, "github.com/docker/buildx ") &&
				!strings.Contains(run.stderr, "[require-buildx] "+tc.version+"\n") {
				t.Errorf("stderr %q does not log the buildx version record", run.stderr)
			}
		})
	}
}

func TestImageScriptsCheckBuildxOnlyBeforeAFallbackBuild(t *testing.T) {
	toolchain, dev := imageTag(t, "toolchain"), imageTag(t, "dev")
	const (
		ensureToolchain = "scripts/ci/ensure-toolchain-image.sh"
		ensureDev       = "scripts/ci/ensure-dev-image.sh"
		current         = "github.com/docker/buildx v0.39.0 360040a78d6113dc84e5d4dcdf0f438e1be7a41f"
		old             = "github.com/docker/buildx v0.37.2 2d379c0c3f22da0d2759d132a0ec81ca949098f0"
	)
	for _, tc := range []struct {
		name, script, version string
		images                fakeImages
		calls                 []string
		pass                  bool
	}{
		{"toolchain cached", ensureToolchain, old, fakeImages{local: []string{toolchain}},
			[]string{"image inspect"}, true},
		{"toolchain pulled", ensureToolchain, old, fakeImages{remote: []string{toolchain}},
			[]string{"image inspect", "pull"}, true},
		{"toolchain built", ensureToolchain, current, fakeImages{},
			[]string{"image inspect", "pull", "buildx version", "buildx build"}, true},
		{"toolchain build refused", ensureToolchain, old, fakeImages{},
			[]string{"image inspect", "pull", "buildx version"}, false},
		{"dev cached", ensureDev, old, fakeImages{local: []string{dev}},
			[]string{"image inspect"}, true},
		{"dev pulled", ensureDev, old, fakeImages{remote: []string{dev}},
			[]string{"image inspect", "pull"}, true},
		{"dev built on a cached toolchain", ensureDev, current, fakeImages{local: []string{toolchain}},
			[]string{"image inspect", "pull", "image inspect", "buildx version", "buildx build"}, true},
		{"dev and toolchain built", ensureDev, current, fakeImages{},
			[]string{"image inspect", "pull", "image inspect", "pull", "buildx version", "buildx build", "buildx version", "buildx build"}, true},
		{"dev build refused", ensureDev, old, fakeImages{remote: []string{toolchain}},
			[]string{"image inspect", "pull", "image inspect", "pull", "buildx version"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run := runWithFakeDocker(t, tc.script, tc.version+"\n", "", tc.images)
			if got := verbs(run.trace); !slices.Equal(got, tc.calls) {
				t.Fatalf("docker calls = %v, want %v\nstderr: %s", got, tc.calls, run.stderr)
			}
			if passed := run.err == nil; passed != tc.pass {
				t.Fatalf("passed = %v, want %v\nstderr: %s", passed, tc.pass, run.stderr)
			}
			want := toolchain
			if tc.script == ensureDev {
				want = dev
			}
			if tc.pass && run.stdout != want+"\n" {
				t.Errorf("stdout = %q, want the image tag %q", run.stdout, want)
			}
			if slices.Contains(tc.calls, "buildx version") && !strings.Contains(run.stderr, "[require-buildx] "+tc.version+"\n") {
				t.Errorf("stderr %q does not log the buildx version", run.stderr)
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
			}
			// The pin must itself clear the gate.
			if run := runWithFakeDocker(t, requireBuildx, "github.com/docker/buildx "+pinned+" 0\n", "", fakeImages{}); run.err != nil {
				t.Errorf("setup-buildx-action version %s fails %s: %s", pinned, requireBuildx, run.stderr)
			}
		case s.Run == requireBuildx:
			check = i
		case strings.HasPrefix(s.Uses, "docker/build-push-action@") && firstBuild < 0:
			firstBuild = i
		}
		if (setup == i || check == i) && (s.If != "" || s.ContinueOnError) {
			t.Errorf("step %d (%s%s) must run unconditionally and fail the job", i, s.Uses, s.Run)
		}
	}
	if setup < 0 || check < 0 || firstBuild < 0 {
		t.Fatalf("setup-buildx = %d, %s = %d, first build-push = %d; want all present", setup, requireBuildx, check, firstBuild)
	}
	if setup >= check || check >= firstBuild {
		t.Errorf("want setup-buildx (%d) < %s (%d) < first build-push (%d)", setup, requireBuildx, check, firstBuild)
	}
}
