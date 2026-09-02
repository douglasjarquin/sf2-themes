# Maintain the authoritative theme catalog.

Apply the repository-wide rules in `../AGENTS.md`, and use this file for every change below `themes/`.
Read `../docs/theme-guidelines.md` and `../docs/roster.md` before changing identities, colors, aliases, or variants.
Use `../src/sf2_theme/parse.py`, `../src/sf2_theme/catalog.py`, and `../src/sf2_theme/validation.py` as the executable catalog contract.

## Preserve catalog identity and order.

- Treat `main.toml`, `main-light.toml`, and `characters/*.toml` as the authoritative palette inputs for every consumer.
- Preserve loader order as `main.toml`, `main-light.toml`, and then character TOMLs sorted by filename.
- Keep exactly 36 committed entries comprising 18 dark themes and one explicit light sibling for each dark theme.
- Name every light sibling `<dark-id>-light`, and append ` Light` to the dark theme's display name.
- Keep `main` as the default dark theme and `main-light` as its explicit light sibling.
- Match each character filename stem to its canonical `meta.id`.
- Keep canonical catalog IDs short because `sf2-themes --theme ken` and `--theme ken-light` resolve those source IDs directly.
- Reserve the `sf2-<catalog-id>` form for adapter-installed and selectable identities, and never store that prefix in `meta.id`.
- Keep each light sibling's kind, character, and introduction era aligned with its dark counterpart.
- Keep light aliases empty, while preserving reviewed aliases on their dark owners.
- Keep `bison` unavailable because it is regionally ambiguous, while `boxer`, `claw`, and `dictator` remain owned by their documented dark themes.

## Author complete and explicit TOML.

- Set `schema_version = 1` in every file.
- Limit top-level data to `schema_version`, `meta`, `ui`, `semantic`, and `ansi`.
- Provide every required metadata field, including an explicit `aliases` array.
- Provide all 17 `ui` colors and all seven `semantic` colors.
- Provide exactly the eight named ANSI colors in both `ansi.normal` and `ansi.bright`.
- Set `meta.character` for character themes and omit it for both main themes.
- Commit every dark and light theme as a fully resolved document containing every required token.
- Keep the catalog as the sole source for Lazygit output; do not add Catppuccin flavor or accent files under `themes/`.
- Use literal reviewed `#RRGGBB` values in committed TOML.
- Keep inheritance, `extends`, runtime inversion, and hidden production color generation out of the catalog.

## Preserve meaning and readability.

- Keep red for errors or destructive states, green for success or health, yellow for warnings or activity, blue for information, magenta for alternate states, cyan for secondary information, and orange for interruption or attention.
- Tint semantic hue families for character identity without exchanging their meanings.
- Maintain at least `7.0:1` contrast for primary foreground on background.
- Maintain at least `4.5:1` contrast for normal text surfaces, selections, and cursor glyphs.
- Maintain at least `3.0:1` contrast for the accent on the background and other essential non-text boundaries.
- Use subtext below `4.5:1` only for genuinely secondary content and review the resulting warning deliberately.
- Keep each ANSI row internally unique and keep every bright slot distinct from its matching normal slot.
- Review all validation warnings because cross-slot color reuse can hide accidental palette collapse even when it is not an error.
- Prefer readability corrections over exact pixel sampling while keeping each light theme recognizably paired with its dark theme.

## Regenerate every dependent representation.

- Rebuild the committed root `sf2-themes` executable because `scripts/build-standalone.py` embeds the catalog's exact TOML sources.
- Regenerate `docs/previews/` because `scripts/generate-previews.py` writes one SVG for every catalog ID.
- Expect `web/src/data/theme-data.mjs` to parse the same files in catalog order and expose all 36 entries through `paletteVariants`.
- Review adapter snapshots and `docs/roster.md` when an intentional token, identity, alias, era, or visual direction changes.

## Verify the focused contract before stopping.

- Run `python3 scripts/build-standalone.py` and `python3 scripts/generate-previews.py` from the repository root after catalog edits.
- Run `uv run --with pytest pytest -q tests/test_catalog.py tests/test_validation.py tests/test_snapshots.py` for catalog, contrast, and adapter-render coverage.
- Run both `uv run sf2-themes validate --all` and `./sf2-themes validate --all` to prove source and embedded catalogs accept the same themes.
- Run `bash tests/test_cli.sh` to exercise the copied standalone CLI and prefixed installed identities.
- Run `uv run --with pytest pytest -q tests/test_lazygit.py tests/test_cli.py tests/test_snapshots.py` to verify complete Lazygit key coverage and all 36 generated identities.
- Run `aube -C web run test:unit`, `mise run web:check`, and `mise run web:build` for browser parsing, palette mapping, and static rendering.
- Run `git diff --check` and inspect the scoped diff for the intended TOML and regenerated dependents.
- Finish only when the source loader, standalone loader, browser catalog, and previews all agree on the same ordered 36-theme catalog.
