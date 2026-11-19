# How it writes files

Paths, overrides, symlinks, and backups. Moved out of the README.

- WezTerm schemes go in `~/.config/wezterm/colors/`.
- WezTerm scheme files, Herdr managed ids, Neovim colorschemes, and Codex themes use the `sf2-<catalog-id>` installed identity.
- The active WezTerm scheme is a managed pointer that auto-switches the selected character's dark and light siblings from host appearance.
  Each WezTerm configuration file has its own pointer and so its own selection, at `sf2-theme/wezterm-current-<digest>.lua` beside the file and named after its real path, so `--config-dir`, `WEZTERM_CONFIG_FILE`, a symlink, or another spelling of the same file all share it; for `~/.wezterm.lua` that is `~/sf2-theme/`.
- A `wezterm.lua` whose sf2 integration still loads a pointer an earlier release wrote, such as the shared `~/.config/sf2-theme/wezterm-current.lua`, keeps that selection and is moved to its own pointer on the next `setup` or `apply`; the older file is never written or removed, since another config may still load it.
- sf2 recognizes only the integration it generated, or its first three lines followed by your own use of `sf2_scheme`, at the top level of a `wezterm.lua` that parses; a comment, string, or function that mentions it changes nothing, and sf2 edits nothing else in the file except removing a whole `config.color_scheme = ...` line when you pass `--adopt`.
- A config sf2 cannot edit safely is left unchanged and `setup` prints the snippet to add by hand.
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
- Commands that write take a lock on `manifest.lock` and decide their changes again under it, so concurrent runs record every target and a queued run never undoes one that finished first; a run that fails part way keeps every earlier record and records only what it wrote.
- Before the first application write, sf2 stages the manifest's next save and checks the existing manifest can be replaced, so a state directory sf2 cannot write, or an immutable manifest, stops the command with the application unchanged; a run that changes nothing writes no state at all.
- If a command is killed after changing an application but before recording it, run the same command again: it records what is on disk and removes the leftover `.manifest.json.*` file.

Override locations with `--config-dir`, `CODEX_HOME`, `CLAUDE_CONFIG_DIR`, `WEZTERM_CONFIG_FILE`, `WEZTERM_CONFIG_DIR`, `HERDR_CONFIG_PATH`, `NVIM_CONFIG_DIR`, or `XDG_CONFIG_HOME`; the state directory follows `XDG_STATE_HOME`.
