package cli

import (
	"fmt"
	"strings"
)

// Help is written by hand and never wraps to the terminal width.

const topUsage = "usage: sf2-themes [-h] [--version] COMMAND ..."

const description = "Street Fighter II theme pack for WezTerm, Herdr TUI, Herdr app, Neovim, " +
	"Codex, Claude Code, Starship, Lazygit, and Slack."

const epilog = `notes:
  apply prepares theme assets, installs or repairs the app integration, and
  selects a theme (default: main) in one idempotent step.
  setup performs the same integration but keeps an existing selection unless
  --theme is given.
  When an app already uses a theme sf2-themes does not manage, apply and
  setup --theme refuse unless --adopt is given; plain setup installs the
  theme files and keeps that theme.
  starship also refreshes ~/.config/sf2-theme/zsh-syntax-highlighting.zsh.
  Lazygit installs one YAML fragment per catalog theme under its themes directory.
  Herdr app installs JSON themes in ~/.config/herdr-app/themes/ and selects
  a light/dark pair in settings.json. Layout themes and the picker take precedence.
  WezTerm, Herdr TUI/app, and Neovim apply both dark and light siblings and auto-switch
  with host appearance. Claude installs one theme file per catalog entry under
  ~/.claude/themes/ and selects it in settings.json; select it or switch
  siblings yourself in /theme, since Claude Code has no host-appearance
  auto-switch.
  Slack prints a paste string and does not install an sf2- identity.
`

var topHelp = buildTopHelp()

func buildTopHelp() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n%s\n\npositional arguments:\n  COMMAND\n", topUsage, description)
	for _, spec := range commandSpecs {
		fmt.Fprintf(&b, "    %-12s %s\n", spec.name, spec.help)
	}
	b.WriteString("\noptions:\n")
	writeOptions(&b, []flagSpec{helpFlag, versionFlag})
	b.WriteString("\n" + epilog)
	return b.String()
}

// positionalLabel is how usage and help name a positional: its choices in braces, or its name.
func positionalLabel(p positionalSpec) string {
	if p.choices != nil {
		return "{" + strings.Join(p.choices, ",") + "}"
	}
	return p.dest
}

func optionLabel(f flagSpec) string {
	label := strings.Join(f.names, ", ")
	if f.metavar != "" {
		label = f.names[0] + " " + f.metavar
	}
	return label
}

// commandUsage is the one-line usage of a command.
func commandUsage(spec *commandSpec) string {
	parts := []string{"usage:", prog, spec.name}
	for _, f := range spec.flags {
		label := f.names[0]
		if f.metavar != "" {
			label += " " + f.metavar
		}
		parts = append(parts, "["+label+"]")
	}
	for _, p := range spec.positionals {
		if p.many {
			parts = append(parts, "["+p.dest+" ...]")
		} else {
			parts = append(parts, positionalLabel(p))
		}
	}
	return strings.Join(parts, " ")
}

func writeOptions(b *strings.Builder, flags []flagSpec) {
	width := 0
	for _, f := range flags {
		width = max(width, len(optionLabel(f)))
	}
	for _, f := range flags {
		writeEntry(b, optionLabel(f), f.help, width)
	}
}

func writeEntry(b *strings.Builder, label, help string, width int) {
	if help == "" {
		fmt.Fprintf(b, "  %s\n", label)
		return
	}
	fmt.Fprintf(b, "  %-*s  %s\n", width, label, help)
}

const maxLabelWidth = 20

// commandHelp is the help a command prints for -h.
func commandHelp(spec *commandSpec) string {
	var b strings.Builder
	b.WriteString(commandUsage(spec) + "\n")
	// Help text starts after the longest label, but no further right than argparse's column 24.
	width := 0
	for _, p := range spec.positionals {
		width = max(width, len(positionalLabel(p)))
	}
	for _, f := range spec.flags {
		width = max(width, len(optionLabel(f)))
	}
	width = min(width, maxLabelWidth)
	if len(spec.positionals) > 0 {
		b.WriteString("\npositional arguments:\n")
		for _, p := range spec.positionals {
			label := positionalLabel(p)
			if len(label) > width {
				// A long choice list sits on its own line, with its help below, as argparse prints it.
				fmt.Fprintf(&b, "  %s\n", label)
				if p.help != "" {
					fmt.Fprintf(&b, "  %-*s  %s\n", width, "", p.help)
				}
				continue
			}
			writeEntry(&b, label, p.help, width)
		}
	}
	b.WriteString("\noptions:\n")
	for _, f := range spec.flags {
		writeEntry(&b, optionLabel(f), f.help, width)
	}
	return b.String()
}
