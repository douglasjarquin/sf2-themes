package parity

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// installScript is the installer at the repository root, run with /bin/sh as users run it.
var installScript = filepath.Join("..", "..", "install.sh")

// fakeRelease writes a release directory whose binaries are shell scripts printing version,
// with a checksums.txt over them. corrupt names an asset whose listed checksum is wrong.
func fakeRelease(t *testing.T, version, corrupt string) string {
	t.Helper()
	dir := t.TempDir()
	var sums strings.Builder
	for _, target := range []string{"darwin_arm64", "darwin_amd64", "linux_arm64", "linux_amd64"} {
		name := "sf2-themes_" + target
		body := []byte("#!/bin/sh\necho " + version + "\n")
		if err := os.WriteFile(filepath.Join(dir, name), body, 0o644); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(body)
		if name == corrupt {
			sum = sha256.Sum256([]byte("something else"))
		}
		fmt.Fprintf(&sums, "%s  %s\n", hex.EncodeToString(sum[:]), name)
	}
	if err := os.WriteFile(filepath.Join(dir, "checksums.txt"), []byte(sums.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

type installRun struct {
	home, bin string
	stdout    string
	stderr    string
	err       error
}

// runInstaller runs install.sh with a fresh HOME. path, when non-empty, replaces PATH.
func runInstaller(t *testing.T, home string, env map[string]string, path string) installRun {
	t.Helper()
	cmd := exec.Command("/bin/sh", installScript)
	cmd.Env = []string{"HOME=" + home}
	if path == "" {
		path = os.Getenv("PATH")
	}
	cmd.Env = append(cmd.Env, "PATH="+path)
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	var stdout, stderr strings.Builder
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	return installRun{home: home, bin: filepath.Join(home, ".local", "bin"), stdout: stdout.String(), stderr: stderr.String(), err: err}
}

func hostAsset() string {
	return "sf2-themes_" + runtime.GOOS + "_" + runtime.GOARCH
}

func TestInstallerInstallsAndLinksSf2(t *testing.T) {
	release := fakeRelease(t, "9.9.9", "")
	run := runInstaller(t, t.TempDir(), map[string]string{"SF2_THEMES_BASE_URL": release}, "")
	if run.err != nil {
		t.Fatalf("install failed: %v\n%s", run.err, run.stderr)
	}
	target, err := os.Readlink(filepath.Join(run.bin, "sf2"))
	if err != nil || target != "sf2-themes" {
		t.Fatalf("sf2 link = %q, %v; want sf2-themes", target, err)
	}
	out, err := exec.Command(filepath.Join(run.bin, "sf2")).Output()
	if err != nil || string(out) != "9.9.9\n" {
		t.Fatalf("sf2 = %q, %v", out, err)
	}
	info, err := os.Stat(filepath.Join(run.bin, "sf2-themes"))
	if err != nil || info.Mode().Perm() != 0o755 {
		t.Fatalf("sf2-themes mode = %v, %v; want 0755", info.Mode().Perm(), err)
	}
	if want := run.bin + " is not on PATH; add it with:\n  export PATH=\"" + run.bin + ":$PATH\"\n"; !strings.HasSuffix(run.stdout, want) {
		t.Fatalf("stdout = %q, want suffix %q", run.stdout, want)
	}
}

func TestInstallerSaysNothingAboutPathWhenTheDirectoryIsOnIt(t *testing.T) {
	home := t.TempDir()
	bin := filepath.Join(home, ".local", "bin")
	run := runInstaller(t, home, map[string]string{"SF2_THEMES_BASE_URL": fakeRelease(t, "9.9.9", "")}, bin+":"+os.Getenv("PATH"))
	if run.err != nil || strings.Contains(run.stdout, "not on PATH") {
		t.Fatalf("err = %v, stdout = %q", run.err, run.stdout)
	}
}

func TestInstallerReplacesItsOwnEarlierInstall(t *testing.T) {
	home := t.TempDir()
	if run := runInstaller(t, home, map[string]string{"SF2_THEMES_BASE_URL": fakeRelease(t, "1.0.0", "")}, ""); run.err != nil {
		t.Fatalf("first install: %v\n%s", run.err, run.stderr)
	}
	run := runInstaller(t, home, map[string]string{"SF2_THEMES_BASE_URL": fakeRelease(t, "2.0.0", "")}, "")
	if run.err != nil {
		t.Fatalf("second install: %v\n%s", run.err, run.stderr)
	}
	out, _ := exec.Command(filepath.Join(run.bin, "sf2")).Output()
	if string(out) != "2.0.0\n" {
		t.Fatalf("after reinstall sf2 = %q, want 2.0.0", out)
	}
}

func TestInstallerRefusesABinaryWhoseChecksumDoesNotMatch(t *testing.T) {
	run := runInstaller(t, t.TempDir(), map[string]string{"SF2_THEMES_BASE_URL": fakeRelease(t, "9.9.9", hostAsset())}, "")
	if run.err == nil || !strings.Contains(run.stderr, "checksum mismatch for "+hostAsset()) {
		t.Fatalf("err = %v, stderr = %q", run.err, run.stderr)
	}
	if _, err := os.Lstat(run.bin); !os.IsNotExist(err) {
		t.Fatalf("install dir exists after a failed checksum: %v", err)
	}
}

func TestInstallerStopsAtASymlinkFromAnOldInstall(t *testing.T) {
	home := t.TempDir()
	bin := filepath.Join(home, ".local", "bin")
	uvTarget := filepath.Join(home, ".local", "share", "uv", "tools", "sf2-themes", "bin", "sf2-themes")
	if err := os.MkdirAll(filepath.Dir(uvTarget), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(uvTarget, []byte("#!/bin/sh\necho 1.0.1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(uvTarget, filepath.Join(bin, "sf2-themes")); err != nil {
		t.Fatal(err)
	}
	run := runInstaller(t, home, map[string]string{"SF2_THEMES_BASE_URL": fakeRelease(t, "9.9.9", "")}, "")
	if run.err == nil || !strings.Contains(run.stderr, "uv tool uninstall sf2-themes") {
		t.Fatalf("err = %v, stderr = %q", run.err, run.stderr)
	}
	body, _ := os.ReadFile(uvTarget)
	entries, _ := os.ReadDir(bin)
	if string(body) != "#!/bin/sh\necho 1.0.1\n" || len(entries) != 1 {
		t.Fatalf("the old install changed: target %q, bin has %d entries", body, len(entries))
	}
}

// shimPath returns a PATH holding links to the named host tools plus any fake scripts.
func shimPath(t *testing.T, tools []string, fakes map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for _, tool := range tools {
		if real, err := exec.LookPath(tool); err == nil {
			if err := os.Symlink(real, filepath.Join(dir, tool)); err != nil {
				t.Fatal(err)
			}
		}
	}
	for name, body := range fakes {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

var installerTools = []string{"uname", "mkdir", "mktemp", "cp", "mv", "chmod", "ln", "rm", "awk", "cut", "readlink", "sha256sum", "shasum", "cat"}

func TestInstallerNamesAnUnsupportedPlatform(t *testing.T) {
	for _, tc := range []struct{ fake, want string }{
		{`[ "$1" = -s ] && echo FreeBSD || echo amd64`, "unsupported operating system: FreeBSD"},
		{`[ "$1" = -s ] && echo Linux || echo riscv64`, "unsupported architecture: riscv64"},
	} {
		path := shimPath(t, installerTools[1:], map[string]string{"uname": tc.fake})
		run := runInstaller(t, t.TempDir(), map[string]string{"SF2_THEMES_BASE_URL": fakeRelease(t, "9.9.9", "")}, path)
		if run.err == nil || !strings.Contains(run.stderr, tc.want) {
			t.Errorf("err = %v, stderr = %q, want %q", run.err, run.stderr, tc.want)
		}
	}
}

func TestInstallerNeedsCurlOrWgetForARemoteRelease(t *testing.T) {
	run := runInstaller(t, t.TempDir(), map[string]string{"SF2_THEMES_BASE_URL": "https://example.invalid/release"}, shimPath(t, installerTools, nil))
	if run.err == nil || !strings.Contains(run.stderr, "curl or wget is required") {
		t.Fatalf("err = %v, stderr = %q", run.err, run.stderr)
	}
}
