# Street Fighter II Theme Pack

A single-binary CLI that installs Street Fighter II color themes into [WezTerm](https://wezterm.org/), [Herdr](https://herdr.dev/) TUI and desktop app, Neovim, [Codex](https://github.com/openai/codex), [Claude Code](https://claude.com/claude-code), [Starship](https://starship.rs/), and [Lazygit](https://github.com/jesseduffield/lazygit), and prints a Slack custom-theme paste string.

## What it is

The pack contains **36 fully resolved themes**: a dark and light variant for the shared `main` family theme plus every arcade roster theme through Super Street Fighter II Turbo.

Unofficial fan project. Street Fighter and related names are trademarks of Capcom. This project is not affiliated with or endorsed by Capcom.

Catalog ids, light variants, and the embedded catalog are in [docs/sf2-themes/design.md](docs/sf2-themes/design.md).

## Features

* **Eight app targets, plus Slack.** WezTerm, Herdr TUI, Herdr app, Neovim, Codex, Claude Code, Starship, and Lazygit install a theme. Slack prints a paste string and does not install an `sf2-` identity. Setup and apply rules are in [docs/sf2-themes/setup-and-apply.md](docs/sf2-themes/setup-and-apply.md).

* **Short catalog ids.** Input stays `ken` or `ryu-light`. File-writing adapters install `sf2-ken` or `sf2-ryu-light`. The id rules are in [docs/sf2-themes/design.md](docs/sf2-themes/design.md).

* **Theme site.** `https://douglasjarquin.github.io/sf2-themes/` previews every fighter in both modes, with install commands for each port. Machine-readable summary: `https://douglasjarquin.github.io/sf2-themes/llms.txt`.

* **CLI.** `apps`, `themes`, `show`, `validate`, `current`, `setup`, and `apply`, installed as both `sf2` and `sf2-themes`. The command list and boss aliases are in [docs/sf2-themes/commands.md](docs/sf2-themes/commands.md).

* **File writes.** Managed paths, marked blocks, symlink refusal, and timestamped backups. Paths and overrides are in [docs/sf2-themes/file-writes.md](docs/sf2-themes/file-writes.md).

* **One binary, no runtime.** A one-line installer downloads the release binary for macOS or Linux, checks it against the published checksums, and links `sf2` to it. Version pins, verification, and upgrading from 1.x are in [docs/sf2-themes/install.md](docs/sf2-themes/install.md).

## Quick Start

### Requirements

macOS or Linux on arm64 or amd64, with `curl` or `wget`.
The binary needs no Python, uv, or other runtime.

The mise toolchain and the Astro site are in [docs/sf2-themes/development.md](docs/sf2-themes/development.md).

### Install

```sh
curl -fsSL https://github.com/douglasjarquin/sf2-themes/releases/latest/download/install.sh | sh
```

That installs `sf2-themes` into `~/.local/bin` and links `sf2` to it.
If you installed 1.x with uv, run `uv tool uninstall sf2-themes` first.
Version pins, another install directory, and checking where the binary was built are in [docs/sf2-themes/install.md](docs/sf2-themes/install.md).

### First run

```sh
sf2 apply wezterm --theme ryu
```

`apply` prepares the theme assets, installs or repairs the app's integration, and selects the theme (default: `main`) in one idempotent step. Repeating it reports `no changes`.

From a checkout, use `mise run apply -- …` in place of `sf2 apply …`. The other apps are in [docs/sf2-themes/setup-and-apply.md](docs/sf2-themes/setup-and-apply.md).

## How it works

```
themes/
 │  short catalog id, default main
 ▼
apply, one step
 │
 └─ installed identity sf2-<catalog-id>
```

WezTerm, Neovim, Herdr TUI, and Herdr app select the same dark and light pair when you apply `ryu` or `ryu-light`. Codex and Claude Code pin the one sibling you name.

Slack apply prints one four-color import line.
Paste it into Preferences, Appearance, Custom theme, Import theme, set Color Mode to match the theme, and turn Darker sidebars off.
Slack writes no file and does not install an `sf2-` identity.

`sf2 apply herdr-app --theme ryu` installs desktop JSON themes and selects the light/dark pair in `~/.config/herdr-app/settings.json`.
Settings apply live; layout themes and the theme picker can override the selection.

Herdr TUI appearance switching needs Herdr v0.9.0 or newer (herdr#2324).

Symlinks are refused unless you pass `--follow-symlinks`. Existing files keep their mode and get a timestamped `.bak.*` copy before the first real change.

Paths, overrides, and removal are in [docs/sf2-themes/file-writes.md](docs/sf2-themes/file-writes.md) and [docs/sf2-themes/uninstall.md](docs/sf2-themes/uninstall.md).

## Documentation

* [docs/sf2-themes/install.md](docs/sf2-themes/install.md) - the installer, version pins, verification, and upgrading from 1.x.

* [docs/sf2-themes/development.md](docs/sf2-themes/development.md) - mise pins and the Astro site.

* [docs/sf2-themes/setup-and-apply.md](docs/sf2-themes/setup-and-apply.md) - setup, apply, and each app.

* [docs/sf2-themes/commands.md](docs/sf2-themes/commands.md) - `apps`, `themes`, `show`, `validate`, `current`, and boss aliases.

* [docs/sf2-themes/file-writes.md](docs/sf2-themes/file-writes.md) - paths, overrides, symlinks, and backups.

* [docs/sf2-themes/uninstall.md](docs/sf2-themes/uninstall.md) - files and settings to remove.

* [docs/sf2-themes/design.md](docs/sf2-themes/design.md) - catalog ids and the embedded catalog.

* [docs/theme-guidelines.md](docs/theme-guidelines.md) - theme guidelines.

* [docs/roster.md](docs/roster.md) - arcade roster.

* [docs/previews/](docs/previews/) - theme previews.

* [themes/](themes/) - theme data.

* [web/AGENTS.md](web/AGENTS.md) - portless setup for the local site.

## Contributing

Ordinary pull requests target `main`. The workflow and checks are in [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Street Fighter II Theme Pack is licensed under the [MIT License](LICENSE).
