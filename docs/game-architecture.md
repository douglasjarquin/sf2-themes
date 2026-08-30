# SF2 Themes Game Architecture

The web game is a static Astro surface under `/sf2-themes/` with one reusable `ArcadeGame.astro` component shared by the homepage and `/game/`.

## Dependency direction

The browser-independent core owns simulation state, deterministic ticks, seeded randomness, fighter definitions, collision, combat, and stage data.

The renderer consumes immutable core snapshots and palette roles without changing combat state.

The browser host owns DOM focus, keyboard input, visibility, intersection, resize, the fixed-step animation scheduler, credits, and lifecycle cleanup.

Astro pages provide static catalog data and compose the shared component without duplicating game logic.

The capture bridge is a browser-only adapter that consumes the same core, renderer, and palette boundaries and exposes deterministic methods only when `/game/?capture=1` is active.

## Fixed-step core

The simulation advances at 60 ticks per second and accepts normalized input frames rather than browser time or DOM events.

The seeded RNG is owned by the core and is reset with the capture seed so identical fighter options, seed, and tick sequence produce identical snapshots.

The browser scheduler converts elapsed frame time into fixed ticks and caps catch-up at eight ticks.

Capture mode advances explicitly through `advanceTicks` and `advanceUntil` and never waits on wall-clock time.

## State machine

The core models boot, title, attract intro, attract fight, attract result, player intro, player fight, player result, and paused phases.

CPU attract matches use the same authored collision and combat rules as player matches.

INSERT COIN interrupts attract states and activates the focused player surface.

The core exposes snapshots containing tick, phase, timers, fighters, projectiles, result, and seeded RNG state.

## Renderer and palette boundary

`TextmodeRenderer` is the only web game module that imports `textmode.js`.

The renderer mounts a 96 by 40 logical grid, clamps device pixel ratio to two, redraws explicit snapshots, resizes through the host, and destroys its canvas and textmode context.

`createGamePalette` converts the TOML-backed UI, semantic, and ANSI catalog roles into the renderer roles used by the stage, fighters, effects, and HUD.

Theme changes replace the renderer palette without restarting the current match.

## Fighter and stage authoring

The validated roster contains the 17 IDs from `docs/roster.md` in canonical order.

Each fighter definition supplies authored multi-cell glyph frames, movement values, hurtboxes, pushbox, moves, projectile, victory quote, and CPU bias.

The dojo stage is an original layered composition with distant, middle, floor, foreground, and restrained environmental animation data.

The art is project-authored terminal glyph content and does not use raster sprites, ROM assets, or copied game files.

## Capture bridge and screenshot archive

Capture URLs accept `theme`, `seed`, `mode`, `stage`, `p1`, `p2`, `tick`, and `moment` and normalize unknown values to safe defaults.

The capture bridge waits for fonts and renderer readiness and exposes `ready`, `reset`, `setTheme`, `advanceTicks`, `advanceUntil`, `getSnapshot`, and `getCaptureState` on `window.__SF2_GAME__` only in capture mode.

The screenshot factory uses Playwright-managed Chromium without a Chrome channel, captures the canonical 36 palette variants at 1280 by 720, records deterministic recipe metadata, and verifies before atomic promotion.

The strict verifier checks catalog order, image set, source and recipe fingerprints, palette hash, dimensions, PNG structure, occupancy, image hashes, and all-17-fighter coverage.

## Routes and commands

`/` renders the compact playable cabinet and preserves the existing home copy and menu.

`/game/` renders the full cabinet with independent theme and P1/P2 fighter controls, keyboard help, and fullscreen affordance.

`/preview/` renders the 36 generated game captures from the strict manifest.

`/palette/` renders every palette variant with its matching generated game capture and source swatches.

`/themes/` remains the adapter catalog and is independent of the game screenshot archive.

Use `mise run web:check`, `mise run web:test`, and `mise run web:build` for the normal web gates.

Use `mise run web:screenshots` to regenerate the archive and `mise run web:screenshots:verify` to verify it without regeneration.

Screenshot-affecting web changes trigger strict verification, a one-theme deterministic smoke archive, full regeneration, and diagnostics artifact upload in CI.
