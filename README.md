# Street Fighter II Theme Pack

A standard-library Python CLI that installs Street Fighter II color themes into [WezTerm](https://wezterm.org/), [Herdr](https://herdr.dev/), and Neovim.

The pack contains **36 fully resolved themes**: a dark and light variant for the shared `main` family theme plus every arcade roster theme through Super Street Fighter II Turbo.

Unofficial fan project. Street Fighter and related names are trademarks of Capcom. This project is not affiliated with or endorsed by Capcom.

## Install

Clone this repository and put the standalone `sf2-themes` script on your `PATH`.

```sh
git clone https://github.com/douglasjarquin/sf2-themes.git
cd sf2-themes
install -m 755 sf2-themes "$HOME/.local/bin/sf2-themes"
```

The CLI has no package dependencies. It needs Python 3.11 or newer.

From a checkout you can also run `python3 -m sf2_theme` with `PYTHONPATH=src`, or `pip install -e .`.

## Setup, then apply

`setup` is one-time application integration.
`apply` selects a theme (default: `main`).

```sh
sf2-themes setup wezterm
sf2-themes apply wezterm
sf2-themes apply wezterm --theme ryu

sf2-themes setup herdr
sf2-themes apply herdr --theme chun-li
herdr server reload-config

sf2-themes setup nvim
sf2-themes apply nvim --theme ryu-light
```

Neovim setup installs every catalog colorscheme under `~/.config/nvim/colors/`, a managed current-theme pointer under `~/.config/nvim/sf2-theme/current.lua`, and a plugin loader under `~/.config/nvim/plugin/sf2-theme.lua`.
The loader applies the selected theme automatically when Neovim starts.

If WezTerm's `wezterm.lua` already selects `street-fighter-2` from an older install, `setup` upgrades that assignment to the managed pointer.
If it selects some other scheme, pass `--adopt` or paste the printed snippet. `setup` will not guess at unknown Lua.

Herdr configs that already have an unmarked `[theme]` section are left alone unless you pass `--adopt`.

`install` still works as a deprecated alias for `apply`.

## Commands

```sh
sf2-themes apps
sf2-themes themes
sf2-themes show ryu
sf2-themes show ryu-light
sf2-themes validate --all
sf2-themes current wezterm
sf2-themes apply herdr --theme boxer --dry-run
sf2-themes current nvim
```

Boss aliases: `boxer` (Balrog), `claw` (Vega), `dictator` (M. Bison).

## How it writes files

- WezTerm schemes go in `~/.config/wezterm/colors/`.
- The active WezTerm scheme is a managed pointer at `~/.config/sf2-theme/wezterm-current.lua`.
- Herdr updates only a marked block in `~/.config/herdr/config.toml`.
- Neovim colorschemes go in `~/.config/nvim/colors/`, with the active theme at `~/.config/nvim/sf2-theme/current.lua`.
- Neovim setup manages the startup loader at `~/.config/nvim/plugin/sf2-theme.lua`.
- Symlinks are refused unless you pass `--follow-symlinks`.
- Existing files keep their mode and get a timestamped `.bak.*` copy before the first real change.

Override locations with `--config-dir`, `WEZTERM_CONFIG_FILE`, `WEZTERM_CONFIG_DIR`, `HERDR_CONFIG_PATH`, `NVIM_CONFIG_DIR`, or `XDG_CONFIG_HOME`.

## Uninstall

Remove `~/.config/wezterm/colors/street-fighter-ii-*.toml`, `~/.config/nvim/colors/street-fighter-ii-*.lua`, `~/.config/nvim/sf2-theme/`, `~/.config/nvim/plugin/sf2-theme.lua`, the WezTerm integration snippet, and the marked Herdr theme block.

## Design

See [docs/theme-guidelines.md](docs/theme-guidelines.md), [docs/roster.md](docs/roster.md), and [docs/previews/](docs/previews/).

Theme data lives in [`themes/`](themes/). Dark themes keep their original IDs, and light variants use the corresponding `<id>-light` ID.
The committed `sf2-themes` script embeds a generated copy of that catalog. Do not hand-edit the embed.
