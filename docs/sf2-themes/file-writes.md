# How it writes files

Paths, overrides, symlinks, and backups. Moved out of the README.

- WezTerm schemes go in `~/.config/wezterm/colors/`.
- WezTerm scheme files, Herdr managed ids, Neovim colorschemes, and Codex themes use the `sf2-<catalog-id>` installed identity.
- The active WezTerm scheme is a managed pointer at `~/.config/wezterm/sf2-theme/wezterm-current.lua` that auto-switches the selected character's dark and light siblings from host appearance.
  Each WezTerm configuration target, including each `--config-dir` and each `WEZTERM_CONFIG_FILE`, has its own pointer and so its own selection; a `wezterm.lua` outside its WezTerm directory gets `sf2-theme/wezterm-current-<digest>.lua` named after its path.
- A `wezterm.lua` whose `local sf2_current = ...` line still names the shared pointer earlier releases wrote at `~/.config/sf2-theme/wezterm-current.lua` keeps that selection and is moved to its own pointer on the next `setup` or `apply`; the shared file is never written or removed, and a comment that mentions it changes nothing.
- An integration that loads any other pointer is left unchanged and `setup` prints the snippet to fix it by hand.
- Herdr app installs `~/.config/herdr-app/themes/sf2-*.json` and selects a light/dark pair in `~/.config/herdr-app/settings.json`.
- Herdr TUI updates only a marked block in `~/.config/herdr/config.toml`, including `auto_switch` and per-mode `[theme.custom.dark]`/`[theme.custom.light]` overlays for the selected character.
- Neovim colorschemes go in `~/.config/nvim/colors/`, with the active character pair at `~/.config/nvim/sf2-theme/current.lua`.
- Neovim also manages the startup loader at `~/.config/nvim/plugin/sf2-theme.lua`.
- Codex custom themes go in `~/.codex/themes/`, with the active theme in `~/.codex/config.toml` under `[tui]`.
- Claude Code custom themes go in `~/.claude/themes/`, with the active theme in `~/.claude/settings.json` under `theme`.
- Starship updates the marked palette before the first table in `~/.config/starship.toml` and refreshes `~/.config/sf2-theme/zsh-syntax-highlighting.zsh`.
- Lazygit themes go in its `themes/` directory, and the selected `gui.theme` plus wildcard author color are managed in `config.yml`.
- Every write and legacy-file removal is planned before the first one happens, so a refusal or a symlink anywhere in the plan leaves every file untouched.
- Symlinks are refused unless you pass `--follow-symlinks`.
- Existing files keep their mode and get a timestamped `.bak.*` copy before the first real change.
- Every `setup` or `apply` that writes records what it owns in `~/.local/state/sf2/manifest.json` (or under `XDG_STATE_HOME`): per app and configuration target, the selection, each managed file with its SHA-256, and the edited setting file.
- The first time sf2 changes a setting file you already had, a copy of your original is kept under `~/.local/state/sf2/backups/<app>/`.
- `--dry-run` writes no state, and an unreadable manifest, or a symlink or wrong kind of file where sf2 keeps its state, stops the command before any file is written.
- Commands that write take a lock on `manifest.lock`, so concurrent runs record every target; a run that fails part way keeps every earlier record and records only what it wrote.

Override locations with `--config-dir`, `CODEX_HOME`, `CLAUDE_CONFIG_DIR`, `WEZTERM_CONFIG_FILE`, `WEZTERM_CONFIG_DIR`, `HERDR_CONFIG_PATH`, `NVIM_CONFIG_DIR`, or `XDG_CONFIG_HOME`; the state directory follows `XDG_STATE_HOME`.
