package cli

import (
	"regexp"
	"slices"
	"strings"

	"github.com/douglasjarquin/sf2-themes/v2/internal/pycompat"
)

// prog is the program name in usage errors, fixed whichever name (sf2 or sf2-themes) started the binary.
const prog = "sf2-themes"

// Request is one parsed command line.
type Request struct {
	Command string
	// App is the application positional of setup, apply, and current.
	App string
	// Names holds the theme positional of show, or the THEME list of validate.
	Names []string
	// Theme is --theme; nil when the flag is absent.
	Theme *string
	// ConfigDir is --config-dir; nil when the flag is absent.
	ConfigDir      *string
	All            bool
	DryRun         bool
	FollowSymlinks bool
	Adopt          bool
	Verbose        bool
}

// flagSpec is one option. names[0] and the slash-joined names are what usage errors print.
type flagSpec struct {
	names   []string
	metavar string
	help    string
	set     func(r *Request, value string)
}

type positionalSpec struct {
	dest    string
	choices []string
	many    bool
	help    string
	set     func(r *Request, value string)
}

type commandSpec struct {
	name        string
	help        string
	positionals []positionalSpec
	flags       []flagSpec
}

var helpFlag = flagSpec{names: []string{"-h", "--help"}, help: "show this help message and exit"}

var versionFlag = flagSpec{names: []string{"--version", "-V"}, help: "show program's version number and exit"}

func themeFlag(help string) flagSpec {
	return flagSpec{
		names: []string{"--theme"}, metavar: "THEME", help: help,
		set: func(r *Request, v string) { r.Theme = &v },
	}
}

func configDirFlag(help string) flagSpec {
	return flagSpec{
		names: []string{"--config-dir"}, metavar: "PATH", help: help,
		set: func(r *Request, v string) { r.ConfigDir = &v },
	}
}

func switchFlag(name, help string, set func(*Request)) flagSpec {
	return flagSpec{names: []string{name}, help: help, set: func(r *Request, _ string) { set(r) }}
}

func appPositional(help string) positionalSpec {
	return positionalSpec{dest: "app", choices: appNames, help: help, set: func(r *Request, v string) { r.App = v }}
}

func mutationSpec(name, help string) commandSpec {
	return commandSpec{
		name:        name,
		help:        help,
		positionals: []positionalSpec{appPositional("application to configure")},
		flags: []flagSpec{
			helpFlag,
			themeFlag("theme id, alias, or display name (default: main)"),
			configDirFlag("write under PATH instead of the app's config location"),
			switchFlag("--dry-run", "print the full plan without writing anything", func(r *Request) { r.DryRun = true }),
			switchFlag("--follow-symlinks", "write through destination symlinks", func(r *Request) { r.FollowSymlinks = true }),
			switchFlag("--adopt", "replace a theme setting sf2-themes does not manage", func(r *Request) { r.Adopt = true }),
			{
				names: []string{"--verbose", "-v"},
				help:  "print every file operation, not just the summary",
				set:   func(r *Request, _ string) { r.Verbose = true },
			},
		},
	}
}

var commandSpecs = []commandSpec{
	{name: "apps", help: "list supported applications", flags: []flagSpec{helpFlag}},
	{name: "themes", help: "list catalog themes", flags: []flagSpec{helpFlag}},
	{
		name: "show",
		help: "print a theme's canonical TOML",
		positionals: []positionalSpec{{
			dest: "theme", help: "theme id, alias, or display name",
			set: func(r *Request, v string) { r.Names = append(r.Names, v) },
		}},
		flags: []flagSpec{helpFlag},
	},
	{
		name: "validate",
		help: "validate theme definitions",
		positionals: []positionalSpec{{
			dest: "THEME", many: true, help: "catalog ids (default: the whole catalog)",
			set: func(r *Request, v string) { r.Names = append(r.Names, v) },
		}},
		flags: []flagSpec{
			helpFlag,
			switchFlag("--all", "validate every catalog theme", func(r *Request) { r.All = true }),
		},
	},
	mutationSpec("setup", "install or repair app integration (keeps an existing selection unless --theme is given)"),
	mutationSpec("apply", "integrate the app and select a theme in one step"),
	{
		name:        "current",
		help:        "print the selected theme id, or none",
		positionals: []positionalSpec{appPositional("")},
		flags:       []flagSpec{helpFlag, configDirFlag("")},
	},
}

// outcome is what a parsed command line asks for.
type outcome int

const (
	runCommand outcome = iota
	showHelp
	showVersion
)

type parsed struct {
	outcome outcome
	request Request
	help    string
}

// usageError is an argparse-style usage error: the usage line, then `<prog>: error: <message>`, exit 2.
type usageError struct {
	prog    string
	usage   string
	message string
}

// negativeNumber is argparse's test for a token that starts with `-` but is a value, not an option.
var negativeNumber = regexp.MustCompile(`^-\d+$|^-\d*\.\d+$`)

// classify follows argparse's option detection without prefix abbreviation:
// it reports whether tok is an option, the matching flag (nil for an unknown option), and an `=value` suffix.
func classify(tok string, flags []flagSpec) (isOption bool, flag *flagSpec, explicit *string) {
	if tok == "" || tok[0] != '-' {
		return false, nil, nil
	}
	if f := findFlag(flags, tok); f != nil {
		return true, f, nil
	}
	if len(tok) == 1 {
		return false, nil, nil
	}
	if name, value, ok := strings.Cut(tok, "="); ok {
		if f := findFlag(flags, name); f != nil {
			return true, f, &value
		}
	}
	if negativeNumber.MatchString(tok) || strings.Contains(tok, " ") {
		return false, nil, nil
	}
	return true, nil, nil
}

// expandShort splits a token of combined single-dash switches such as `-vh` into `-v -h`, the way
// argparse does, and reports ok=false when tok is not such a token. From the first character that is
// not a known switch, the rest of the token is one unrecognized argument (`-vx` gives `-v` and `-x`).
// A dash there is argparse's "ignored explicit argument" error on the switch before it.
func expandShort(prog, usage, tok string, flags []flagSpec) (switches []string, extra string, ok bool, err *usageError) {
	if len(tok) < 3 || tok[0] != '-' || tok[1] == '-' || strings.Contains(tok, "=") || findFlag(flags, tok) != nil {
		return nil, "", false, nil
	}
	current := findFlag(flags, tok[:2])
	if current == nil || current.metavar != "" {
		return nil, "", false, nil
	}
	switches = []string{tok[:2]}
	for rest := tok[2:]; rest != ""; rest = rest[1:] {
		if rest[0] == '-' {
			return nil, "", true, ignoredExplicit(prog, usage, current, rest)
		}
		next := findFlag(flags, "-"+rest[:1])
		if next == nil || next.metavar != "" {
			return switches, "-" + rest, true, nil
		}
		switches = append(switches, "-"+rest[:1])
		current = next
	}
	return switches, "", true, nil
}

func findFlag(flags []flagSpec, name string) *flagSpec {
	for i := range flags {
		if slices.Contains(flags[i].names, name) {
			return &flags[i]
		}
	}
	return nil
}

func ignoredExplicit(prog, usage string, f *flagSpec, value string) *usageError {
	return &usageError{prog, usage, "argument " + strings.Join(f.names, "/") + ": ignored explicit argument " + pycompat.Repr(value)}
}

func invalidChoice(prog, usage, dest, value string, choices []string) *usageError {
	quoted := make([]string, len(choices))
	for i, choice := range choices {
		quoted[i] = pycompat.Repr(choice)
	}
	return &usageError{prog, usage, "argument " + dest + ": invalid choice: " + pycompat.Repr(value) +
		" (choose from " + strings.Join(quoted, ", ") + ")"}
}

func commandNames() []string {
	names := make([]string, len(commandSpecs))
	for i, spec := range commandSpecs {
		names[i] = spec.name
	}
	return names
}

// parseArgs parses a command line. Options may come before or after positionals, `--name=value` and
// `--name value` are equivalent, and `--` makes every later argument positional.
// Option names must be spelled in full. Help and --version act as soon as they are reached;
// unknown options and surplus positionals are reported together after the command parses.
func parseArgs(args []string) (parsed, *usageError) {
	topFlags := []flagSpec{helpFlag, versionFlag}
	var extras []string
	for i := 0; i < len(args); i++ {
		tok := args[i]
		if switches, extra, ok, err := expandShort(prog, topUsage, tok, topFlags); err != nil {
			return parsed{}, err
		} else if ok {
			if extra != "" {
				extras = append(extras, extra)
			}
			args = slices.Concat(args[:i], switches, args[i+1:])
			tok = args[i]
		}
		if tok == "--" {
			if i == len(args)-1 {
				extras = append(extras, tok)
				break
			}
			return parsed{}, invalidChoice(prog, topUsage, "COMMAND", tok, commandNames())
		}
		isOption, flag, explicit := classify(tok, topFlags)
		switch {
		case isOption && flag == nil:
			extras = append(extras, tok)
		case isOption && explicit != nil:
			return parsed{}, ignoredExplicit(prog, topUsage, flag, *explicit)
		case isOption && flag.names[0] == helpFlag.names[0]:
			return parsed{outcome: showHelp, help: topHelp}, nil
		case isOption:
			return parsed{outcome: showVersion}, nil
		default:
			index := slices.Index(commandNames(), tok)
			if index < 0 {
				return parsed{}, invalidChoice(prog, topUsage, "COMMAND", tok, commandNames())
			}
			result, subExtras, err := parseCommand(&commandSpecs[index], args[i+1:])
			if err != nil || result.outcome != runCommand {
				return result, err
			}
			if extras = append(extras, subExtras...); len(extras) > 0 {
				return parsed{}, &usageError{prog, topUsage, "unrecognized arguments: " + strings.Join(extras, " ")}
			}
			return result, nil
		}
	}
	if len(extras) > 0 {
		return parsed{}, &usageError{prog, topUsage, "unrecognized arguments: " + strings.Join(extras, " ")}
	}
	return parsed{outcome: showHelp, help: topHelp}, nil
}

func parseCommand(spec *commandSpec, args []string) (parsed, []string, *usageError) {
	subProg := prog + " " + spec.name
	usage := commandUsage(spec)
	request := Request{Command: spec.name}
	var extras []string
	filled := 0
	manyTaken := false
	positionalOnly := false
	lastWasPositional := false
	for i := 0; i < len(args); i++ {
		tok := args[i]
		wasPositional := lastWasPositional
		lastWasPositional = false
		if !positionalOnly && tok == "--" {
			lastWasPositional = wasPositional
			positionalOnly = true
			// argparse drops `--` when a positional still has room or the run of positionals just before it
			// absorbs it; otherwise it is an unrecognized argument.
			if filled == len(spec.positionals) && !lastWasPositional {
				extras = append(extras, tok)
			}
			continue
		}
		if !positionalOnly {
			if switches, extra, ok, err := expandShort(subProg, usage, tok, spec.flags); err != nil {
				return parsed{}, nil, err
			} else if ok {
				if extra != "" {
					extras = append(extras, extra)
				}
				args = slices.Concat(args[:i], switches, args[i+1:])
				tok = args[i]
			}
			isOption, flag, explicit := classify(tok, spec.flags)
			// argparse gives a zero-or-more positional one run of consecutive values; values after an option are surplus.
			if isOption && filled < len(spec.positionals) && spec.positionals[filled].many && manyTaken {
				filled++
			}
			switch {
			case isOption && flag == nil:
				extras = append(extras, tok)
				continue
			case isOption && flag.metavar == "" && explicit != nil:
				return parsed{}, nil, ignoredExplicit(subProg, usage, flag, *explicit)
			case isOption && flag.names[0] == helpFlag.names[0]:
				return parsed{outcome: showHelp, help: commandHelp(spec)}, nil, nil
			case isOption && flag.metavar == "":
				flag.set(&request, "")
				continue
			case isOption && explicit != nil:
				flag.set(&request, *explicit)
				continue
			case isOption:
				if i+1 == len(args) || args[i+1] == "--" {
					return parsed{}, nil, expectedOne(subProg, usage, flag)
				}
				if next, _, _ := classify(args[i+1], spec.flags); next {
					return parsed{}, nil, expectedOne(subProg, usage, flag)
				}
				i++
				flag.set(&request, args[i])
				continue
			}
		}
		if filled == len(spec.positionals) {
			extras = append(extras, tok)
			continue
		}
		lastWasPositional = true
		positional := spec.positionals[filled]
		if positional.choices != nil && !slices.Contains(positional.choices, tok) {
			return parsed{}, nil, invalidChoice(subProg, usage, positional.dest, tok, positional.choices)
		}
		positional.set(&request, tok)
		manyTaken = positional.many
		if !positional.many {
			filled++
		}
	}
	var missing []string
	for _, positional := range spec.positionals[filled:] {
		if !positional.many {
			missing = append(missing, positional.dest)
		}
	}
	if len(missing) > 0 {
		return parsed{}, nil, &usageError{subProg, usage, "the following arguments are required: " + strings.Join(missing, ", ")}
	}
	return parsed{outcome: runCommand, request: request}, extras, nil
}

func expectedOne(prog, usage string, flag *flagSpec) *usageError {
	return &usageError{prog, usage, "argument " + strings.Join(flag.names, "/") + ": expected one argument"}
}
