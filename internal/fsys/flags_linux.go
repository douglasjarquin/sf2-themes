package fsys

import (
	"errors"
	"os"
	"syscall"
	"unsafe"
)

// Inode flags from <linux/fs.h>: FS_IOC_GETFLAGS reads them, and these two forbid replacing or unlinking a file.
const (
	getFlags      = 0x80086601
	immutableFlag = 0x00000010
	appendFlag    = 0x00000020
)

// protectedFile reports whether path carries an inode flag that makes rename refuse to replace it. A filesystem that
// keeps no such flags has none set.
func protectedFile(path string) (bool, error) {
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return false, nil
		}
		return false, err
	}
	defer func() { _ = file.Close() }()
	var flags uint32
	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), getFlags, uintptr(unsafe.Pointer(&flags))); errno != 0 {
		if errno == syscall.ENOTTY || errno == syscall.EOPNOTSUPP || errno == syscall.EINVAL {
			return false, nil
		}
		return false, errno
	}
	return flags&(immutableFlag|appendFlag) != 0, nil
}
