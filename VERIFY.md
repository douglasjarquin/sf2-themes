# Verify

How to verify changes in this repository before opening a pull request.

Prefer [mise](https://mise.jdx.dev/) tasks from the repo root so local runs match CI.

## Setup

```sh
mise install
mise run deps
```

`mise run deps` syncs Python extras, installs the Astro toolchain via aube, and installs Playwright Chromium for web e2e.

## Python / CLI

Run when you touch paths that select the Python CI jobs, including `src/`, `tests/`, `themes/`, `mise-tasks/`, `scripts/`, `docker/`, `.cursor/`, `pyproject.toml`, `uv.lock`, `mise.toml`, `mise.lock`, or the committed `sf2-themes` executable:

```sh
mise run test
mise run lint
mise run validate-catalog
mise run shellcheck
```

- `mise run test` runs pytest and the standalone CLI shell harness.
- `mise run lint` runs ruff over `src`, `tests`, and `mise-tasks`.
- `mise run validate-catalog` validates every theme in `themes/`.
- `mise run shellcheck` matches the CI toolchain job (it runs whenever the Python path set is selected, not only when shell scripts change).

The committed root `sf2-themes` executable embeds `src/sf2_theme` and `themes/`. CI always runs `mise run standalone-freshness` on the Python path set, so edits under `src/` or `themes/` (or other embed inputs) commonly make that binary stale even if you did not regenerate it by hand:

```sh
mise run standalone-freshness
```

If freshness fails, regenerate the standalone executable with the mise build task, then re-run freshness and commit both the regenerated file and your source changes.

## Web / Astro

Run when you touch paths that select the web CI jobs, including `web/`, `themes/`, `mise-tasks/generate-web-theme-data`, `docker/`, `scripts/ci/`, `mise.toml`, or `mise.lock`:

```sh
mise run web:install
mise run web:check
mise run web:build
mise run web:test
```

## CI

Pull requests and pushes to `main` run the GitHub Actions workflow **Verify changes** (`.github/workflows/verify.yml`).

It path-selects jobs for workflows, Python, and web, then requires the selected jobs to pass through the `gate` job. When path filtering cannot decide, CI fails open and runs the full set.

Local mise tasks above are the intended preflight for that workflow.