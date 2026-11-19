---
title: Develop SF2 Themes
applies_when:
  - Planning, implementing, or reviewing changes in the sf2-themes repository
  - Changing the theme catalog, the Go CLI or its adapters, the installer, or the Astro site
tags: [go, cli, astro, catalog, mise, verification]
---

# Develop SF2 Themes

Read `AGENTS.md` first, then the nested `AGENTS.md` for the area you touch (`internal/`, `internal/adapters/`, `web/`) and `VERIFY.md`.
They are the source for the stack, commands, ownership boundaries, and checks; this rule does not repeat them.

Run everything through the `mise` tasks in `mise.toml`, which pin Go, golangci-lint, Node, aube, and shellcheck.
Use aube for web installs and scripts (`mise run web:*` or `aube -C web ...`); never npm, npx, pnpm, or yarn.
`github.com/BurntSushi/toml` is the only runtime Go dependency; do not add another for something the standard library or an existing package covers.

`themes/` is the single source of truth for palettes.
Adapters and web pages consume the validated `catalog` package or the generated site data; they never parse TOML or restate semantic color rules themselves.
After a catalog change, run `mise run generate-previews` and `mise run generate-web-theme-data`, and `mise run generate-favicons` when the Ryu colors change.
Never hand-edit `docs/previews/`, `web/src/data/generated-theme-data.json`, or the `web/public/` favicons; the web build fails while the site data is stale, and `mise run test` fails while the favicons are.

The parity goldens in `internal/parity/` are the CLI's behavior contract, byte for byte.
Re-record them with `mise run parity:record` only for an intended behavior change, and treat every golden diff in review as a behavior change that needs its own justification.
Never re-record or hand-edit goldens to make a failing test pass.

Adapters return plan steps and never write; `fsys.Commit` is the only writer and owns dry-run, backups, and the symlink policy.
Preserve unrelated user configuration, unknown WezTerm Lua, and unmarked Herdr blocks, and never follow a configuration symlink unless `--follow-symlinks` selects it.

Tests never read or write the real home directory.
Give every test that resolves a config path a synthetic `HOME` and `XDG_CONFIG_HOME` under `t.TempDir()` (with `t.Setenv`), and set or clear the app-specific variables it depends on, as the parity harness does.
Use synthetic configs and data in evidence as well.

The site is static Astro under the `/sf2-themes` Pages base; build internal URLs with `sitePath()` and keep routes useful before scripts run.

`mise run verify` is the pre-PR gate and runs the checks `VERIFY.md` lists.
Attach a CLI transcript for CLI behavior changes, and before and after screenshots for visual site changes.

A release is a `v<semver>` tag on `main`, and only Doug pushes one or merges.
