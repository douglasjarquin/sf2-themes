# Uninstall

Files and settings the CLI leaves behind. Moved out of the README.

Remove `~/.config/wezterm/colors/sf2-*.toml` and any remaining `street-fighter-ii-*.toml`, `~/.config/nvim/colors/sf2-*.lua` and any remaining `street-fighter-ii-*.lua`, `~/.config/nvim/sf2-theme/`, `~/.config/nvim/plugin/sf2-theme.lua`, `~/.codex/themes/sf2-*.tmTheme`, `~/.claude/themes/sf2-*.json`, the Lazygit `themes/sf2-*.yml` files and marked config sections, the Codex `[tui]` theme setting, the Claude Code `theme` setting, the WezTerm integration snippet, and the marked Herdr theme block.

Each WezTerm config's selection pointer is a `wezterm-current-<digest>.lua` file in the `sf2-theme/` directory beside that config: `~/.config/wezterm/sf2-theme/` for the default config, `~/sf2-theme/` for `~/.wezterm.lua`, and beside any `--config-dir` or `WEZTERM_CONFIG_FILE` config.
The manifest lists every pointer it recorded under `files`; remove those files, then remove each `sf2-theme/` directory only if it is empty, because a directory beside your config may hold other files.
Earlier releases also wrote `wezterm-current.lua` pointers: the shared `~/.config/sf2-theme/wezterm-current.lua`, and `~/.config/wezterm/sf2-theme/wezterm-current.lua`, which an alternate config set with `WEZTERM_CONFIG_FILE` may also have loaded; remove them once no `wezterm.lua` names them.

The ownership record and the original settings it kept live in `~/.local/state/sf2/`; restore any original you want from `backups/`, then remove the directory.

Then remove the binary and its link from the install directory, `~/.local/bin` unless you set `SF2_THEMES_INSTALL_DIR`:

```sh
rm ~/.local/bin/sf2-themes ~/.local/bin/sf2
```

A 1.x install made with uv is removed with `uv tool uninstall sf2-themes` instead.
