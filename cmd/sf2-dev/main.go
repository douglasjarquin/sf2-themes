// Command sf2-dev holds the maintainer generators: the SVG previews and the web theme data.
// It is never released.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/douglasjarquin/sf2-themes/v2/internal/devtools"
	"github.com/douglasjarquin/sf2-themes/v2/internal/pycompat"
)

const (
	usage         = "usage: sf2-dev {previews,web-data} ..."
	previewsUsage = "usage: sf2-dev previews [-h] [--output OUTPUT]"
	webDataUsage  = "usage: sf2-dev web-data [-h] [--source SOURCE] [--output OUTPUT] [--check]"
)

const (
	catalogDir    = "themes"
	previewsDir   = "docs/previews"
	webDataOutput = "web/src/data/generated-theme-data.json"
)

func main() {
	// The TOML library switches to TOML 1.1 when this variable is set; the catalog is TOML 1.0.
	_ = os.Unsetenv("BURNTSUSHI_TOML_110")
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		_, _ = fmt.Fprintf(stdout, "%s\n\ncommands:\n  previews  write an SVG swatch for every theme to docs/previews\n"+
			"  web-data  generate web/src/data/generated-theme-data.json from the catalog\n", usage)
		return 0
	}
	if len(args) == 0 {
		return usageError(stderr, usage, "sf2-dev", "the following arguments are required: command")
	}
	switch args[0] {
	case "previews":
		return previews(args[1:], stdout, stderr)
	case "web-data":
		return webData(args[1:], stdout, stderr)
	}
	return usageError(stderr, usage, "sf2-dev",
		fmt.Sprintf("argument command: invalid choice: %s (choose from 'previews', 'web-data')", pycompat.Repr(args[0])))
}

func previews(args []string, stdout, stderr io.Writer) int {
	opts, err := parseOptions(args, []string{"--output"}, nil)
	if errors.Is(err, errHelp) {
		_, _ = fmt.Fprintf(stdout, "%s\n\nWrite an SVG swatch for every theme.\n\noptions:\n"+
			"  -h, --help       show this help message and exit\n  --output OUTPUT\n", previewsUsage)
		return 0
	}
	if err != nil {
		return usageError(stderr, previewsUsage, "sf2-dev previews", err.Error())
	}
	root, err := moduleRoot()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	output, ok := opts.values["--output"]
	if !ok {
		output = pycompat.Path(root, previewsDir)
	}
	return devtools.Previews(filepath.Join(root, catalogDir), pycompat.Path(output), stdout, stderr)
}

func webData(args []string, stdout, stderr io.Writer) int {
	opts, err := parseOptions(args, []string{"--source", "--output"}, []string{"--check"})
	if errors.Is(err, errHelp) {
		_, _ = fmt.Fprintf(stdout, "%s\n\nGenerate browser theme data from the TOML catalog.\n\noptions:\n"+
			"  -h, --help       show this help message and exit\n  --source SOURCE\n  --output OUTPUT\n  --check\n",
			webDataUsage)
		return 0
	}
	if err != nil {
		return usageError(stderr, webDataUsage, "sf2-dev web-data", err.Error())
	}
	source, hasSource := opts.values["--source"]
	output, hasOutput := opts.values["--output"]
	if !hasSource || !hasOutput {
		root, err := moduleRoot()
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 1
		}
		if !hasSource {
			source = filepath.Join(root, catalogDir)
		}
		if !hasOutput {
			output = pycompat.Path(root, webDataOutput)
		}
	}
	return devtools.WebData(source, pycompat.Path(output), opts.flags["--check"], stdout, stderr)
}

func usageError(stderr io.Writer, usage, prog, message string) int {
	_, _ = fmt.Fprintf(stderr, "%s\n%s: error: %s\n", usage, prog, message)
	return 2
}

var errHelp = errors.New("help requested")

type options struct {
	values map[string]string
	flags  map[string]bool
}

// parseOptions reads `--name value`, `--name=value`, and boolean flags the way argparse does for these
// commands: a later occurrence wins, -h or --help asks for help, and anything else is unrecognized.
func parseOptions(args, valued, booleans []string) (options, error) {
	opts := options{values: map[string]string{}, flags: map[string]bool{}}
	var unrecognized []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-h" || arg == "--help" {
			return opts, errHelp
		}
		name, value, inline := strings.Cut(arg, "=")
		switch {
		case contains(valued, name):
			if !inline {
				if i+1 >= len(args) || (strings.HasPrefix(args[i+1], "-") && args[i+1] != "-") {
					return opts, fmt.Errorf("argument %s: expected one argument", name)
				}
				i++
				value = args[i]
			}
			opts.values[name] = value
		case contains(booleans, arg):
			opts.flags[arg] = true
		case inline && contains(booleans, name):
			return opts, fmt.Errorf("argument %s: ignored explicit argument %s", name, pycompat.Repr(value))
		default:
			unrecognized = append(unrecognized, arg)
		}
	}
	if len(unrecognized) > 0 {
		return opts, fmt.Errorf("unrecognized arguments: %s", strings.Join(unrecognized, " "))
	}
	return opts, nil
}

func contains(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}

// moduleRoot returns the directory holding go.mod, found by walking up from the working directory,
// with symbolic links resolved.
func moduleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		dir = resolved
	}
	for {
		if info, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && info.Mode().IsRegular() {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("go.mod not found; run sf2-dev from inside the sf2-themes checkout")
		}
		dir = parent
	}
}
