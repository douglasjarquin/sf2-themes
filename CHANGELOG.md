# Changelog

## 1.0.0

- Replace the five-color brand palette with 18 fully resolved themes (`main` plus the Super Turbo roster).
- Give WezTerm real ANSI slots, distinct bright colors, and navy-on-gold selection.
- Populate every Herdr theme token so Catppuccin no longer leaks through.
- Split `setup` (one-time integration) from `apply` (select a theme).
- Stop rewriting unknown WezTerm Lua.
- Refuse symlink replacement by default, preserve file modes, write timestamped backups, and support `--dry-run`.
- Validate `#RRGGBB`, contrast, aliases, and renderer snapshots in CI.
