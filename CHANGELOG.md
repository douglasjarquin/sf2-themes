# Changelog

## Unreleased

- WezTerm, Herdr, and Neovim apply a selected character as a dark/light pair and auto-switch with host appearance instead of pinning a single variant.
- Herdr writes `auto_switch` plus per-mode `[theme.custom.dark]`/`[theme.custom.light]` overlays (needs a Herdr build with herdr#2324; not yet in a stable release). WezTerm's pointer follows `get_appearance()`. Neovim's pointer follows `TERM_THEME` or `'background'`.
- Neovim's pointer now re-syncs on `FocusGained`/`VimResume` and a timer instead of only at startup, since `TERM_THEME` freezes in the process environment and a host appearance change made after launch never reached an already-running Neovim.

## 1.0.1

- Upgrade a previous `config.color_scheme = "street-fighter-2"` assignment during `setup wezterm` instead of leaving WezTerm pinned to the old scheme file.
- Keep an already-selected theme when `setup wezterm` is re-run without `--theme`.
- Support `--adopt` for WezTerm, matching Herdr.

## 1.0.0

- Replace the five-color brand palette with 18 fully resolved themes (`main` plus the Super Turbo roster).
- Give WezTerm real ANSI slots, distinct bright colors, and navy-on-gold selection.
- Populate every Herdr theme token so Catppuccin no longer leaks through.
- Split `setup` (one-time integration) from `apply` (select a theme).
- Stop rewriting unknown WezTerm Lua.
- Refuse symlink replacement by default, preserve file modes, write timestamped backups, and support `--dry-run`.
- Validate `#RRGGBB`, contrast, aliases, and renderer snapshots in CI.
