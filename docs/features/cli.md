# sf2-themes CLI and app adapters

`cmd/sf2-themes` builds the `sf2-themes` binary (linked as `sf2` by the installer; commands: `apps`, `themes`, `show`, `validate`, `current`, `setup`, `apply`) over `internal/cli` and the app adapters under `internal/adapters/`: WezTerm, Herdr, Herdr app, Neovim, Codex, Claude Code, Starship, Lazygit, and Slack.
`themes/embed.go` embeds the catalog, so one static binary per platform is the whole product; releases publish it for macOS and Linux on arm64 and amd64.

## Entry points

- `sf2 <command>` or `sf2-themes <command>` from the binary `install.sh` puts in `~/.local/bin`; from a checkout via `mise run apply -- …`, `mise run setup -- …`, or `go run ./cmd/sf2-themes …`.
- `mise run test` for the Go unit tests, the parity goldens and parser corpus in `internal/parity`, and the repository contract tests in `internal/repometa`.

## Scenarios

| ID | Scenario | Driver | Evidence |
| --- | --- | --- | --- |
| `cli.entry-points` | One binary answers as both `sf2-themes` and `sf2`, `--version` prints the bare stamped release version, and the removed `install` command is a usage error | automated: `mise run test` runs `cmd/sf2-themes/main_test.go` (`TestVersionPrintsStampedVersionOnly`) and `internal/parity/install_test.go` (`TestInstallerInstallsAndLinksSf2`), plus the parity cases for `install` | aggregate run record |
| `cli.commands` | `apps`, `themes`, `show`, `validate`, `current`, and boss aliases behave per `docs/sf2-themes/commands.md`; per-command `--help` documents real flags and never depends on terminal width | automated: `mise run test` runs `internal/cli/cli_test.go`, `internal/cli/args_test.go`, `internal/catalog`, and the parity goldens | aggregate run record |
| `cli.adapters` | `apply` is one complete idempotent step (assets + integration + selection) for every app; ownership of each app's theme setting decides one outcome per class (a foreign theme refuses `apply` unless `--adopt`, plain `setup` keeps it), and a refusal writes nothing; `setup` repairs without reselecting and reports what it kept; the summary names the resolved config target, says whether the selection follows host appearance or stays fixed, and gives any reload step; all writes preserve unrelated config, refuse symlinks, and back up before overwriting | automated: `mise run test` runs the adapter suites under `internal/adapters/`, `internal/plan`, `internal/fsys`, `internal/cli/mutate_test.go`, and the parity goldens in `internal/parity` | aggregate run record |
| `cli.standalone` | A copied binary with no sibling files lists and applies all 36 themes, and the release build installs through `install.sh` into a clean machine, verifies its checksum, and applies a theme | automated: `TestBinaryCarriesTheCatalog` and the `TestInstaller*` cases inside `mise run test`, plus the release rehearsal job in `.github/workflows/verify.yml` (`scripts/ci/build-release.sh` and `scripts/ci/rehearse-install.sh` in `debian:bookworm-slim` and on macOS) | aggregate run record and CI job |
| `cli.apply-real` | `apply` against a real WezTerm/Neovim/etc. config produces working theme activation | manual: `mise run apply -- <app> --theme <id>` against a throwaway HOME, then open the app | operator confirmation |

## Gotchas and manual gaps

- Unit tests use synthetic config trees; real app quirks (WezTerm reload timing, Herdr server reload, Starship sourcing order) are only proven manually.
- The parity goldens pin CLI output and written files; re-record them with `mise run parity:record` only for an intended behavior change, and treat a golden diff in review as one.
- A published release is proven only by installing it; the rehearsal job proves the build, not the GitHub Release upload.
