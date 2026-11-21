// Package store keeps sf2's ownership manifest: which files sf2 wrote for each application target, their
// fingerprints, the selection it made there, and the backup of the setting it first replaced.
//
// The manifest lives at $XDG_STATE_HOME/sf2/manifest.json, or ~/.local/state/sf2/manifest.json. It is the proof
// of ownership later commands rely on; a filename starting with `sf2-` is not.
package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/douglasjarquin/sf2-themes/v2/internal/fsys"
	"github.com/douglasjarquin/sf2-themes/v2/internal/pycompat"
)

// Version is the manifest schema this binary reads and writes.
const Version = 1

// File is one path sf2 wrote and the SHA-256 of what it wrote there.
type File struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

// Target is everything sf2 owns for one application configuration target. App and Target together are its key:
// Target is the configuration path the app's `target:` summary line prints, so two --config-dir targets of one app
// are two entries.
type Target struct {
	App    string `json:"app"`
	Target string `json:"target"`
	// Selection is the sf2 identity selected for this target, or "" when the user's own theme was kept.
	Selection string `json:"selection,omitempty"`
	// Setting is the user configuration file sf2 edited to select the theme, when it edited one.
	Setting *File `json:"setting,omitempty"`
	// Prior is the store's copy of Setting from before sf2 first changed it, kept across later runs.
	Prior string `json:"prior,omitempty"`
	// Files are the files sf2 wholly owns for this target, sorted by path.
	Files []File `json:"files"`
}

// Manifest is the whole ownership record, targets sorted by app and then target.
type Manifest struct {
	Version int      `json:"version"`
	Targets []Target `json:"targets"`
}

// Path is the manifest's location under XDG_STATE_HOME, which is honored only when it is an absolute path.
func Path() (string, error) {
	root := os.Getenv("XDG_STATE_HOME")
	if root != "" {
		expanded, err := pycompat.ExpandUser(root)
		if err != nil {
			return "", err
		}
		root = expanded
	}
	if !filepath.IsAbs(root) {
		home, err := pycompat.Home()
		if err != nil {
			return "", err
		}
		root = pycompat.Path(home, ".local", "state")
	}
	return pycompat.Path(root, "sf2", "manifest.json"), nil
}

// Load reads the manifest at path; a missing file is an empty manifest. A manifest that is not valid JSON, or that
// a newer binary wrote, is an error so no command acts on ownership it cannot read.
func Load(path string) (Manifest, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Manifest{Version: Version}, nil
	}
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("%s is not a valid sf2 manifest: %w", path, err)
	}
	// The whole file must be that one document; trailing data is never silently dropped by the next save.
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return Manifest{}, fmt.Errorf("%s is not a valid sf2 manifest: unexpected data after the manifest", path)
	}
	if manifest.Version != Version {
		return Manifest{}, fmt.Errorf("%s has manifest version %d; this sf2 reads version %d", path, manifest.Version, Version)
	}
	return manifest, nil
}

// Find returns the entry for app and target, or nil when sf2 has recorded nothing there.
func (m *Manifest) Find(app, target string) *Target {
	for i := range m.Targets {
		if m.Targets[i].App == app && m.Targets[i].Target == target {
			return &m.Targets[i]
		}
	}
	return nil
}

// Wrote reports whether any entry records sf2 writing exactly the bytes with SHA-256 sha at path.
func (m *Manifest) Wrote(path, sha string) bool {
	for _, target := range m.Targets {
		for _, file := range target.Files {
			if file.SHA256 == sha && (file.Path == path || fsys.SamePath(file.Path, path)) {
				return true
			}
		}
	}
	return false
}

// Record replaces the entry for entry's app and target. An entry without a Prior keeps the one already recorded,
// because only the first change to a setting holds what the user had before sf2.
func (m *Manifest) Record(entry Target) {
	entry.Files = slices.Clone(entry.Files)
	slices.SortFunc(entry.Files, func(a, b File) int { return strings.Compare(a.Path, b.Path) })
	if existing := m.Find(entry.App, entry.Target); existing != nil {
		if entry.Prior == "" {
			entry.Prior = existing.Prior
		}
		*existing = entry
		return
	}
	m.Targets = append(m.Targets, entry)
	slices.SortFunc(m.Targets, func(a, b Target) int {
		if c := strings.Compare(a.App, b.App); c != 0 {
			return c
		}
		return strings.Compare(a.Target, b.Target)
	})
}

// encode is the manifest's on-disk form.
func encode(manifest Manifest) (string, error) {
	manifest.Version = Version
	if manifest.Targets == nil {
		manifest.Targets = []Target{}
	}
	for i := range manifest.Targets {
		if manifest.Targets[i].Files == nil {
			manifest.Targets[i].Files = []File{}
		}
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	return string(data) + "\n", err
}

// KeepPrior keeps data, the bytes of a user's setting file named name before sf2 first changed it, under
// backups/<app>/ beside the manifest at manifestPath and returns the copy's path. Each copy is published exclusively
// at the next free `<name>.<n>`, so no later or concurrent copy can replace it.
func KeepPrior(manifestPath, app, name string, data []byte) (string, error) {
	dir := pycompat.Path(pycompat.Parent(manifestPath), "backups", app)
	for n := 1; ; n++ {
		kept := pycompat.Path(dir, fmt.Sprintf("%s.%d", name, n))
		err := fsys.CreateExclusive(kept, data, 0o600)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		return kept, err
	}
}

// Check refuses, writing nothing, the state Open and a staged save would refuse for app: a symlink or the wrong kind
// of file where sf2 keeps its state or its lock, or a manifest the next save could not replace. A command checks it
// before taking the lock, so such a refusal leaves no state directory or lock behind.
func Check(app string) error {
	path, err := Path()
	if err != nil {
		return err
	}
	dir := pycompat.Parent(path)
	for _, required := range []string{dir, pycompat.Path(dir, "backups"), pycompat.Path(dir, "backups", app)} {
		if err := fsys.RequireDir(required); err != nil {
			return err
		}
	}
	for _, file := range []string{path, pycompat.Path(dir, "manifest.lock")} {
		if err := fsys.RequireFile(file); err != nil {
			return err
		}
	}
	return fsys.CanReplace(path)
}

// Session is the manifest held under the store's lock for one mutating command, so concurrent commands apply their
// records one after another instead of saving over each other.
type Session struct {
	Path     string
	Manifest Manifest
	unlock   func()
	staged   *fsys.Staged
	// original is the manifest's bytes when the session opened, nil when there was none.
	original []byte
	// made are the state directories that did not exist when the session opened, deepest first.
	made []string
	// modified is the state directory's modification time once the session held the lock.
	modified time.Time
}

// Open checks that the state directory can take app's records, takes the store's lock, loads the manifest, and stages
// the next save. It refuses before any application is touched when a state path is a symlink or the wrong kind of
// file, the manifest is unreadable, or the manifest could not be published. Close releases the lock.
func Open(app string) (*Session, error) {
	path, err := Path()
	if err != nil {
		return nil, err
	}
	dir := pycompat.Parent(path)
	var made []string
	for _, required := range []string{pycompat.Path(dir, "backups", app), pycompat.Path(dir, "backups")} {
		if err := fsys.RequireDir(required); err != nil {
			return nil, err
		}
		if _, err := os.Lstat(required); errors.Is(err, fs.ErrNotExist) {
			made = append(made, required)
		}
	}
	if err := fsys.RequireDir(dir); err != nil {
		return nil, err
	}
	unlock, err := fsys.Lock(pycompat.Path(dir, "manifest.lock"))
	if err != nil {
		return nil, err
	}
	session := &Session{Path: path, unlock: unlock, made: made}
	if err := fsys.RequireFile(path); err != nil {
		session.Close()
		return nil, err
	}
	if session.Manifest, err = Load(path); err != nil {
		session.Close()
		return nil, err
	}
	if session.original, err = os.ReadFile(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		session.Close()
		return nil, err
	}
	if info, err := os.Stat(dir); err == nil {
		session.modified = info.ModTime()
	}
	if err := removeStranded(dir); err != nil {
		session.Close()
		return nil, err
	}
	return session, nil
}

// removeStranded deletes temporary manifests a command left behind when it was killed. Only the holder of the lock
// stages one, so under the lock every one found is stranded.
func removeStranded(dir string) error {
	stranded, err := filepath.Glob(filepath.Join(dir, ".manifest.json.*"))
	if err != nil {
		return err
	}
	for _, path := range stranded {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	return nil
}

// Stage proves the next Save can publish before a change the manifest must record: it creates the temporary file
// Save publishes, which refuses a state directory sf2 cannot write, and probes that the existing manifest can be
// replaced. It does nothing when a save is already staged.
func (s *Session) Stage() error {
	if s.staged != nil {
		return nil
	}
	staged, err := fsys.Stage(s.Path)
	if err != nil {
		return err
	}
	if err := staged.Probe(); err != nil {
		staged.Discard()
		return err
	}
	s.staged = staged
	return nil
}

// Save atomically publishes the session's manifest through the staged temporary file, staging one when none is. A
// manifest identical to the one on disk writes nothing.
func (s *Session) Save() error {
	data, err := encode(s.Manifest)
	if err != nil {
		return err
	}
	if existing, err := os.ReadFile(s.Path); err == nil && string(existing) == data {
		return nil
	}
	if err := s.Stage(); err != nil {
		return err
	}
	staged := s.staged
	s.staged = nil
	return staged.Publish(data, 0o644)
}

// Restore puts the manifest back as the session found it and removes the backup directories it created, for a
// command whose changes were undone. The state directory and its lock stay: another command may be waiting on the
// lock.
func (s *Session) Restore() error {
	if s.staged != nil {
		s.staged.Discard()
		s.staged = nil
	}
	if s.original == nil {
		if err := os.Remove(s.Path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	} else {
		data, err := os.ReadFile(s.Path)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if !bytes.Equal(data, s.original) {
			if err := s.Stage(); err != nil {
				return err
			}
			staged := s.staged
			s.staged = nil
			if err := staged.Publish(string(s.original), 0o644); err != nil {
				return err
			}
		}
	}
	for _, dir := range s.made {
		// A directory something else has put a file in since is left alone.
		if err := os.Remove(dir); err != nil && !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, syscall.ENOTEMPTY) &&
			!errors.Is(err, syscall.EEXIST) {
			return err
		}
	}
	if s.modified.IsZero() {
		return nil
	}
	return os.Chtimes(pycompat.Parent(s.Path), time.Time{}, s.modified)
}

// Close removes an unused staged save and releases the lock.
func (s *Session) Close() {
	if s.staged != nil {
		s.staged.Discard()
		s.staged = nil
	}
	s.unlock()
}
