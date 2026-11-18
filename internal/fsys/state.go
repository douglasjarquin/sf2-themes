package fsys

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"

	"github.com/douglasjarquin/sf2-themes/v2/internal/pycompat"
)

// Lock takes an exclusive advisory lock on path, creating it and its directory when missing, and blocks until no
// other process holds it. The returned function releases it. A symlink or non-regular file at path is refused.
func Lock(path string) (func(), error) {
	if err := os.MkdirAll(pycompat.Parent(path), 0o777); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|syscall.O_NOFOLLOW, 0o600)
	if err != nil {
		if errors.Is(err, syscall.ELOOP) {
			return nil, fmt.Errorf("refusing to lock through symlink %s", path)
		}
		return nil, err
	}
	info, err := file.Stat()
	if err == nil && !info.Mode().IsRegular() {
		err = fmt.Errorf("%s is not a regular file", path)
	}
	if err == nil {
		for {
			err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX)
			if !errors.Is(err, syscall.EINTR) {
				break
			}
		}
	}
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	return func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
	}, nil
}

// CreateExclusive publishes data at path only when nothing is there yet, and reports fs.ErrExist otherwise. The
// file appears complete or not at all.
func CreateExclusive(path string, data []byte, mode fs.FileMode) (err error) {
	dir := pycompat.Parent(path)
	if err := os.MkdirAll(dir, 0o777); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(dir, "."+pycompat.Name(path)+".*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(temporary.Name()) }()
	if _, err = temporary.Write(data); err != nil {
		_ = temporary.Close()
		return err
	}
	if err = temporary.Close(); err != nil {
		return err
	}
	if err = os.Chmod(temporary.Name(), mode); err != nil {
		return err
	}
	// A hard link fails when path exists, which makes publishing exclusive.
	return os.Link(temporary.Name(), path)
}

// RequireDir refuses a path that exists and is not a directory, or is a symlink, so state written under it later
// cannot fail or escape after an application was already changed. A missing path is fine.
func RequireDir(path string) error {
	info, err := os.Lstat(path)
	switch {
	case err != nil && ignorable(err):
		return nil
	case err != nil:
		return err
	case info.Mode()&fs.ModeSymlink != 0:
		return fmt.Errorf("refusing to use symlink %s for sf2 state", path)
	case !info.IsDir():
		return fmt.Errorf("%s is not a directory; sf2 keeps its state there", path)
	}
	return nil
}

// RequireFile refuses a path that exists and is not a regular file, or is a symlink. A missing path is fine.
func RequireFile(path string) error {
	info, err := os.Lstat(path)
	switch {
	case err != nil && ignorable(err):
		return nil
	case err != nil:
		return err
	case info.Mode()&fs.ModeSymlink != 0:
		return fmt.Errorf("refusing to replace symlink %s", path)
	case !info.Mode().IsRegular():
		return fmt.Errorf("%s is not a regular file; sf2 keeps its state there", path)
	}
	return nil
}
