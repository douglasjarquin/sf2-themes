package cli

import (
	"syscall"
	"testing"
)

// userImmutable is UF_IMMUTABLE, the flag `chflags uchg` sets.
const userImmutable = 0x2

// setImmutable marks path immutable for one test, as `chflags uchg` does, and clears the flag after it.
func setImmutable(t *testing.T, path string) {
	t.Helper()
	if err := syscall.Chflags(path, userImmutable); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = syscall.Chflags(path, 0) })
}
