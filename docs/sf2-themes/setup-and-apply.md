# Apply, and the setup fallback

One command per app, then nothing. Moved out of the README.

`apply` prepares the theme assets, installs or repairs the app's integration, and selects a theme (default: `main`) in one step.
It is idempotent: the first run and the hundredth do the same work, and a repeat with nothing to do reports `no changes`.
When an integration step cannot be completed safely — for example a `wezterm.lua` the CLI cannot edit — `apply` reports the manual step on stderr and exits nonzero instead of pretending success.

`setup` runs the same integration but keeps an existing selection unless `--theme` is given.
Use it to repair integration without changing the active theme.

`install` still works as a deprecated alias for `apply`.

The CLI keeps short catalog ids such as `ken` and `ryu-light` for input.
Every adapter installs and selects the corresponding prefixed ids such as `sf2-ken` and `sf2-ryu-light`.

Every mutation command takes `--dry-run` to print the full plan without writing, `--verbose` to list every file operation, `--config-dir` to write under a different directory, and `--follow-symlinks` to write through destination symlinks.
`--adopt` is valid only for `wezterm`, `herdr`, and `lazygit`.

Examples below assume a uv tool install; `sf2-themes` works in place of `sf2`.
From a checkout, use `mise run apply -- …` or `scripts/sf2 …` in place of `sf2 …`.

```sh
sf2 apply wezterm
sf2 apply wezterm --theme ryu

sf2 apply herdr --theme chun-li
herdr server reload-config

sf2 apply nvim --theme ryu-light

sf2 apply codex --theme ryu-light

sf2 apply claude --theme ryu-light

sf2 apply starship --theme vega

sf2 apply lazygit --theme vega
```

Starship apply also refreshes `~/.config/sf2-theme/zsh-syntax-highlighting.zsh`; source it after `zsh-syntax-highlighting.zsh` to tint zsh command tokens.

Lazygit installs all 36 complete theme fragments under its `themes/` directory and selects the requested theme in `config.yml`.
Use `--adopt` when an existing `gui.theme` section is not managed by sf2-themes.

WezTerm apply writes every catalog scheme, a pointer that returns the selected character's dark or light sibling from `wezterm.gui.get_appearance()`, and a Lua integration in `wezterm.lua` that loads the pointer.
Applying `ryu` or `ryu-light` selects the same pair.
An empty or missing `wezterm.lua` gets a starter config, and a recognized `wezterm.config_builder()` shape gets the integration inserted before its return.
If `wezterm.lua` already selects `street-fighter-2` from an older install, the assignment is upgraded to the managed pointer.
Other shapes are left byte-for-byte unchanged: `apply` prints a pasteable snippet and exits nonzero, and `--adopt` is accepted when the config only assigns a foreign `color_scheme`.
`setup` will not guess at unknown Lua either, but reports the snippet as a warning and still exits zero.

Neovim apply installs every catalog colorscheme as `sf2-<catalog-id>.lua` under `~/.config/nvim/colors/`, a managed current-theme pointer under `~/.config/nvim/sf2-theme/current.lua`, and a plugin loader under `~/.config/nvim/plugin/sf2-theme.lua` so the pointer is read on every startup.
The pointer selects the matching dark or light colorscheme from `TERM_THEME` (set by the WezTerm integration) or `'background'`. Applying `ryu` or `ryu-light` selects the same pair.

Codex apply writes every catalog theme as `sf2-<catalog-id>.tmTheme` under `$CODEX_HOME/themes/` and selects the active prefixed theme with `[tui].theme` in `$CODEX_HOME/config.toml`.
Restart Codex after applying a theme, or reselect it with `/theme` in an existing session.

Claude Code apply writes every catalog theme as `sf2-<catalog-id>.json` under `~/.claude/themes/` and selects it with `theme` in `~/.claude/settings.json`. Reselect it with `/theme` in an existing session. Claude Code has no host-appearance auto-switch for custom themes, so applying `ryu` or `ryu-light` just pins that one sibling, like Codex.

Applying or setting up a theme replaces the managed unprefixed files from older versions so the old and `sf2-` identities are not left side by side.

Herdr apply enables `auto_switch` for the selected character (or `main`) and writes per-mode `[theme.custom.dark]`/`[theme.custom.light]` overlays, so the character palette itself follows host appearance (needs a Herdr build with herdr#2324; not in a stable release yet). Applying `chun-li` or `chun-li-light` selects the same pair. Reload with `herdr server reload-config` after applying.

Herdr configs that already have an unmarked `[theme]` section are left alone unless you pass `--adopt`.
