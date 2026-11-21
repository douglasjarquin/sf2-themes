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
}

// changesOrder is the order of the counts on the `changes:` line.
var changesOrder = []fsys.Action{
	fsys.Created, fsys.Updated, fsys.WouldCreate, fsys.WouldUpdate, fsys.Unchanged,
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
}

// gate decides a run from the config's ownership:
//
//	ownership       --adopt  selects a theme  outcome
//	broken          any      any              refuse
//	unknown shape   any      any              refuse
//	foreign         no       yes              refuse
//	foreign         no       no               files only, keep the user's theme
//	foreign         yes      any              write the setting
//	absent, managed, legacy                   write the setting
//
// A run selects a theme unless it is `setup` without --theme. Every refusal comes before the first write, so a
// refused run leaves the disk and sf2's state exactly as they were.
func gate(ownership plan.Ownership, adopt, selectsTheme bool) decision {
	switch ownership {
	case plan.Broken, plan.UnknownShape:
		return decision{refuse: true}
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
		if inspection.Detail != "" {
			return errors.New(app.Label + " already uses a theme sf2-themes does not manage, and sf2-themes will not " +
				"edit it for you.\n" + pycompat.RStrip(inspection.Detail) + "\n\nThen run the command again.")
		}
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
	// A first pass refuses before the state lock is taken, so a refused run leaves sf2's state untouched too. It reads
	// the manifest without the lock; the pass under the lock decides again from the manifest it holds.
	recorded, err := readManifest()
	if err != nil {
		return 0, err
	}
	run, err := prepare(req, app, themes, configDir, requested, selectsTheme, opts, recorded)
	if err != nil {
		return 0, err
	}
	if !req.DryRun {
		// State that the run could not record in refuses it here, before the lock creates anything.
		if changing, err := changesFiles(run, opts); err != nil {
			return 0, err
		} else if changing {
			if err := store.Check(req.App); err != nil {
				return 0, err
			}
		}
	}
	var owned *ownership
	if !req.DryRun {
		awaitingLock()
		if owned, err = openOwnership(req.App, run.target); err != nil {
			return 0, err
		}
		defer owned.session.Close()
		// Another command may have changed the configuration while this one waited for the lock, so everything the
		// commit acts on is decided again under it.
		if run, err = prepare(req, app, themes, configDir, requested, selectsTheme, opts, owned.session.Manifest); err != nil {
			return 0, err
		}
		owned.target = run.target
		if err := owned.prepareWrites(run, opts); err != nil {
			return 0, err
		}
	}
	transaction, err := fsys.Apply(run.steps, opts)
	if owned != nil {
		err = owned.finish(transaction, err, func(complete bool) error {
			var disowns func(string) bool
			if complete && app.Disowns != nil {
				disowns = func(path string) bool { return app.Disowns(configDir, path) }
			}
			return owned.record(transaction.Results, complete && !run.decided.slot, disowns, func() (*catalog.Theme, error) {
				inspected, err := app.Inspect(themes, configDir)
				return inspected.Current, err
			})
		})
	}
	if err != nil {
		return 0, err
	}
	results := transaction.Results
	report(ctx, results, req.Verbose)
	label, err := selectionText(app, run.theme, run.how, themes)
	if err != nil {
		return 0, err
	}
	summarize(ctx, req.Command, req.App, app, run.decided, label, run.target, results)
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

// prepare inspects the app, gates its ownership, chooses the theme, plans and preflights the writes, and refuses one
// that would replace a file sf2 cannot show it wrote. It writes nothing.
func prepare(req Request, app adapters.App, themes []catalog.Theme, configDir string, requested *catalog.Theme,
	selectsTheme bool, opts fsys.Options, recorded store.Manifest) (prepared, error) {
	var run prepared
	var err error
	if run.inspection, err = app.Inspect(themes, configDir); err != nil {
		return run, err
	}
	run.decided = gate(run.inspection.Ownership, req.Adopt, selectsTheme)
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
	if run.destinations, err = fsys.Preflight(run.steps, opts); err != nil {
		return run, err
	}
	return run, claim(run, app, themes, configDir, recorded)
}

// readManifest reads the manifest without taking the lock, for the first decision only.
func readManifest() (store.Manifest, error) {
	path, err := store.Path()
	if err != nil {
		return store.Manifest{}, err
	}
	return store.Load(path)
}

// claim refuses a write that would replace a file sf2 cannot show it wrote. The configuration file at the target is
// the user's own and is edited with a backup; every other file sf2 writes must be missing, already hold what sf2
// writes, or hold bytes sf2 is known to have written there: recorded in the manifest with their SHA-256, or exactly
// what this sf2 writes at that path for some catalog selection, which is what releases before the manifest wrote. A
// name, such as an `sf2-` prefix, proves nothing.
func claim(run prepared, app adapters.App, themes []catalog.Theme, configDir string, recorded store.Manifest) error {
	var known map[string]map[string]bool
	var unproven []string
	for i, step := range run.steps {
		destination := run.destinations[i]
		if fsys.SamePath(destination, run.target) {
			continue
		}
		raw, err := os.ReadFile(destination)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if string(raw) == step.Content {
			continue
		}
		if text, err := pycompat.DecodeText(raw); err == nil && text == step.Content {
			continue
		}
		sha := fsys.Fingerprint(string(raw))
		if recorded.Wrote(destination, sha) {
			continue
		}
		if known == nil {
			known = renderings(app, themes, configDir, run.decided.slot)
		}
		if known[step.Path][sha] {
			continue
		}
		unproven = append(unproven, destination)
	}
	switch len(unproven) {
	case 0:
		return nil
	case 1:
		return fmt.Errorf("refusing to replace %s: sf2-themes did not write it, or it changed after sf2-themes wrote "+
			"it. Move it aside and run the command again", unproven[0])
	}
	return fmt.Errorf("refusing to replace %d files sf2-themes did not write, or that changed after sf2-themes wrote "+
		"them. Move them aside and run the command again:\n  %s", len(unproven), strings.Join(unproven, "\n  "))
}

// renderings is the SHA-256 of everything app writes at each path for any catalog selection.
func renderings(app adapters.App, themes []catalog.Theme, configDir string, slot bool) map[string]map[string]bool {
	known := map[string]map[string]bool{}
	for _, theme := range themes {
		steps, err := app.Plan(theme, themes, configDir, slot)
		if err != nil {
			continue
		}
		for _, step := range steps {
			if known[step.Path] == nil {
				known[step.Path] = map[string]bool{}
			}
			known[step.Path][fsys.Fingerprint(step.Content)] = true
		}
	}
	return known
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
	case !decided.slot:
		printf(ctx.Stdout, "%s %s: kept your own theme (config left unchanged)\n", verb, name)
	default:
		printf(ctx.Stdout, "%s %s: %s\n", verb, name, selection)
	}
	printf(ctx.Stdout, "  target: %s\n", target)
	if !decided.slot && app.Pointer {
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
	case inspection.Ownership == plan.Foreign && !app.Pointer:
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
	// kept is the copy of the user's setting this run kept, removed again when the run is undone.
	kept string
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

// awaitingLock runs as a command that writes is about to wait for the state lock, after its first, lock-free
// decision. Tests replace it to hold that moment.
var awaitingLock = func() {}

// prepareWrites readies the state for a commit that changes the application, before its first write: it keeps the
// user's prior setting and proves the manifest can be published afterwards. A commit that changes nothing writes no
// state, not even a temporary file.
func (o *ownership) prepareWrites(run prepared, opts fsys.Options) error {
	if changing, err := changesFiles(run, opts); err != nil || !changing {
		return err
	}
	if err := o.session.Stage(); err != nil {
		return err
	}
	if err := o.keepPrior(run.steps, run.destinations); err != nil {
		return err
	}
	return o.session.Stage()
}

// changesFiles reports whether committing run would change any file.
func changesFiles(run prepared, opts fsys.Options) (bool, error) {
	preview := opts
	preview.DryRun, preview.Verbose = true, false
	predicted, err := fsys.Commit(run.steps, preview)
	if err != nil {
		return false, err
	}
	return slices.ContainsFunc(predicted, func(r fsys.Result) bool { return r.Action.Planned() }), nil
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
		if !fsys.SamePath(destinations[i], o.target) {
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
		o.kept = entry.Prior
		o.session.Manifest.Record(entry)
		return o.session.Save()
	}
	return nil
}

// finish settles sf2's state after the commit. A complete commit is recorded; when the record cannot be saved, the
// commit is undone. A commit that failed has undone itself. Once the application files are as they were, the state
// is put back as well: the kept setting copy goes and the manifest returns to what the run found, so a failed run
// leaves nothing behind. Changes that could not be undone are recorded instead, so ownership is never lost.
func (o *ownership) finish(transaction *fsys.Transaction, err error, record func(complete bool) error) error {
	if err == nil && len(transaction.Results) > 0 {
		if err = record(true); err == nil {
			return nil
		}
		if rollbackErr := transaction.Rollback(); rollbackErr != nil {
			err = errors.Join(err, rollbackErr)
		}
	}
	if err == nil {
		return nil
	}
	if slices.ContainsFunc(transaction.Results, func(r fsys.Result) bool { return !r.Action.Planned() && r.Action != fsys.Unchanged }) {
		if recordErr := record(false); recordErr != nil {
			err = errors.Join(err, recordErr)
		}
		return err
	}
	if o.kept != "" {
		if removeErr := os.Remove(o.kept); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
			return errors.Join(err, removeErr)
		}
		o.kept = ""
	}
	if restoreErr := o.session.Restore(); restoreErr != nil {
		err = errors.Join(err, restoreErr)
	}
	return err
}

// record merges what one commit did into the target's entry and saves it. The file at the target is the user's
// own configuration, recorded as the setting sf2 edited; every other written file is wholly sf2's. A record the
// commit did not reach is kept as it was. releasedSetting drops the setting
// record after a complete run that left the user's own setting in place, and disowns, after a complete run, drops
// records the app says the target can no longer own, such as a pointer an earlier release shared; it judges the path
// alone, so a run after an interrupted migration drops the claim too. The selection is what the configuration
// selects after the commit, read back through inspect, and stays as recorded when it cannot be read.
//
// Running a command again after one was interrupted between its application writes and this save records what is on
// disk, so the manifest catches up.
func (o *ownership) record(results []fsys.Result, releasedSetting bool, disowns func(string) bool, current func() (*catalog.Theme, error)) error {
	entry := o.entry()
	files := map[string]string{}
	for _, file := range entry.Files {
		if disowns == nil || !disowns(file.Path) {
			files[file.Path] = file.SHA256
		}
	}
	for _, result := range results {
		switch {
		case fsys.SamePath(result.Path, o.target):
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
