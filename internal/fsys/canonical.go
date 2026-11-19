package fsys

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// maxLinks bounds symlink resolution, as the kernel's ELOOP limit does.
const maxLinks = 40

// CheckSpelling refuses a path sf2 would act on unless it is spelled the way the filesystem stores it: no `..`
// component, and every existing component named exactly as its directory entry, in letter case and Unicode
// normalization. A path through a missing directory and back out with `..` reaches a file inspection cannot see,
// and an alternative spelling of an existing file names one target two ways, so both are refused rather than
// reconciled; the user passes the real path instead.
func CheckSpelling(path string) error { return checkSpelling(path, map[string][]string{}) }

// checkSpelling is CheckSpelling with directory listings shared through listed, so a plan of many files in one
// directory reads it once.
func checkSpelling(path string, listed map[string][]string) error {
	parts := strings.Split(absolute(path), string(filepath.Separator))
	if slices.Contains(parts, "..") {
		return fmt.Errorf("refusing %s: it goes through `..`; pass the real path", path)
	}
	resolved := string(filepath.Separator)
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		next := filepath.Join(resolved, part)
		if _, err := os.Lstat(next); err != nil {
			if ignorable(err) {
				return nil
			}
			return err
		}
		names, ok := listed[resolved]
		if !ok {
			var err error
			if names, err = entryNames(resolved); err != nil {
				return err
			}
			listed[resolved] = names
		}
		if !slices.Contains(names, part) {
			return fmt.Errorf("refusing %s: %q is not spelled as it is on disk; pass the real path", path, part)
		}
		real, err := filepath.EvalSymlinks(next)
		if err != nil {
			// A dangling symlink reaches nothing further to check; the writer applies its symlink policy.
			return nil
		}
		resolved = real
	}
	return nil
}

func entryNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	return names, nil
}

// Canonical is the location path opens, as the filesystem resolves it: each existing component is followed through
// symlinks one at a time, and `..` inside a link target steps up from what was actually reached. A tail that does not
// exist yet is appended as written. Callers pass paths CheckSpelling accepted.
func Canonical(path string) string {
	resolved, missing := walk(string(filepath.Separator), absolute(path), 0)
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
