# PYTHON ADAPTER OWNERSHIP

## Scope

This guidance applies to `src/sf2_theme/adapters/` and adapter-facing changes in `src/sf2_theme/filesystem.py`.
Adapters translate validated `Theme` values into consumer files and conservative configuration edits.
Keep path discovery, rendering, ownership classification, and current-theme reads inside the owning adapter.
Treat `theme.metadata.selectable_id` as the generated filename and configured identity while CLI catalog IDs remain short.

## Adapter contract

- Plan, never write: `plan_<app>(theme, themes, config_dir, slot)` returns `Write` and `Remove` steps from `sf2_theme.plan`; only `filesystem.commit` touches disk, and it owns dry-run and the symlink policy.
- Inspect, never refuse: `inspect_<app>(themes, config_dir)` returns one `Inspection` holding the ownership class (absent, managed, legacy, foreign, unknown shape, broken) and the current sf2 theme when it resolves through `installed_theme()`.
- Broken and unknown-shape inspections carry the reason or the manual fix in `detail`; the CLI turns them into the one refusal message.
- The CLI's ownership gate decides whether to write the theme setting; `slot=False` means plan the theme files only and leave the setting untouched.
- The CLI resolves the selection once and passes a concrete theme; merges replace whatever owns the setting, because a foreign one only reaches them after `--adopt`.
- `current <app>` reads through the same inspection, so a selection `setup` reports as kept is always one `current` prints.

## WezTerm

- The adapter owns catalog scheme files under `<wezterm-config>/colors/` and the pointer `<XDG_CONFIG_HOME>/sf2-theme/wezterm-current.lua`, and edits `wezterm.lua` only when the gate allows the setting.
- The pointer returns the dark or light sibling from `wezterm.gui.get_appearance()` so a selected character follows host light/dark instead of pinning one scheme.
- Selecting either sibling writes the same pair; the managed identity is the dark `sf2-<catalog-id>`.
- Ownership comes from `wezterm.lua` and the selection from the pointer, so a foreign Lua config with a readable pointer is foreign for the gate and kept for the selection.
- Every run that writes anything plans the pointer, including plain setup over a foreign config and an unknown shape.
- Classification ignores Lua comment lines and recognizes pointer integration before the config shape, so a pasted snippet in any shape is managed.
- Empty Lua is absent and receives a starter config; a recognized `wezterm.config_builder()` shape receives pointer integration before its matching return.
- Integration without the `TERM_THEME` guard, or a literal SF2 `color_scheme` on a builder, is legacy and is upgraded while preserving unrelated Lua.
- A non-SF2 `color_scheme` on a builder is foreign; adoption removes recognized color-scheme assignments and preserves every other Lua line.
- Any other shape is unknown: `wezterm.lua` stays byte-for-byte unchanged, scheme files and pointer are still written, and the run exits nonzero with the snippet.
- Scheme files include compose_cursor, visual_bell, indexed 16/17, and retro `tab_bar` colors; fancy-tab `window_frame` stays Lua-only.

## Starship

- `apply starship` owns the marked palette block in `starship.toml` and refreshes `sf2-theme/zsh-syntax-highlighting.zsh`.
- The block sits in the top-level key region, before the first table, because Starship reads `palette` only at the top level; a block found elsewhere moves there.
- A top-level `palette` outside the markers is foreign; when plain setup keeps it, neither file is written.
- Parse the merged document before returning the plan.
- Named Starship styles (`blue`, `purple`, …) resolve through the managed `sf2` palette onto accent and semantic tokens.
- The zsh snippet must be sourced after `zsh-syntax-highlighting.zsh`; setup prints the one-line hint.

## Herdr

- The adapter owns exactly the block between the `sf2-themes managed theme` markers.
- Resolve the selected catalog id to its dark/light sibling pair and write `auto_switch`, `light_name`, `dark_name`, and per-mode `[theme.custom.dark]`/`[theme.custom.light]` overlays (requires a Herdr build with herdr#2324; stable releases before it silently drop these tables and stay pinned to one palette).
- Selecting either sibling (`chun-li` or `chun-li-light`) writes the same pair; the managed identity is the dark `sf2-<catalog-id>`.
- Replace an existing complete managed block in place; incomplete markers or invalid TOML inspect as broken.
- An unmarked `[theme]`, `[theme.*]`, or top-level `theme =` is foreign.
- Adoption removes only the existing theme namespace and preserves unrelated tables, keys, and top-level values.
- Parse the merged document before writing so preservation never produces invalid Herdr configuration.

## Herdr app

- `herdr-app` is the desktop JSON adapter; `herdr` remains the independent TUI TOML adapter.
- Install the catalog under `<config-dir>/themes/sf2-<catalog-id>.json` and merge only `theme.light`/`theme.dark` in `settings.json`.
- Default to `$XDG_CONFIG_HOME/herdr-app` or `~/.config/herdr-app`; preserve appearance, layout, top-level colors, and other settings.
- Only a matching catalog pair is managed; any other selection, including two different SF2 families, is foreign.
- Current reports the dark installed ID only for a matching catalog pair; it reads settings, not effective layout/picker state.
- Terminal fields use canonical UI/ANSI colors; per-file UI overrides map canvas/background, chrome/surface, surface/overlay, border/border, and accent/accent.
- Invalid JSON, a non-object document, or a `theme` that is neither a name nor an object inspects as broken.
- Verify through `tests/test_herdr_app.py` and the copied standalone harness.

## Neovim

- The adapter owns catalog files `<nvim-config>/colors/sf2-*.lua` and the selected pointer `<nvim-config>/sf2-theme/current.lua`.
- The pointer selects the dark or light sibling from `TERM_THEME` or `'background'` so Neovim follows the same character pair as WezTerm and Herdr.
- Selecting either sibling writes the same pair; the managed identity is the dark `sf2-<catalog-id>`.
- Every run also owns `<nvim-config>/plugin/sf2-theme.lua` as the startup loader.
- Nothing in these files is ever foreign; a pointer whose id does not resolve is an unreadable selection, and a legacy short ID resolves and is rewritten with the `sf2-` identity.
- Limit legacy cleanup to enumerated `street-fighter-ii-<catalog-id>.lua` files and leave every other `colors/` or `plugin/` entry untouched.

## Codex

- The adapter owns catalog files `<codex-home>/themes/sf2-*.tmTheme` and only the `theme` assignment inside `[tui]` in `<codex-home>/config.toml`.
- Preserve every other Codex table, key, comment, and existing `[tui]` setting when selecting a theme.
- An inline `tui` form is an unknown shape whose manual fix is converting it to a `[tui]` table; never restructure it.
- A `[tui].theme` equal to a short catalog ID is legacy and is rewritten with the `sf2-` identity; any non-`sf2-` value is foreign.
- Limit legacy cleanup to unprefixed `<catalog-id>.tmTheme` files represented by the current catalog.

## Lazygit

- The adapter owns `<lazygit-config>/themes/sf2-<catalog-id>.yml` for every catalog entry and the selected theme sections in `<lazygit-config>/config.yml`.
- Theme fragments must cover every current Lazygit `gui.theme` key plus wildcard `authorColors['*']`, with SF2 semantic roles rather than Catppuccin hues.
- Preserve unrelated `gui` settings and named author colors, treat an unmarked existing `gui.theme` as foreign, and keep the managed blocks idempotent.
- Incomplete or out-of-order markers inspect as broken.
- Resolve the config directory from `--config-dir`, `LAZYGIT_CONFIG_DIR`, or the platform's Lazygit default.

## Claude Code

- The adapter owns catalog files `<claude-home>/themes/sf2-*.json` and only the `theme` key in `<claude-home>/settings.json`, resolved from `--config-dir`, `CLAUDE_CONFIG_DIR`, or `~/.claude`.
- Claude Code's custom-theme schema (`name`/`base`/`overrides`) is an experimental component with no publicly documented full token list; `OVERRIDE_TOKENS` is a best-effort mapping onto SF2 roles, confirmed live in a running session for the `claude`/`error`/`success` tokens the docs do specify.
- Claude Code has no host-appearance auto-switch for custom themes (confirmed absent from the docs), so selecting a light or dark sibling here just pins that one theme in settings.json like Codex, not a following pair like WezTerm/Herdr/Neovim.
- Preserve every other `settings.json` key; only ever touch the top-level `theme` string, written as `custom:<selectable-id>`.
- A `custom:sf2-*` theme is managed, and unreadable when it does not resolve; any other `theme` is foreign.

## Slack

- Slack has no config file and stays outside the inspect, plan, and commit path. `render_theme` returns the four-color import line for the requested catalog theme.
- Do not install an `sf2-` identity, resolve a dark/light pair, or call `project_adapter_colors()`.
- `current slack` fails closed, prints no theme id, and writes nothing.
- `--config-dir`, `--follow-symlinks`, and `--adopt` do not apply.

## Shared write contract

- Return every managed text write and legacy-file removal as a plan step so every adapter shares one mutation contract.
- `commit` resolves every destination before the first write; refuse destination symlinks by default, follow only when explicitly requested, write the resolved target, and retain the symlink itself.
- Dry-run reports `would_create`, `would_update`, or `would_remove` with a diff and creates no file, directory, backup, or legacy cleanup mutation.
- A changed existing file receives a UTC-stamped backup before atomic replacement, with its mode preserved, while unchanged content creates no backup.
- Never broaden cleanup beyond exact managed paths and enumerated legacy names or remove unknown files, stale-looking aliases, unmarked blocks, or user-owned configuration.

## Focused verification

- Run `uv run pytest -q tests/test_wezterm.py tests/test_cli.py tests/test_snapshots.py` for WezTerm ownership or Lua integration changes.
- Run `uv run pytest -q tests/test_herdr.py tests/test_snapshots.py` for Herdr merge or rendering changes.
- Run `uv run pytest -q tests/test_nvim.py tests/test_snapshots.py` for Neovim layout or rendering changes.
- Run `uv run pytest -q tests/test_codex.py` for Codex theme or config changes.
- Run `uv run pytest -q tests/test_lazygit.py tests/test_cli.py tests/test_snapshots.py` for Lazygit rendering, merge, or dispatch changes.
- Run `uv run pytest -q tests/test_claude.py` for Claude Code theme or settings changes.
- Run `uv run pytest -q tests/test_slack.py` for Slack import-line or dispatch changes.
- Run `uv run pytest -q tests/test_ownership.py tests/test_cli.py` for ownership gate or selection changes.
- Run `uv run pytest -q tests/test_plan.py tests/test_filesystem.py` plus every touched adapter suite for shared write changes.
- After adapter behavior changes, follow the inherited standalone regeneration rule and run `mise run test-cli` to exercise the copied CLI across all adapters.
- Completion requires focused tests to pass with dry-run, backup, symlink, preservation, and exact-path ownership expectations intact.
