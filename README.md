# Street Fighter II Theme Pack

A standard-library Python CLI that installs Street Fighter II color themes into [WezTerm](https://wezterm.org/), [Herdr](https://herdr.dev/), Neovim, and [Codex](https://github.com/openai/codex).

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

## Development with mise

The repository's `mise.toml` pins the local toolchain to Python 3.11, Node 24, and uv 0.11.

```sh
mise install
mise run test
mise run web:test
mise run web:check
mise run web:build
mise run web:dev
```

`mise run web:dev` starts the Astro site at `http://127.0.0.1:4321`.

## Setup, then apply

`setup` is one-time application integration.
`apply` selects a theme (default: `main`).

The CLI keeps short catalog ids such as `ken` and `ryu-light` for input.
Every adapter installs and selects the corresponding prefixed ids such as `sf2-ken` and `sf2-ryu-light`.

```sh
sf2-themes setup wezterm
sf2-themes apply wezterm
sf2-themes apply wezterm --theme ryu

sf2-themes setup herdr
sf2-themes apply herdr --theme chun-li
herdr server reload-config

sf2-themes setup nvim
sf2-themes apply nvim --theme ryu-light

sf2-themes setup codex
sf2-themes apply codex --theme ryu-light
```

Neovim setup installs every catalog colorscheme as `sf2-<catalog-id>.lua` under `~/.config/nvim/colors/`, a managed current-theme pointer under `~/.config/nvim/sf2-theme/current.lua`, and a plugin loader under `~/.config/nvim/plugin/sf2-theme.lua`.
The loader applies the selected theme automatically when Neovim starts.

Codex setup writes every catalog theme as `sf2-<catalog-id>.tmTheme` under `$CODEX_HOME/themes/` and selects the active prefixed theme with `[tui].theme` in `$CODEX_HOME/config.toml`.
Restart Codex after applying a theme, or reselect it with `/theme` in an existing session.

If WezTerm's `wezterm.lua` already selects `street-fighter-2` from an older install, `setup` upgrades that assignment to the managed pointer.
If it selects some other scheme, pass `--adopt` or paste the printed snippet. `setup` will not guess at unknown Lua.

Applying or setting up a theme replaces the managed unprefixed files from older versions so the old and `sf2-` identities are not left side by side.

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
sf2-themes current codex
```

Boss aliases: `boxer` (Balrog), `claw` (Vega), `dictator` (M. Bison).

## How it writes files

- WezTerm schemes go in `~/.config/wezterm/colors/`.
- WezTerm scheme files, Herdr managed ids, Neovim colorschemes, and Codex themes use the `sf2-<catalog-id>` installed identity.
- The active WezTerm scheme is a managed pointer at `~/.config/sf2-theme/wezterm-current.lua`.
- Herdr updates only a marked block in `~/.config/herdr/config.toml`.
- Neovim colorschemes go in `~/.config/nvim/colors/`, with the active theme at `~/.config/nvim/sf2-theme/current.lua`.
- Neovim setup manages the startup loader at `~/.config/nvim/plugin/sf2-theme.lua`.
- Codex custom themes go in `~/.codex/themes/`, with the active theme in `~/.codex/config.toml` under `[tui]`.
- Symlinks are refused unless you pass `--follow-symlinks`.
- Existing files keep their mode and get a timestamped `.bak.*` copy before the first real change.

Override locations with `--config-dir`, `CODEX_HOME`, `WEZTERM_CONFIG_FILE`, `WEZTERM_CONFIG_DIR`, `HERDR_CONFIG_PATH`, `NVIM_CONFIG_DIR`, or `XDG_CONFIG_HOME`.

## Uninstall

Remove `~/.config/wezterm/colors/sf2-*.toml` and any remaining `street-fighter-ii-*.toml`, `~/.config/nvim/colors/sf2-*.lua` and any remaining `street-fighter-ii-*.lua`, `~/.config/nvim/sf2-theme/`, `~/.config/nvim/plugin/sf2-theme.lua`, `~/.codex/themes/sf2-*.tmTheme`, the Codex `[tui]` theme setting, the WezTerm integration snippet, and the marked Herdr theme block.

## Design

See [docs/theme-guidelines.md](docs/theme-guidelines.md), [docs/roster.md](docs/roster.md), and [docs/previews/](docs/previews/).

Theme data lives in [`themes/`](themes/). Dark themes keep their original IDs, and light variants use the corresponding `<id>-light` ID.
Those catalog ids remain the short CLI names; generated adapter identities add the `sf2-` prefix.
The committed `sf2-themes` script embeds a generated copy of that catalog. Do not hand-edit the embed.
