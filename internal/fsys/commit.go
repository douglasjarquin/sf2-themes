// Package fsys performs a planned mutation with the safe-write contract: symlink refusal, backups, atomic
// replacement, preserved modes, and dry-run. It is format-agnostic and receives final content.
package fsys

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
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
)

// Planned reports whether the action is a dry-run outcome.
func (a Action) Planned() bool {
	return a == WouldCreate || a == WouldUpdate
}

// Result is the outcome of one step. Requested is the step's path and Path the destination written, which is the
// resolved target of a followed symlink. Diff is the unified diff of a write that changes content when the commit is
// a dry run or verbose, and "" otherwise. Fingerprint is the SHA-256 of the bytes the file holds afterwards, and
// Backup the backup a changed file received.
type Result struct {
	Requested   string
	Path        string
	Action      Action
	Diff        string
	Fingerprint string
	Backup      string
}

// Options are the command-line switches that shape a commit.
type Options struct {
	DryRun         bool
	FollowSymlinks bool
	Verbose        bool
}

// now is the clock that stamps backup names.
var now = time.Now

// rename publishes a written file; tests replace it to fail part way through a commit.
var rename = os.Rename

// Transaction is a commit's results and what undoes it, so a caller whose own record of the commit fails can put
// every file back.
type Transaction struct {
	Results []Result
	undo    []func() error
}

// Rollback restores every file the transaction changed, newest first: a replaced file gets its original bytes and
// mode back, and a created file, its backup, and the directories made for it are removed. What could not be undone
// stays in Results.
func (t *Transaction) Rollback() error {
	var errs []error
	var results []Result
	var undo []func() error
	for i := len(t.undo) - 1; i >= 0; i-- {
		if err := t.undo[i](); err != nil {
			errs = append(errs, err)
			results = append(results, t.Results[i])
			undo = append(undo, t.undo[i])
		}
	}
	slices.Reverse(results)
	slices.Reverse(undo)
	t.Results, t.undo = results, undo
	if len(errs) > 0 {
		return fmt.Errorf("could not undo every change: %w", errors.Join(errs...))
	}
	return nil
}

// Commit performs a planned mutation, refusing before the first write when any destination is unsafe or cannot be
// replaced. A write that fails part way undoes the ones before it, so a failed commit leaves the files as they were.
func Commit(steps []plan.Step, opts Options) ([]Result, error) {
	transaction, err := Apply(steps, opts)
	return transaction.Results, err
}

// Apply is Commit returning the Transaction, so the caller can still undo a commit that succeeded. When Apply fails,
// it has undone its writes; any it could not undo stay in the returned Transaction's Results.
func Apply(steps []plan.Step, opts Options) (*Transaction, error) {
	transaction := &Transaction{}
	destinations, err := Preflight(steps, opts)
	if err != nil {
		return transaction, err
	}
	for i, step := range steps {
		result, undo, err := writeFile(destinations[i], step.Content, opts)
		if err != nil {
			if rollbackErr := transaction.Rollback(); rollbackErr != nil {
				err = errors.Join(err, rollbackErr)
			}
			return transaction, err
		}
		result.Requested = step.Path
		if result.Action == Created || result.Action == Updated {
			transaction.Results = append(transaction.Results, result)
			transaction.undo = append(transaction.undo, undo)
			continue
		}
		// An unchanged or dry-run result has nothing to undo.
		transaction.Results = append(transaction.Results, result)
		transaction.undo = append(transaction.undo, func() error { return nil })
	}
	return transaction, nil
}

// Preflight resolves every destination of steps and refuses an unsafe one, or one a write would fail to replace,
// writing nothing. It returns the destinations Commit would use, so a caller can refuse a run before touching
// anything else.
func Preflight(steps []plan.Step, opts Options) ([]string, error) {
	destinations := make([]string, len(steps))
	listed := map[string][]string{}
	for i, step := range steps {
		resolved, err := resolve(step.Path, listed)
		if err != nil {
			return nil, err
		}
		destination, err := resolveDestination(step.Path, opts.FollowSymlinks)
		if err != nil {
			return nil, err
		}
		if destination != step.Path {
			// A symlink the writer follows lands on the file Resolve names, the one inspection read and the manifest
			// keys, never on a lexical reading of its target.
			destination = resolved
		}
		if err := writable(destination, step.Content); err != nil {
			return nil, err
		}
		destinations[i] = destination
	}
	return destinations, nil
}

// writable refuses a destination a write of content would fail on part way: something other than a regular file in
// its place, a file marked immutable or append-only, or a directory sf2 cannot create files in, where the temporary
// file, the backup, and the rename go. A file that already holds content is not written, so it passes.
func writable(destination, content string) error {
	info, err := os.Lstat(destination)
	switch {
	case err != nil && !ignorable(err):
		return err
	case err == nil && !info.Mode().IsRegular():
		return fmt.Errorf("cannot write %s: it is not a regular file", destination)
	case err == nil:
		raw, err := os.ReadFile(destination)
		if err != nil {
			return err
		}
		if string(raw) == content {
			return nil
		}
		if text, err := pycompat.DecodeText(raw); err == nil && text == content {
			return nil
		}
		if protected, err := protectedFile(destination); err != nil {
			return err
		} else if protected {
			return fmt.Errorf("cannot write %s: it is marked immutable or append-only", destination)
		}
		if err := replaceableIn(destination, info); err != nil {
			return err
		}
	}
	dir := pycompat.Parent(destination)
	for {
		info, err := os.Stat(dir)
		if err == nil {
			if !info.IsDir() {
				return fmt.Errorf("cannot write %s: %s is not a directory", destination, dir)
			}
			break
		}
		if !ignorable(err) || pycompat.Parent(dir) == dir {
			return err
		}
		dir = pycompat.Parent(dir)
	}
	if protected, err := protectedFile(dir); err != nil {
		return err
	} else if protected {
		return fmt.Errorf("cannot write %s: %s is marked immutable or append-only", destination, dir)
	}
	if err := syscall.Access(dir, accessWrite|accessSearch); err != nil {
		return fmt.Errorf("cannot write %s: %s: %w", destination, dir, err)
	}
	return nil
}

// The access(2) modes writable checks a directory for.
const (
	accessWrite  = 0x2
	accessSearch = 0x1
)

// replaceableIn refuses a file in a sticky directory that neither it nor the directory belongs to the user, which
// the rename could not replace.
func replaceableIn(path string, info fs.FileInfo) error {
	dir, err := os.Stat(pycompat.Parent(path))
	if err != nil {
		return err
	}
	uid := uint32(os.Geteuid())
	if dir.Mode()&fs.ModeSticky != 0 && uid != 0 && owner(info) != uid && owner(dir) != uid {
		return fmt.Errorf("%s cannot be replaced: another user owns it in a sticky directory", path)
	}
	return nil
}

// writeFile atomically replaces destination, already resolved, with content, keeping its mode and backing up what it
// held. It returns what undoes the write.
func writeFile(destination, content string, opts Options) (Result, func() error, error) {
	isFile, err := statIs(os.Stat, destination, fs.FileMode.IsRegular)
	if err != nil {
		return Result{}, nil, err
	}
	var existing *string
	var raw []byte
	if isFile {
		if raw, err = os.ReadFile(destination); err != nil {
			return Result{}, nil, err
		}
		text, err := pycompat.DecodeText(raw)
		if err != nil {
			return Result{}, nil, fmt.Errorf("%s: %w", destination, err)
		}
		existing = &text
	}
	fingerprint := Fingerprint(content)
	if existing != nil && (*existing == content || string(raw) == content) {
		// Equal text is a no-op, but the file may differ in bytes, such as CRLF line ends; ownership records the bytes.
		return Result{Path: destination, Action: Unchanged, Fingerprint: Fingerprint(string(raw))}, nil, nil
	}
	var diff string
	if opts.DryRun || opts.Verbose {
		diff = unifiedDiff(destination, existing, content)
	}
	if opts.DryRun {
		action := WouldCreate
		if existing != nil {
			action = WouldUpdate
		}
		return Result{Path: destination, Action: action, Diff: diff, Fingerprint: fingerprint}, nil, nil
	}
	var undo []func() error
	undoAll := func() error {
		var errs []error
		for i := len(undo) - 1; i >= 0; i-- {
			if err := undo[i](); err != nil {
				errs = append(errs, err)
			}
		}
		return errors.Join(errs...)
	}
	fail := func(err error) (Result, func() error, error) {
		if undoErr := undoAll(); undoErr != nil {
			err = errors.Join(err, undoErr)
		}
		return Result{}, nil, err
	}
	// The directory's times come back last, after everything the write put in it is gone again.
	if dir, err := os.Stat(pycompat.Parent(destination)); err == nil {
		modified := dir.ModTime()
		undo = append(undo, func() error { return os.Chtimes(pycompat.Parent(destination), time.Time{}, modified) })
	}
	mode := fs.FileMode(0o644)
	var saved string
	var modified time.Time
	if existing != nil {
		info, err := os.Stat(destination)
		if err != nil {
			return Result{}, nil, err
		}
		mode, modified = info.Mode().Perm(), info.ModTime()
		if saved, err = backup(destination, *existing); err != nil {
			return Result{}, nil, err
		}
		undo = append(undo, func() error { return removeIfPresent(saved) })
	}
	made, err := makeDirs(pycompat.Parent(destination))
	undo = append(undo, func() error { return removeDirs(made) })
	if err != nil {
		return fail(err)
	}
	if err := replace(destination, content, mode); err != nil {
		return fail(err)
	}
	if existing != nil {
		original := string(raw)
		undo = append(undo, func() error {
			if err := replace(destination, original, mode); err != nil {
				return err
			}
			return os.Chtimes(destination, time.Time{}, modified)
		})
	} else {
		undo = append(undo, func() error { return removeIfPresent(destination) })
	}
	action := Created
	if existing != nil {
		action = Updated
	}
	return Result{Path: destination, Action: action, Diff: diff, Fingerprint: fingerprint, Backup: saved}, undoAll, nil
}

// makeDirs creates dir and its missing parents, and returns the ones it created, deepest first.
func makeDirs(dir string) ([]string, error) {
	var missing []string
	for current := dir; ; current = pycompat.Parent(current) {
		if _, err := os.Lstat(current); err == nil || !ignorable(err) || pycompat.Parent(current) == current {
			break
		}
		missing = append(missing, current)
	}
	return missing, os.MkdirAll(dir, 0o777)
}

// removeDirs removes directories makeDirs created, deepest first, skipping any that are gone.
func removeDirs(dirs []string) error {
	for _, dir := range dirs {
		if err := removeIfPresent(dir); err != nil {
			return err
		}
	}
	return nil
}

func removeIfPresent(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

// Fingerprint is the hex SHA-256 of content, the identity a manifest records for a managed file.
func Fingerprint(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
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

// backup writes existing beside path with a UTC-stamped name and returns that name. It never replaces a file: when
// the name is taken, a counter follows the stamp.
func backup(path, existing string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	stamp := now().UTC().Format("20060102T150405Z")
	name := pycompat.WithName(path, pycompat.Name(path)+".bak."+stamp)
	for n := 1; ; n++ {
		file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if errors.Is(err, fs.ErrExist) {
			name = pycompat.WithName(path, fmt.Sprintf("%s.bak.%s.%d", pycompat.Name(path), stamp, n))
			continue
		}
		if err != nil {
			return "", err
		}
		_, err = file.WriteString(existing)
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
		if err == nil {
			err = os.Chmod(name, info.Mode().Perm())
		}
		if err != nil {
			_ = os.Remove(name)
			return "", err
		}
		return name, nil
	}
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
	if err = rename(temporary.Name(), path); err != nil {
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

// Exists is pathlib's exists: false when the path does not exist or cannot be reached.
func Exists(path string) (bool, error) {
	return statIs(os.Stat, path, func(fs.FileMode) bool { return true })
}

// IsFile is pathlib's is_file: false when the path does not exist, cannot be reached, or is not a regular file.
func IsFile(path string) (bool, error) {
	return statIs(os.Stat, path, fs.FileMode.IsRegular)
}

// ignorable is pathlib's _ignore_error: the errors that make an existence check answer false instead of raising.
func ignorable(err error) bool {
	return errors.Is(err, syscall.ENOENT) || errors.Is(err, syscall.ENOTDIR) ||
		errors.Is(err, syscall.EBADF) || errors.Is(err, syscall.ELOOP)
}
