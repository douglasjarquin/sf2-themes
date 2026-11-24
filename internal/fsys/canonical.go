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

// Resolve returns the location path opens, as the filesystem resolves it, and is the one identity sf2 inspects,
// backs up, writes, and records for that path. Each existing component is followed through symlinks one at a time,
// and `..` inside a link target steps up from the directory actually reached; a tail that does not exist yet is
// appended as written.
//
// It refuses a path, rather than reconciling it, when the path as given goes through `..`, when any name it or a
// symlink it follows reaches is spelled other than as its directory entry (letter case or Unicode normalization), or
// when a missing directory is followed by `..`, which the filesystem would refuse but a lexical join would quietly
// turn into another file. The user passes the real path instead.
func Resolve(path string) (string, error) { return resolve(path, map[string][]string{}) }

// resolve is Resolve with directory listings shared through listed, so a plan of many files in one directory reads
// it once.
func resolve(path string, listed map[string][]string) (string, error) {
	abs := absolute(path)
	if slices.Contains(strings.Split(abs, string(filepath.Separator)), "..") {
		return "", fmt.Errorf("refusing %s: it goes through `..`; pass the real path", path)
	}
	r := resolver{path: path, listed: listed}
	reached, missing, err := r.walk(string(filepath.Separator), abs, 0)
	if err != nil {
		return "", err
	}
	return filepath.Join(append([]string{reached}, missing...)...), nil
}

// resolver walks one path, checking every name it reaches.
type resolver struct {
	path   string
	listed map[string][]string
	// lenient skips the checks, for comparing paths sf2 did not choose, such as a pointer named in a config.
	lenient bool
}

// walk resolves path's components from the existing directory resolved. It returns the deepest existing location
// reached and the components after it that do not exist.
func (r resolver) walk(resolved, path string, links int) (string, []string, error) {
	parts := strings.Split(path, string(filepath.Separator))
	for i, part := range parts {
		if part == "" || part == "." {
			continue
		}
		// The filesystem looks a name up only inside a directory; `file/..` is refused, not stepped out of.
		if info, err := os.Stat(resolved); err == nil && !info.IsDir() && !r.lenient {
			return "", nil, fmt.Errorf("refusing %s: %s is not a directory; pass the real path", r.path, resolved)
		}
		if part == ".." {
			resolved = filepath.Dir(resolved)
			continue
		}
		next := filepath.Join(resolved, part)
		info, err := os.Lstat(next)
		if err != nil {
			if !ignorable(err) && !r.lenient {
				return "", nil, err
			}
			return r.tail(resolved, parts[i:])
		}
		if err := r.spelled(resolved, part); err != nil {
			return "", nil, err
		}
		if info.Mode()&fs.ModeSymlink == 0 {
			resolved = next
			continue
		}
		target, err := os.Readlink(next)
		if err != nil || links >= maxLinks {
			// The writer reports a loop or an unreadable link when it reaches this one.
			return r.tail(resolved, parts[i:])
		}
		if !filepath.IsAbs(target) {
			target = resolved + string(filepath.Separator) + target
		}
		var missing []string
		if resolved, missing, err = r.walk(string(filepath.Separator), target, links+1); err != nil {
			return "", nil, err
		}
		if len(missing) > 0 {
			return r.tail(resolved, append(missing, parts[i+1:]...))
		}
	}
	return resolved, nil, nil
}

// tail returns the components that do not exist below resolved, refusing `..` among them: the filesystem cannot step
// out of a directory that is not there.
func (r resolver) tail(resolved string, parts []string) (string, []string, error) {
	missing := nonEmpty(parts)
	if slices.Contains(missing, "..") && !r.lenient {
		return "", nil, fmt.Errorf("refusing %s: it reaches `..` after %s, which does not exist; pass the real path",
			r.path, filepath.Join(resolved, missing[0]))
	}
	return resolved, missing, nil
}

// spelled refuses name unless dir holds an entry spelled exactly so.
func (r resolver) spelled(dir, name string) error {
	if r.lenient {
		return nil
	}
	names, ok := r.listed[dir]
	if !ok {
		var err error
		if names, err = entryNames(dir); err != nil {
			return err
		}
		r.listed[dir] = names
	}
	if !slices.Contains(names, name) {
		return fmt.Errorf("refusing %s: %q in %s is not spelled as it is on disk; pass the real path", r.path, name, dir)
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
// by os.SameFile, and the parts that do not exist yet are spelled alike. A path Resolve refuses is the same only as
// itself.
func SamePath(a, b string) bool {
	aDir, aMissing, aErr := resolver{lenient: true}.walk(string(filepath.Separator), absolute(a), 0)
	bDir, bMissing, bErr := resolver{lenient: true}.walk(string(filepath.Separator), absolute(b), 0)
	if aErr != nil || bErr != nil || slices.Contains(aMissing, "..") || slices.Contains(bMissing, "..") {
		return a == b
	}
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
