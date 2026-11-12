// Package parity holds the case table, normalizer, and golden store that prove the Go
// sf2-themes binary prints and writes exactly what the Python reference does.
//
// The harness itself lives in the package tests. Goldens are recorded from the
// committed Python standalone and replayed against whichever binary is under test.
package parity

import "strings"

// Sandbox stands for the case's sandbox root, resolved through symlinks, in arguments,
// environment values, seed files, and everything a golden records.
const Sandbox = "{SANDBOX}"

// Group is a family of cases the Go binary gains together, one adapter or command family at a time.
type Group string

// The groups a case can belong to.
const (
	GroupCLI      Group = "cli"
	GroupCatalog  Group = "catalog"
	GroupSlack    Group = "slack"
	GroupWezterm  Group = "wezterm"
	GroupNvim     Group = "nvim"
	GroupHerdr    Group = "herdr"
	GroupHerdrApp Group = "herdr-app"
	GroupStarship Group = "starship"
	GroupClaude   Group = "claude"
	GroupCodex    Group = "codex"
	GroupLazygit  Group = "lazygit"
)

// PendingInGo names, per group, the capability the Go binary still lacks. A case whose
// groups appear here is skipped against the Go binary with that reason. Delete a group's
// line when the Go binary gains it; delete the map and its one use once it is empty.
var PendingInGo = map[Group]string{
	GroupSlack:    "slack paste line is not ported to Go yet",
	GroupWezterm:  "wezterm adapter not ported to Go yet",
	GroupNvim:     "nvim adapter not ported to Go yet",
	GroupHerdr:    "herdr adapter not ported to Go yet",
	GroupHerdrApp: "herdr-app adapter not ported to Go yet",
	GroupStarship: "starship adapter not ported to Go yet",
	GroupClaude:   "claude adapter not ported to Go yet",
	GroupCodex:    "codex adapter not ported to Go yet",
	GroupLazygit:  "lazygit adapter not ported to Go yet",
}

// Stream names one output stream of a step.
type Stream string

// The two output streams.
const (
	Stdout Stream = "stdout"
	Stderr Stream = "stderr"
)

// Mask exempts output the Go binary is allowed to print differently.
//
// A tail mask replaces the text after the first occurrence of After on each line of
// Stream that contains it; the text up to and including After is still compared, so
// a changed fixed prefix still fails. A whole mask exempts the stream and is reserved
// for help and usage text.
type Mask struct {
	Stream Stream
	After  string
	Whole  bool
	// ErrorLine keeps only the `<prog>: error: <message>` line of the stream.
	ErrorLine bool
}

// HelpText exempts stdout of a help request: help text is rewritten in Go and does not
// depend on terminal width. The exit code and stderr are still compared.
func HelpText() Mask { return Mask{Stream: Stdout, Whole: true} }

// UsageError drops the usage text from stderr of a usage error, because it is rewritten in
// Go, and still compares the `<prog>: error: <message>` line, the exit code, which must stay
// 2, and stdout.
func UsageError() Mask { return Mask{Stream: Stderr, ErrorLine: true} }

// VersionLine masks the version line: the Go binary prints its own release version.
func VersionLine() Mask { return Mask{Stream: Stdout} }

// Tail masks stderr text after a decode or operating-system error's fixed prefix,
// which ends with after.
func Tail(after string) Mask { return Mask{Stream: Stderr, After: after} }

// Markers that end the fixed prefix of a reference error whose tail may differ.
const (
	tomlTail = " is not valid TOML: "
	jsonTail = " is not valid JSON: "
	// errorTail ends the prefix of an operating-system error, or of a TOML decode
	// error the reference reports with no prefix of its own.
	errorTail = "error: "
)

// Step is one invocation of the binary under test.
type Step struct {
	Args  []string
	Masks []Mask
}

// File is one entry seeded into a sandbox before the first step. Path is relative to the
// sandbox root. Link, when set, makes the entry a symlink to that target. Mode defaults to 0644.
type File struct {
	Path    string
	Content string
	Mode    uint32
	Link    string
}

// GoExpectation is what the Go binary must do where the reference behaves differently
// on purpose. Such a case has no golden: it never runs against the reference.
type GoExpectation struct {
	Exit         int
	StderrPrefix string
}

// Case is one sandboxed scenario.
type Case struct {
	Name   string
	Groups []Group
	// Env sets variables on top of the cleared environment. Every adapter variable is unset unless set here.
	Env  map[string]string
	Seed []File
	// SeedFrom names a recorded case whose golden tree seeds this sandbox before Seed is applied.
	SeedFrom string
	Steps    []Step
	// PerOS records one golden per operating system, for the one default-path case that differs by OS.
	PerOS bool
	// NoChanges requires the last step to report no changes and leave no backup file.
	NoChanges bool
	// GuardRealHome checks that the real home directory of the test process is untouched.
	GuardRealHome bool
	// GoOnly holds the Go binary's expectation where the reference has none to record.
	GoOnly *GoExpectation
}

func step(args ...string) Step { return Step{Args: args} }

func (s Step) masked(masks ...Mask) Step {
	s.Masks = append(s.Masks, masks...)
	return s
}

func cfg(app string) string { return Sandbox + "/config/" + app }

// cmd builds `<verb> <app> [extra...] --config-dir <sandbox>/config/<app>`.
func cmd(verb, app string, extra ...string) Step {
	args := append([]string{verb, app}, extra...)
	return step(append(args, "--config-dir", cfg(app))...)
}

func seed(app, name, content string) File {
	return File{Path: "config/" + app + "/" + name, Content: content}
}

func lines(text ...string) string { return strings.Join(text, "\n") + "\n" }

// Cases returns the whole case table.
func Cases() []Case {
	var cases []Case
	cases = append(cases, cliCases()...)
	cases = append(cases, catalogCases()...)
	cases = append(cases, slackCases()...)
	cases = append(cases, gateCases()...)
	cases = append(cases, scenarioCases()...)
	cases = append(cases, locationCases()...)
	cases = append(cases, goOnlyCases()...)
	return cases
}

func cliCases() []Case {
	cli := []Group{GroupCLI}
	usage := func(name string, args ...string) Case {
		return Case{Name: "cli/" + name, Groups: cli, Steps: []Step{step(args...).masked(UsageError())}}
	}
	return []Case{
		{Name: "cli/version", Groups: cli, Steps: []Step{step("--version").masked(VersionLine())}},
		{Name: "cli/no-command", Groups: cli, Steps: []Step{step().masked(HelpText())}},
		{Name: "cli/help", Groups: cli, Steps: []Step{step("--help").masked(HelpText())}},
		{Name: "cli/apply-help", Groups: cli, Steps: []Step{step("apply", "--help").masked(HelpText())}},
		// The command list no longer offers the removed install command.
		{Name: "cli/unknown-command", Groups: cli, Steps: []Step{step("frob").masked(UsageError(), Tail(" (choose from "))}},
		usage("apply-without-app", "apply"),
		usage("apply-unknown-app", "apply", "vim"),
		usage("apply-unknown-flag", "apply", "codex", "--bogus"),
		usage("apply-theme-without-value", "apply", "codex", "--theme"),
		usage("show-without-theme", "show"),
		usage("current-without-app", "current"),
		usage("validate-names-after-an-option", "validate", "ryu", "--all", "ken"),
		usage("double-dash-without-positional", "apps", "--", "x"),
		usage("combined-switch-with-unknown-char", "apply", "codex", "-vx"),
		usage("combined-switch-with-dash", "apply", "codex", "-v-"),
		{Name: "cli/combined-short-help", Groups: cli, Steps: []Step{step("apply", "codex", "-vh").masked(HelpText())}},
		{Name: "cli/combined-top-level-help", Groups: cli, Steps: []Step{step("-hV").masked(HelpText())}},
		{Name: "cli/double-dash-inside-names", Groups: []Group{GroupCLI, GroupCatalog}, Steps: []Step{step("validate", "ryu", "--", "ken")}},
		{Name: "cli/double-dash-after-name", Groups: []Group{GroupCLI, GroupCatalog}, Steps: []Step{step("show", "ryu", "--")}},
		{Name: "cli/double-dash-names-an-option", Groups: []Group{GroupCLI, GroupCatalog}, Steps: []Step{step("validate", "ryu", "--", "--all")}},
	}
}

func catalogCases() []Case {
	catalog := []Group{GroupCatalog}
	one := func(name string, args ...string) Case {
		return Case{Name: "catalog/" + name, Groups: catalog, Steps: []Step{step(args...)}}
	}
	return []Case{
		one("apps", "apps"),
		one("themes", "themes"),
		one("show-main", "show", "main"),
		one("show-light-sibling", "show", "ryu-light"),
		one("show-alias", "show", "street-fighter-2"),
		one("show-display-name", "show", "Street Fighter II - Chun-Li"),
		one("show-case-insensitive", "show", "RYU"),
		one("show-unknown", "show", "nope"),
		one("validate", "validate"),
		one("validate-all", "validate", "--all"),
		one("validate-named", "validate", "main", "ryu"),
		one("validate-light-sibling", "validate", "ryu-light"),
		one("validate-unknown", "validate", "nope"),
	}
}

func slackCases() []Case {
	slack := []Group{GroupSlack}
	one := func(name string, args ...string) Case {
		return Case{Name: "slack/" + name, Groups: slack, Steps: []Step{step(args...)}}
	}
	return []Case{
		one("apply", "apply", "slack"),
		one("apply-theme", "apply", "slack", "--theme", "ken"),
		one("apply-light-sibling", "apply", "slack", "--theme", "ken-light"),
		one("setup", "setup", "slack"),
		one("apply-dry-run", "apply", "slack", "--dry-run"),
		one("apply-unknown-theme", "apply", "slack", "--theme", "nope"),
		// The copied-CLI shell test: a config location writes nothing.
		{Name: "slack/apply-config-dir", Groups: slack, Steps: []Step{cmd("apply", "slack", "--theme", "ken")}},
		{Name: "slack/current", Groups: slack, Steps: []Step{cmd("current", "slack")}},
	}
}

// gateCommand is one way of invoking a file-writing app against its ownership state.
type gateCommand struct {
	name  string
	verb  string
	extra []string
}

var gateCommands = []gateCommand{
	{"setup", "setup", nil},
	{"setup-theme", "setup", []string{"--theme", "ken"}},
	{"apply", "apply", nil},
	{"apply-theme", "apply", []string{"--theme", "ken"}},
	{"adopt", "apply", []string{"--adopt"}},
	{"dry-run", "apply", []string{"--theme", "ken", "--dry-run"}},
	{"verbose", "apply", []string{"--theme", "ken", "--verbose"}},
	{"current", "current", nil},
}

// Command subsets for state variants that need less than the full cross product.
var (
	allCommands     = []string{"setup", "setup-theme", "apply", "apply-theme", "adopt", "dry-run", "verbose", "current"}
	refusalCommands = []string{"setup", "apply", "current"}
	writeCommands   = []string{"setup", "apply", "adopt", "dry-run"}
)

// gateState is one ownership state of an app's config, as a seed.
type gateState struct {
	name     string
	files    []File
	seedFrom bool // seed from the app's managed-source golden
	commands []string
	masks    []Mask
}

// gateApp is a file-writing app and the ownership states it has.
type gateApp struct {
	name   string
	group  Group
	file   string // the config file holding the theme setting; empty when the app owns every file it writes
	states []gateState
}

func builderLua(body string) string {
	return lines(
		`local wezterm = require("wezterm")`,
		"local config = wezterm.config_builder()",
		body,
		"return config",
	)
}

const pointerPath = Sandbox + "/home/.config/sf2-theme/wezterm-current.lua"

func gateApps() []gateApp {
	toml := func(app, file string) []Mask { return []Mask{Tail(cfg(app) + "/" + file + tomlTail)} }
	json := func(app, file string) []Mask { return []Mask{Tail(cfg(app) + "/" + file + jsonTail)} }
	return []gateApp{
		{name: "wezterm", group: GroupWezterm, file: "wezterm.lua", states: []gateState{
			{name: "absent", commands: allCommands},
			{name: "absent-builder", files: []File{seed("wezterm", "wezterm.lua", builderLua("config.font_size = 13"))}, commands: writeCommands},
			{name: "managed", seedFrom: true, commands: allCommands},
			{name: "legacy-scheme", files: []File{seed("wezterm", "wezterm.lua", builderLua(`config.color_scheme = "sf2-ryu"`))}, commands: allCommands},
			{name: "legacy-pointer", files: []File{
				seed("wezterm", "wezterm.lua", builderLua(strings.Join([]string{
					`local sf2_current = "` + pointerPath + `"`,
					"wezterm.add_to_config_reload_watch_list(sf2_current)",
					"config.color_scheme = dofile(sf2_current)",
				}, "\n"))),
				{Path: "home/.config/sf2-theme/wezterm-current.lua", Content: lines("-- sf2-themes: sf2-ryu", `return "sf2-ryu"`)},
			}, commands: allCommands},
			{name: "foreign", files: []File{seed("wezterm", "wezterm.lua", builderLua(`config.color_scheme = "Builtin Dark"`))}, commands: allCommands},
			{name: "unknown-shape", files: []File{seed("wezterm", "wezterm.lua", "return {\n  font_size = 13,\n}\n")}, commands: allCommands},
			{name: "legacy-files", files: []File{
				seed("wezterm", "colors/street-fighter-ii-ryu.toml", "old managed scheme\n"),
				seed("wezterm", "colors/street-fighter-ii-main.toml", "old managed scheme\n"),
			}, commands: []string{"apply", "dry-run", "verbose"}},
		}},
		{name: "nvim", group: GroupNvim, states: []gateState{
			{name: "absent", commands: allCommands},
			{name: "managed", seedFrom: true, commands: allCommands},
			{name: "managed-unreadable", files: []File{seed("nvim", "sf2-theme/current.lua", lines("-- sf2-themes: sf2-nope"))}, commands: []string{"setup", "apply", "current"}},
			{name: "legacy-files", files: []File{seed("nvim", "colors/street-fighter-ii-ryu.lua", "old managed scheme\n")}, commands: []string{"apply", "dry-run", "verbose"}},
		}},
		{name: "herdr", group: GroupHerdr, file: "config.toml", states: []gateState{
			{name: "absent", commands: allCommands},
			{name: "absent-other-settings", files: []File{seed("herdr", "config.toml", lines("[ui]", `theme = "follow-system"`))}, commands: writeCommands},
			{name: "managed", seedFrom: true, commands: allCommands},
			{name: "managed-unreadable", files: []File{seed("herdr", "config.toml", lines("# >>> sf2-themes managed theme", "# sf2-themes: sf2-nope", "[theme]", `name = "catppuccin"`, "# <<< sf2-themes managed theme"))}, commands: []string{"setup", "apply", "current"}},
			{name: "foreign", files: []File{seed("herdr", "config.toml", lines("[theme]", `name = "dracula"`))}, commands: allCommands},
			{name: "foreign-top-level", files: []File{seed("herdr", "config.toml", lines(`theme = "dracula"`, "", "[ui]", "sidebar = true"))}, commands: []string{"setup", "apply", "adopt", "current"}},
			{name: "broken", files: []File{seed("herdr", "config.toml", "[theme\n")}, commands: allCommands, masks: toml("herdr", "config.toml")},
			{name: "broken-markers", files: []File{seed("herdr", "config.toml", lines("# >>> sf2-themes managed theme", "[theme]", `name = "x"`))}, commands: refusalCommands},
		}},
		{name: "herdr-app", group: GroupHerdrApp, file: "settings.json", states: []gateState{
			{name: "absent", commands: allCommands},
			{name: "absent-other-settings", files: []File{seed("herdr-app", "settings.json", `{"fontSize": 14}`)}, commands: writeCommands},
			{name: "managed", seedFrom: true, commands: allCommands},
			{name: "managed-extra-keys", files: []File{seed("herdr-app", "settings.json", `{"theme": {"dark": "sf2-ryu", "light": "sf2-ryu-light", "mode": "auto"}, "fontSize": 14}`)}, commands: []string{"setup", "apply", "current"}},
			{name: "foreign", files: []File{seed("herdr-app", "settings.json", `{"theme": {"dark": "graphite", "light": "paper"}}`)}, commands: allCommands},
			{name: "foreign-name", files: []File{seed("herdr-app", "settings.json", `{"theme": "graphite"}`)}, commands: []string{"setup", "apply", "adopt", "current"}},
			{name: "foreign-mixed-pair", files: []File{seed("herdr-app", "settings.json", `{"theme": {"dark": "sf2-ryu", "light": "sf2-ken-light"}}`)}, commands: []string{"setup", "apply", "adopt", "current"}},
			{name: "broken", files: []File{seed("herdr-app", "settings.json", "{")}, commands: allCommands, masks: json("herdr-app", "settings.json")},
			{name: "broken-not-object", files: []File{seed("herdr-app", "settings.json", "[]\n")}, commands: refusalCommands},
			{name: "broken-theme-type", files: []File{seed("herdr-app", "settings.json", `{"theme": 3}`)}, commands: refusalCommands},
		}},
		{name: "starship", group: GroupStarship, file: "starship.toml", states: []gateState{
			{name: "absent", commands: allCommands},
			{name: "absent-tables", files: []File{seed("starship", "starship.toml", lines("add_newline = false", "", "[directory]", `style = "blue"`))}, commands: writeCommands},
			{name: "absent-multiline-format", files: []File{seed("starship", "starship.toml", lines(`format = """`, "[a]", `$all"""`, "", "# directory", "[directory]", `style = "blue"`))}, commands: []string{"apply"}},
			{name: "managed", seedFrom: true, commands: allCommands},
			{name: "managed-after-tables", files: []File{seed("starship", "starship.toml", lines(
				"[directory]", `style = "blue"`, "",
				"# >>> sf2-themes managed theme", "# sf2-themes: sf2-ken", `palette = "sf2"`, "", "[palettes.sf2]", `blue = "#000000"`, "# <<< sf2-themes managed theme",
			))}, commands: []string{"setup", "apply", "current"}},
			{name: "foreign", files: []File{seed("starship", "starship.toml", lines(`palette = "mine"`, "", "[palettes.mine]", `blue = "#0000ff"`))}, commands: allCommands},
			{name: "broken", files: []File{seed("starship", "starship.toml", "palette = \n")}, commands: allCommands, masks: toml("starship", "starship.toml")},
			{name: "broken-markers", files: []File{seed("starship", "starship.toml", "# <<< sf2-themes managed theme\n")}, commands: refusalCommands},
			// A config tomllib accepts whose merge declares the palette table twice.
			{name: "merge-collision", files: []File{seed("starship", "starship.toml", "palettes.sf2.x = 1\n")}, commands: []string{"apply"}, masks: []Mask{Tail(errorTail)}},
		}},
		{name: "claude", group: GroupClaude, file: "settings.json", states: []gateState{
			{name: "absent", commands: allCommands},
			{name: "absent-empty-object", files: []File{seed("claude", "settings.json", "{}\n")}, commands: writeCommands},
			{name: "absent-other-settings", files: []File{seed("claude", "settings.json", `{"model": "opus", "nested": {"a": [1, 2.5, null, true]}, "unicode": "café 😀"}`)}, commands: writeCommands},
			{name: "managed", seedFrom: true, commands: allCommands},
			{name: "managed-unreadable", files: []File{seed("claude", "settings.json", `{"theme": "custom:sf2-not-a-theme"}`)}, commands: []string{"setup", "apply", "current"}},
			{name: "managed-private-mode", files: []File{{Path: "config/claude/settings.json", Content: `{"theme": "custom:sf2-ryu"}`, Mode: 0o600}}, commands: []string{"apply"}},
			{name: "foreign", files: []File{seed("claude", "settings.json", `{"theme": "dark"}`)}, commands: allCommands},
			{name: "broken", files: []File{seed("claude", "settings.json", "{")}, commands: allCommands, masks: json("claude", "settings.json")},
			{name: "broken-not-object", files: []File{seed("claude", "settings.json", "[1]\n")}, commands: refusalCommands},
		}},
		{name: "codex", group: GroupCodex, file: "config.toml", states: []gateState{
			{name: "absent", commands: allCommands},
			{name: "absent-tui-table", files: []File{seed("codex", "config.toml", lines(`model = "o3"`, "", "[tui]", "animations = false", "", "[other]", "x = 1"))}, commands: writeCommands},
			{name: "managed", seedFrom: true, commands: allCommands},
			{name: "legacy", files: []File{seed("codex", "config.toml", lines("[tui]", `theme = "ryu"`))}, commands: allCommands},
			{name: "foreign", files: []File{seed("codex", "config.toml", lines("[tui]", `theme = "dracula"`))}, commands: allCommands},
			{name: "foreign-with-context", files: []File{seed("codex", "config.toml", lines("# mine", `model = "o3"`, "", "[tui]", `theme = "dracula"`, "animations = false"))}, commands: []string{"adopt"}},
			{name: "unknown-shape", files: []File{seed("codex", "config.toml", lines(`tui = { theme = "dracula" }`))}, commands: allCommands},
			{name: "unknown-shape-dotted", files: []File{seed("codex", "config.toml", lines(`tui.theme = "dracula"`))}, commands: refusalCommands},
			{name: "broken", files: []File{seed("codex", "config.toml", "[tui\n")}, commands: allCommands, masks: toml("codex", "config.toml")},
			{name: "legacy-files", files: []File{seed("codex", "themes/ryu.tmTheme", "old managed theme\n")}, commands: []string{"apply", "dry-run"}},
		}},
		{name: "lazygit", group: GroupLazygit, file: "config.yml", states: []gateState{
			{name: "absent", commands: allCommands},
			{name: "absent-other-settings", files: []File{seed("lazygit", "config.yml", "gui:\n  sidePanelWidth: 0.3\n\nnotATheme: true\n")}, commands: writeCommands},
			{name: "absent-author-colors", files: []File{seed("lazygit", "config.yml", lines("gui:", "  authorColors:", "    'alice': '#ff0000'", "    '*': '#00ff00'", "git:", "  paging: {}"))}, commands: []string{"apply"}},
			{name: "managed", seedFrom: true, commands: allCommands},
			{name: "foreign", files: []File{seed("lazygit", "config.yml", "gui:\n  theme:\n    activeBorderColor:\n      - red\n")}, commands: allCommands},
			{name: "foreign-with-authors", files: []File{seed("lazygit", "config.yml", lines("gui:  # interface", "  theme:", "    activeBorderColor:", "      - red", "  authorColors:", "    'bob': '#0000ff'", "os:", "  edit: vim"))}, commands: []string{"adopt"}},
			{name: "broken", files: []File{seed("lazygit", "config.yml", "gui:\n  # >>> sf2-themes managed theme\n")}, commands: allCommands},
		}},
	}
}

// gateCases crosses each file-writing app with its ownership states and the gate's commands,
// and adds the managed-state source, the no-changes check over a reference-written tree,
// and the symlink policy per app.
func gateCases() []Case {
	commands := map[string]gateCommand{}
	for _, command := range gateCommands {
		commands[command.name] = command
	}
	var cases []Case
	for _, app := range gateApps() {
		groups := []Group{app.group}
		source := app.name + "/managed-source"
		cases = append(cases,
			Case{Name: source, Groups: groups, Steps: []Step{cmd("apply", app.name, "--theme", "ryu")}},
			// A tree the reference wrote, applied again with the same arguments.
			Case{Name: app.name + "/managed/apply-same", Groups: groups, SeedFrom: source, NoChanges: true,
				Steps: []Step{cmd("apply", app.name, "--theme", "ryu")}},
		)
		for _, state := range app.states {
			for _, name := range state.commands {
				command := commands[name]
				c := Case{
					Name:   app.name + "/" + state.name + "/" + command.name,
					Groups: groups,
					Seed:   state.files,
					Steps:  []Step{cmd(command.verb, app.name, command.extra...).masked(state.masks...)},
				}
				if state.seedFrom {
					c.SeedFrom = source
				}
				cases = append(cases, c)
			}
		}
		cases = append(cases, symlinkCases(app)...)
	}
	return cases
}

func symlinkCases(app gateApp) []Case {
	groups := []Group{app.group}
	file := app.file
	content := ""
	switch app.name {
	case "nvim":
		file, content = "sf2-theme/current.lua", lines("-- sf2-themes: sf2-ryu")
	case "wezterm":
		content = builderLua("config.font_size = 13")
	case "herdr", "codex", "starship":
		content = lines("# kept")
	case "herdr-app", "claude":
		content = "{}\n"
	case "lazygit":
		content = lines("os:", "  edit: vim")
	}
	target := "elsewhere/" + app.name + "/" + file
	files := []File{
		{Path: target, Content: content},
		{Path: "config/" + app.name + "/" + file, Link: Sandbox + "/" + target},
	}
	return []Case{
		{Name: app.name + "/symlink/apply", Groups: groups, Seed: files, Steps: []Step{cmd("apply", app.name)}},
		{Name: app.name + "/symlink/follow-symlinks", Groups: groups, Seed: files, Steps: []Step{cmd("apply", app.name, "--follow-symlinks")}},
		{Name: app.name + "/symlink/follow-symlinks-dry-run", Groups: groups, Seed: files, Steps: []Step{cmd("apply", app.name, "--follow-symlinks", "--dry-run")}},
	}
}

var fileApps = []struct {
	name  string
	group Group
}{
	{"wezterm", GroupWezterm}, {"herdr", GroupHerdr}, {"herdr-app", GroupHerdrApp}, {"nvim", GroupNvim},
	{"codex", GroupCodex}, {"starship", GroupStarship}, {"lazygit", GroupLazygit}, {"claude", GroupClaude},
}

// scenarioCases carries over the multi-step black-box scenarios from the Python tests and the
// copied-CLI shell test.
func scenarioCases() []Case {
	var cases []Case
	for _, app := range fileApps {
		groups := []Group{app.group}
		cases = append(cases,
			Case{Name: app.name + "/repeat-apply", Groups: groups, NoChanges: true, Steps: []Step{
				cmd("apply", app.name, "--theme", "ryu"),
				cmd("apply", app.name, "--theme", "ryu"),
			}},
			Case{Name: app.name + "/kept-selection-is-current", Groups: groups, Steps: []Step{
				cmd("apply", app.name, "--theme", "chun-li-light"),
				cmd("setup", app.name),
				cmd("current", app.name),
			}},
		)
	}
	for _, app := range []struct {
		name  string
		group Group
	}{{"codex", GroupCodex}, {"claude", GroupClaude}, {"lazygit", GroupLazygit}} {
		cases = append(cases, Case{Name: app.name + "/setup-then-light-sibling", Groups: []Group{app.group}, Steps: []Step{
			cmd("setup", app.name),
			cmd("apply", app.name, "--theme", "ryu-light"),
			cmd("current", app.name),
		}})
	}
	wezterm := []Group{GroupWezterm}
	xdg := map[string]string{"XDG_CONFIG_HOME": Sandbox + "/xdg"}
	pointer := Sandbox + "/xdg/sf2-theme/wezterm-current.lua"
	snippet := strings.Join([]string{
		`local sf2_current = "` + pointer + `"`,
		"wezterm.add_to_config_reload_watch_list(sf2_current)",
		"local sf2_scheme = dofile(sf2_current)",
		"local sf2_env = {}",
		`sf2_env.TERM_THEME = "dark"`,
	}, "\n")
	pasted := lines(`local wezterm = require("wezterm")`, snippet, "return { color_scheme = sf2_scheme }")
	cases = append(cases,
		Case{Name: "wezterm/light-sibling-current-without-config-dir", Groups: wezterm, Env: xdg, Steps: []Step{
			cmd("apply", "wezterm", "--theme", "ryu-light"),
			step("current", "wezterm"),
		}},
		Case{Name: "wezterm/setup-then-reapply", Groups: wezterm, Seed: []File{seed("wezterm", "wezterm.lua", lines(`local wezterm = require("wezterm")`, "local config = wezterm.config_builder()", "return config"))}, NoChanges: true, Steps: []Step{
			cmd("setup", "wezterm"),
			cmd("apply", "wezterm"),
		}},
		Case{Name: "wezterm/pasted-snippet-apply", Groups: wezterm, Env: xdg, Seed: []File{seed("wezterm", "wezterm.lua", pasted)}, Steps: []Step{cmd("apply", "wezterm")}},
		Case{Name: "wezterm/pasted-snippet-setup", Groups: wezterm, Env: xdg, Seed: []File{seed("wezterm", "wezterm.lua", pasted)}, Steps: []Step{cmd("setup", "wezterm")}},
		Case{Name: "wezterm/commented-out-integration", Groups: wezterm, Env: xdg, Seed: []File{seed("wezterm", "wezterm.lua", builderLua(`-- local sf2_current = "`+pointer+`" -- dofile(sf2_current) TERM_THEME`))}, Steps: []Step{cmd("apply", "wezterm")}},
		Case{Name: "wezterm/unreadable-pointer", Groups: wezterm, Env: xdg, Seed: []File{{Path: "xdg/sf2-theme/wezterm-current.lua", Content: lines(`return "sf2-ryu"`)}}, Steps: []Step{
			cmd("setup", "wezterm"),
			cmd("current", "wezterm"),
		}},
		Case{Name: "wezterm/foreign-setup-then-current", Groups: wezterm, Env: xdg, Seed: []File{seed("wezterm", "wezterm.lua", builderLua(`config.color_scheme = "Builtin Dark"`))}, Steps: []Step{
			cmd("setup", "wezterm"),
			cmd("current", "wezterm"),
		}},
		Case{Name: "wezterm/unknown-shape-theme", Groups: wezterm, Env: xdg, Seed: []File{seed("wezterm", "wezterm.lua", "return {\n  font_size = 13,\n}\n")}, Steps: []Step{cmd("apply", "wezterm", "--theme", "ryu")}},
		Case{Name: "nvim/setup-keeps-pointer", Groups: []Group{GroupNvim}, Steps: []Step{
			cmd("apply", "nvim", "--theme", "ryu-light"),
			cmd("setup", "nvim"),
		}},
		Case{Name: "codex/legacy-setup-verbose", Groups: []Group{GroupCodex}, Seed: []File{seed("codex", "config.toml", lines("[tui]", `theme = "ryu"`))}, Steps: []Step{cmd("setup", "codex", "--verbose")}},
		Case{Name: "herdr/foreign-setup-adopt", Groups: []Group{GroupHerdr}, Seed: []File{seed("herdr", "config.toml", lines("[theme]", `name = "dracula"`))}, Steps: []Step{cmd("setup", "herdr", "--adopt")}},
		Case{Name: "herdr/light-sibling-current", Groups: []Group{GroupHerdr}, Steps: []Step{
			cmd("apply", "herdr", "--theme", "chun-li-light"),
			cmd("current", "herdr"),
		}},
		Case{Name: "herdr-app/light-sibling-setup-current", Groups: []Group{GroupHerdrApp}, Steps: []Step{
			cmd("apply", "herdr-app", "--theme", "ryu-light"),
			cmd("setup", "herdr-app"),
			cmd("current", "herdr-app"),
		}},
		Case{Name: "starship/apply-then-theme", Groups: []Group{GroupStarship}, Seed: []File{seed("starship", "starship.toml", lines("add_newline = false", "", "[directory]", `style = "blue"`))}, Steps: []Step{
			cmd("apply", "starship"),
			cmd("apply", "starship", "--theme", "ken"),
		}},
		Case{Name: "starship/adopt-harmless", Groups: []Group{GroupStarship}, Steps: []Step{cmd("apply", "starship", "--adopt")}},
		Case{Name: "nvim/adopt-harmless", Groups: []Group{GroupNvim}, Steps: []Step{cmd("apply", "nvim", "--adopt")}},
		Case{Name: "codex/config-dir-is-a-file", Groups: []Group{GroupCodex}, Seed: []File{{Path: "config/codex", Content: "not a directory\n"}}, Steps: []Step{cmd("apply", "codex").masked(Tail(errorTail))}},
		Case{Name: "codex/unknown-theme", Groups: []Group{GroupCodex}, Steps: []Step{cmd("apply", "codex", "--theme", "nope")}},
		Case{Name: "codex/relative-config-dir", Groups: []Group{GroupCodex}, Steps: []Step{step("apply", "codex", "--config-dir", "relative/codex", "--dry-run")}},
		Case{Name: "codex/home-relative-config-dir", Groups: []Group{GroupCodex}, Steps: []Step{step("apply", "codex", "--config-dir", "~/codex")}},
		Case{Name: "codex/config-dir-symlink", Groups: []Group{GroupCodex}, Seed: []File{
			{Path: "elsewhere/codex/config.toml", Content: lines("[tui]", `theme = "sf2-ryu"`)},
			{Path: "config/codex", Link: Sandbox + "/elsewhere/codex"},
		}, Steps: []Step{cmd("apply", "codex")}},
	)
	return cases
}

// locationCases select each app's config location through its environment variables, then
// cover the default locations: the isolation scenario and the one Lazygit default path that
// differs by operating system.
func locationCases() []Case {
	env := func(name, group string, g Group, vars map[string]string, args ...string) Case {
		return Case{Name: group + "/env-" + name, Groups: []Group{g}, Env: vars, Steps: []Step{step(args...)}}
	}
	xdg := Sandbox + "/xdg"
	return []Case{
		env("xdg-config-home", "wezterm", GroupWezterm, map[string]string{"XDG_CONFIG_HOME": xdg}, "apply", "wezterm"),
		env("wezterm-config-dir", "wezterm", GroupWezterm, map[string]string{"XDG_CONFIG_HOME": xdg, "WEZTERM_CONFIG_DIR": xdg + "/wezterm"}, "apply", "wezterm"),
		env("wezterm-config-file", "wezterm", GroupWezterm, map[string]string{"WEZTERM_CONFIG_FILE": Sandbox + "/lua/custom.lua"}, "apply", "wezterm"),
		env("xdg-config-home", "herdr", GroupHerdr, map[string]string{"XDG_CONFIG_HOME": xdg}, "apply", "herdr"),
		env("herdr-config-path", "herdr", GroupHerdr, map[string]string{"HERDR_CONFIG_PATH": Sandbox + "/herdr.toml"}, "apply", "herdr"),
		env("xdg-config-home", "herdr-app", GroupHerdrApp, map[string]string{"XDG_CONFIG_HOME": xdg}, "apply", "herdr-app"),
		env("xdg-config-home", "nvim", GroupNvim, map[string]string{"XDG_CONFIG_HOME": xdg}, "apply", "nvim"),
		env("nvim-config-dir", "nvim", GroupNvim, map[string]string{"NVIM_CONFIG_DIR": Sandbox + "/nvim"}, "apply", "nvim"),
		env("xdg-config-home", "starship", GroupStarship, map[string]string{"XDG_CONFIG_HOME": xdg}, "apply", "starship"),
		env("starship-config", "starship", GroupStarship, map[string]string{"STARSHIP_CONFIG": Sandbox + "/prompt.toml"}, "apply", "starship"),
		env("codex-home", "codex", GroupCodex, map[string]string{"CODEX_HOME": Sandbox + "/codex"}, "apply", "codex"),
		env("claude-config-dir", "claude", GroupClaude, map[string]string{"CLAUDE_CONFIG_DIR": Sandbox + "/claude"}, "apply", "claude"),
		env("lazygit-config-dir", "lazygit", GroupLazygit, map[string]string{"LAZYGIT_CONFIG_DIR": Sandbox + "/lazygit"}, "apply", "lazygit"),
		{Name: "wezterm/home-dotfile", Groups: []Group{GroupWezterm}, Seed: []File{{Path: "home/.wezterm.lua", Content: builderLua("config.font_size = 13")}}, Steps: []Step{step("apply", "wezterm")}},
		{Name: "isolation/default-locations", GuardRealHome: true,
			Groups: []Group{GroupWezterm, GroupHerdr, GroupHerdrApp, GroupNvim, GroupCodex, GroupStarship, GroupClaude},
			Steps: []Step{
				step("apply", "wezterm"), step("apply", "herdr"), step("apply", "herdr-app"), step("apply", "nvim"),
				step("apply", "codex"), step("apply", "starship"), step("apply", "claude"),
			}},
		{Name: "lazygit/default-location", PerOS: true, GuardRealHome: true, Groups: []Group{GroupLazygit}, Steps: []Step{
			step("apply", "lazygit"),
			step("current", "lazygit"),
		}},
	}
}

// goOnlyCases hold where the Go binary departs from the reference on purpose: removed
// commands and flags, exact flag names, and an error line instead of a traceback for a
// config file that is not valid UTF-8.
func goOnlyCases() []Case {
	usage := &GoExpectation{Exit: 2}
	notUTF8 := &GoExpectation{Exit: 1, StderrPrefix: "error: "}
	cases := []Case{
		{Name: "cli/install-is-gone", Groups: []Group{GroupCLI}, GoOnly: usage, Steps: []Step{cmd("install", "codex")}},
		{Name: "cli/current-theme-is-gone", Groups: []Group{GroupCLI}, GoOnly: usage, Steps: []Step{cmd("current", "codex", "--theme", "ryu")}},
		{Name: "cli/flag-abbreviation-is-rejected", Groups: []Group{GroupCLI}, GoOnly: usage, Steps: []Step{cmd("apply", "codex", "--dry")}},
	}
	for _, app := range gateApps() {
		if app.file == "" {
			continue
		}
		cases = append(cases, Case{
			Name:   app.name + "/not-utf8",
			Groups: []Group{app.group},
			GoOnly: notUTF8,
			Seed:   []File{seed(app.name, app.file, "\xff\xfe theme\n")},
			Steps:  []Step{cmd("apply", app.name)},
		})
	}
	return cases
}
