# SF2 Themes Game Architecture

The web game is a static Astro surface under `/sf2-themes/` with one reusable `ArcadeGame.astro` component shared by `/game/`.

## Dependency direction

The browser-independent core owns simulation state, deterministic ticks, seeded randomness, fighter definitions, collision, combat, and stage data.

The renderer consumes immutable core snapshots and palette roles without changing combat state.

The browser host owns DOM focus, keyboard input, visibility, intersection, resize, the fixed-step animation scheduler, credits, and lifecycle cleanup.

Astro pages provide static catalog data and compose the shared component without duplicating game logic.

## Fixed-step core

The simulation advances at 60 ticks per second and accepts normalized input frames rather than browser time or DOM events.

The seeded RNG is owned by the core so identical fighter options, seed, and tick sequence produce identical snapshots.

The browser scheduler converts elapsed frame time into fixed ticks and caps catch-up at eight ticks.

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

## Routes and commands

`/` remains the catalog home and is independent of the live cabinet.

`/game/` renders the full cabinet with independent theme and P1/P2 fighter controls, keyboard help, and fullscreen affordance.

`/preview/` renders the 36 canonical palettes through code, terminal, and swatch previews.

`/palette/` renders every palette variant with source swatches.

`/themes/` remains the adapter catalog.

Use `mise run web:check`, `mise run web:test`, and `mise run web:build` for the normal web gates.
