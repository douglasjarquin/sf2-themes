# Verify

How to verify changes in this repository before opening a pull request.

Prefer [mise](https://mise.jdx.dev/) tasks from the repo root so local runs match CI.

```verify
entrypoint = "mise run verify"
feature_maps = "docs/features/README.md"
artifacts = ".artifacts/verification"
evidence = ".artifacts/evidence"
task_owner = "."
timeout_seconds = 3600
policy_files = ["VERIFY.md", "docs/theme-guidelines.md", "themes/", "mise.toml"]

[requires]
commands = ["git", "mise", "python3", "uv", "node", "aube"]
```

`mise run verify` is the aggregate pre-PR gate: it runs `web:install`, `test`, `lint`, `validate-catalog`, `shellcheck`, `standalone-freshness`, `web:check`, `web:build`, and `web:test` in order and stops at the first failure.

## Setup

```sh
mise install
mise run deps
```

`mise run deps` syncs Python extras, installs the Astro toolchain via aube, and installs Playwright Chromium for web e2e.

## Readiness

`git rev-parse --show-toplevel` must print this repository's root.
`mise tasks ls` must list `verify` with a source inside this repository; a `verify` task inherited from a parent directory is another project's command.
`mise run deps` must have run once so `uv`, the `web/node_modules/` tree, and the Playwright Chromium build are present.

## Automated checks

`mise run verify` is the canonical aggregate entrypoint.
Each verification role runs that aggregate once for its final candidate; targeted iteration does not require repeating every subcommand before the aggregate.
It runs, in order:

| Check | Command | Proves |
| --- | --- | --- |
| Web dependencies | `mise run web:install` | `web/` installs from `web/aube-lock.yaml` |
| Python and CLI | `mise run test` | pytest under `tests/` plus the copied-standalone harness in `mise-tasks/test-cli` |
| Python lint | `mise run lint` | ruff over `src`, `tests`, and `mise-tasks` |
| Catalog | `mise run validate-catalog` | every theme in `themes/` passes schema and semantic validation |
| Shell | `mise run shellcheck` | the shell file tasks and CI orchestration scripts lint clean |
| Standalone freshness | `mise run standalone-freshness` | the committed `sf2-themes` executable matches current `src/` and `themes/` |
| Web types | `mise run web:check` | `astro check` plus the theme-data freshness gate |
| Web build | `mise run web:build` | all routes emit into `web/dist` under the `/sf2-themes/` base |
| Web tests | `mise run web:test` | Node contract tests plus the Playwright e2e suite |

Scoped tasks stay available for iteration: `mise run pytest`, `mise run lint`, `mise run validate-catalog`, `mise run shellcheck`, `mise run standalone-freshness`, and the `web:`-prefixed tasks.

## Scenarios

The feature maps under `docs/features/` describe the real user journeys and which of them the automated checks exercise.
Rows marked `automated` are covered by `mise run verify`; rows marked `manual` need a real app config, a live browser, or a deployed Pages environment and are recorded `not-run` unless a person reports them.
A green `mise run verify` is evidence for the automated rows only, never for a scenario that was not exercised.

## Isolation

Pytest writes only under test-managed temporary directories, and `mise-tasks/test-cli` runs the copied standalone against a synthetic HOME.
The web e2e suite builds `web/dist` and serves it through Astro's own preview on `PLAYWRIGHT_PORT`; it never touches a deployed site.
No check needs credentials, network services, or the user's real app configuration.

## Artifacts

Each verification run may record `.artifacts/verification/<run-id>/` and captured evidence under `.artifacts/evidence/`; both are Git-ignored.
`web/dist`, `web/test-results/`, and `web/playwright-report/` are regenerated or diagnostic output and are also Git-ignored.
Reference records by path in reports rather than committing them.

## Teardown

The checks leave nothing running: pytest uses temporary directories, and Playwright closes the preview server it started.
Remove `.artifacts/` when the records are no longer needed.

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
