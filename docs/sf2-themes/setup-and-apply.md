# Setup, then apply

One-time integration, then a theme selection. Moved out of the README.

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

sf2-themes setup claude
sf2-themes apply claude --theme ryu-light

sf2-themes setup starship
sf2-themes apply starship --theme vega

sf2-themes setup lazygit
sf2-themes apply lazygit --theme vega
```

Starship apply also refreshes `~/.config/sf2-theme/zsh-syntax-highlighting.zsh`.

Lazygit setup and apply install all 36 complete theme fragments under its `themes/` directory and select the requested theme in `config.yml`.
Use `--adopt` when an existing `gui.theme` section is not managed by sf2-themes.

If WezTerm's `wezterm.lua` already selects `street-fighter-2` from an older install, `setup` upgrades that assignment to the managed pointer.
If it selects some other scheme, pass `--adopt` or paste the printed snippet. `setup` will not guess at unknown Lua.

WezTerm apply writes every catalog scheme and a pointer that returns the selected character's dark or light sibling from `wezterm.gui.get_appearance()`. Applying `ryu` or `ryu-light` selects the same pair.

Neovim setup installs every catalog colorscheme as `sf2-<catalog-id>.lua` under `~/.config/nvim/colors/`, a managed current-theme pointer under `~/.config/nvim/sf2-theme/current.lua`, and a plugin loader under `~/.config/nvim/plugin/sf2-theme.lua`.
The pointer selects the matching dark or light colorscheme from `TERM_THEME` (set by the WezTerm integration) or `'background'`. Applying `ryu` or `ryu-light` selects the same pair.

Codex setup writes every catalog theme as `sf2-<catalog-id>.tmTheme` under `$CODEX_HOME/themes/` and selects the active prefixed theme with `[tui].theme` in `$CODEX_HOME/config.toml`.
Restart Codex after applying a theme, or reselect it with `/theme` in an existing session.

Claude Code setup writes every catalog theme as `sf2-<catalog-id>.json` under `~/.claude/themes/` and selects it with `theme` in `~/.claude/settings.json`. Reselect it with `/theme` in an existing session. Claude Code has no host-appearance auto-switch for custom themes, so applying `ryu` or `ryu-light` just pins that one sibling, like Codex.

Applying or setting up a theme replaces the managed unprefixed files from older versions so the old and `sf2-` identities are not left side by side.

Herdr apply enables `auto_switch` for the selected character (or `main`) and writes per-mode `[theme.custom.dark]`/`[theme.custom.light]` overlays, so the character palette itself follows host appearance (needs a Herdr build with herdr#2324; not in a stable release yet). Applying `chun-li` or `chun-li-light` selects the same pair. Reload with `herdr server reload-config` after applying.

Herdr configs that already have an unmarked `[theme]` section are left alone unless you pass `--adopt`.

`install` still works as a deprecated alias for `apply`.
