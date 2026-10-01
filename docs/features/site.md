# Static theme site

The Astro site under `web/` builds to `web/dist` and deploys to `https://douglasjarquin.github.io/sf2-themes/` via the Pages workflow.
It is a single-page catalog: hero, live fighter preview with a `SampleBlock` seven-pane sample, a `⌘K` theme palette, three-round install steps for seven apps, and a ports section, all re-themed at runtime by `web/src/scripts/site-theme.mjs` from the payload embedded by `web/src/layouts/SiteLayout.astro`.
The retired routes (`/themes/`, `/palette/`, `/preview/`, `/install/`, `/themes/<id>/`) are noindex meta-refresh stubs pointing at home anchors; the arcade cabinet and `/game/` were removed.

## Entry points

- `https://douglasjarquin.github.io/sf2-themes/` in production, `aube -C web run preview` or `mise run web:dev:local` locally.
- Source lives under `web/src/`; browser contracts are exercised by `web/tests/e2e/` and Node contract tests under `web/test/` and `web/tests/unit/`.
- SEO artifacts under `web/public/` are generated from `web/src/lib/seo.mjs`.

## Scenarios

| ID | Scenario | Driver | Evidence |
| --- | --- | --- | --- |
| `site.install` | Web dependencies resolve from `web/package.json` through `web/aube-lock.yaml` | automated: `mise run web:install` inside `mise run verify` against `web/aube-lock.yaml` | aggregate run record |
| `site.check` | `astro check` is clean over `web/src/` | automated: `mise run web:check` inside `mise run verify` over `web/src/` | aggregate run record |
| `site.build` | `astro build` emits the home page, 404, and all stub routes under the `/sf2-themes/` base in `web/dist` | automated: `mise run web:build` inside `mise run verify`; route emission is asserted by `web/tests/e2e/static-site.spec.mjs` | aggregate run record |
| `site.theme` | Header select, dark/light buttons, `t` key, and left/right arrows switch the palette and persist to localStorage | automated: `web/tests/e2e/shell.spec.mjs` inside `mise run web:test` | playwright report |
| `site.palette` | `⌘K`/`Ctrl+K` opens the palette; filtering (including boss aliases) and Enter apply a family/mode, Escape restores focus | automated: `web/tests/e2e/shell.spec.mjs` inside `mise run web:test` | playwright report |
| `site.deeplink` | `?theme=`/`?mode=` deep links override stored state at boot, before paint | automated: `web/tests/e2e/shell.spec.mjs` inside `mise run web:test` | playwright report |
| `site.preview` | SampleBlock tabs switch seven panes and `data-t` bindings rewrite fighter name, file id, and palette values on theme change | automated: `web/tests/e2e/home.spec.mjs` inside `mise run web:test` | playwright report |
| `site.install-steps` | Port selection rewrites the two install commands (persistent install, then one `sf2 apply`); copy buttons write the shown command and hide when the clipboard API is missing | automated: `web/tests/e2e/home.spec.mjs` inside `mise run web:test` | playwright report |
| `site.nojs` | The page remains useful as static HTML with scripts disabled | automated: `web/tests/e2e/home.spec.mjs` inside `mise run web:test` | playwright report |
| `site.seo` | Canonical, description, JSON-LD, `web/public/robots.txt`, `web/public/sitemap.xml`, `web/public/llms.txt`, and `web/dist/index.html` canonicalization are correct | automated: `web/tests/unit/seo.test.mjs` plus `web/tests/e2e/static-site.spec.mjs` inside `mise run web:test` | aggregate run record |
| `site.stubs` | Retired routes serve noindex meta-refresh stubs targeting the right home anchors, and `/game/` is absent | automated: `web/tests/e2e/static-site.spec.mjs` inside `mise run web:test` | playwright report |
| `site.404` | The 404 page is branded, noindexed, and links known routes | automated: `web/tests/e2e/static-site.spec.mjs` inside `mise run web:test` | playwright report |
| `site.responsive` | The 760px column reflows with no page-level horizontal overflow at 375px | automated: `web/tests/e2e/home.spec.mjs` inside `mise run web:test` | playwright report |
| `site.visual` | The page matches the v2 design across themes and modes at desktop and mobile widths | manual: `mise run web:dev:local` or a `web/dist` preview, reviewed against `DESIGN.md` | reviewer confirmation |
| `site.deploy` | Merging to `main` deploys `web/dist` to GitHub Pages | manual: `.github/workflows/deploy.yml` on a `main`-ref push | Pages deployment URL |

## Gotchas and manual gaps

- Automated e2e runs in desktop Chromium only; WebKit/Firefox and real-mobile rendering are manual.
- The arcade cabinet is deleted: the game source tree, the `/game/` route, its component, and every game test were removed, and `site.stubs` asserts the route now 404s.
- `web/test-results/` and `web/playwright-report/` hold Playwright diagnostics and are gitignored.
