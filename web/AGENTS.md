# STATIC ASTRO WEB APPLICATION

The repository-level guidance in `../AGENTS.md` still applies, and this file records only web-specific seams that are easy to miss.

## DEPLOYMENT AND ROUTING

- Keep `astro.config.mjs` on static output with the `/sf2-themes` GitHub Pages base and the configured production site origin.
- Build every internal page and asset URL with `sitePath()` so local preview, Playwright, and GitHub Pages retain the same trailing-slash base contract.
- Use relative route URLs against Playwright's configured base URL, and assert `/sf2-themes/.../` when the prefix itself is part of the behavior under test.
- Treat `web/dist/` as the disposable static package consumed by preview and the Pages artifact upload.

## SOURCE OWNERSHIP

- `src/layouts/SiteLayout.astro` owns the document shell, metadata, global theme properties, `SiteHeader`, `SiteFooter`, and the single import of `global.css`.
- Keep shared navigation and footer behavior in their components instead of reproducing site chrome in route files.
- Keep shared tokens, focus treatment, shell layout, and cross-route responsive rules in `src/styles/global.css`, while route-only styling stays co-located with its route or component.
- Route files under `src/pages/` own page composition and progressive route-local interactions, and every normal route renders through `SiteLayout`.
- `src/components/ArcadeGame.astro` owns reusable cabinet markup, the serialized palette payload, bootstrap, controls, and the static poster used by the full-game route.
- `/game/` configures the full palette and fighter selectors without forking the cabinet implementation.
- Browser scheduling, input, observation, rendering, and teardown belong under `src/game/`, while route scripts should remain thin adapters around those APIs.
- Treat `data-*` hooks on the shell and arcade as a tested browser contract, and update their Node and Playwright consumers with any intentional change.
- `src/data/theme-data.mjs` is the web adapter over the root TOML catalog, so pages and components consume its exports instead of duplicating theme parsing or palette order.
- Run source-importing web commands with `web/` as the working directory because the default theme-data paths resolve the root catalog relative to that directory.

## ACCESSIBILITY AND FALLBACKS

- Keep every route useful as static HTML before client scripts run, with native links, buttons, labels, selects, headings, landmarks, and readable content.
- Preserve the existing `aria-current`, `aria-pressed`, accessible names, live regions, status roles, keyboard order, and visible `:focus-visible` treatment when changing interactions.
- Keep the arcade poster and text status as a bounded visible fallback when palette parsing, WebGL, or renderer startup fails.
- Respect reduced motion by removing transitions and leaving the arcade paused on its static poster until explicit activation.
- Feature-detect clipboard and fullscreen APIs, hide unsupported controls where appropriate, and announce success only after the browser operation succeeds.
- Prefer logical CSS properties and verify both desktop and mobile behavior when changing shared layout or controls.

## AUBE AND MISE BOUNDARIES

- From the repository root, use the `mise run web:*` tasks so Node 24 and Aube 2.2.4 come from the pinned toolchain.
- From `web/`, use `aube run <script>` or use `aube -C web ...` from the root for focused package scripts.
- Treat `aubr` as package-script composition used inside `package.json`, not as the repository-level entry point.
- Install from `aube-lock.yaml` with `mise run web:install` locally or `aube -C web ci` in CI, matching the lockfile-backed path both `verify.yml` and `deploy.yml` run.
- Keep `web/.npmrc` on `node-linker=hoisted` so Astro prerender can resolve native bindings and client entry files from an npm-compatible tree.
- Keep `astro check`, unit tests, browser tests, and the production build as separate evidence because none substitutes for another.
- The normal web CI path installs with aube, installs Chromium, builds, then runs check, Node tests, and Playwright tests, while the Pages workflow uploads `web/dist/` and deploys only outside pull requests.

## TEST SURFACES AND OWNED SERVERS

- `tests/unit/*.test.mjs` and `test/*.test.mjs` are Node tests for config, data, deterministic game logic, renderer contracts, and import boundaries without a browser page.
- `tests/e2e/*.spec.mjs` are Playwright tests against a production build served under `/sf2-themes/` in desktop Chromium.
- Use `aube -C web run test:unit` or `test:e2e` for a focused layer, while `mise run web:test` runs both layers in sequence.
- Playwright builds and starts its own Astro preview on `PLAYWRIGHT_PORT`, which defaults to `4321`, and local runs may reuse a server already listening there.
- For concurrent or manual E2E runs, select an unused `PLAYWRIGHT_PORT` and confirm that any reused server belongs to this worktree.
- Let Playwright close the preview it starts, and when starting a preview manually record its PID and port so cleanup stops only that owned process.

## GENERATED OUTPUTS

- Treat `node_modules/`, `.astro/`, `dist/`, `test-results/`, and `playwright-report/` as generated dependency, cache, build, or diagnostic output rather than source.
- Preserve traces and screenshots long enough to diagnose browser failures, then remove only artifacts created by the current run.
