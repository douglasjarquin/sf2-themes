# Theme catalog and generated data

The 36-entry TOML catalog under `themes/` is the single source of truth for every palette.
`mise-tasks/generate-web-theme-data` compiles it to `web/src/data/generated-theme-data.json`, and `web/src/data/theme-data.mjs` plus `web/src/data/site-theme-data.mjs` adapt it for the site payload the layout embeds.

## Entry points

- `themes/*.toml` files, authored per `docs/theme-guidelines.md`.
- `sf2-themes validate --all` and the `mise run validate-catalog` task.
- `aube -C web run themes:generate` and the `--check` freshness gate baked into `web:build`, `web:check`, and `web:test`.

## Scenarios

| ID | Scenario | Driver | Evidence |
| --- | --- | --- | --- |
| `catalog.validate` | Every catalog entry validates against the schema and semantic rules | automated: `mise run validate-catalog` inside `mise run verify`, exercised by `tests/test_catalog.py` | aggregate run record |
| `catalog.freshness` | The committed `web/src/data/generated-theme-data.json` byte-matches a fresh generation from `themes/` | automated: `mise run web:test` runs `web/test/theme-data.test.mjs`, which regenerates and compares | aggregate run record |
| `catalog.shape` | The site payload exposes 18 families with dark and light variants, aliases, and the flattened token set the boot script consumes | automated: `web/test/theme-data.test.mjs` plus the e2e suite in `web/tests/e2e/` inside `mise run web:test` | aggregate run record |
| `catalog.rejects` | Malformed TOML, missing fields, and unknown keys fail generation before they can reach the site | automated: `web/test/theme-data.test.mjs` rejection cases inside `mise run web:test` | aggregate run record |
| `catalog.previews` | `docs/previews/` SVG/PNG assets reflect the current catalog | manual: `mise run generate-previews` then inspect the diff | regenerated files |

## Gotchas and manual gaps

- The catalog is data; visual contrast and family cohesion are judged by a person against `docs/theme-guidelines.md`.
- `docs/previews/` is generated output; only a person can confirm it was regenerated after a theme change.
