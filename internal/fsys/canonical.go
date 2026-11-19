package fsys

import "path/filepath"

// Canonical is path made absolute with every symlink in its existing part resolved, so two spellings of one location,
// such as `a/../b` and `b` or a symlinked directory and its target, compare equal. A part that does not exist yet is
// appended as written, cleaned.
func Canonical(path string) string {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return resolveExisting(absolute)
}

func resolveExisting(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	parent := filepath.Dir(path)
	if parent == path {
		return path
	}
	return filepath.Join(resolveExisting(parent), filepath.Base(path))
}

// SamePath reports whether a and b name the same location on the filesystem.
func SamePath(a, b string) bool { return Canonical(a) == Canonical(b) }
