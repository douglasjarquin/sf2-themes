# Street Fighter II Theme Pack

A standard-library Python CLI that installs Street Fighter II color themes into [WezTerm](https://wezterm.org/) and [Herdr](https://herdr.dev/).

The pack contains **36 fully resolved themes**: a dark and light variant for the shared `main` family theme plus every arcade roster theme through Super Street Fighter II Turbo.

Unofficial fan project. Street Fighter and related names are trademarks of Capcom. This project is not affiliated with or endorsed by Capcom.

## Install

Clone this repository and put the standalone `sf2-theme` script on your `PATH`.

```sh
git clone https://github.com/douglasjarquin/street-fighter-2-theme.git
cd street-fighter-2-theme
install -m 755 sf2-theme "$HOME/.local/bin/sf2-theme"
```

The CLI has no package dependencies. It needs Python 3.11 or newer.

From a checkout you can also run `python3 -m sf2_theme` with `PYTHONPATH=src`, or `pip install -e .`.

## Setup, then apply

`setup` is one-time application integration.
`apply` selects a theme (default: `main`).

```sh
sf2-theme setup wezterm
sf2-theme apply wezterm
sf2-theme apply wezterm --theme ryu

sf2-theme setup herdr
sf2-theme apply herdr --theme chun-li
herdr server reload-config
```

If WezTerm's `wezterm.lua` already selects `street-fighter-2` from an older install, `setup` upgrades that assignment to the managed pointer.
If it selects some other scheme, pass `--adopt` or paste the printed snippet. `setup` will not guess at unknown Lua.

Herdr configs that already have an unmarked `[theme]` section are left alone unless you pass `--adopt`.

`install` still works as a deprecated alias for `apply`.

## Commands

```sh
sf2-theme apps
sf2-theme themes
sf2-theme show ryu
sf2-theme show ryu-light
sf2-theme validate --all
sf2-theme current wezterm
sf2-theme apply herdr --theme boxer --dry-run
```

Boss aliases: `boxer` (Balrog), `claw` (Vega), `dictator` (M. Bison).

## How it writes files

- WezTerm schemes go in `~/.config/wezterm/colors/`.
- The active WezTerm scheme is a managed pointer at `~/.config/sf2-theme/wezterm-current.lua`.
- Herdr updates only a marked block in `~/.config/herdr/config.toml`.
- Symlinks are refused unless you pass `--follow-symlinks`.
- Existing files keep their mode and get a timestamped `.bak.*` copy before the first real change.

Override locations with `--config-dir`, `WEZTERM_CONFIG_FILE`, `WEZTERM_CONFIG_DIR`, `HERDR_CONFIG_PATH`, or `XDG_CONFIG_HOME`.

## Uninstall

Remove `~/.config/wezterm/colors/street-fighter-ii-*.toml`, `~/.config/sf2-theme/`, the WezTerm integration snippet, and the marked Herdr theme block.

## Design

See [docs/theme-guidelines.md](docs/theme-guidelines.md), [docs/roster.md](docs/roster.md), and [docs/previews/](docs/previews/).

Theme data lives in [`themes/`](themes/). Dark themes keep their original IDs, and light variants use the corresponding `<id>-light` ID.
The committed `sf2-theme` script embeds a generated copy of that catalog. Do not hand-edit the embed.
