package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"

	"github.com/douglasjarquin/sf2-themes/v2/internal/adapters"
	"github.com/douglasjarquin/sf2-themes/v2/internal/adapters/slack"
	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
	"github.com/douglasjarquin/sf2-themes/v2/internal/fsys"
	"github.com/douglasjarquin/sf2-themes/v2/internal/plan"
	"github.com/douglasjarquin/sf2-themes/v2/internal/pycompat"
	"github.com/douglasjarquin/sf2-themes/v2/internal/store"
)

// actionText is how the plan listing and the `changes:` line name each outcome.
var actionText = map[fsys.Action]string{
	fsys.Created:     "created",
	fsys.Updated:     "updated",
	fsys.Unchanged:   "unchanged",
	fsys.WouldCreate: "would create",
	fsys.WouldUpdate: "would update",
	fsys.Removed:     "removed",
	fsys.WouldRemove: "would remove",
}

// changesOrder is the order of the counts on the `changes:` line.
var changesOrder = []fsys.Action{
	fsys.Created, fsys.Updated, fsys.Removed, fsys.WouldCreate, fsys.WouldUpdate, fsys.WouldRemove, fsys.Unchanged,
}

type selection int

const (
	selected selection = iota
	kept
	noneFound
	reset
)

// decision is what the ownership gate allows for one run.
type decision struct {
	refuse bool
	slot   bool
	// incomplete writes the theme files, leaves the setting alone, and then exits 1 with the manual fix.
	incomplete bool
}

// gate decides a run from the config's ownership:
//
//	ownership       app keeps files on unknown shape  --adopt  selects a theme  outcome
//	broken          any                               any      any              refuse
//	unknown shape   no                                any      any              refuse
//	unknown shape   yes                               any      any              files only, then refuse (incomplete)
//	foreign         any                               no       yes              refuse
//	foreign         any                               no       no               files only, keep the user's theme
//	foreign         any                               yes      any              write the setting
//	absent, managed, legacy                                                     write the setting
//
// A run selects a theme unless it is `setup` without --theme.
func gate(ownership plan.Ownership, app adapters.App, adopt, selectsTheme bool) decision {
	switch ownership {
	case plan.Broken:
		return decision{refuse: true}
	case plan.UnknownShape:
		if !app.OwnsFilesOnUnknownShape {
			return decision{refuse: true}
		}
		return decision{incomplete: true}
	case plan.Foreign:
		if adopt {
			return decision{slot: true}
		}
		return decision{refuse: selectsTheme}
	}
	return decision{slot: true}
}

func refusal(name string, app adapters.App, inspection plan.Inspection) error {
	switch inspection.Ownership {
	case plan.Foreign:
		return fmt.Errorf("%s already uses a theme sf2-themes does not manage; pass --adopt to replace it, "+
			"or run `setup %s` without --theme to keep it and install the theme files", app.Label, name)
	case plan.UnknownShape:
		return errors.New(app.Label + " config has a shape sf2-themes cannot edit safely.\n" +
			pycompat.RStrip(inspection.Detail) + "\n\nThen run the command again.")
	}
	return fmt.Errorf("%s config cannot be changed safely: %s", app.Label, inspection.Detail)
}

// mutate runs setup or apply: inspect, gate, select, plan, commit, report.
func mutate(ctx *Context, req Request) (int, error) {
	themes, err := catalog.Load()
	if err != nil {
		return 0, err
	}
	var requested *catalog.Theme
	if req.Theme != nil {
		theme, err := catalog.GetTheme(*req.Theme, themes)
		if err != nil {
			return 0, err
		}
		requested = &theme
	}
	if req.App == "slack" {
		return printSlack(ctx, requested, themes)
	}
	app := apps[req.App]
	configDir, err := expandConfigDir(req)
	if err != nil {
		return 0, err
	}
	selectsTheme := req.Command != "setup" || requested != nil
	opts := fsys.Options{DryRun: req.DryRun, FollowSymlinks: req.FollowSymlinks, Verbose: req.Verbose}
	// A first pass refuses before the state lock is taken, so a refused run leaves sf2's state untouched too.
	run, err := prepare(req, app, themes, configDir, requested, selectsTheme, opts)
	if err != nil {
		return 0, err
	}
	var owned *ownership
	if !req.DryRun {
		if owned, err = openOwnership(req.App, run.target); err != nil {
			return 0, err
		}
		defer owned.session.Close()
		// Another command may have changed the configuration while this one waited for the lock, so everything the
		// commit acts on is decided again under it.
		if run, err = prepare(req, app, themes, configDir, requested, selectsTheme, opts); err != nil {
			return 0, err
		}
		owned.target = run.target
		if err := owned.keepPrior(run.steps, run.destinations); err != nil {
			return 0, err
		}
		// Prove the record of this run can be published before the first application write.
		if err := owned.session.Stage(); err != nil {
			return 0, err
		}
	}
	results, err := fsys.Commit(run.steps, opts)
	if owned != nil && len(results) > 0 {
		// Record whatever was written, even by a commit that failed part way, so ownership is never lost.
		if recordErr := owned.record(results, err == nil && !run.decided.slot, func() (*catalog.Theme, error) {
			inspected, err := app.Inspect(themes, configDir)
			return inspected.Current, err
		}); recordErr != nil && err == nil {
			err = recordErr
		}
	}
	if err != nil {
		return 0, err
	}
	report(ctx, results, req.Verbose)
	label, err := selectionText(app, run.theme, run.how, themes)
	if err != nil {
		return 0, err
	}
	summarize(ctx, req.Command, req.App, app, run.decided, label, run.target, results)
	if run.decided.incomplete {
		return 0, refusal(req.App, app, run.inspection)
	}
	if app.StderrHint != "" && run.decided.slot {
		printf(ctx.Stderr, "%s\n", app.StderrHint)
	}
	return 0, nil
}

// prepared is everything one run commits, decided from the configuration as it is on disk.
type prepared struct {
	inspection   plan.Inspection
	decided      decision
	theme        catalog.Theme
	how          selection
	steps        []plan.Step
	destinations []string
	target       string
}

// prepare inspects the app, gates its ownership, chooses the theme, and plans and preflights the writes. It writes
// nothing.
func prepare(req Request, app adapters.App, themes []catalog.Theme, configDir string, requested *catalog.Theme,
	selectsTheme bool, opts fsys.Options) (prepared, error) {
	var run prepared
	var err error
	if run.inspection, err = app.Inspect(themes, configDir); err != nil {
		return run, err
	}
	run.decided = gate(run.inspection.Ownership, app, req.Adopt, selectsTheme)
	if run.decided.refuse {
		return run, refusal(req.App, app, run.inspection)
	}
	if run.theme, run.how, err = choose(requested, selectsTheme, run.inspection, themes); err != nil {
		return run, err
	}
	if run.steps, err = app.Plan(run.theme, themes, configDir, run.decided.slot); err != nil {
		return run, err
	}
	if run.target, err = app.Target(configDir); err != nil {
		return run, err
	}
	run.destinations, err = fsys.Preflight(run.steps, opts)
	return run, err
}

func expandConfigDir(req Request) (string, error) {
	if req.ConfigDir == nil {
		return "", nil
	}
	return pycompat.ExpandUser(*req.ConfigDir)
}

func choose(requested *catalog.Theme, selectsTheme bool, inspection plan.Inspection, themes []catalog.Theme) (catalog.Theme, selection, error) {
	switch {
	case selectsTheme && requested != nil:
		return *requested, selected, nil
	case !selectsTheme && inspection.Current != nil:
		return *inspection.Current, kept, nil
	}
	theme, err := catalog.DefaultTheme(themes)
	switch {
	case selectsTheme:
		return theme, selected, err
	case inspection.Unreadable:
		return theme, reset, err
	}
	return theme, noneFound, err
}

// report prints the plan: dry-run outcomes always, every outcome with --verbose, each followed by its diff.
func report(ctx *Context, results []fsys.Result, verbose bool) {
	for _, result := range results {
		if !verbose && !result.Action.Planned() {
			continue
		}
		printf(ctx.Stdout, "%s: %s\n", actionText[result.Action], result.Path)
		printf(ctx.Stdout, "%s", result.Diff)
	}
}

// changes is the `changes:` summary: counts in changesOrder, or `no changes` when nothing changed.
func changes(results []fsys.Result) string {
	counts := map[fsys.Action]int{}
	for _, result := range results {
		counts[result.Action]++
	}
	if len(results) == counts[fsys.Unchanged] {
		return "no changes"
	}
	var parts []string
	for _, action := range changesOrder {
		if counts[action] > 0 {
			parts = append(parts, fmt.Sprintf("%d %s", counts[action], actionText[action]))
		}
	}
	return strings.Join(parts, ", ")
}

func selectionText(app adapters.App, theme catalog.Theme, how selection, themes []catalog.Theme) (string, error) {
	label := theme.Metadata.SelectableID()
	if app.Pair {
		dark, light, err := catalog.ThemePair(theme, themes)
		if err != nil {
			return "", err
		}
		label = dark.Metadata.SelectableID() + " + " + light.Metadata.SelectableID() + " (follows host appearance)"
	} else {
		label += " (fixed; does not follow host appearance)"
	}
	switch how {
	case kept:
		return "kept " + label, nil
	case noneFound:
		return "no selection found, selected " + label, nil
	case reset:
		return "selection unreadable, reset to " + label, nil
	}
	return label, nil
}

func summarize(ctx *Context, verb, name string, app adapters.App, decided decision, selection, target string, results []fsys.Result) {
	switch {
	case decided.incomplete:
		printf(ctx.Stdout, "%s %s: incomplete (config left unchanged)\n", verb, name)
	case !decided.slot:
		printf(ctx.Stdout, "%s %s: kept your own theme (config left unchanged)\n", verb, name)
	default:
		printf(ctx.Stdout, "%s %s: %s\n", verb, name, selection)
	}
	printf(ctx.Stdout, "  target: %s\n", target)
	if !decided.slot && app.OwnsFilesOnUnknownShape {
		printf(ctx.Stdout, "  pointer: %s\n", selection)
	}
	printf(ctx.Stdout, "  changes: %s\n", changes(results))
	if decided.slot && app.Reload != "" {
		printf(ctx.Stdout, "  note: %s\n", app.Reload)
	}
}

// printSlack prints the import line on stdout and the steps to use it on stderr; Slack has no file to write.
func printSlack(ctx *Context, requested *catalog.Theme, themes []catalog.Theme) (int, error) {
	theme := requested
	if theme == nil {
		main, err := catalog.DefaultTheme(themes)
		if err != nil {
			return 0, err
		}
		theme = &main
	}
	mode := "Dark"
	if theme.Metadata.Variant == catalog.VariantLight {
		mode = "Light"
	}
	printf(ctx.Stdout, "%s\n", slack.RenderTheme(*theme))
	printf(ctx.Stderr, "%s\nSet Slack Color Mode to %s.\nTurn Darker sidebars off.\n"+
		"Paste the line into Preferences, Appearance, Custom theme, Import theme.\n", theme.Metadata.ID, mode)
	return 0, nil
}

var errNoSlackTheme = errors.New("no Slack theme to read")

// runCurrent prints the sf2 identity an app selects, through the same inspection setup uses.
func runCurrent(ctx *Context, req Request) (int, error) {
	configDir, err := expandConfigDir(req)
	if err != nil {
		return 0, err
	}
	if req.App == "slack" {
		return 0, errNoSlackTheme
	}
	app := apps[req.App]
	themes, err := catalog.Load()
	if err != nil {
		return 0, err
	}
	inspection, err := app.Inspect(themes, configDir)
	if err != nil {
		return 0, err
	}
	if inspection.Current != nil {
		printf(ctx.Stdout, "%s\n", inspection.Current.Metadata.SelectableID())
		return 0, nil
	}
	switch {
	case inspection.Ownership == plan.Broken:
		return 0, errors.New(inspection.Detail)
	case inspection.Ownership == plan.Foreign && !app.OwnsFilesOnUnknownShape:
		return 0, fmt.Errorf("%s uses a theme sf2-themes does not manage", app.Label)
	case inspection.Unreadable:
		return 0, fmt.Errorf("could not read the sf2 theme %s selects", app.Label)
	}
	return 0, fmt.Errorf("no %s theme applied yet", app.Label)
}

// ownership is one run's record for its app target, held under the store's lock from before the first application
// write until the record is saved.
type ownership struct {
	session *store.Session
	app     string
	target  string
}

// openOwnership refuses an unusable state directory or an unreadable manifest before anything is written.
func openOwnership(app, target string) (*ownership, error) {
	session, err := store.Open(app)
	if err != nil {
		return nil, err
	}
	return &ownership{session: session, app: app, target: target}, nil
}

// entry is a copy of the target's recorded entry, or a new one.
func (o *ownership) entry() store.Target {
	if previous := o.session.Manifest.Find(o.app, o.target); previous != nil {
		entry := *previous
		entry.Files = slices.Clone(previous.Files)
		return entry
	}
	return store.Target{App: o.app, Target: o.target}
}

// keepPrior keeps the bytes of the user's setting file before sf2 first changes it, and records the copy before the
// commit, so neither a failed commit nor a failed save can lose what the user had. A target keeps one prior: the
// first one sf2 ever took.
func (o *ownership) keepPrior(steps []plan.Step, destinations []string) error {
	entry := o.entry()
	if entry.Prior != "" || entry.Setting != nil {
		return nil
	}
	for i, step := range steps {
		if step.Remove || step.Path != o.target {
			continue
		}
		raw, err := os.ReadFile(destinations[i])
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if text, err := pycompat.DecodeText(raw); err == nil && text == step.Content {
			return nil
		}
		if entry.Prior, err = store.KeepPrior(o.session.Path, o.app, pycompat.Name(o.target), raw); err != nil {
			return err
		}
		o.session.Manifest.Record(entry)
		return o.session.Save()
	}
	return nil
}

// record merges what one commit did into the target's entry and saves it. The file at the target is the user's
// own configuration, recorded as the setting sf2 edited; every other written file is wholly sf2's. A removed file is
// no longer owned, and a record the commit did not reach is kept as it was. releasedSetting drops the setting
// record after a complete run that left the user's own setting in place. The selection is what the configuration
// selects after the commit, read back through inspect, and stays as recorded when it cannot be read.
func (o *ownership) record(results []fsys.Result, releasedSetting bool, current func() (*catalog.Theme, error)) error {
	entry := o.entry()
	files := map[string]string{}
	for _, file := range entry.Files {
		files[file.Path] = file.SHA256
	}
	for _, result := range results {
		switch {
		case result.Action == fsys.Removed:
			delete(files, result.Path)
		case result.Requested == o.target:
			entry.Setting = &store.File{Path: result.Path, SHA256: result.Fingerprint}
		default:
			files[result.Path] = result.Fingerprint
		}
	}
	if releasedSetting {
		entry.Setting = nil
	}
	entry.Files = entry.Files[:0]
	for path, sha := range files {
		entry.Files = append(entry.Files, store.File{Path: path, SHA256: sha})
	}
	if theme, err := current(); err == nil {
		entry.Selection = ""
		if theme != nil {
			entry.Selection = theme.Metadata.SelectableID()
		}
	}
	o.session.Manifest.Record(entry)
	return o.session.Save()
}
