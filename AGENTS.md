# PROJECT KNOWLEDGE BASE

**Generated:** 2026-11-12
**Commit:** e2f1b3d
**Branch:** refactor/2026-11-12-1022-refactor-go-port-plan-pr2

## OVERVIEW

SF2 Themes is a Go CLI and static Astro site for one TOML theme catalog shared by terminal adapters and documentation.
The CLI ships as one static `sf2-themes` binary per platform (macOS and Linux, arm64 and amd64) on GitHub Releases, with the catalog embedded; `install.sh` installs it and links `sf2` to it.

## MAINTAINING THIS FILE

This is the committed home for project-intrinsic build, test, release, architecture, and sharp-edge knowledge.
Keep only guidance useful to future sessions across the repository.
Point to authoritative files and commands instead of repeating implementation details.
Rewrite or prune stale notes instead of appending history.
`CLAUDE.md`, `GEMINI.md`, and `GROK.md` are symlinks to this file; edit only `AGENTS.md`.

## STRUCTURE

```text
cmd/sf2-themes/      The shipped binary's main package
cmd/sf2-dev/         Maintainer generators: SVG previews and the site's theme data
internal/            CLI, catalog, plan/commit, adapters, Python-compat helpers, parity proof, repo contracts
themes/              Authoritative 36-entry TOML catalog; embed.go embeds it into the binary
install.sh           One-line installer published with every release
scripts/ci/          Release build, install rehearsal, and container orchestration
docker/              toolchain (mise + Go + Node + aube, no app source) and dev (+ prefetched deps) images
.cursor/             Cursor Cloud environment.json; builds docker/dev/Dockerfile
web/                 Static Astro site, Node tests, and Stagehand e2e
docs/                Authored contracts and generated SVG previews
.github/workflows/   Path-selected CI, release, Pages build/deploy, image publishing, and issue automation
```

## WHERE TO LOOK

| Task | Location | Notes |
|---|---|---|
| CLI dispatch | `internal/cli/cli.go`, `args.go`, `help.go` | Public commands are `sf2` and `sf2-themes`; one binary answers to both |
| Mutating commands | `internal/cli/mutate.go` | `setup`, `apply`, `current`, and the ownership gate |
| Theme model and lookup | `internal/catalog/` | `themes/` remains the source of truth; `internal/AGENTS.md` maps the package |
| Adapter mutation | `internal/adapters/<app>/`, `internal/plan/`, `internal/fsys/` | Preserve unrelated user configuration and symlink policy; see `internal/adapters/AGENTS.md` |
| Python-identical bytes | `internal/pycompat/` | Text, paths, `repr`, JSON, and diffs exactly as the 1.x Python CLI produced them |
| Parity proof | `internal/parity/` | Goldens for every command and app, the TOML/JSON parser corpus, and the installer cases |
| Repository contracts | `internal/repometa/` | Tests that pin the CI workflow, dev image, and Cursor environment |
| Release | `.github/workflows/release.yml`, `scripts/ci/build-release.sh`, `install.sh`, `.github/release-notes.md` | Version is stamped into `internal/version` at build time |
| Generators | `cmd/sf2-dev`, `internal/devtools/` | `previews` writes `docs/previews/`; `web-data` writes the site's theme data |
| Theme design contract | `docs/theme-guidelines.md`, `docs/roster.md` | Covers IDs, dark/light pairs, semantics, and validation |
| Astro shell and routes | `web/src/layouts/`, `web/src/pages/`, `web/src/lib/site-path.mjs` | Static Pages base is `/sf2-themes` |
| Site theme runtime | `web/src/scripts/site-theme.mjs`, `web/src/data/site-theme-data.mjs` | Header controls, `⌘K` palette, `?theme=` deep links |
| Site install command and SEO | `web/src/lib/seo.mjs`, `web/public/llms.txt` | `llms.txt` is generated from `seo.mjs` and a unit test holds them equal |
| Go verification | `mise.toml`, `internal/*/..._test.go` | `mise run test` runs everything uncached |
| Web verification | `web/test/`, `web/tests/unit/`, `web/tests/e2e/` | Node contracts plus real browser coverage through Stagehand |
| Starship / zsh prompt | `internal/adapters/starship/` | Managed palette + sourcable command highlight snippet |
| Containerized dev/CI | `docker/`, `scripts/ci/run-in-dev-container.sh` | `mise run <task>` runs identically on bare host or through the wrapper |
| Cursor Cloud | `.cursor/environment.json`, `docker/dev/Dockerfile` | Cloud agents build the dev image with no toolchain build-arg |

## CODE MAP

| Symbol | Type | Location | Role |
|---|---|---|---|
| `catalog.Theme` | data model | `internal/catalog/model.go` | Shared immutable palette contract |
| `catalog.Load` | function | `internal/catalog/catalog.go` | Validated catalog entry point over the embedded `themes.FS` |
| `cli.Main` | function | `internal/cli/cli.go` | Binary entry; returns the exit code to `cmd/sf2-themes` |
| `fsys.Commit` | function | `internal/fsys/commit.go` | The only writer: atomic writes, backups, dry-run, and symlinks |
| `adapters.App` | struct | `internal/adapters/adapter.go` | One per app: inspect, plan, and labels |
| `siteThemeFamilies` | data | `web/src/data/site-theme-data.mjs` | Flattened site payload from the canonical catalog |

## CONVENTIONS

- Tool versions and stable developer commands live in `mise.toml`: Go 1.27, golangci-lint 2.15, Node 24, Aube 2.2.4, and shellcheck 0.11.0.
- The Go module is `github.com/douglasjarquin/sf2-themes/v2`, and `github.com/BurntSushi/toml` is its only runtime dependency.
- Catalog IDs stay short; adapter-installed identities use `sf2-<catalog-id>`.
- Catalog order is `main.toml`, optional `main-light.toml`, then sorted character TOMLs.
- The binary always reads the embedded catalog; `SF2_THEME_DIR` is not honored.
- The parity goldens are the behavior contract: CLI output and written files change only through `mise run parity:record`, and a golden diff in review is a behavior change.
- The Astro app uses static output and the `/sf2-themes` Pages base; internal URLs go through `sitePath()`.
- CI path filtering fails open when filter data is missing or invalid.
- Pull requests build the Pages artifact and rehearse the release install, but deployment occurs only on non-PR events.
- A release is a `v<semver>` tag on `main`; push one only with the owner's explicit go-ahead.

## ANTI-PATTERNS

- Do not hand-edit `docs/previews/` or `web/src/data/generated-theme-data.json`; run `mise run generate-previews` or `mise run generate-web-theme-data`.
- Do not edit parity goldens by hand or re-record them to make a failing test pass without an intended behavior change.
- Do not duplicate catalog parsing or semantic color rules inside adapters or pages.
- Do not write files from an adapter; return plan steps for `fsys.Commit`.
- Do not modify unknown WezTerm Lua, unmarked Herdr theme blocks, or unrelated adapter configuration.
- Do not follow configuration symlinks unless the caller explicitly selects `--follow-symlinks`.
- Do not use npm, npx, pnpm, or yarn for web installs or scripts; use aube through `mise run web:*` or `aube -C web ...`.

## COMMANDS

```bash
mise run deps
mise run build
mise run test
mise run lint
mise run validate-catalog
mise run shellcheck
mise run apply -- wezterm --theme vega
go run ./cmd/sf2-themes --version
mise run generate-previews
mise run generate-web-theme-data
mise run parity:record
mise run web:install
mise run web:check
mise run web:build
mise run web:test
mise run web:dev:local
mise run verify
scripts/ci/run-in-dev-container.sh mise run test
```

## NOTES

- Catalog changes require `mise run generate-previews` and `mise run generate-web-theme-data`; the web build fails while the site's theme data is stale.
- `mise run verify` is the pre-PR gate; `VERIFY.md` lists what it runs.
- The Neovim adapter owns managed `colors/`, `sf2-theme/current.lua`, and `plugin/sf2-theme.lua` output.
- The installer verifies SHA-256 against `checksums.txt` (integrity, not origin); `gh attestation verify` proves origin.
- The installer refuses to write through a symlink, which protects a leftover 1.x uv install; the fix it names is `uv tool uninstall sf2-themes`.
- Generated `.omo/`, caches, and build output do not count as source complexity when placing child guidance.
- CI and `.made.yml` run `web:install`/`web:build`/`web:check`/`web:test` through aube. Keep `web/.npmrc` on `node-linker=hoisted` so Astro prerender can resolve native bindings.
- `web:dev:local` serves the Astro site at `https://sf2-themes.test` via portless instead of a raw port; see `web/AGENTS.md` for setup and the Astro agent-detection caveat.

## Cursor Cloud

Cursor Cloud uses the mise environment from `.cursor/environment.json`.
It builds `docker/dev/Dockerfile` with no toolchain build-arg, so that file bootstraps mise when `FROM` is Ubuntu.
CI still passes the published toolchain image into the same Dockerfile.

Do not start nested Docker.
Do not wrap commands in `scripts/ci/run-in-dev-container.sh`.
The agent already is the environment.
Use `mise run test`, `mise run lint`, `mise run web:check`, and `mise run web:test`.
`mise run web:dev:container` is already running in the `web` terminal on port 4322.
Cloud verification uses the aube web tasks.
