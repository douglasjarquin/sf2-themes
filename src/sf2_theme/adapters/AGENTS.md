# PYTHON ADAPTER OWNERSHIP

## Scope

This guidance applies to `src/sf2_theme/adapters/` and adapter-facing changes in `src/sf2_theme/filesystem.py`.
Adapters translate validated `Theme` values into consumer files and conservative configuration edits.
Keep path discovery, rendering, ownership decisions, and current-theme reads inside the owning adapter.
Treat `theme.metadata.selectable_id` as the generated filename and configured identity while CLI catalog IDs remain short.

## WezTerm

- `apply_wezterm` owns catalog scheme files under `<wezterm-config>/colors/` and the pointer `<XDG_CONFIG_HOME>/sf2-theme/wezterm-current.lua`; it never edits `wezterm.lua`.
- `setup_wezterm` adds one-time Lua integration after writing schemes, with the pointer replaced only when requested or absent.
- Empty Lua receives a starter config, while a recognized `wezterm.config_builder()` shape receives pointer integration before its matching return.
- Existing managed pointer integration is idempotent.
- Unknown Lua shapes remain byte-for-byte unchanged and return a pasteable integration snippet.
- A foreign `color_scheme` remains untouched unless `--adopt` is explicit.
- Adoption may remove recognized color-scheme assignments from a safe builder but must preserve every other Lua line.
- A known legacy SF2 assignment may be upgraded without adoption while preserving unrelated Lua.
- Scheme files include compose_cursor, visual_bell, indexed 16/17, and retro `tab_bar` colors; fancy-tab `window_frame` stays Lua-only.

## Starship

- `apply starship` owns the marked palette block in `starship.toml` and refreshes `sf2-theme/zsh-syntax-highlighting.zsh`.
- Named Starship styles (`blue`, `purple`, …) resolve through the managed `sf2` palette onto accent and semantic tokens.
- The zsh snippet must be sourced after `zsh-syntax-highlighting.zsh`; setup prints the one-line hint.

## Herdr

- Both `setup herdr` and `apply herdr` delegate to `apply_herdr`, which owns exactly the block between the `sf2-themes managed theme` markers.
- Resolve the selected catalog id to its dark/light sibling pair and write `auto_switch`, `light_name`, `dark_name`, `[theme.custom.dark]`, and `[theme.custom.light]` so Herdr can follow host-terminal appearance.
- Selecting either sibling (`chun-li` or `chun-li-light`) writes the same pair; the managed identity is the dark `sf2-<catalog-id>`.
- Replace an existing complete managed block in place and fail closed on incomplete markers or invalid TOML.
- An unmarked `[theme]`, `[theme.*]`, or top-level `theme =` remains user-owned unless `--adopt` is explicit.
- Adoption removes only the existing theme namespace and preserves unrelated tables, keys, and top-level values.
- Parse the merged document before writing so preservation never produces invalid Herdr configuration.

## Neovim

- The adapter owns catalog files `<nvim-config>/colors/sf2-*.lua` and the selected pointer `<nvim-config>/sf2-theme/current.lua`.
- `apply_nvim` writes the catalog and pointer, while `setup_nvim` additionally owns `<nvim-config>/plugin/sf2-theme.lua` as the startup loader.
- Setup without an explicit theme preserves a valid managed pointer and only normalizes a recognized legacy short ID.
- Limit legacy cleanup to enumerated `street-fighter-ii-<catalog-id>.lua` files and leave every other `colors/` or `plugin/` entry untouched.

## Codex

- The adapter owns catalog files `<codex-home>/themes/sf2-*.tmTheme` and only the `theme` assignment inside `[tui]` in `<codex-home>/config.toml`.
- Preserve every other Codex table, key, comment, and existing `[tui]` setting when selecting a theme.
- Refuse ambiguous inline `tui` forms instead of restructuring user configuration.
- Setup without an explicit theme preserves an existing selection and normalizes only recognized legacy short IDs, while apply selects the requested or default theme.
- Limit legacy cleanup to unprefixed `<catalog-id>.tmTheme` files represented by the current catalog.

## Lazygit

- `apply_lazygit` and `setup_lazygit` own `<lazygit-config>/themes/sf2-<catalog-id>.yml` for every catalog entry and the selected theme sections in `<lazygit-config>/config.yml`.
- Theme fragments must cover every current Lazygit `gui.theme` key plus wildcard `authorColors['*']`, with SF2 semantic roles rather than Catppuccin hues.
- Preserve unrelated `gui` settings and named author colors, refuse an unmarked existing `gui.theme` unless `--adopt` is explicit, and keep the managed blocks idempotent.
- Resolve the config directory from `--config-dir`, `LAZYGIT_CONFIG_DIR`, or the platform's Lazygit default, and route every write through `write_file`.

## Shared write contract

- Route managed text writes through `write_file` so every adapter shares one mutation contract.
- Refuse destination symlinks by default; follow only when explicitly requested, write the resolved target, and retain the symlink itself.
- Dry-run reports `would_create` or `would_update` with a diff and creates no file, directory, backup, or legacy cleanup mutation.
- A changed existing file receives a UTC-stamped backup before atomic replacement, with its mode preserved, while unchanged content creates no backup.
- Never broaden cleanup beyond exact managed paths and enumerated legacy names or remove unknown files, stale-looking aliases, unmarked blocks, or user-owned configuration.

## Focused verification

- Run `uv run --with pytest pytest -q tests/test_wezterm.py tests/test_cli.py tests/test_snapshots.py` for WezTerm ownership or Lua integration changes.
- Run `uv run --with pytest pytest -q tests/test_herdr.py tests/test_snapshots.py` for Herdr merge or rendering changes.
- Run `uv run --with pytest pytest -q tests/test_nvim.py tests/test_snapshots.py` for Neovim layout or rendering changes.
- Run `uv run --with pytest pytest -q tests/test_codex.py` for Codex theme or config changes.
- Run `uv run --with pytest pytest -q tests/test_lazygit.py tests/test_cli.py tests/test_snapshots.py` for Lazygit rendering, merge, or dispatch changes.
- Run `uv run --with pytest pytest -q tests/test_filesystem.py` plus every touched adapter suite for shared write changes.
- After adapter behavior changes, follow the inherited standalone regeneration rule and run `bash tests/test_cli.sh` to exercise the copied CLI across all adapters.
- Completion requires focused tests to pass with dry-run, backup, symlink, preservation, and exact-path ownership expectations intact.
