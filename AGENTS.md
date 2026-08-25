# Project agent memory

This file is the project's committed home for project-intrinsic agent knowledge: build, test, release, architecture, and sharp-edge notes that should travel with the code.

- Add durable project-specific notes here as they are discovered through real work.

## Maintaining this file

Keep this file for knowledge useful to almost every future agent session in this project.
Do not repeat what the codebase already shows; point to the authoritative file or command instead.
Prefer rewriting or pruning existing entries over appending new ones.

## Architecture

- The Neovim adapter owns the managed `colors/`, `sf2-theme/current.lua`, and `plugin/sf2-theme.lua` layout in `src/sf2_theme/adapters/nvim.py`.
- Rebuild the committed standalone CLI with `python3 scripts/build-standalone.py` after changing source modules or catalog behavior.

## Verification

- Run `uv run --with pytest pytest -q` and `bash tests/test_cli.sh` for the Python and standalone CLI suites.
When updating this file, preserve this bar for all agents and keep entries concise.
