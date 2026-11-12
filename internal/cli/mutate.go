package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/douglasjarquin/sf2-themes/v2/internal/adapters"
	"github.com/douglasjarquin/sf2-themes/v2/internal/adapters/slack"
	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
	"github.com/douglasjarquin/sf2-themes/v2/internal/fsys"
	"github.com/douglasjarquin/sf2-themes/v2/internal/plan"
	"github.com/douglasjarquin/sf2-themes/v2/internal/pycompat"
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

// selection is how a run arrived at the theme it writes.
type selection int

const (
	selected selection = iota
	kept
	noneFound
	reset
)

// decision is what the ownership gate allows for one run.
type decision struct {
	// refuse stops the run before anything is planned or written.
	refuse bool
	// slot lets the plan write the app's theme setting.
	slot bool
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

// refusal is the one message for an ownership class the gate refuses.
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

func runSetup(ctx *Context, req Request) (int, error) {
	return mutate(ctx, req)
}

func runApply(ctx *Context, req Request) (int, error) {
	return mutate(ctx, req)
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
	app, err := lookupApp(req.App)
	if err != nil {
		return 0, err
	}
	configDir, err := expandConfigDir(req)
	if err != nil {
		return 0, err
	}
	selectsTheme := req.Command != "setup" || requested != nil
	inspection, err := app.Inspect(themes, configDir)
	if err != nil {
		return 0, err
	}
	decided := gate(inspection.Ownership, app, req.Adopt, selectsTheme)
	if decided.refuse {
		return 0, refusal(req.App, app, inspection)
	}
	theme, how, err := choose(requested, selectsTheme, inspection, themes)
	if err != nil {
		return 0, err
	}
	steps, err := app.Plan(theme, themes, configDir, decided.slot)
	if err != nil {
		return 0, err
	}
	results, err := fsys.Commit(steps, fsys.Options{DryRun: req.DryRun, FollowSymlinks: req.FollowSymlinks})
	if err != nil {
		return 0, err
	}
	report(ctx, results, req.Verbose)
	label, err := selectionText(app, theme, how, themes)
	if err != nil {
		return 0, err
	}
	summarize(ctx, req.Command, req.App, app, decided, label, results)
	if decided.incomplete {
		return 0, refusal(req.App, app, inspection)
	}
	if app.StderrHint != "" && decided.slot {
		printf(ctx.Stderr, "%s\n", app.StderrHint)
	}
	return 0, nil
}

// expandConfigDir returns the expanded --config-dir, or "" when the flag is absent.
func expandConfigDir(req Request) (string, error) {
	if req.ConfigDir == nil {
		return "", nil
	}
	return pycompat.ExpandUser(*req.ConfigDir)
}

// choose returns the theme a run writes and how it was chosen.
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

func summarize(ctx *Context, verb, name string, app adapters.App, decided decision, selection string, results []fsys.Result) {
	switch {
	case decided.incomplete:
		printf(ctx.Stdout, "%s %s: incomplete (config left unchanged)\n", verb, name)
	case !decided.slot:
		printf(ctx.Stdout, "%s %s: kept your own theme (config left unchanged)\n", verb, name)
	default:
		printf(ctx.Stdout, "%s %s: %s\n", verb, name, selection)
	}
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
	app, err := lookupApp(req.App)
	if err != nil {
		return 0, err
	}
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
