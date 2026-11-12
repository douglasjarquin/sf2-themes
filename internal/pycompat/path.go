package pycompat

import (
	"errors"
	"os"
	"os/user"
	"strings"
)

// Paths here are strings in the form str(PurePosixPath(...)) prints: repeated slashes and `.` components are
// dropped, `..` is kept for the filesystem to resolve, a trailing slash is dropped, and an empty path is `.`.

// Path is PurePosixPath(*parts): the parts joined left to right, where an absolute part replaces what came before.
// Path(a, b) is also `Path(a) / b`.
func Path(parts ...string) string {
	root := ""
	var names []string
	for _, part := range parts {
		if partRoot, rest := splitRoot(part); partRoot != "" {
			root, names = partRoot, nil
			part = rest
		}
		for name := range strings.SplitSeq(part, "/") {
			if name != "" && name != "." {
				names = append(names, name)
			}
		}
	}
	if root == "" && len(names) == 0 {
		return "."
	}
	return root + strings.Join(names, "/")
}

// splitRoot returns the root of a part and the rest: POSIX keeps exactly two leading slashes, and treats one or
// three or more as a single slash.
func splitRoot(part string) (root, rest string) {
	rest = strings.TrimLeft(part, "/")
	switch len(part) - len(rest) {
	case 0:
		return "", part
	case 2:
		return "//", rest
	default:
		return "/", rest
	}
}

// Name is Path.name: the final component, or "" for the root and for `.`.
func Name(path string) string {
	root, rest := splitRoot(Path(path))
	if rest == "." && root == "" {
		return ""
	}
	return rest[strings.LastIndex(rest, "/")+1:]
}

// Parent is Path.parent: the path without its final component; the root and `.` are their own parents.
func Parent(path string) string {
	path = Path(path)
	root, rest := splitRoot(path)
	if rest == "" || (root == "" && rest == ".") {
		return path
	}
	i := strings.LastIndex(rest, "/")
	if i < 0 {
		return Path(root)
	}
	return root + rest[:i]
}

// WithName is Path.with_name(name): the same parent with name as the final component.
// The path must have a final component and name must be a single component, as Python requires.
func WithName(path, name string) string {
	return Path(Parent(path), name)
}

// ErrNoHome is returned by ExpandUser where pathlib raises "Could not determine home directory."
var ErrNoHome = errors.New("could not determine home directory")

// ExpandUser is Path.expanduser(): a leading `~` becomes $HOME (or the account's home directory when HOME is unset),
// and a leading `~name` becomes that user's home directory.
func ExpandUser(path string) (string, error) {
	path = Path(path)
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	first, rest, _ := strings.Cut(path, "/")
	var home string
	if first == "~" {
		if env, ok := os.LookupEnv("HOME"); ok {
			home = env
		} else if current, err := user.Current(); err == nil {
			home = current.HomeDir
		} else {
			return "", ErrNoHome
		}
	} else {
		account, err := user.Lookup(first[1:])
		if err != nil {
			return "", ErrNoHome
		}
		home = account.HomeDir
	}
	home = strings.TrimRight(home, "/")
	if home == "" {
		home = "/"
	}
	if strings.HasPrefix(home, "~") {
		return "", ErrNoHome
	}
	return Path(home, rest), nil
}

// Home is Path.home().
func Home() (string, error) {
	return ExpandUser("~")
}

// Resolve is Path.resolve() (os.path.realpath without strict): the absolute path with every symbolic link that
// exists followed and `..` applied after the links before it; components that do not exist are kept as written.
// A symbolic-link loop leaves the rest of the path unresolved, as realpath does.
func Resolve(path string) string {
	resolved, _ := joinRealPath("", path, map[string]*string{})
	if !strings.HasPrefix(resolved, "/") {
		cwd, err := os.Getwd()
		if err == nil {
			resolved = posixJoin(cwd, resolved)
		}
	}
	return Path(normPath(resolved))
}

// joinRealPath is posixpath._joinrealpath. seen maps a link to its resolution, or to nil while it is being resolved.
func joinRealPath(path, rest string, seen map[string]*string) (string, bool) {
	if strings.HasPrefix(rest, "/") {
		rest = rest[1:]
		path = "/"
	}
	for rest != "" {
		var name string
		name, rest, _ = strings.Cut(rest, "/")
		if name == "" || name == "." {
			continue
		}
		if name == ".." {
			if path == "" {
				path = ".."
				continue
			}
			var last string
			path, last = posixSplit(path)
			if last == ".." {
				path = posixJoin(posixJoin(path, ".."), "..")
			}
			continue
		}
		next := posixJoin(path, name)
		info, err := os.Lstat(next)
		if err != nil || info.Mode()&os.ModeSymlink == 0 {
			path = next
			continue
		}
		if cached, ok := seen[next]; ok {
			if cached != nil {
				path = *cached
				continue
			}
			return posixJoin(next, rest), false
		}
		seen[next] = nil
		target, err := os.Readlink(next)
		if err != nil {
			path = next
			continue
		}
		var ok bool
		path, ok = joinRealPath(path, target, seen)
		if !ok {
			return posixJoin(path, rest), false
		}
		resolved := path
		seen[next] = &resolved
	}
	return path, true
}

// posixJoin is os.path.join for two components.
func posixJoin(a, b string) string {
	switch {
	case strings.HasPrefix(b, "/"):
		return b
	case a == "" || strings.HasSuffix(a, "/"):
		return a + b
	default:
		return a + "/" + b
	}
}

// posixSplit is os.path.split.
func posixSplit(p string) (head, tail string) {
	i := strings.LastIndex(p, "/") + 1
	head, tail = p[:i], p[i:]
	if trimmed := strings.TrimRight(head, "/"); trimmed != "" {
		head = trimmed
	}
	return head, tail
}

// normPath is os.path.normpath, which, unlike Path, collapses `..` lexically.
func normPath(p string) string {
	if p == "" {
		return "."
	}
	slashes := 0
	if strings.HasPrefix(p, "/") {
		slashes = 1
		if strings.HasPrefix(p, "//") && !strings.HasPrefix(p, "///") {
			slashes = 2
		}
	}
	var comps []string
	for comp := range strings.SplitSeq(p, "/") {
		switch {
		case comp == "" || comp == ".":
		case comp != ".." || (slashes == 0 && len(comps) == 0) || (len(comps) > 0 && comps[len(comps)-1] == ".."):
			comps = append(comps, comp)
		case len(comps) > 0:
			comps = comps[:len(comps)-1]
		}
	}
	out := strings.Repeat("/", slashes) + strings.Join(comps, "/")
	if out == "" {
		return "."
	}
	return out
}
