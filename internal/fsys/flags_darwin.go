package fsys

import (
	"os"
	"syscall"
)

// File flags from <sys/stat.h> that forbid replacing or unlinking a file.
const (
	userImmutable   = 0x00000002
	userAppend      = 0x00000004
	systemImmutable = 0x00020000
	systemAppend    = 0x00040000
)

// protectedFile reports whether path carries a flag that makes rename refuse to replace it.
func protectedFile(path string) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Flags&(userImmutable|userAppend|systemImmutable|systemAppend) != 0, nil
}
