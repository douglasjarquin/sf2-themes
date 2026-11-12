// Package cli parses the sf2-themes command line and runs its commands.
package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/douglasjarquin/sf2-themes/v2/internal/catalog"
	"github.com/douglasjarquin/sf2-themes/v2/internal/version"
)

// appNames lists the configurable applications, in `apps` order; slack only prints a paste string.
var appNames = []string{"wezterm", "herdr", "herdr-app", "nvim", "codex", "starship", "lazygit", "claude", "slack"}

// Context carries the output streams to a command handler.
type Context struct {
	Stdout io.Writer
	Stderr io.Writer
}

// handler runs one parsed command. A returned error prints as `error: <message>` on stderr with exit status 1;
// otherwise the returned status is the exit status.
type handler func(ctx *Context, req Request) (int, error)

var handlers = map[string]handler{
	"apps":     runApps,
	"themes":   runThemes,
	"show":     runShow,
	"validate": runValidate,
	"setup":    mutate,
	"apply":    mutate,
	"current":  runCurrent,
}

// Main runs one command line (without the program name) and returns the process exit status:
// 0 on success, 1 for an `error:` line, 2 for a usage error.
func Main(args []string, stdout, stderr io.Writer) int {
	result, usageErr := parseArgs(args)
	if usageErr != nil {
		printf(stderr, "%s\n%s: error: %s\n", usageErr.usage, usageErr.prog, usageErr.message)
		return 2
	}
	switch result.outcome {
	case showHelp:
		printf(stdout, "%s", result.help)
		return 0
	case showVersion:
		printf(stdout, "%s\n", version.Version)
		return 0
	}
	ctx := &Context{Stdout: stdout, Stderr: stderr}
	status, err := handlers[result.request.Command](ctx, result.request)
	if err != nil {
		printf(stderr, "error: %s\n", err)
		return 1
	}
	return status
}

// printf writes command output. A failed write to stdout or stderr has nowhere to be reported, so it is dropped.
func printf(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

func runApps(ctx *Context, _ Request) (int, error) {
	printf(ctx.Stdout, "%s\n", strings.Join(appNames, "\n"))
	return 0, nil
}

func runThemes(ctx *Context, _ Request) (int, error) {
	themes, err := catalog.Load()
	if err != nil {
		return 0, err
	}
	for _, theme := range themes {
		printf(ctx.Stdout, "%s\t%s\n", theme.Metadata.ID, theme.Metadata.DisplayName)
	}
	return 0, nil
}

func runShow(ctx *Context, req Request) (int, error) {
	themes, err := catalog.Load()
	if err != nil {
		return 0, err
	}
	theme, err := catalog.GetTheme(req.Names[0], themes)
	if err != nil {
		return 0, err
	}
	printf(ctx.Stdout, "%s", catalog.ShowTheme(theme))
	return 0, nil
}

func runValidate(ctx *Context, req Request) (int, error) {
	parsed, err := catalog.Parse()
	if err != nil {
		return 0, err
	}
	themes := parsed
	var lines []string
	if req.All || len(req.Names) == 0 {
		lines = catalog.CatalogIssues(themes)
	} else {
		themes = nil
		for _, name := range req.Names {
			theme, err := catalog.GetTheme(name, parsed)
			if err != nil {
				return 0, err
			}
			themes = append(themes, theme)
		}
		for _, theme := range themes {
			for _, issue := range catalog.ValidateTheme(theme) {
				lines = append(lines, issue.Line())
			}
		}
	}
	failed := false
	for _, line := range lines {
		if strings.HasPrefix(line, "error:") {
			printf(ctx.Stderr, "%s\n", line)
			failed = true
		} else {
			printf(ctx.Stdout, "%s\n", line)
		}
	}
	if failed {
		return 1, nil
	}
	ids := make([]string, len(themes))
	for i, theme := range themes {
		ids[i] = theme.Metadata.ID
	}
	printf(ctx.Stdout, "valid: %s\n", strings.Join(ids, ", "))
	return 0, nil
}
