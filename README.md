# Street Fighter II Theme Pack

A standard-library Python CLI that installs Street Fighter II color themes into [WezTerm](https://wezterm.org/), [Herdr](https://herdr.dev/), Neovim, [Codex](https://github.com/openai/codex), and [Starship](https://starship.rs/).

The pack contains **36 fully resolved themes**: a dark and light variant for the shared `main` family theme plus every arcade roster theme through Super Street Fighter II Turbo.

Unofficial fan project. Street Fighter and related names are trademarks of Capcom. This project is not affiliated with or endorsed by Capcom.

## Install

Run the CLI from GitHub with uv.
You do not need a checkout or a globally installed binary.

One-shot:

```sh
uvx --from git+https://github.com/douglasjarquin/sf2-themes.git sf2-themes --version
```

Install once as a uv tool, then use the bare command:

```sh
uv tool install git+https://github.com/douglasjarquin/sf2-themes.git
sf2-themes --version
```

From a checkout, use the committed CLI, a mise task, or the repo wrapper:

```sh
./sf2-themes --version
mise run apply -- wezterm --theme vega
scripts/sf2 --version
```

`./sf2-themes` needs Python 3.11 on `PATH`.
`mise run apply` and `mise run setup` forward arguments to the project CLI.
`scripts/sf2` does the same through `uv run --project .`.

The older `uv run --with git+https://github.com/douglasjarquin/sf2-themes.git sf2-themes …` form still works.

The command requires uv and Python 3.11 or newer.

## Development with mise

The repository's `mise.toml` pins the local toolchain to Python 3.11, Node 24, uv 0.11, and aube 2.1.

```sh
mise install
mise run web:install
mise run test
mise run apply -- wezterm --theme vega
mise run setup -- wezterm
mise run web:test
mise run web:check
mise run web:build
mise run web:screenshots
mise run web:screenshots:verify
mise run web:dev
```

`mise run web:dev` starts the Astro site at `http://127.0.0.1:4321`.
`mise run web:screenshots` captures all 36 palette variants at 1280 × 720 and writes the verified archive to `web/public/screenshots/game/` with a strict manifest.
`mise run web:screenshots:verify` validates the archive without regenerating it.
The generated archive is available at `/sf2-themes/preview/` and is also used by `/sf2-themes/palette/`.

## Arcade game

The homepage includes a compact playable ASCII cabinet, and `/sf2-themes/game/` provides the full cabinet with independent theme, Player One, and Player Two selectors.
Use Arrow keys or WASD to move, Z or J for a light attack, X or K for a heavy attack, C for a projectile, Shift to block, Escape to pause, and Enter to insert a coin.
Theme changes recolor the current snapshot, while fighter selections apply to the next match.
The game is a static browser experience with a deterministic fixed-step core, authored terminal glyph art, and a visible static poster fallback when the renderer is unavailable.
The screenshot archive uses the same `/game/` capture surface and contains original project-authored terminal captures rather than raster or commercial game assets.

## Setup, then apply

`setup` is one-time application integration.
`apply` selects a theme (default: `main`).

The CLI keeps short catalog ids such as `ken` and `ryu-light` for input.
Every adapter installs and selects the corresponding prefixed ids such as `sf2-ken` and `sf2-ryu-light`.

Examples below use a uv tool install.
From a checkout, use `mise run apply -- …`, `mise run setup -- …`, or `scripts/sf2 …` in place of `sf2-themes …`.

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

sf2-themes setup starship
sf2-themes apply starship --theme vega
```

Starship apply also refreshes `~/.config/sf2-theme/zsh-syntax-highlighting.zsh`.

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
sf2-themes current starship
```

From a checkout the same commands work as `scripts/sf2 …` or `./sf2-themes …`.

Boss aliases: `boxer` (Balrog), `claw` (Vega), `dictator` (M. Bison).

## How it writes files

- WezTerm schemes go in `~/.config/wezterm/colors/`.
- WezTerm scheme files, Herdr managed ids, Neovim colorschemes, and Codex themes use the `sf2-<catalog-id>` installed identity.
- The active WezTerm scheme is a managed pointer at `~/.config/sf2-theme/wezterm-current.lua`.
- Herdr updates only a marked block in `~/.config/herdr/config.toml`.
- Neovim colorschemes go in `~/.config/nvim/colors/`, with the active theme at `~/.config/nvim/sf2-theme/current.lua`.
- Neovim setup manages the startup loader at `~/.config/nvim/plugin/sf2-theme.lua`.
- Codex custom themes go in `~/.codex/themes/`, with the active theme in `~/.codex/config.toml` under `[tui]`.
- Starship updates the marked palette in `~/.config/starship.toml` and refreshes `~/.config/sf2-theme/zsh-syntax-highlighting.zsh`.
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
