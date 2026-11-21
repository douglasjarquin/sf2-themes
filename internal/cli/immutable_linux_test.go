package cli

import (
	"os"
	"syscall"
	"testing"
	"unsafe"
)

// The FS_IOC_SETFLAGS ioctl and FS_IMMUTABLE_FL, which `chattr +i` sets.
const (
	setFlags      = 0x40086602
	immutableFlag = 0x10
)

// setImmutable marks path immutable for one test, as `chattr +i` does, and clears the flag after it. Only a process
// with CAP_LINUX_IMMUTABLE may, so the test is skipped without it.
func setImmutable(t *testing.T, path string) {
	t.Helper()
	if err := ioctlFlags(path, immutableFlag); err != nil {
		t.Skipf("cannot mark %s immutable: %v", path, err)
	}
	t.Cleanup(func() { _ = ioctlFlags(path, 0) })
}

func ioctlFlags(path string, flags uint32) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	value := uintptr(flags)
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), setFlags, uintptr(unsafe.Pointer(&value))); errno != 0 {
		return errno
	}
	return nil
}
