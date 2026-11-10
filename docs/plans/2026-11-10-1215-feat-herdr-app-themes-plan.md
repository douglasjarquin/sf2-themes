---
title: "feat: Add Herdr app themes"
type: feat
date: 2026-11-10
artifact_contract: ce-unified-plan/v1
product_contract_source: ce-plan-bootstrap
execution: code
---

# Add Herdr app themes

## Goal Capsule

**Objective:** SF2 users can install a fighter's light and dark palettes in the Herdr desktop app with one CLI command.
**Means:** A separate `herdr-app` adapter and paired settings selection (KTD1).
**Authority:** The user's feature request, repository adapter contracts, and the app's documented JSON schema govern this work.
**Execution profile:** Two ordered units in one PR; lfg owns review, publication, and CI follow-up.
**Stop conditions:** Preserve unrelated working changes and stop if the app schema contradicts the requested paired themes.

## Product Contract

### Summary

Add `herdr-app` to SF2's supported applications, with setup, apply, and current commands for independent desktop light and dark themes.

### Problem Frame

The existing `herdr` target writes a TUI TOML configuration and cannot configure the desktop app's JSON themes.

### Requirements

- **R1:** Expose `herdr-app` separately while retaining the existing `herdr` TUI behavior.
- **R2:** Install every catalog theme as `themes/sf2-<catalog-id>.json` under `--config-dir`, or `~/.config/herdr-app` by default, honoring the normal XDG config root convention.
- **R3:** Render background, foreground, selection, cursor, all 16 ANSI slots in normal-then-bright order, and mode-specific canvas, chrome, surface, border, and accent overrides from canonical SF2 roles.
- **R4:** Applying either sibling selects its catalog-resolved dark/light pair through `settings.json`'s `theme.dark` and `theme.light` keys; preserve all other settings, including appearance, layout, colors, and unrelated theme keys.
- **R5:** Setup without `--theme` refreshes assets while preserving any existing selection, including foreign or independently mixed light/dark choices, byte-for-byte; explicit setup selection behaves like apply.
- **R6:** `current herdr-app` reports the canonical dark installed ID for a valid matching SF2 pair and fails clearly for absent, foreign, malformed, or mixed-family selections.
- **R7:** Use the shared safe-write contract for dry-run, backups, idempotence, and symlinks; malformed settings must produce an actionable error rather than an exception traceback.
- **R8:** Document app paths, commands, live application, appearance preservation, and layout/theme-picker precedence; the copied standalone CLI supports the new target.

### Acceptance Examples

- **AE1:** Applying `ryu-light` writes `sf2-ryu.json` and `sf2-ryu-light.json`, selects them by mode, and reports `sf2-ryu` through current.
- **AE2:** Setup without an explicit theme leaves a user's independently chosen light/dark selection unchanged while refreshing catalog assets.
- **AE3:** Dry-run against existing settings prints changes without modifying settings, theme files, or backups.

### Scope Boundaries

Live user configuration is not changed as part of implementation; verification uses isolated directories.
The TUI adapter is not renamed or migrated.
Web UI changes, new app environment variables, layout rewriting, theme-picker automation, and appearance forcing are outside this change.

## Planning Contract

### Key Technical Decisions

- **KTD1:** Implement `src/sf2_theme/adapters/herdr_app.py` as a separate JSON adapter, following the existing Claude adapter's catalog-install pattern and the shared pair resolver.
- **KTD2:** Map canvas to UI background, chrome to UI surface, surface to UI overlay, border to UI border, and accent to UI accent; map terminal fields directly to canonical UI/ANSI roles.
- **KTD3:** Parse and validate the settings object before writing assets; merge only selected theme keys, preserve existing selection during setup, and route mutations through `write_file()`.
- **KTD4:** Treat current as the configured SF2 pair rather than the effective on-screen theme, because layout and picker overrides can take precedence.

### High-Level Technical Design

The following behavior table is directional guidance for dispatch and selection preservation.

| Command | Existing selection | Result |
|---|---|---|
| Apply, or setup with a theme | Any valid settings object | Select the requested catalog pair. |
| Setup without a theme | Any existing theme selection | Refresh assets; retain settings bytes. |
| Setup without a theme | No selection | Select the default main pair. |
| Current | Matching managed pair | Return the dark installed ID. |
| Current | Missing, foreign, malformed, or mixed pair | Fail clearly without outputting an ID. |

### Sources

The authoritative app schema was inspected at `/Users/douglasjarquin/.config/herdr-app/AGENTS.md`, in the themes and look section beginning around line 298.
The user supplied the same schema and precedence rules in the request.
Implementation patterns live in `src/sf2_theme/adapters/claude.py`, `src/sf2_theme/catalog.py`, and `src/sf2_theme/filesystem.py`.
The standalone module roster lives in `mise-tasks/build-standalone`.

## Implementation Units

### U1. Add the desktop adapter and CLI contract

**Goal:** Users can apply and inspect paired Herdr app themes with the installed or source CLI.
**Requirements:** R1-R7, AE1-AE3; KTD1-KTD4.
**Files:** `src/sf2_theme/adapters/herdr_app.py`, `src/sf2_theme/cli.py`, `mise-tasks/build-standalone`, `tests/test_herdr_app.py`, `tests/test_cli.py`, `tests/test_snapshots.py`, and an owned JSON snapshot if useful.
**Approach:** Install the canonical catalog, preserve unrelated JSON fields, add paired selection and current dispatch, and regenerate the standalone executable through its generator.
**Test scenarios:** Check both variants and ANSI ordering, pair equivalence, catalog completeness, setup preservation, explicit selection, malformed root/theme JSON, missing/foreign/mixed current selection, dry-run, repeat no-op, backup contents, settings/theme-file symlink behavior, and default/config-dir path resolution.
The adapter and CLI tests protect generated files and configuration outcomes; they do not prove the running desktop app's rendering.
**Verification:** Focused adapter/CLI/snapshot suites pass, the legacy Herdr tests stay green, and the regenerated executable runs outside the checkout.

### U2. Publish the consumer contract and copied-CLI proof

**Goal:** Documentation and the distribution harness expose the desktop target distinctly from the TUI.
**Dependencies:** U1.
**Requirements:** R8.
**Files:** `README.md`, `docs/sf2-themes/commands.md`, `docs/sf2-themes/setup-and-apply.md`, `src/sf2_theme/adapters/AGENTS.md`, `mise-tasks/test-cli`.
**Approach:** Add desktop commands and paths, distinguish Herdr app from TUI, explain mode and override precedence, and exercise the copied standalone against an isolated app directory.
**Test scenarios:** The copied executable applies a light sibling, validates JSON payload/pair and current output, and repeats setup without selecting a different pair.
The copied harness proves packaging and the user-facing command path; native visual acceptance remains separate.
**Verification:** Repository Python and copied-CLI gates plus lint pass; no hand-edited generated artifacts remain.

## Verification Contract

Regenerate the standalone with `mise run build-standalone` before Python verification.
Run focused Herdr app, CLI, snapshot, and existing Herdr tests, then the canonical `mise run test` and `mise run lint` gates.
Run `mise run test-cli` for the copied standalone consumer contract if it is not already included in the aggregate gate.
Finish with the repository-required pre-PR `mise run verify` gate.
Any native desktop check must use an isolated app configuration and be reported separately from JSON/CLI proof; lack of a safe isolated native launch is a disclosed verification limitation.
No web behavior changes are planned, so browser testing is not a feature acceptance requirement.

## Definition of Done

Both units are complete with all R-IDs protected by the stated checks.
The source and copied standalone produce valid desktop files and preserve unrelated settings.
Documentation clearly distinguishes the two Herdr targets.
The full pre-PR `mise run verify` gate passes, abandoned implementation attempts are removed, and lfg receives the verified implementation for review and PR delivery.
