# sf2-themes CLI and app adapters

`src/sf2_theme/` implements the `sf2` Python CLI (also installed as `sf2-themes`; commands: `apps`, `themes`, `show`, `validate`, `current`, `setup`, `apply`, `install`) and the seven app adapters: WezTerm, Herdr, Neovim, Codex, Claude Code, Starship, and Lazygit.
The committed root `sf2-themes` executable is a generated standalone build of the same package, with freshness enforced by `mise run standalone-freshness` over `mise-tasks/build-standalone`.

## Entry points

- `sf2 <command>` or `sf2-themes <command>` from a uv tool install; from a checkout via `uv run sf2-themes`, the `scripts/sf2` alias, or the committed standalone.
- `mise run test` for pytest under `tests/` plus the copied-standalone harness in `mise-tasks/test-cli`.

## Scenarios

| ID | Scenario | Driver | Evidence |
| --- | --- | --- | --- |
| `cli.entry-points` | `pyproject.toml` installs both `sf2` and `sf2-themes` console scripts onto `sf2_theme.cli:main`, and `install` remains a working deprecated alias for `apply` | automated: `mise run test` runs `tests/test_cli.py` (`test_sf2_entry_point_declared`, `test_install_warns_and_applies`, per-command `--help` coverage) | aggregate run record |
| `cli.commands` | `apps`, `themes`, `show`, `validate`, `current`, and boss aliases behave per `docs/sf2-themes/commands.md`; per-command `--help` documents real flags | automated: `mise run test` runs `tests/test_catalog.py` and the command coverage in `tests/` | aggregate run record |
| `cli.adapters` | `apply` is one complete idempotent step (assets + integration + selection) for all seven apps and exits nonzero when integration is blocked; `setup` repairs without reselecting; all writes preserve unrelated config, refuse symlinks, and back up before overwriting | automated: `mise run test` runs the adapter suites in `tests/` (e.g. `tests/test_wezterm.py`, `tests/test_nvim.py`, `tests/test_cli.py`) | aggregate run record |
| `cli.standalone` | The committed standalone embeds the current `src/` and `themes/` and passes the copied-CLI shell harness | automated: `mise run standalone-freshness` inside `mise run verify` plus `mise-tasks/test-cli` inside `mise run test` | aggregate run record |
| `cli.apply-real` | `apply` against a real WezTerm/Neovim/etc. config produces working theme activation | manual: `mise run apply -- <app> --theme <id>` against a throwaway HOME, then open the app | operator confirmation |

## Gotchas and manual gaps

- Unit tests use synthetic config trees; real app quirks (WezTerm reload timing, Herdr server reload, Starship sourcing order) are only proven manually.
- The standalone freshness check fails the aggregate whenever `src/` or `themes/` changed without a regenerated binary.
