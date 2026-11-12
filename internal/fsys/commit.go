// Package fsys performs a planned mutation with the safe-write contract: symlink refusal, backups, atomic
// replacement, preserved modes, and dry-run. It is format-agnostic and receives final content.
package fsys

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"
	"time"

	"github.com/douglasjarquin/sf2-themes/v2/internal/plan"
	"github.com/douglasjarquin/sf2-themes/v2/internal/pycompat"
)

// Action is the outcome of one step.
type Action string

// The outcomes a step can have.
const (
	Created     Action = "created"
	Updated     Action = "updated"
	Unchanged   Action = "unchanged"
	WouldCreate Action = "would_create"
	WouldUpdate Action = "would_update"
	Removed     Action = "removed"
	WouldRemove Action = "would_remove"
)

// Planned reports whether the action is a dry-run outcome.
func (a Action) Planned() bool {
	return a == WouldCreate || a == WouldUpdate || a == WouldRemove
}

// Result is the outcome of one step. Path is the destination written, which is the resolved target of a followed
// symlink. Diff is the unified diff of a write that changes content, and "" otherwise.
type Result struct {
	Path   string
	Action Action
	Diff   string
}

// Options are the command-line switches that shape a commit.
type Options struct {
	DryRun         bool
	FollowSymlinks bool
}

// now is the clock that stamps backup names.
var now = time.Now

// Commit performs a planned mutation, refusing before the first write when any destination is unsafe.
// Removing a file that does not exist reports nothing.
func Commit(steps []plan.Step, opts Options) ([]Result, error) {
	destinations := make([]string, len(steps))
	for i, step := range steps {
		destinations[i] = step.Path
		if step.Remove {
			continue
		}
		destination, err := resolveDestination(step.Path, opts.FollowSymlinks)
		if err != nil {
			return nil, err
		}
		destinations[i] = destination
	}
	var results []Result
	for i, step := range steps {
		destination := destinations[i]
		if !step.Remove {
			result, err := writeFile(destination, step.Content, opts.DryRun)
			if err != nil {
				return results, err
			}
			results = append(results, result)
			continue
		}
		isFile, err := statIs(os.Stat, destination, fs.FileMode.IsRegular)
		if err != nil {
			return results, err
		}
		isLink, err := statIs(os.Lstat, destination, isSymlink)
		if err != nil {
			return results, err
		}
		if !isFile && !isLink {
			continue
		}
		action := WouldRemove
		if !opts.DryRun {
			if err := os.Remove(destination); err != nil {
				return results, err
			}
			action = Removed
		}
		results = append(results, Result{Path: destination, Action: action})
	}
	return results, nil
}

// writeFile atomically replaces path, already resolved, with content, keeping its mode and backing up what it held.
func writeFile(path, content string, dryRun bool) (Result, error) {
	destination, err := resolveDestination(path, false)
	if err != nil {
		return Result{}, err
	}
	isFile, err := statIs(os.Stat, destination, fs.FileMode.IsRegular)
	if err != nil {
		return Result{}, err
	}
	var existing *string
	if isFile {
		text, err := pycompat.ReadText(destination)
		if err != nil {
			return Result{}, err
		}
		existing = &text
	}
	if existing != nil && *existing == content {
		return Result{Path: destination, Action: Unchanged}, nil
	}
	diff := unifiedDiff(destination, existing, content)
	if dryRun {
		action := WouldCreate
		if existing != nil {
			action = WouldUpdate
		}
		return Result{Path: destination, Action: action, Diff: diff}, nil
	}
	if existing != nil {
		if err := backup(destination, *existing); err != nil {
			return Result{}, err
		}
	}
	if err := os.MkdirAll(pycompat.Parent(destination), 0o777); err != nil {
		return Result{}, err
	}
	mode := fs.FileMode(0o644)
	if info, err := os.Stat(destination); err == nil {
		mode = info.Mode().Perm()
	} else if !ignorable(err) {
		return Result{}, err
	}
	if err := replace(destination, content, mode); err != nil {
		return Result{}, err
	}
	action := Created
	if existing != nil {
		action = Updated
	}
	return Result{Path: destination, Action: action, Diff: diff}, nil
}

// resolveDestination refuses a symlinked destination unless the caller follows symlinks, and then returns its
// resolved target.
func resolveDestination(path string, follow bool) (string, error) {
	link, err := statIs(os.Lstat, path, isSymlink)
	if err != nil {
		return "", err
	}
	if !link {
		return path, nil
	}
	if !follow {
		return "", fmt.Errorf("refusing to replace symlink %s; pass --follow-symlinks to write through", path)
	}
	resolved := pycompat.Resolve(path)
	// Path.resolve() stats its result and raises on a symlink loop, which realpath alone leaves unresolved.
	if _, err := os.Stat(resolved); errors.Is(err, syscall.ELOOP) {
		return "", errors.New("Symlink loop from " + pycompat.Repr(resolved))
	}
	return resolved, nil
}

func backup(path, existing string) error {
	stamp := now().UTC().Format("20060102T150405Z")
	name := pycompat.WithName(path, pycompat.Name(path)+".bak."+stamp)
	if err := os.WriteFile(name, []byte(existing), 0o666); err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	return os.Chmod(name, info.Mode().Perm())
}

// replace writes content to a temporary file beside path and renames it over path, so a reader never sees a
// partial file. A failed replace removes the temporary file.
func replace(path, content string, mode fs.FileMode) (err error) {
	temporary, err := os.CreateTemp(pycompat.Parent(path), "."+pycompat.Name(path)+".*")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = os.Remove(temporary.Name())
		}
	}()
	if _, err = temporary.WriteString(content); err != nil {
		_ = temporary.Close()
		return err
	}
	if err = temporary.Close(); err != nil {
		return err
	}
	if err = os.Chmod(temporary.Name(), mode|0o600); err != nil {
		return err
	}
	if err = os.Rename(temporary.Name(), path); err != nil {
		return err
	}
	return os.Chmod(path, mode)
}

func unifiedDiff(path string, existing *string, content string) string {
	var before []string
	from := "/dev/null"
	if existing != nil {
		before = pycompat.SplitLinesKeepEnds(*existing)
		from = path
	}
	return pycompat.UnifiedDiff(before, pycompat.SplitLinesKeepEnds(content), from, path)
}

func isSymlink(mode fs.FileMode) bool { return mode&fs.ModeSymlink != 0 }

// statIs is pathlib's is_file/is_symlink: false when the path does not exist or cannot be reached, and an error for
// any other failure.
func statIs(stat func(string) (fs.FileInfo, error), path string, test func(fs.FileMode) bool) (bool, error) {
	info, err := stat(path)
	if err != nil {
		if ignorable(err) {
			return false, nil
		}
		return false, err
	}
	return test(info.Mode()), nil
}

// ignorable is pathlib's _ignore_error: the errors that make an existence check answer false instead of raising.
func ignorable(err error) bool {
	return errors.Is(err, syscall.ENOENT) || errors.Is(err, syscall.ENOTDIR) ||
		errors.Is(err, syscall.EBADF) || errors.Is(err, syscall.ELOOP)
}
