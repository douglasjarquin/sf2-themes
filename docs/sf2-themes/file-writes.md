# How it writes files

Paths, overrides, symlinks, and backups. Moved out of the README.

- WezTerm schemes go in `~/.config/wezterm/colors/`.
- WezTerm scheme files, Herdr managed ids, Neovim colorschemes, and Codex themes use the `sf2-<catalog-id>` installed identity.
- The active WezTerm scheme is a managed pointer that auto-switches the selected character's dark and light siblings from host appearance.
  Each WezTerm configuration file has its own pointer, at `sf2-theme/wezterm-current-<digest>.lua` beside the file the filesystem opens and named after its path, so `--config-dir`, `WEZTERM_CONFIG_FILE`, or a symlink to the same file all share it; for `~/.wezterm.lua` that is `~/sf2-theme/`.
- A `wezterm.lua` whose sf2-themes integration still loads a pointer an earlier release wrote, such as the shared `~/.config/sf2-theme/wezterm-current.lua`, keeps that selection and is moved to its own pointer on the next `setup` or `apply`; the older file is never written or removed, since another config may still load it.
- sf2-themes writes `wezterm.lua` only as a new file, by replacing its own integration as a whole, or, with `--adopt`, by adding its integration to a file whose every line it can verify; everything else is left to you with the exact steps.
- A config path that goes through `..`, or spells an existing directory or file differently from the disk (letter case or Unicode normalization), is refused for every app; pass the real path.
  The same holds for every symlink the path follows: a link to another spelling of a file, or through a missing directory and back out with `..`, is refused rather than resolved to whatever file a lexical reading names.
- Herdr app installs `~/.config/herdr-app/themes/sf2-*.json` and selects a light/dark pair in `~/.config/herdr-app/settings.json`.
- Herdr TUI updates only a marked block in `~/.config/herdr/config.toml`, including `auto_switch` and per-mode `[theme.custom.dark]`/`[theme.custom.light]` overlays for the selected character.
- Neovim colorschemes go in `~/.config/nvim/colors/`, with the active character pair at `~/.config/nvim/sf2-theme/current.lua`.
- Neovim also manages the startup loader at `~/.config/nvim/plugin/sf2-theme.lua`.
- Codex custom themes go in `~/.codex/themes/`, with the active theme in `~/.codex/config.toml` under `[tui]`.
- Claude Code custom themes go in `~/.claude/themes/`, with the active theme in `~/.claude/settings.json` under `theme`.
- Starship updates the marked palette before the first table in `~/.config/starship.toml` and refreshes `~/.config/sf2-theme/zsh-syntax-highlighting.zsh`.
- Lazygit themes go in its `themes/` directory, and the selected `gui.theme` plus wildcard author color are managed in `config.yml`.
- Every write is planned and checked before the first one happens, so a refusal, a symlink, or a file sf2 cannot replace anywhere in the plan leaves every file untouched.
- Before writing, sf2 checks that each write can succeed: nothing but a regular file stands where it writes, the file is not immutable or append-only, and its directory takes new files.
- sf2 deletes nothing, and it replaces a file other than the app's own config only when it wrote that file: the manifest records its SHA-256, or it holds exactly what sf2 writes there for some theme, as releases before the manifest wrote. A file you made or changed, whatever its name, stops the command until you move it aside.
- Files earlier releases named `street-fighter-ii-*` (WezTerm and Neovim) or `<id>.tmTheme` (Codex) are left in place; remove them yourself when you no longer use them.
- Symlinks are refused unless you pass `--follow-symlinks`.
- Existing files keep their mode and get a timestamped `.bak.*` copy before the first real change; a second copy in the same second gets a `.1` suffix instead of replacing the first.
- Every `setup` or `apply` that writes records what it owns in `~/.local/state/sf2/manifest.json` (or under `XDG_STATE_HOME`): per app and configuration target, the selection, each managed file with its SHA-256, and the edited setting file.
- The first time sf2 changes a setting file you already had, a copy of your original is kept under `~/.local/state/sf2/backups/<app>/`.
- `--dry-run` writes no state, and an unreadable manifest, or a symlink or wrong kind of file where sf2 keeps its state, stops the command before any file is written.
- Commands that write take a lock on `manifest.lock` and decide their changes again under it, so concurrent runs record every target and a queued run never undoes one that finished first.
- A run that fails part way, or whose record cannot be saved, puts back every file it wrote (bytes, mode, and times), removes the backups and directories it made, and returns the manifest and kept copies to what they were. Only the state directory and its `manifest.lock` stay, because another command may be waiting on the lock.
- Before the first application write, sf2 stages the manifest's next save and checks the existing manifest can be replaced (not immutable or append-only, and replaceable in a sticky directory), so a state directory sf2 cannot write, or an immutable manifest, stops the command with the application unchanged; a run that changes nothing writes no state at all.
- If a command is killed after changing an application but before recording it, run the same command again: it records what is on disk and removes the leftover `.manifest.json.*` file.

Override locations with `--config-dir`, `CODEX_HOME`, `CLAUDE_CONFIG_DIR`, `WEZTERM_CONFIG_FILE`, `WEZTERM_CONFIG_DIR`, `HERDR_CONFIG_PATH`, `NVIM_CONFIG_DIR`, or `XDG_CONFIG_HOME`; the state directory follows `XDG_STATE_HOME`.
