package main

import (
	"os/exec"
	"path/filepath"
	"testing"
)

const versionVar = "github.com/douglasjarquin/sf2-themes/v2/internal/version.Version"

func buildAndRun(t *testing.T, ldflags string, args ...string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "sf2-themes")
	build := []string{"build", "-o", bin}
	if ldflags != "" {
		build = append(build, "-ldflags", ldflags)
	}
	if out, err := exec.Command("go", append(build, ".")...).CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	out, err := exec.Command(bin, args...).Output()
	if err != nil {
		t.Fatalf("run %v: %v", args, err)
	}
	return string(out)
}

func TestVersionIsDevWithoutStamp(t *testing.T) {
	if got := buildAndRun(t, "", "--version"); got != "dev\n" {
		t.Fatalf("--version = %q, want %q", got, "dev\n")
	}
}

func TestVersionPrintsStampedVersionOnly(t *testing.T) {
	if got := buildAndRun(t, "-X "+versionVar+"=2.0.0", "--version"); got != "2.0.0\n" {
		t.Fatalf("--version = %q, want %q", got, "2.0.0\n")
	}
}
