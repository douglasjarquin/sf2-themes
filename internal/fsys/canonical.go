package fsys

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// maxLinks bounds symlink resolution, as the kernel's ELOOP limit does.
const maxLinks = 40

// Canonical is the location path opens, as the filesystem resolves it: each existing component is followed through
// symlinks one at a time and spelled with its name on disk, and `..` steps up from what was actually reached, so
// `link/..` is the link target's parent. A tail that does not exist yet is appended as written.
//
// Two spellings of one location give one result, including different letter case on a case-insensitive volume, and
// distinct locations give distinct results.
func Canonical(path string) string {
	if !filepath.IsAbs(path) {
		if wd, err := os.Getwd(); err == nil {
			path = wd + string(filepath.Separator) + path
		}
	}
	resolved, missing := walk(string(filepath.Separator), path, 0)
	return filepath.Join(append([]string{resolved}, missing...)...)
}

// walk resolves path's components from the existing directory resolved. It returns the deepest existing location
// reached and the components after it that do not exist.
func walk(resolved, path string, links int) (string, []string) {
	parts := strings.Split(path, string(filepath.Separator))
	for i, part := range parts {
		switch part {
		case "", ".":
			continue
		case "..":
			resolved = filepath.Dir(resolved)
			continue
		}
		next := filepath.Join(resolved, part)
		info, err := os.Lstat(next)
		if err != nil {
			return resolved, nonEmpty(parts[i:])
		}
		next = filepath.Join(resolved, nameOnDisk(resolved, part, info))
		if info.Mode()&fs.ModeSymlink == 0 {
			resolved = next
			continue
		}
		target, err := os.Readlink(next)
		if err != nil || links >= maxLinks {
			return resolved, nonEmpty(parts[i:])
		}
		if !filepath.IsAbs(target) {
			target = resolved + string(filepath.Separator) + target
		}
		var missing []string
		if resolved, missing = walk(string(filepath.Separator), target, links+1); len(missing) > 0 {
			return resolved, append(missing, nonEmpty(parts[i+1:])...)
		}
	}
	return resolved, nil
}

// nameOnDisk is the directory entry in dir that part opened: part itself when an entry has exactly that name,
// otherwise the entry a case-insensitive volume matched.
func nameOnDisk(dir, part string, info fs.FileInfo) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return part
	}
	for _, entry := range entries {
		if entry.Name() == part {
			return part
		}
	}
	for _, entry := range entries {
		if !strings.EqualFold(entry.Name(), part) {
			continue
		}
		if candidate, err := os.Lstat(filepath.Join(dir, entry.Name())); err == nil && os.SameFile(candidate, info) {
			return entry.Name()
		}
	}
	return part
}

func nonEmpty(parts []string) []string {
	var kept []string
	for _, part := range parts {
		if part != "" && part != "." {
			kept = append(kept, part)
		}
	}
	return kept
}

// SamePath reports whether a and b name one location: the existing parts they reach are the same file or directory
// by os.SameFile, and the parts that do not exist yet are spelled alike.
func SamePath(a, b string) bool {
	aDir, aMissing := walk(string(filepath.Separator), absolute(a), 0)
	bDir, bMissing := walk(string(filepath.Separator), absolute(b), 0)
	if strings.Join(aMissing, "/") != strings.Join(bMissing, "/") {
		return false
	}
	aInfo, aErr := os.Stat(aDir)
	bInfo, bErr := os.Stat(bDir)
	if aErr != nil || bErr != nil {
		return aDir == bDir
	}
	return os.SameFile(aInfo, bInfo)
}

func absolute(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	if wd, err := os.Getwd(); err == nil {
		return wd + string(filepath.Separator) + path
	}
	return path
}
