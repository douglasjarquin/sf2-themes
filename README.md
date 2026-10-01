# Street Fighter II Theme Pack

A standard-library Python CLI that installs Street Fighter II color themes into [WezTerm](https://wezterm.org/), [Herdr](https://herdr.dev/), Neovim, [Codex](https://github.com/openai/codex), [Claude Code](https://claude.com/claude-code), [Starship](https://starship.rs/), and [Lazygit](https://github.com/jesseduffield/lazygit).

## What it is

The pack contains **36 fully resolved themes**: a dark and light variant for the shared `main` family theme plus every arcade roster theme through Super Street Fighter II Turbo.

Unofficial fan project. Street Fighter and related names are trademarks of Capcom. This project is not affiliated with or endorsed by Capcom.

Catalog ids, light variants, and the generated embed are in [docs/sf2-themes/design.md](docs/sf2-themes/design.md).

## Features

* **Seven apps.** WezTerm, Herdr, Neovim, Codex, Claude Code, Starship, and Lazygit. Setup and apply rules are in [docs/sf2-themes/setup-and-apply.md](docs/sf2-themes/setup-and-apply.md).

* **Short catalog ids.** Input stays `ken` or `ryu-light`. Every adapter installs `sf2-ken` or `sf2-ryu-light`. The id rules are in [docs/sf2-themes/design.md](docs/sf2-themes/design.md).

* **Theme site.** `https://douglasjarquin.github.io/sf2-themes/` previews every fighter in both modes, with install commands for each port.

* **CLI.** `apps`, `themes`, `show`, `validate`, `current`, `setup`, and `apply`, installed as both `sf2` and `sf2-themes`. The command list and boss aliases are in [docs/sf2-themes/commands.md](docs/sf2-themes/commands.md).

* **File writes.** Managed paths, marked blocks, symlink refusal, and timestamped backups. Paths and overrides are in [docs/sf2-themes/file-writes.md](docs/sf2-themes/file-writes.md).

* **GitHub or a checkout.** uv installs the CLI, runs it one-shot, or launches it from a checkout. All three are in [docs/sf2-themes/install.md](docs/sf2-themes/install.md).

## Quick Start

### Requirements

The command requires uv and Python 3.11 or newer.

`./sf2-themes` needs Python 3.11 on `PATH`. You do not need a checkout or a globally installed binary.

The mise toolchain and the Astro site are in [docs/sf2-themes/development.md](docs/sf2-themes/development.md).

### Install

```sh
uv tool install git+https://github.com/douglasjarquin/sf2-themes.git
```

That installs the `sf2` command (also available as `sf2-themes`) once, from GitHub. One-shot and checkout launchers are in [docs/sf2-themes/install.md](docs/sf2-themes/install.md).

### First run

```sh
sf2 apply wezterm --theme ryu
```

`apply` prepares the theme assets, installs or repairs the app's integration, and selects the theme (default: `main`) in one idempotent step. Repeating it reports `no changes`.

From a checkout, use `mise run apply -- …` or `scripts/sf2 …` in place of `sf2 …`. The other apps are in [docs/sf2-themes/setup-and-apply.md](docs/sf2-themes/setup-and-apply.md).

## How it works

```
themes/
 │  short catalog id, default main
 ▼
apply, one step
 │
 └─ installed identity sf2-<catalog-id>
```

WezTerm, Neovim, and Herdr select the same dark and light pair when you apply `ryu` or `ryu-light`. Codex and Claude Code pin the one sibling you name.

Herdr appearance switching needs a build with herdr#2324 and is not in a stable release yet.

Symlinks are refused unless you pass `--follow-symlinks`. Existing files keep their mode and get a timestamped `.bak.*` copy before the first real change.

Paths, overrides, and removal are in [docs/sf2-themes/file-writes.md](docs/sf2-themes/file-writes.md) and [docs/sf2-themes/uninstall.md](docs/sf2-themes/uninstall.md).

## Documentation

* [docs/sf2-themes/install.md](docs/sf2-themes/install.md) — uv, a uv tool install, and a checkout.

* [docs/sf2-themes/development.md](docs/sf2-themes/development.md) — mise pins and the Astro site.

* [docs/sf2-themes/setup-and-apply.md](docs/sf2-themes/setup-and-apply.md) — setup, apply, and each app.

* [docs/sf2-themes/commands.md](docs/sf2-themes/commands.md) — `apps`, `themes`, `show`, `validate`, `current`, and boss aliases.

* [docs/sf2-themes/development.md](docs/sf2-themes/development.md) — the Astro site and its tooling.

* [docs/sf2-themes/file-writes.md](docs/sf2-themes/file-writes.md) — paths, overrides, symlinks, and backups.

* [docs/sf2-themes/uninstall.md](docs/sf2-themes/uninstall.md) — files and settings to remove.

* [docs/sf2-themes/design.md](docs/sf2-themes/design.md) — catalog ids and the generated embed.

* [docs/theme-guidelines.md](docs/theme-guidelines.md) — theme guidelines.

* [docs/roster.md](docs/roster.md) — arcade roster.

* [docs/previews/](docs/previews/) — theme previews.

* [themes/](themes/) — theme data.

* [web/AGENTS.md](web/AGENTS.md) — portless setup for the local site.

## Contributing

Ordinary pull requests target `main`. The workflow and checks are in [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Street Fighter II Theme Pack is licensed under the [MIT License](LICENSE).
