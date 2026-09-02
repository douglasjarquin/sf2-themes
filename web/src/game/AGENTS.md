# Maintain the browser game subsystem.

## Use the local sources of truth.

- This file governs `web/src/game/` and its browser-facing integration through `ArcadeGame.astro` and the `/game/` route.
- Read `docs/game-architecture.md` before changing a cross-boundary contract, then inspect the owning source and focused test.
- Treat `types.ts`, `config.ts`, `fighter-registry.ts`, and `content/fighters/index.ts` as the local contracts for state, timing, validation, and roster order.

## Preserve dependency direction.

- Keep `types.ts`, `config.ts`, `fighter-registry.ts`, and `core/` independent of the DOM, Astro, rendering, and input devices.
- Let the core import fighter validation at its input boundary, while authored roster selection remains outside `core/`.
- Let fighter content use `render/glyph-sprite.ts` only as its authoring schema, and keep other renderer behavior out of content modules.
- Let renderers consume immutable snapshots, authored content, stage data, and palette roles without mutating simulation state.
- Let `BrowserGameHost`, `bootstrap.ts`, and keyboard input compose the pure core with browser APIs.
- Expose composition APIs to Astro components and pages without importing Astro into game modules.

## Keep the simulation deterministic.

- Advance gameplay at 60 fixed ticks per second from normalized input frames, integer world coordinates, and explicit phase transitions.
- Route every random decision through the core-owned seeded RNG so an equal seed and equal input sequence produce equal snapshots.
- Keep wall-clock reads, DOM APIs, timers, animation frames, and `Math.random` outside `core/`.
- Convert animation-frame elapsed time through `core/clock.ts`, preserve the eight-tick catch-up cap, and render the resulting snapshot.

## Enforce the closed fighter roster.

- `FIGHTER_IDS` defines the canonical ordered set of exactly 17 playable IDs, and the `FighterId` union plus `FIGHTER_ROSTER` must match it exactly.
- Accept fighter definitions only through `validateFighterDefinition`, and accept a complete registry only when every canonical fighter appears once.
- Keep theme IDs, light-variant IDs, duplicate fighters, and unknown fighter IDs outside the playable boundary.
- Preserve each fighter's required poses, authored moves, projectile, victory data, AI bias, directional glyph frames, four palette regions, matching theme metadata, and original-project-authored provenance.
- Coordinate any roster change across types, registry validation, content exports, renderer coverage, route selectors, and `docs/roster.md` in one change.

## Keep the renderer boundary narrow.

- `render/TextmodeRenderer.ts` is the sole source module allowed to import `textmode.js`.
- Preserve the fixed 96x40 logical grid, explicit non-looping redraws, top-left public coordinates, and complete in-grid scene composition.
- Keep physical canvas fitting and resizing separate from logical coordinates, and clamp device pixel ratio to two.
- Map TOML-backed tokens through `createGamePalette`, and change themes by replacing renderer roles without resetting the match.
- Destroy the textmode context and remove its owned canvas on every teardown or failed mount.

## Keep the browser host lifecycle explicit.

- `BrowserGameHost` mounts the renderer, owns credits and status, schedules fixed ticks, and coordinates resize, focus, visibility, and intersection state.
- In reduced-motion mode, `start()` must leave the phase at boot, the tick at zero, the poster visible, and the loop paused until explicit coin activation.
- Activate keyboard input only after player activation while the cabinet itself is focused, visible, and intersecting.
- Release every held or pulsed input when focus leaves, the page hides, the cabinet goes offscreen, input deactivates, or the host is destroyed.
- Preserve native keyboard behavior for controls outside the focused cabinet, and prevent defaults only for mapped game keys accepted by the active input source.
- Host teardown must cancel its animation frame, destroy input, disconnect both observers, remove every listener, and destroy the renderer exactly once.

## Preserve accessible observability.

- Keep the labelled `tabindex="0"` cabinet, polite atomic `[data-game-live-status]` region, credit and game status nodes, and palette `aria-pressed` state synchronized with the rendered snapshot.
- Treat renderer, loop, phase, tick, fighter, pose, move, and theme `data-*` attributes as browser-test observability contracts.
- On renderer failure, pause the loop, expose the failed status, restore the static poster, and announce the fallback without an uncaught page error.

## Run focused verification.

- Run `aube -C web run check` before browser tests so strict TypeScript and Astro diagnostics are clean.
- Run `aube -C web exec node --test test/game-core.test.mjs test/fighter-registry.test.mjs test/fighter-content.test.mjs test/fighter-roster.test.mjs test/renderer-contract.test.mjs test/palette-mapping.test.mjs test/textmode-import-guard.test.mjs test/browser-game-host.test.mjs` for core, roster, content, palette, renderer, import, and host contracts.
- Run `aube -C web exec playwright test tests/e2e/home.spec.mjs tests/e2e/game.spec.mjs` for real-browser focus, teardown, accessibility status, route, and renderer behavior.
