# PROJECT KNOWLEDGE BASE

**Generated:** 2026-08-27
**Commit:** e63002d
**Branch:** codex/feat-sf2-ascii-arcade

## OVERVIEW

SF2 Themes is a Python CLI and static Astro site for one TOML theme catalog shared by terminal adapters, documentation, and a deterministic browser arcade game.
The committed standalone CLI is a generated product with a freshness check in CI.

## MAINTAINING THIS FILE

This is the committed home for project-intrinsic build, test, release, architecture, and sharp-edge knowledge.
Keep only guidance useful to future sessions across the repository.
Point to authoritative files and commands instead of repeating implementation details.
Rewrite or prune stale notes instead of appending history.
`CLAUDE.md` imports this file and must remain a redirect, not a second source of guidance.

## STRUCTURE

```text
src/sf2_theme/       Python model, catalog, CLI, safe writes, and adapters
mise-tasks/          Standalone/preview/theme-data generation and the copied-CLI shell test, as mise file tasks
themes/              Authoritative 36-entry TOML catalog
scripts/             scripts/sf2 (a plain `sf2-themes` alias for checkout use) and scripts/ci/ (container orchestration)
docker/              toolchain (mise + Python + Node + aube, no app source) and dev (+ prefetched deps) images
tests/               Pytest contracts and snapshots
web/                 Static Astro site, Node tests, and Playwright
web/src/game/        Fixed-step game core plus browser, renderer, and input adapters
docs/                Authored contracts and generated SVG previews
.github/workflows/   Path-selected CI, Pages build/deploy, image publishing, and issue automation
```

## WHERE TO LOOK

| Task | Location | Notes |
|---|---|---|
| CLI dispatch | `src/sf2_theme/cli.py` | Public command is `sf2-themes`; `install` is deprecated |
| Theme model and lookup | `src/sf2_theme/model.py`, `catalog.py`, `validation.py` | `themes/` remains the source of truth |
| Adapter mutation | `src/sf2_theme/adapters/`, `filesystem.py` | Preserve unrelated user configuration and symlink policy |
| Standalone generation | `mise-tasks/build-standalone` | Owns the committed root `sf2-themes` executable |
| Theme design contract | `docs/theme-guidelines.md`, `docs/roster.md` | Covers IDs, dark/light pairs, semantics, and validation |
| Astro shell and routes | `web/src/layouts/`, `web/src/pages/`, `web/src/lib/site-path.mjs` | Static Pages base is `/sf2-themes` |
| Browser game | `web/src/game/`, `docs/game-architecture.md` | Shared by `/game/` |
| Python verification | `tests/`, `mise.toml` | Pytest plus the copied standalone CLI harness |
| Web verification | `web/test/`, `web/tests/e2e/`, `web/playwright.config.mjs` | Node contracts plus real browser coverage |
| Starship / zsh prompt | `adapters/starship.py`, `adapters/zsh_syntax.py` | Managed palette + sourcable command highlight snippet |
| Containerized dev/CI | `docker/`, `scripts/ci/run-in-dev-container.sh` | `mise run <task>` runs identically on bare host or through the wrapper |

## CODE MAP

| Symbol | Type | Location | Graph refs | Role |
|---|---|---|---:|---|
| `Theme` | data model | `src/sf2_theme/model.py` | 10 callers | Shared immutable palette contract |
| `load_catalog` | function | `src/sf2_theme/catalog.py` | 5 callers | Validated catalog entry point |
| `main` | function | `src/sf2_theme/cli.py` | 1 caller | Installed and module CLI entry |
| `write_file` | function | `src/sf2_theme/filesystem.py` | adapter boundary | Atomic writes, backups, dry-run, and symlinks |
| `createGameCore` | function | `web/src/game/core/state-machine.ts` | 7 callers | Deterministic simulation API |
| `validateFighterDefinition` | function | `web/src/game/fighter-registry.ts` | 6 callers | Closed 17-fighter runtime contract |
| `BrowserGameHost` | class | `web/src/game/BrowserGameHost.ts` | 2 callers | DOM lifecycle, input, scheduling, and status |
| `TextmodeRenderer` | class | `web/src/game/render/TextmodeRenderer.ts` | shared boundary | Sole `textmode.js` integration |

## CONVENTIONS

- Tool versions and stable developer commands live in `mise.toml`: Python 3.11, Node 24, uv 0.11, and Aube 2.1.
- Catalog IDs stay short; adapter-installed identities use `sf2-<catalog-id>`.
- Catalog order is `main.toml`, optional `main-light.toml`, then sorted character TOMLs.
- The Astro app uses static output and the `/sf2-themes` Pages base; internal URLs go through `sitePath()`.
- The game uses integer fixed ticks, seeded RNG, immutable snapshots, and explicit browser adapters.
- CI path filtering fails open when filter data is missing or invalid.
- Pull requests build the Pages artifact but deployment occurs only on non-PR events.

## ANTI-PATTERNS

- Do not hand-edit the generated root `sf2-themes` executable.
- Do not hand-edit `docs/previews/`; run `mise run generate-previews`.
- Do not duplicate catalog parsing or semantic color rules inside adapters or pages.
- Do not modify unknown WezTerm Lua, unmarked Herdr theme blocks, or unrelated adapter configuration.
- Do not follow configuration symlinks unless the caller explicitly selects `--follow-symlinks`.
- Do not introduce DOM, browser globals, timers, animation frames, wall-clock reads, or `Math.random` into `web/src/game/core/`.
- Do not import `textmode.js` outside `TextmodeRenderer.ts`.

## COMMANDS

```bash
mise run test
mise run lint
mise run apply -- wezterm --theme vega
./sf2-themes --version
mise run build-standalone
mise run web:install
mise run web:check
mise run web:install:npm
mise run web:build:npm
mise run web:test:npm
scripts/ci/run-in-dev-container.sh mise run test
```

## NOTES

- Python or catalog changes require standalone regeneration before `mise run test`.
- Theme changes also require `mise run generate-previews`.
- The Neovim adapter owns managed `colors/`, `sf2-theme/current.lua`, and `plugin/sf2-theme.lua` output.
- Generated `.omo/`, caches, and build output do not count as source complexity when placing child guidance.
- Use `web:build:npm`/`web:check`/`web:test:npm` (not the plain aube-based `web:build`/`web:test`) for anything that runs a real Astro build — see `web/AGENTS.md` for why.
