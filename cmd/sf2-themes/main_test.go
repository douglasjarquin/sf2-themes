package main

import (
	"os/exec"
	"path/filepath"
	"strings"
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

func TestBinaryCarriesTheCatalog(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "sf2-themes")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	command := exec.Command(bin, "themes")
	command.Dir = t.TempDir()
	out, err := command.Output()
	if err != nil {
		t.Fatalf("themes: %v", err)
	}
	if lines := strings.Count(string(out), "\n"); lines != 36 {
		t.Errorf("themes printed %d lines from a directory with no catalog, want 36", lines)
	}
}
