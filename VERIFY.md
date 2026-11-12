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
commands = ["git", "mise", "go", "node", "aube"]
```

`mise run verify` is the aggregate pre-PR gate: it runs `web:install`, `build`, `test`, `lint`, `validate-catalog`, `shellcheck`, `web:check`, `web:build`, and `web:test` in order and stops at the first failure.

## Setup

```sh
mise install
mise run deps
```

`mise run deps` downloads the Go modules, installs the Astro toolchain via aube, and installs Playwright Chromium for web e2e.

## Readiness

`git rev-parse --show-toplevel` must print this repository's root.
`mise tasks ls` must list `verify` with a source inside this repository; a `verify` task inherited from a parent directory is another project's command.
`mise run deps` must have run once so the Go module cache, the `web/node_modules/` tree, and the Playwright Chromium build are present.

## Automated checks

`mise run verify` is the canonical aggregate entrypoint.
Each verification role runs that aggregate once for its final candidate; targeted iteration does not require repeating every subcommand before the aggregate.
It runs, in order:

| Check | Command | Proves |
| --- | --- | --- |
| Web dependencies | `mise run web:install` | `web/` installs from `web/aube-lock.yaml` |
| Go build | `mise run build` | every Go package and command under `cmd/` and `internal/` compiles |
| Go tests | `mise run test` | Go unit tests, the parity goldens and parser corpus in `internal/parity`, the installer cases, and the repository contract tests in `internal/repometa` |
| Go lint | `mise run lint` | `gofmt`, `go vet`, and `golangci-lint` over `cmd`, `internal`, and `themes` |
| Catalog | `mise run validate-catalog` | every theme in `themes/` passes schema and semantic validation |
| Shell | `mise run shellcheck` | `install.sh`, the CI scripts, and the dev image entrypoint lint clean |
| Web types | `mise run web:check` | `astro check` plus the theme-data freshness gate |
| Web build | `mise run web:build` | all routes emit into `web/dist` under the `/sf2-themes/` base |
| Web tests | `mise run web:test` | Node contract tests plus the Playwright e2e suite |

Scoped tasks stay available for iteration: `mise run build`, `mise run test`, `mise run lint`, `mise run validate-catalog`, `mise run shellcheck`, and the `web:`-prefixed tasks.
For one package, run `go test -count=1 ./internal/<package>/...`.

## Scenarios

The feature maps under `docs/features/` describe the real user journeys and which of them the automated checks exercise.
Rows marked `automated` are covered by `mise run verify`; rows marked `manual` need a real app config, a live browser, or a deployed Pages environment and are recorded `not-run` unless a person reports them.
A green `mise run verify` is evidence for the automated rows only, never for a scenario that was not exercised.

## Isolation

Go tests write only under `t.TempDir()` directories, and the parity harness and installer cases run the binary against a synthetic HOME.
The web e2e suite builds `web/dist` and serves it through Astro's own preview on `PLAYWRIGHT_PORT`; it never touches a deployed site.
No check needs credentials, network services, or the user's real app configuration.

## Artifacts

Each verification run may record `.artifacts/verification/<run-id>/` and captured evidence under `.artifacts/evidence/`; both are Git-ignored.
`web/dist`, `web/test-results/`, and `web/playwright-report/` are regenerated or diagnostic output and are also Git-ignored.
Reference records by path in reports rather than committing them.

## Teardown

The checks leave nothing running: Go tests use temporary directories, and Playwright closes the preview server it started.
Remove `.artifacts/` when the records are no longer needed.

## Go / CLI

Run when you touch paths that select the Go CI jobs, including `cmd/`, `internal/`, `themes/`, `go.mod`, `go.sum`, `.golangci.yml`, `install.sh`, `scripts/ci/`, `docker/`, `.cursor/`, `.github/workflows/`, `.github/release-notes.md`, `mise.toml`, or `mise.lock`:

```sh
mise run build
mise run test
mise run lint
mise run validate-catalog
mise run shellcheck
```

- `mise run test` runs every Go test uncached, including the parity goldens, the parser corpus, and the installer cases.
- `mise run lint` runs `gofmt`, `go vet`, and `golangci-lint`.
- `mise run validate-catalog` validates every theme in `themes/`.
- `mise run shellcheck` matches the CI toolchain job.

The parity goldens in `internal/parity/testdata/` pin CLI output and written files.
After an intended behavior change, re-record them with `mise run parity:record` and commit the diff with the change; a golden diff in review is a behavior change.
A catalog change also needs `mise run generate-previews` and `mise run generate-web-theme-data`.

CI also runs a release rehearsal on Linux and macOS: it builds the release assets with `scripts/ci/build-release.sh`, checks their checksums, and installs them with `install.sh` into a clean `debian:bookworm-slim` container and a fresh macOS HOME.

## Web / Astro

Run when you touch paths that select the web CI jobs, including `web/`, `themes/`, `cmd/sf2-dev/`, `internal/`, `go.mod`, `go.sum`, `docker/`, `scripts/ci/`, `mise.toml`, or `mise.lock`:

```sh
mise run web:install
mise run web:check
mise run web:build
mise run web:test
```

## CI

Pull requests and pushes to `main` run the GitHub Actions workflow **Verify changes** (`.github/workflows/verify.yml`).

It path-selects jobs for workflows, tooling, Go, and web, then requires the selected jobs to pass through the `gate` job. When path filtering cannot decide, CI fails open and runs the full set.

Local mise tasks above are the intended preflight for that workflow.
