package repometa

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

type cursorEnvironment struct {
	Build struct {
		Dockerfile string `json:"dockerfile"`
		Context    string `json:"context"`
	} `json:"build"`
	User      string `json:"user"`
	Install   string `json:"install"`
	Terminals []struct {
		Name    string `json:"name"`
		Command string `json:"command"`
	} `json:"terminals"`
}

func loadCursorEnvironment(t *testing.T) cursorEnvironment {
	t.Helper()
	var env cursorEnvironment
	if err := json.Unmarshal([]byte(readFile(t, ".cursor/environment.json")), &env); err != nil {
		t.Fatal(err)
	}
	return env
}

// instructions returns the Dockerfile's lines that are neither blank nor comments.
func instructions(t *testing.T, rel string) []string {
	t.Helper()
	var lines []string
	for _, line := range strings.Split(readFile(t, rel), "\n") {
		if line != "" && !strings.HasPrefix(line, "#") {
			lines = append(lines, line)
		}
	}
	return lines
}

func lineIndex(t *testing.T, lines []string, match func(string) bool, what string) int {
	t.Helper()
	i := slices.IndexFunc(lines, match)
	if i < 0 {
		t.Fatalf("no line %s", what)
	}
	return i
}

func TestCursorEnvironmentPointsAtTheDevDockerfile(t *testing.T) {
	env := loadCursorEnvironment(t)
	if env.Build.Context != ".." {
		t.Errorf("build context = %q, want ..", env.Build.Context)
	}
	dockerfile := filepath.Clean(filepath.Join(root, ".cursor", env.Build.Dockerfile))
	if want := filepath.Clean(filepath.Join(root, "docker", "dev", "Dockerfile")); dockerfile != want {
		t.Errorf("dockerfile = %s, want %s", dockerfile, want)
	}
	if _, err := os.Stat(dockerfile); err != nil {
		t.Error(err)
	}
}

func TestCursorEnvironmentInstallRefreshesLockedToolsAndDeps(t *testing.T) {
	env := loadCursorEnvironment(t)
	if env.User != "dev" {
		t.Errorf("user = %q, want dev", env.User)
	}
	if want := "mise install --locked && mise run deps"; env.Install != want {
		t.Errorf("install = %q, want %q", env.Install, want)
	}
	if len(env.Terminals) == 0 || env.Terminals[0].Name != "web" || env.Terminals[0].Command != "mise run web:dev:container" {
		t.Errorf("terminals = %+v, want the web terminal running mise run web:dev:container first", env.Terminals)
	}
}

func TestDevDockerfileDefaultsToTheSameUbuntuPinAsTheToolchain(t *testing.T) {
	var ubuntu string
	for _, line := range strings.Split(readFile(t, "docker/toolchain/Dockerfile"), "\n") {
		if value, ok := strings.CutPrefix(line, "ARG UBUNTU_IMAGE="); ok {
			ubuntu = value
		}
	}
	if ubuntu == "" {
		t.Fatal("the toolchain Dockerfile has no ARG UBUNTU_IMAGE")
	}
	dev := readFile(t, "docker/dev/Dockerfile")
	if !strings.Contains(dev, "ARG TOOLCHAIN_IMAGE="+ubuntu) {
		t.Errorf("the dev Dockerfile does not default TOOLCHAIN_IMAGE to %s", ubuntu)
	}
	if strings.Contains(dev, "ARG TOOLCHAIN_IMAGE=sf2-themes-toolchain") {
		t.Error("the dev Dockerfile defaults to a host-local toolchain image")
	}
	if !strings.Contains(dev, "COPY docker/toolchain/apt-packages.txt") {
		t.Error("the dev Dockerfile does not install the toolchain apt packages")
	}
}

// aube's xx::fslock writes /tmp/fslock/<hash>. Root mise install leaves that directory 0755,
// so USER dev cannot create lockfiles during the image build.
func TestDevDockerfileMakesFslockWritableBeforeNonrootAube(t *testing.T) {
	lines := instructions(t, "docker/dev/Dockerfile")
	lineIndex(t, lines, func(l string) bool { return strings.Contains(l, "mkdir -p /tmp/fslock") }, "creating /tmp/fslock")
	chmod := lineIndex(t, lines, func(l string) bool { return strings.Contains(l, "chmod 1777 /tmp/fslock") }, "opening /tmp/fslock")
	userDev := lineIndex(t, lines, func(l string) bool { return l == "USER dev" }, "switching to USER dev")
	aubeCI := lineIndex(t, lines, func(l string) bool { return strings.Contains(l, "aube ci") }, "running aube ci")
	if chmod >= userDev || userDev >= aubeCI {
		t.Errorf("chmod at %d, USER dev at %d, aube ci at %d: want them in that order", chmod, userDev, aubeCI)
	}
	if n := strings.Count(readFile(t, "scripts/ci/run-in-dev-container.sh"), "chmod 1777 /tmp/fslock"); n != 1 {
		t.Errorf("run-in-dev-container.sh opens /tmp/fslock %d times, want 1", n)
	}
}

// Cursor Cloud's install step runs `mise install --locked` as dev, and mise rebuilds its shims on every install, so
// dev must own the shims directory even though root installed the tools.
func TestDevDockerfileGivesDevTheMiseShimsBeforeNonrootInstall(t *testing.T) {
	lines := instructions(t, "docker/dev/Dockerfile")
	chown := lineIndex(t, lines, func(l string) bool {
		return strings.Contains(l, "chown -R dev:dev") && strings.Contains(l, `"$MISE_DATA_DIR/shims"`)
	}, "giving dev the mise shims directory")
	install := lineIndex(t, lines, func(l string) bool { return strings.Contains(l, "mise install --locked") }, "running mise install")
	if chown <= install {
		t.Errorf("shims chown at %d, root mise install at %d: want the chown after the root install", chown, install)
	}
	if env := loadCursorEnvironment(t); env.User != "dev" || !strings.HasPrefix(env.Install, "mise install") {
		t.Errorf("Cursor runs %q as %q; this test assumes mise install as dev", env.Install, env.User)
	}
}

func TestDepsTaskDownloadsGoModulesAndInstallsAubeAndChromium(t *testing.T) {
	var mise struct {
		Tools map[string]any `toml:"tools"`
		Tasks map[string]struct {
			Run string `toml:"run"`
		} `toml:"tasks"`
	}
	if _, err := toml.Decode(readFile(t, "mise.toml"), &mise); err != nil {
		t.Fatal(err)
	}
	run := mise.Tasks["deps"].Run
	for _, want := range []string{"go mod download", "aube -C web install", "playwright-core install chromium"} {
		if !strings.Contains(run, want) {
			t.Errorf("deps runs %q, which misses %q", run, want)
		}
	}
	for _, tool := range []string{"python", "uv", "ruff"} {
		if _, ok := mise.Tools[tool]; ok {
			t.Errorf("mise.toml still pins %s", tool)
		}
	}
	for name, task := range mise.Tasks {
		for _, banned := range []string{"python", "uv ", "pytest", "ruff"} {
			if strings.Contains(task.Run, banned) {
				t.Errorf("task %s runs %q: %s", name, banned, task.Run)
			}
		}
	}
}

func TestDevImagePrefetchesGoModulesAndNoPython(t *testing.T) {
	dev := readFile(t, "docker/dev/Dockerfile")
	for _, want := range []string{"COPY --chown=dev:dev go.mod go.sum ./", "go mod download"} {
		if !strings.Contains(dev, want) {
			t.Errorf("the dev Dockerfile misses %q", want)
		}
	}
	for _, rel := range []string{
		"docker/dev/Dockerfile", "docker/dev/entrypoint.sh", "docker/toolchain/apt-packages.txt",
		"docker/dev/apt-packages.txt", "scripts/ci/run-in-dev-container.sh", "scripts/ci/dev-image-fingerprint.sh",
	} {
		text := readFile(t, rel)
		for _, banned := range []string{"python", "pyproject", "uv.lock", "uv sync", ".venv"} {
			if strings.Contains(text, banned) {
				t.Errorf("%s still names %q", rel, banned)
			}
		}
	}
}

// The dev image bakes the Go module cache, so its tag must change when go.mod or go.sum does.
func TestDevImageFingerprintChangesWithGoModules(t *testing.T) {
	inputs := []string{
		"mise.toml", "mise.lock", "docker/toolchain/Dockerfile", "docker/toolchain/apt-packages.txt",
		"go.mod", "go.sum", "web/package.json", "web/aube-lock.yaml", "web/.npmrc",
		"docker/dev/Dockerfile", "docker/dev/apt-packages.txt", "docker/dev/entrypoint.sh",
		"scripts/ci/dev-image-fingerprint.sh",
	}
	repo := t.TempDir()
	for _, rel := range inputs {
		path := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(readFile(t, rel)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	fingerprint := func(target string) string {
		t.Helper()
		out, err := exec.Command("bash", filepath.Join(repo, "scripts/ci/dev-image-fingerprint.sh"), "--target", target).Output()
		if err != nil {
			t.Fatalf("fingerprint --target %s: %v", target, err)
		}
		return strings.TrimSpace(string(out))
	}
	devBefore, toolchainBefore := fingerprint("dev"), fingerprint("toolchain")
	for _, rel := range []string{"go.mod", "go.sum"} {
		path := filepath.Join(repo, rel)
		original, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(slices.Clone(original), "// changed\n"...), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := fingerprint("dev"); got == devBefore {
			t.Errorf("the dev fingerprint %s did not change when %s changed", got, rel)
		}
		if got := fingerprint("toolchain"); got != toolchainBefore {
			t.Errorf("the toolchain fingerprint changed from %s to %s when %s changed", toolchainBefore, got, rel)
		}
		if err := os.WriteFile(path, original, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if got := fingerprint("dev"); got != devBefore {
		t.Errorf("the dev fingerprint is %s after restoring go.mod and go.sum, want %s", got, devBefore)
	}
}
