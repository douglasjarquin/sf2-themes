# How it writes files

Paths, overrides, symlinks, and backups. Moved out of the README.

- WezTerm schemes go in `~/.config/wezterm/colors/`.
- WezTerm scheme files, Herdr managed ids, Neovim colorschemes, and Codex themes use the `sf2-<catalog-id>` installed identity.
- The active WezTerm scheme is a managed pointer at `~/.config/sf2-theme/wezterm-current.lua` that auto-switches the selected character's dark and light siblings from host appearance.
- Herdr updates only a marked block in `~/.config/herdr/config.toml`, including `auto_switch` and per-mode `[theme.custom.dark]`/`[theme.custom.light]` overlays for the selected character.
- Neovim colorschemes go in `~/.config/nvim/colors/`, with the active character pair at `~/.config/nvim/sf2-theme/current.lua`.
- Neovim setup manages the startup loader at `~/.config/nvim/plugin/sf2-theme.lua`.
- Codex custom themes go in `~/.codex/themes/`, with the active theme in `~/.codex/config.toml` under `[tui]`.
- Claude Code custom themes go in `~/.claude/themes/`, with the active theme in `~/.claude/settings.json` under `theme`.
- Starship updates the marked palette in `~/.config/starship.toml` and refreshes `~/.config/sf2-theme/zsh-syntax-highlighting.zsh`.
- Lazygit themes go in its `themes/` directory, and the selected `gui.theme` plus wildcard author color are managed in `config.yml`.
- Symlinks are refused unless you pass `--follow-symlinks`.
- Existing files keep their mode and get a timestamped `.bak.*` copy before the first real change.

Override locations with `--config-dir`, `CODEX_HOME`, `CLAUDE_CONFIG_DIR`, `WEZTERM_CONFIG_FILE`, `WEZTERM_CONFIG_DIR`, `HERDR_CONFIG_PATH`, `NVIM_CONFIG_DIR`, or `XDG_CONFIG_HOME`.
