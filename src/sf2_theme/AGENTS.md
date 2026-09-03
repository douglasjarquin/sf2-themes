# PYTHON PACKAGE BOUNDARY

## OVERVIEW

`themes/` is the canonical data source; this package turns its TOML into typed, validated `Theme` values and adapter output.
The runtime path is `tomllib -> parse_theme() -> Theme -> load_catalog() -> CLI/adapters`.
`parse_catalog()` is the intentionally unvalidated discovery path used by validation and reporting; `load_catalog()` adds catalog uniqueness and all error-severity theme checks.
The committed root executable mirrors this package by embedding the standalone generator's explicit module roster and theme payload.

## WHERE TO LOOK

| Concern | Owner | Boundary |
|---|---|---|
| Theme values and names | `model.py` | Immutable palette types, exact hex parsing, canonical IDs, and adapter-facing IDs. |
| TOML shape | `parse.py` | Schema, required fields, enum values, unknown keys, and typed construction. |
| Semantic validity | `validation.py` | Contrast, ANSI distinctions, lookup-key uniqueness, and error versus warning severity. |
| Catalog lifecycle | `catalog.py` | Embedded or filesystem discovery, deterministic order, validated loading, lookup, pairing, and defaults. |
| Public dispatch | `cli.py`, `__main__.py` | Argument handling, command behavior, error presentation, and adapter orchestration. |
| Application formats | `adapters/` | Rendering, recognized configuration merges, managed markers, and app-specific paths. |
| Mutation primitive | `filesystem.py` | No-op detection, dry-run diffs, backups, mode preservation, symlink policy, and atomic replacement. |
| Standalone mirror | `mise-tasks/build-standalone` | Runtime module allowlist and embedded theme files for the generated `sf2-themes`. |

## CONVENTIONS

- Keep structural rejection in `parse.py`, semantic checks in `validation.py`, and their composition in `catalog.py`.
- Route normal runtime consumers through `load_catalog()`; reserve `parse_catalog()` for callers that must inspect or report invalid catalog data.
- Pass an already loaded catalog into repeated lookups and adapters so one operation uses one deterministic snapshot.
- Resolve a selected catalog id to its dark and light siblings through `theme_pair()` instead of reconstructing `-light` names in adapters.
- Resolve a managed `sf2-` pointer identity through `installed_theme()` so setup can refresh an existing pair without `--theme`.
- Keep catalog and CLI selection IDs short, such as `ryu-light`.
- Derive installed names through `ThemeMetadata.selectable_id` or `selectable_id()` so adapter-facing identities are consistently `sf2-<catalog-id>`.
- Preserve the intentional spelling split: public distribution and command `sf2-themes`, Python import package `sf2_theme`, and managed state directory `sf2-theme`.
- In source mode, catalog discovery honors `SF2_THEME_DIR` before the repository catalog; the standalone uses `_embedded.THEME_FILES` instead of filesystem discovery.
- Let adapters own syntax-aware reads and merges, then send final content through `write_file()`.
- Keep runtime imports standard-library or package-local because the copied standalone runs without an installed project environment.

## ANTI-PATTERNS

- Do not use `parse_catalog()` as a shortcut for runtime behavior that promises a valid catalog.
- Do not duplicate TOML parsing, contrast rules, catalog ordering, or lookup semantics inside adapters or commands.
- Do not hand-build `sf2-` names from aliases, display names, or storage paths.
- Do not write live adapter configuration with `Path.write_text()`; preserve merge semantics and the `write_file()` safety contract.
- Do not move application parsing into `filesystem.py`; it is format-agnostic and receives final content.
- Do not add a runtime module without adding it to the standalone generator's `MODULES` roster.
- Do not create a source `_embedded.py`; the standalone installs that module virtually at runtime.

## VERIFICATION

- Pair catalog, model, parser, or validator changes with `tests/test_catalog.py` and `tests/test_validation.py`.
- Pair safe-write changes with `tests/test_filesystem.py` and every adapter suite whose mutation path changed.
- Pair CLI dispatch changes with `tests/test_cli.py` and the affected adapter suite.
- Pair rendered output changes with `tests/test_snapshots.py` and the owning adapter tests.
- Any package or catalog change triggers standalone regeneration before verification; a new import also requires a generator roster update.
- Treat `mise-tasks/test-cli` as the installed-consumer contract because it copies the generated executable away from the repository and exercises all adapters.
- Prove standalone freshness by regenerating from inputs and confirming the committed root executable has no remaining diff.
- Finish with the root Python verification gate rather than duplicating its command sequence here.
