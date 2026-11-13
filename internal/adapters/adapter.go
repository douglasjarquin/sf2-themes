// Package adapters holds the contract every file-writing application adapter implements; each adapter lives in
// its own subpackage. See AGENTS.md beside this file.
package adapters

import (
	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
	"github.com/douglasjarquin/sf2-themes/v2/internal/plan"
)

// App is how the CLI drives one configurable application. An adapter supplies Inspect and Plan; the CLI owns the
// ownership gate, theme selection, the commit, and every line printed.
//
// configDir is the expanded --config-dir, or "" when the flag is absent and the adapter resolves its default
// location (environment variables, then the home directory).
type App struct {
	// Label names the app in summaries and refusals, such as "Herdr app".
	Label string
	// Inspect reads the app's configuration once and classifies who owns the theme setting. It never refuses: a
	// broken or unknown-shape config is an Inspection with Detail, and only an I/O or decode failure is an error.
	Inspect func(themes []catalog.Theme, configDir string) (plan.Inspection, error)
	// Plan returns every write and removal for theme. slot=false means plan the theme files only and leave the
	// app's theme setting untouched. It never writes.
	Plan func(theme catalog.Theme, themes []catalog.Theme, configDir string, slot bool) ([]plan.Step, error)
	// Target resolves the configuration file or directory that holds the app's theme setting, printed as the
	// `target:` summary line.
	Target func(configDir string) (string, error)
	// Pair reports a dark and light sibling pair that follows host appearance in the summary.
	Pair bool
	// OwnsFilesOnUnknownShape lets an unknown-shape config still receive the theme files, then exit nonzero
	// with the manual fix, and adds the `pointer:` summary line when the setting is left alone.
	OwnsFilesOnUnknownShape bool
	// Reload is the `note:` summary line printed when the setting was written; "" prints none.
	Reload string
	// StderrHint is printed on stderr, as is, after a run that wrote the setting; "" prints none.
	StderrHint string
}
