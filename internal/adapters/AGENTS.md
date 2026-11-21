# GO ADAPTER OWNERSHIP

## Scope

This guidance applies to `internal/adapters/` and adapter-facing changes in `internal/plan/` and `internal/fsys/`.
Adapters translate validated `catalog.Theme` values into consumer files and conservative configuration edits.
Keep path discovery, rendering, ownership classification, and current-theme reads inside the owning adapter.
Treat `theme.Metadata.SelectableID()` as the generated filename and configured identity while CLI catalog IDs remain short.
The parity goldens for the app are the behavior contract, byte for byte; they were first recorded from the 1.x Python adapter of the same name.

## Adapter contract

- Each app is one package, `internal/adapters/<app>/`, that exports `var App = adapters.App{...}` (see `adapter.go`).
- `App.Inspect(themes, configDir)` returns one `plan.Inspection` and never refuses.
- `App.Plan(theme, themes, configDir, slot)` returns `plan.Write` and `plan.Remove` steps and never writes.
- `configDir` is the expanded `--config-dir`, or `""` when the flag is absent and the adapter resolves its default location.
- An `error` from either function is an I/O or decode failure the CLI prints as one `error:` line with exit 1; an ownership problem is never an error.
- `Label`, `Pair`, `Pointer`, `Reload`, and `StderrHint` carry the app's labels, pairing, reload hint, and stderr hint; change them only with the goldens that print them.
- Register the app with one line in its own file, `internal/cli/app_<name>.go`: `func init() { register("<name>", <name>.App) }`.
- Only `fsys.Commit` touches disk, and it owns dry-run and the symlink policy.
- The inspection holds the ownership class (absent, managed, legacy, foreign, unknown shape, broken) and the current sf2 theme when it resolves through `catalog.InstalledTheme`.
- Broken and unknown-shape inspections carry the reason or the manual fix in `Detail`; the CLI turns them into the one refusal message.
- The CLI's ownership gate decides whether to write the theme setting; `slot=false` means plan the theme files only and leave the setting untouched.
- The CLI resolves the selection once and passes a concrete theme; merges replace whatever owns the setting, because a foreign one only reaches them after `--adopt`.
- `current <app>` reads through the same inspection, so a selection `setup` reports as kept is always one `current` prints.

## Helpers

- Read a config with `plan.ReadExisting(path)`: it is `Path.read_text` with universal newlines, returns `""` for a missing file, and returns an error for a file that is not UTF-8.
- Read a file that must exist with `pycompat.ReadText`.
- Find a recorded managed id with `plan.RecordedID(text, pattern)`, which matches each stripped line at its start like Python's `pattern.match`; the pattern needs no leading `^`.
- Turn a recorded id into an inspection with `plan.Recorded(ownership, id, found, themes)`.
- Build paths with `pycompat.Path`, `Parent`, `Name`, `WithName`, and `ExpandUser`, never `filepath`, so printed paths match pathlib.
- Find the home directory with `pycompat.Home()`.
- Write Python's `\s`, `\S`, and `\w` in a regular expression as `pycompat.Space`, `pycompat.NotSpace`, and `pycompat.Word`.
- Split and strip with `pycompat.SplitLines`, `SplitLinesKeepEnds`, `Strip`, `LStrip`, and `RStrip`.
- Decode TOML with `catalog.DecodeTOML` and JSON with `pycompat.Loads`; encode JSON with `pycompat.Dumps` or `DumpsUnicode`, and quote with `pycompat.Repr`.
- Resolve a dark and light pair with `catalog.ThemePair` and a managed identity with `catalog.InstalledTheme`.

## WezTerm

- The adapter owns catalog scheme files under `<wezterm-config>/colors/` and the target's pointer; it writes `wezterm.lua` only as a new file, as a whole replacement of one of its own forms, or, with `--adopt`, by inserting its block into a file it verifies line by line.
- A target is the `wezterm.lua` the filesystem opens through any symlink (`fsys.Resolve`); inspection, backup, commit, pointer, and manifest key all use that one identity, and setting results are matched with `fsys.SamePath`, never as strings.
- `fsys.Resolve`, also run by `fsys.Preflight` on every planned path for every adapter, refuses a path with any `..` component, any name it or a symlink it follows reaches that is spelled unlike its directory entry (letter case, Unicode normalization), and `..` after a missing directory in a link target; sf2 never reconciles spellings, and a followed link writes to the file `Resolve` names.
- The pointer is `sf2-theme/wezterm-current-<12 hex of the SHA-256 of the identity>.lua` beside the file, so `WEZTERM_CONFIG_DIR` never moves it and no pointer is a plain `wezterm-current.lua` that earlier releases shared; a path with a control character is refused before any Lua could carry it.
- `parse.go` verifies a whole Lua 5.4 chunk (syntax, misplaced `break` and `...`, and at most 200 locals declared at the top level of the chunk), refusing `goto`, labels, and `<const>`/`<close>` as unverifiable; a file that does not parse is broken and nothing is written.
- Other compile-time limits are not checked: locals active inside nested blocks or function bodies, upvalues, nesting depth, and constants. A result that sf2's block would push past the top-level limit fails re-verification and takes the manual path.
- `lua.go` recognizes only sf2's own forms, as exact generated lines under one indent that are whole top-level statements: the full block, the unguarded block (`<builder>.color_scheme = dofile(sf2_current)`), and the first releases' `<builder>.color_scheme = "<sf2 scheme>"` line.
- A form counts only after a top-level `local wezterm = require("wezterm")` and `local <builder>`, in a chunk that ends `return <builder>`, with no other `color_scheme` and no other `sf2_current`, `sf2_scheme`, or `sf2_env` in code; a hand-adapted snippet, a single-quoted literal, or a form inside a function, comment, or string is not sf2's.
- The full form is managed and the others are legacy; writing the setting replaces the form as a whole with the full block naming the target's pointer, which also moves an integration off a former pointer without writing that file.
- `App.Disowns` drops a recorded pointer other than the target's own after any complete run, judged from the path, so a run after an interrupted migration stops the old claim too.
- A file with no sf2 form is foreign; `--adopt` inserts the full block before the final return only when every line is blank, a line comment, the wezterm require, one `local <builder> = wezterm.config_builder()`, a single-line literal `<builder>.<setting> = ...` other than `color_scheme`, or `return <builder>`, and the result re-verifies as managed.
- sf2 inserts or replaces its block only where every name the block reads from its surroundings keeps its standard meaning. `blockNames` reads those names (today `wezterm`, `dofile`, `type`) and the block's own locals from the block's text, so the check cannot drift from the block. Before the insertion point, none of them, nor `require`, may be bound anywhere: as a local (the builder counts), a parameter, a loop variable, a `function <name>`, or an assignment. The one exception is the top-level `local wezterm = require("wezterm")`. The builder also must not be one of the block's locals. Anything else takes the manual path, which names the binding to rename.
- `wezterm.lua` is read as written (`readLua`, no newline translation): an adopted file is the original bytes plus the block, ended with the file's one line break, and a replaced form keeps its own breaks; mixed line breaks get the manual steps.
- Every other case is refused with nothing written and the exact manual steps (`analysis.manual`): where to paste the block, and to remove other `color_scheme` assignments and any older loader.
- A valid shape sf2 cannot edit is unknown and refused like any other, with or without `--adopt`: nothing is written, not even scheme files or the pointer.
- The pointer returns the dark or light sibling from `wezterm.gui.get_appearance()` so a selected character follows host light/dark instead of pinning one scheme.
- Selecting either sibling writes the same pair; the managed identity is the dark `sf2-<catalog-id>`.
- Only a managed or legacy form reports a selection; a foreign or unknown config reports none, even beside a readable pointer.
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
- Resolve the selected catalog id to its dark/light sibling pair and write `auto_switch`, `light_name`, `dark_name`, and per-mode `[theme.custom.dark]`/`[theme.custom.light]` overlays.
- Those overlays require a Herdr build with herdr#2324; stable releases before it silently drop these tables and stay pinned to one palette.
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
- An `XDG_CONFIG_HOME` that is set but empty resolves to the relative `herdr-app`, reproducing the reference; fixing it is a separate behavior change.

## Neovim

- The adapter owns catalog files `<nvim-config>/colors/sf2-*.lua` and the selected pointer `<nvim-config>/sf2-theme/current.lua`.
- The pointer selects the dark or light sibling from `TERM_THEME` or `'background'` so Neovim follows the same character pair as WezTerm and Herdr.
- Selecting either sibling writes the same pair; the managed identity is the dark `sf2-<catalog-id>`.
- Every run also owns `<nvim-config>/plugin/sf2-theme.lua` as the startup loader.
- Nothing in these files is ever foreign; a pointer whose id does not resolve is an unreadable selection, and a legacy short ID resolves and is rewritten with the `sf2-` identity.
- Leave every other `colors/` or `plugin/` entry untouched, including `street-fighter-ii-*.lua` files earlier releases wrote.

## Codex

- The adapter owns catalog files `<codex-home>/themes/sf2-*.tmTheme` and only the `theme` assignment inside `[tui]` in `<codex-home>/config.toml`.
- Preserve every other Codex table, key, comment, and existing `[tui]` setting when selecting a theme.
- An inline `tui` form is an unknown shape whose manual fix is converting it to a `[tui]` table; never restructure it.
- A `[tui].theme` equal to a short catalog ID is legacy and is rewritten with the `sf2-` identity; any non-`sf2-` value is foreign.
- Leave unprefixed `<catalog-id>.tmTheme` files earlier releases wrote in place.
- A `[tui]` header followed by a comment is not recognized, so the config inspects as absent and a second `[tui]` table is appended; this reproduces the reference and is fixed only as a separate behavior change.

## Lazygit

- The adapter owns `<lazygit-config>/themes/sf2-<catalog-id>.yml` for every catalog entry and the selected theme sections in `<lazygit-config>/config.yml`.
- Theme fragments must cover every current Lazygit `gui.theme` key plus wildcard `authorColors['*']`, with SF2 semantic roles rather than Catppuccin hues.
- Preserve unrelated `gui` settings and named author colors, treat an unmarked existing `gui.theme` as foreign, and keep the managed blocks idempotent.
- Incomplete or out-of-order markers inspect as broken.
- Resolve the config directory from `--config-dir`, `LAZYGIT_CONFIG_DIR`, or the platform's Lazygit default: `~/Library/Application Support/lazygit` when `runtime.GOOS` is darwin, otherwise `$XDG_CONFIG_HOME/lazygit` or `~/.config/lazygit`, so its parity golden is recorded per OS.

## Claude Code

- The adapter owns catalog files `<claude-home>/themes/sf2-*.json` and only the `theme` key in `<claude-home>/settings.json`, resolved from `--config-dir`, `CLAUDE_CONFIG_DIR`, or `~/.claude`.
- Claude Code's custom-theme schema (`name`/`base`/`overrides`) is an experimental component with no publicly documented full token list.
- `overrideTokens` is a best-effort mapping onto SF2 roles, confirmed live in a running session for the `claude`/`error`/`success` tokens the docs do specify.
- Claude Code has no host-appearance auto-switch for custom themes, so selecting a light or dark sibling pins that one theme in `settings.json` like Codex, not a following pair like WezTerm, Herdr, or Neovim.
- Preserve every other `settings.json` key; only ever touch the top-level `theme` string, written as `custom:<selectable-id>`.
- A `custom:sf2-*` theme is managed, and unreadable when it does not resolve; any other `theme` is foreign.

## Slack

- Slack has no config file and stays outside the inspect, plan, and commit path; it is not an `adapters.App` and is not registered.
- `slack.RenderTheme` returns the four-color import line for the requested catalog theme, and the CLI prints the instructions.
- Do not install an `sf2-` identity, resolve a dark/light pair, or call `catalog.ProjectAdapterColors`.
- `current slack` fails closed, prints no theme id, and writes nothing.
- `--config-dir`, `--follow-symlinks`, `--adopt`, and `--dry-run` do not apply.

## Shared write contract

- The CLI, not an adapter, records each committed write in the `internal/store` manifest; the file at the app's `Target` is recorded as the edited setting and every other write as a wholly owned file.
- The CLI refuses unsafe destinations with `fsys.Preflight` before it locks state, keeps the setting's original bytes before the commit, merges the commit's results into the earlier record, and reads the selection back through `Inspect`.
- The CLI replaces a file other than the target only when the manifest records its SHA-256 or it holds exactly what the adapter's `Plan` writes there for some catalog theme; an adapter never claims a file by its name.
- `fsys.Apply` undoes its own writes when one fails, and the CLI rolls the `Transaction` back, then restores the manifest and removes the kept copy, when the record cannot be saved.

- Return every managed text write as a plan step so every adapter shares one mutation contract; sf2 plans no deletions, because no record proves it wrote a file under an older name.
- `fsys.Commit` resolves every destination before the first write; it refuses destination symlinks by default, follows only when explicitly requested, writes the resolved target, and retains the symlink itself.
- Dry-run reports `would_create` or `would_update` with a diff and creates no file, directory, or backup.
- A changed existing file receives a UTC-stamped backup (`<name>.bak.<YYYYMMDDTHHMMSSZ>`, then `.1`, `.2` within one second) before atomic replacement, with its mode preserved, while unchanged content creates no backup.
- `fsys.Preflight` also refuses a write that would fail part way: a non-regular file in its place, an immutable or append-only file or directory, or a directory the user cannot create files in.
- Never remove unknown files, stale-looking aliases, unmarked blocks, or user-owned configuration.

## Focused verification

- Keep each adapter's tests beside the package, with expected values from committed testdata or goldens, never from the Go code under test.
- Run `go test -count=1 ./internal/adapters/<app>/ ./internal/cli/` for an adapter change.
- Run `go test -count=1 ./internal/parity/` for any change that reaches output or written files, and re-record with `mise run parity:record` only for an intended behavior change.
- Run `go test -count=1 ./internal/plan/ ./internal/fsys/ ./internal/cli/` plus every touched adapter for shared write or gate changes.
- Finish with `gofmt`, `go vet`, and `golangci-lint run` on the changed packages.
