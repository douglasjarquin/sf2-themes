# Contributing

Thanks for wanting to contribute to SF2 Themes.

This repository welcomes ordinary pull requests against `main`. Keep changes focused, keep verification green, and open a normal GitHub PR.

## Workflow

1. Fork the repo (or clone it if you have write access).
2. Create a branch from current `main`.
3. Make your changes.
4. Run the checks in [VERIFY.md](VERIFY.md) that match what you touched.
5. Commit with a clear message.
6. Push your branch and open a pull request against `main`.

Keep the PR description short: what changed, why, and which VERIFY.md commands you ran.

## Verification

Before you open a PR, follow [VERIFY.md](VERIFY.md).

That file is the source of truth for local and CI verification. Path-selected GitHub Actions jobs mirror the same mise tasks; the `gate` job in the `Verify changes` workflow must pass on the PR.

## Repo conventions

- Python 3.11 CLI package under `src/sf2_theme/`. Public command is `sf2-themes`.
- Tool versions and developer commands live in `mise.toml` (Python 3.11, Node 24, uv, aube). Prefer `mise run …` over ad-hoc tool invocations.
- Theme catalog lives in `themes/` (36 TOML entries). Catalog ids stay short (`ryu`, `ken-light`); installed adapter identities use `sf2-<catalog-id>`.
- Adapters live in `src/sf2_theme/adapters/` (WezTerm, Herdr, Neovim, Codex, Claude Code, Starship, Lazygit, and related helpers). Preserve unrelated user config when mutating files.
- Do not hand-edit the generated root `sf2-themes` executable; regenerate it with the mise standalone build task when needed.
- The Astro site lives in `web/` with static output and the `/sf2-themes` Pages base. Use `mise run web:install`, `web:check`, `web:build`, and `web:test` for site work.
- Broader project guidance for agents and maintainers lives in [AGENTS.md](AGENTS.md).

## Questions

Open a GitHub issue in this repository.