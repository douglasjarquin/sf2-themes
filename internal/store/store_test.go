package store

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestPathHonorsAnAbsoluteXDGStateHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, tc := range []struct{ xdg, want string }{
		{"/state", "/state/sf2/manifest.json"},
		{"", home + "/.local/state/sf2/manifest.json"},
		{"relative/state", home + "/.local/state/sf2/manifest.json"},
		{"~/st", home + "/st/sf2/manifest.json"},
	} {
		t.Setenv("XDG_STATE_HOME", tc.xdg)
		if got, err := Path(); err != nil || got != tc.want {
			t.Errorf("XDG_STATE_HOME=%q: %q, %v; want %q", tc.xdg, got, err, tc.want)
		}
	}
}

func TestAMissingManifestIsEmpty(t *testing.T) {
	got, err := Load(filepath.Join(t.TempDir(), "manifest.json"))
	if err != nil || !reflect.DeepEqual(got, Manifest{Version: Version}) {
		t.Errorf("%+v, %v", got, err)
	}
}

func TestAnUnreadableManifestIsAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.json")
	for content, want := range map[string]string{
		"{":                               "is not a valid sf2 manifest",
		`{"version": 1, "extra": true}`:   "is not a valid sf2 manifest",
		`{"version": 2, "targets": []}`:   "has manifest version 2; this sf2 reads version 1",
		`{"version": 0, "targets": null}`: "has manifest version 0",
	} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v, want %q", content, err, want)
		}
	}
}

func TestRecordReplacesATargetKeepsItsFirstPriorAndSorts(t *testing.T) {
	var m Manifest
	m.Record(Target{App: "wezterm", Target: "/b/wezterm.lua", Selection: "sf2-ken"})
	m.Record(Target{App: "wezterm", Target: "/a/wezterm.lua", Selection: "sf2-ryu", Prior: "/a/wezterm.lua.bak.1",
		Files: []File{{Path: "/a/z", SHA256: "1"}, {Path: "/a/b", SHA256: "2"}}})
	m.Record(Target{App: "codex", Target: "/c/config.toml"})
	m.Record(Target{App: "wezterm", Target: "/a/wezterm.lua", Selection: "sf2-guile", Files: []File{{Path: "/a/b", SHA256: "3"}}})

	want := []Target{
		{App: "codex", Target: "/c/config.toml"},
		{App: "wezterm", Target: "/a/wezterm.lua", Selection: "sf2-guile", Prior: "/a/wezterm.lua.bak.1", Files: []File{{Path: "/a/b", SHA256: "3"}}},
		{App: "wezterm", Target: "/b/wezterm.lua", Selection: "sf2-ken"},
	}
	if !reflect.DeepEqual(m.Targets, want) {
		t.Errorf("targets\n%+v\nwant\n%+v", m.Targets, want)
	}
	if m.Find("wezterm", "/b/wezterm.lua").Selection != "sf2-ken" || m.Find("nvim", "/a") != nil {
		t.Error("Find")
	}
}

func TestSaveWritesTheDocumentedShapeAndRoundTrips(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", root)
	path := filepath.Join(root, "sf2", "manifest.json")
	session, err := Open("codex")
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	m := &session.Manifest
	m.Record(Target{App: "codex", Target: "/c/config.toml", Selection: "sf2-ryu",
		Setting: &File{Path: "/c/config.toml", SHA256: "aa"}, Prior: "/c/config.toml.bak.1",
		Files: []File{{Path: "/c/themes/sf2-ryu.tmTheme", SHA256: "bb"}}})
	m.Record(Target{App: "herdr", Target: "/h/config.toml"})
	if err := session.Save(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := `{
  "version": 1,
  "targets": [
    {
      "app": "codex",
      "target": "/c/config.toml",
      "selection": "sf2-ryu",
      "setting": {
        "path": "/c/config.toml",
        "sha256": "aa"
      },
      "prior": "/c/config.toml.bak.1",
      "files": [
        {
          "path": "/c/themes/sf2-ryu.tmTheme",
          "sha256": "bb"
        }
      ]
    },
    {
      "app": "herdr",
      "target": "/h/config.toml",
      "files": []
    }
  ]
}
`
	if string(data) != want {
		t.Errorf("manifest:\n%s", data)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Find("codex", "/c/config.toml").Setting.SHA256 != "aa" || len(loaded.Targets) != 2 {
		t.Errorf("round trip %+v", loaded)
	}
}

func TestKeepPriorNeverReplacesAnEarlierCopy(t *testing.T) {
	dir := t.TempDir()
	manifest := filepath.Join(dir, "state", "sf2", "manifest.json")
	var kept []string
	for _, content := range []string{"mine\r\n", "later\n"} {
		path, err := KeepPrior(manifest, "codex", "config.toml", []byte(content))
		if err != nil {
			t.Fatal(err)
		}
		kept = append(kept, path)
	}
	base := filepath.Join(dir, "state", "sf2", "backups", "codex")
	if kept[0] != base+"/config.toml.1" || kept[1] != base+"/config.toml.2" {
		t.Errorf("kept %v", kept)
	}
	for i, want := range []string{"mine\r\n", "later\n"} {
		data, err := os.ReadFile(kept[i])
		info, statErr := os.Stat(kept[i])
		if err != nil || statErr != nil || string(data) != want || info.Mode().Perm() != 0o600 {
			t.Errorf("%s: %q %v %v", kept[i], data, err, statErr)
		}
	}
}

func TestTrailingDataAfterTheManifestIsAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.json")
	for _, content := range []string{`{"version":1,"targets":[]}TRAILING`, `{"version":1,"targets":[]}{}`, `{"version":1,"targets":[]}]`} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "is not a valid sf2 manifest") {
			t.Errorf("%s: %v", content, err)
		}
	}
	if err := os.WriteFile(path, []byte("{\"version\":1,\"targets\":[]}\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err != nil {
		t.Errorf("trailing whitespace: %v", err)
	}
}

func TestOpenRefusesUnusableStateBeforeLocking(t *testing.T) {
	for name, setup := range map[string]func(t *testing.T, sf2 string){
		"manifest symlink": func(t *testing.T, sf2 string) {
			real := filepath.Join(filepath.Dir(sf2), "real.json")
			writeFile(t, real, `{"version":1,"targets":[]}`)
			if err := os.Symlink(real, filepath.Join(sf2, "manifest.json")); err != nil {
				t.Fatal(err)
			}
		},
		"manifest directory": func(t *testing.T, sf2 string) { mkdir(t, filepath.Join(sf2, "manifest.json")) },
		"backups file":       func(t *testing.T, sf2 string) { writeFile(t, filepath.Join(sf2, "backups"), "x") },
		"app backups file":   func(t *testing.T, sf2 string) { writeFile(t, filepath.Join(sf2, "backups", "codex"), "x") },
		"state dir file":     func(t *testing.T, sf2 string) { writeFile(t, sf2, "x") },
	} {
		t.Run(name, func(t *testing.T) {
			state := t.TempDir()
			t.Setenv("XDG_STATE_HOME", state)
			sf2 := filepath.Join(state, "sf2")
			if name != "state dir file" {
				mkdir(t, sf2)
			}
			setup(t, sf2)
			if session, err := Open("codex"); err == nil {
				session.Close()
				t.Error("opened")
			}
		})
	}
}

func TestSessionsSerializeConcurrentUpdates(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	const writers = 16
	var group sync.WaitGroup
	errs := make(chan error, writers)
	for i := range writers {
		group.Add(1)
		go func() {
			defer group.Done()
			session, err := Open("wezterm")
			if err != nil {
				errs <- err
				return
			}
			defer session.Close()
			session.Manifest.Record(Target{App: "wezterm", Target: fmt.Sprintf("/t/%02d/wezterm.lua", i)})
			errs <- session.Save()
		}()
	}
	group.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	path, _ := Path()
	manifest, err := Load(path)
	if err != nil || len(manifest.Targets) != writers {
		t.Errorf("%d targets, %v", len(manifest.Targets), err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	mkdir(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}
