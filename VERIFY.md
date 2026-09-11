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

Run when you touch `src/`, `tests/`, `themes/`, `mise-tasks/`, `scripts/`, `docker/`, `pyproject.toml`, `uv.lock`, or `mise.toml`:

```sh
mise run test
mise run lint
mise run validate-catalog
```

- `mise run test` runs pytest and the standalone CLI shell harness.
- `mise run lint` runs ruff over `src`, `tests`, and `mise-tasks`.
- `mise run validate-catalog` validates every theme in `themes/`.

Optional when you change shell scripts:

```sh
mise run shellcheck
```

If you regenerate the committed `sf2-themes` executable, also run:

```sh
mise run standalone-freshness
```

## Web / Astro

Run when you touch `web/` or theme data that feeds the site:

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