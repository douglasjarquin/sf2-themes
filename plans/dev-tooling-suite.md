# Dev Tooling Suite: mise file-tasks, aube everywhere, containerized dev/CI

## TL;DR
> **Summary**: Eliminate every mise task that shims out to a separate script file (move the script into a real mise file-task instead); replace the remaining raw npm/npx calls with aube; containerize local dev and CI behind a two-tier Docker image (toolchain + dev) published to GHCR, with job-scoped volume cleanup so nothing leaks onto a runner across runs.
> **Deliverables**: `mise-tasks/` directory (6 relocated scripts as real file-tasks), updated `mise.toml` (lockfile + new tasks), `docker/toolchain/` + `docker/dev/` images, `scripts/ci/*.sh` container-orchestration wrappers, restructured `verify.yml`/`deploy.yml` + new `publish-images.yml`, updated `.made.yml`, and every doc/test reference kept in sync.
> **Effort**: XL
> **Parallel**: YES - 7 waves
> **Critical Path**: Wave 1 (mise.lock + dependency reconciliation) -> Wave 2 (script relocations) -> Wave 3 (mise task definitions) -> Wave 4 (Docker images) -> Wave 5 (CI restructure) -> Wave 6 (aube-in-CI verification) -> Wave 7 (`.made.yml` + doc sweep) -> Final Verification

## Context

### Original Request
Boss wants a "suite of dev tools" for sf2-themes: (1) switch all tooling/tasks to mise, with no task that just bashes out to an existing script — the script's body moves into a real mise file-based task; (2) replace every remaining npm/pnpm call with aube; (3) a container-based dev environment (tooling install, test, and run-the-website all happen inside a Docker container), and CI should also run inside that container so self-hosted runners don't accumulate leftover artifacts across runs. Inspiration (explicitly not verbatim): boss's `niceuptime` project's `docker/dev/Dockerfile`, `docker/toolchain/Dockerfile`, and `scripts/ci/{run-in-dev-container,ensure-dev-image,gc-job-overlay-volumes}.sh`.

### Interview Summary
- **Two-tier images, not one Dockerfile**: boss corrected the planner's initial single-Dockerfile recommendation — a single Dockerfile's layer cache only pays off within one repo; a separately-published toolchain image is meant to be reused across a future "suite of dev tools" beyond just this repo.
- CI stays on GitHub-hosted `ubuntu-latest` (not moving to self-hosted for this repo).
- Adopt `mise.lock` (`[settings] lockfile = true`) for reproducible tool installs.
- Local dev container supports both one-shot task runs and a long-running dev-server mode.
- **Every** script under `scripts/` gets a mise-tasks/ relocation, including the two one-off theme-authoring tools — not just the routine test/build/CI ones.
- CI's inline step logic folds into mise tasks too; `verify.yml`/`deploy.yml` steps shrink to calling `mise run <task>` (via the container wrapper) instead of duplicating commands in YAML.
- `.made.yml` (a separate agent-review/CI gate tool's config, currently hardcoding absolute host paths and duplicating `mise run test` plus raw npm/ruff/shellcheck) gets updated to call the same mise task names directly on host (not through the container — its job is a fast local self-check, not CI parity).
- Registry: GHCR, authenticated via the repo's own `GITHUB_TOKEN`, no new secret.
- `scripts/sf2` is explicitly excluded from relocation — README documents it as a user-facing convenience entry point (`scripts/sf2 …` in place of `sf2-themes …`), not an internal build/test shim.
- `scripts/ci/*.sh` (the new container-orchestration bootstrap scripts) are explicitly NOT mise tasks — they're host-level scripts whose job is to launch the container the mise tasks then run inside; a mise task can't bootstrap its own container.

### Metis Review (gaps addressed)
- **Python version-matrix contradiction**: the confirmed single pinned-Python-3.11 toolchain image cannot serve the existing 4-version (`3.11`-`3.14`) pytest compatibility matrix. Resolved: the matrix job is orthogonal to tooling-consistency containerization (it tests the *package's* cross-version compatibility, not dev-tooling consistency) — it stays exactly as-is, using bare `actions/setup-python`, untouched by this migration. Only lint/shellcheck/standalone-freshness/catalog-validate/shell-integration and the web job move into the containerized `toolchain-checks`/`web` jobs.
- **Unverified "aube fixes the fresh-runner virtual-store failure" assumption**: `web/AGENTS.md` records a *specific prior finding* that aube's install fails on a fresh CI runner — nothing proves containerizing fixes that. Resolved: Wave 6 adds an explicit reproduction task with a documented npm-inside-the-container fallback if aube still fails (this still satisfies "no npm on the host" even if it doesn't satisfy "no npm anywhere").
- **Test files hardcode the exact strings being replaced**: `tests/test_ci_workflow.py` asserts the literal `npm --prefix web ci`/`npm --prefix web run build` strings and the exact `gate.needs` set; `tests/test_import_revised_themes.py`, `web/test/theme-data.test.mjs`, and `web/test/palette-terminal-mapping.test.mjs` hardcode the old `scripts/*.py` paths. All four are now explicit tasks, not overlooked side effects.
- **Extensionless file-task lint/shellcheck blind spot** (Metis verified empirically: `ruff check mise-tasks` silently finds nothing and exits 0; `shellcheck mise-tasks/*` hard-errors on the Python files): lint tasks now enumerate files per-language explicitly instead of globbing the directory, and `pyproject.toml` gets `extend-include` for the Python file-tasks.
- **Named-volume masking, UID/GID, port collision, fingerprint scope, multi-arch, GHCR fork-PR visibility, layer-cache-on-ephemeral-runners, build-standalone.py's self-referencing embedded path string, rollback path**: each has a concrete resolution baked into the relevant task below (see References/Acceptance Criteria).
- **Scope-creep sequencing suggestion**: adopted as this plan's wave structure (relocations -> task definitions -> images -> CI -> aube verification -> `.made.yml`), so each wave is independently revertible even though it's one plan.
- Two items are genuinely outside agent-executable scope and are called out explicitly rather than silently assumed: making the new GHCR packages **Public** (so fork PRs' read-only `GITHUB_TOKEN` can pull them) and updating any branch-protection required-status-check names — both are GitHub UI/settings actions for the boss, listed in Final Verification.

## Work Objectives

### Core Objective
Move sf2-themes' tooling entirely onto mise (no script-shim tasks), route every package-manager call through aube, and make local dev + CI run inside a versioned, cache-friendly, GHCR-published container so nothing installs onto — or leaks state onto — the host or a shared runner.

### Deliverables
- `mise-tasks/{test-cli,build-standalone,generate-previews,generate-web-theme-data,author-character-themes,import-revised-themes}` — real, executable, `#MISE`-annotated file tasks replacing the old `scripts/*.py` + `tests/test_cli.sh`.
- Updated `mise.toml`: `[settings] lockfile = true`, `mise.lock` committed, `ruff`/`shellcheck` added as pinned tools, new `lint`/`shellcheck`/`validate-catalog`/`standalone-freshness`/`web:dev:container` tasks, old shim body removed from `[tasks.test]`.
- `docker/toolchain/{Dockerfile,apt-packages.txt}` — mise + Python 3.11 + Node 24 + aube 2.1 + ruff + shellcheck, no app source, multi-arch (`linux/amd64` + `linux/arm64`).
- `docker/dev/{Dockerfile,entrypoint.sh,apt-packages.txt}` — layered `FROM` the toolchain image, app manifests + deps prefetched (uv + aube + Playwright Chromium), non-root `dev` user.
- `scripts/ci/{dev-image-fingerprint.sh,ensure-toolchain-image.sh,ensure-dev-image.sh,run-in-dev-container.sh,gc-job-overlay-volumes.sh}`.
- `.github/workflows/publish-images.yml` (new), restructured `verify.yml` and `deploy.yml`.
- Updated `.made.yml`, and every doc/test reference to a relocated path (11+ files, enumerated per-task below).

### Definition of Done (verifiable conditions with commands)
- `mise run test` passes locally on a clean checkout with only `mise` on PATH (no manual `pip install`/`npm install` beforehand).
- `docker run $(scripts/ci/ensure-dev-image.sh) mise run web:test` passes with zero host-side Node/Python install.
- `.github/workflows/verify.yml` passes end-to-end on a real PR (both the untouched Python-matrix job and the new containerized jobs).
- `docker volume ls | grep sf2-themes-ci-<run>-<job>` returns empty after any CI job completes (GC verified).
- No `npm`/`npx`/`pnpm` string appears anywhere except inside a Dockerfile's apt/OS-dependency layer or the documented aube-fallback exception from Wave 6: `grep -rn "npm \|npx \|pnpm" --include='*.yml' --include='*.mjs' --include='*.json' . | grep -v node_modules`.
- `grep -rln "scripts/build-standalone.py\|scripts/generate-previews.py\|scripts/generate-web-theme-data.py\|scripts/author_character_themes.py\|scripts/import-revised-themes.py\|tests/test_cli.sh" . --include='*.md' --include='*.py' --include='*.json' --include='*.mjs' --include='*.yml'` returns nothing (every reference updated).

### Must Have
- Every relocated script keeps its exact runtime behavior (same output, same exit codes) — verified by re-running its existing test coverage after the move, before touching anything downstream.
- `mise.lock` committed and honored (`mise install --locked`) in both images.
- The `dev` image's baked deps are actually usable, not masked by an empty mounted volume (explicit seed-on-first-run check).
- UID/GID fixup so bind-mounted files stay host-owned on both the Linux CI runner and a macOS/OrbStack dev machine.
- The 4-version Python compatibility matrix keeps running exactly as today.

### Must NOT Have
- `scripts/sf2` is not moved, renamed, or wrapped.
- `scripts/ci/*.sh` are not converted into mise tasks.
- No new task whose entire body is `run = "bash <path-to-another-script>"` or `run = "python3 <path-to-another-script>"` — if a task needs more than one line, it becomes a `mise-tasks/` file, not a one-line shim.
- Do not touch the Python-version test matrix's structure (job name, matrix versions, `allow-prereleases`) — only its non-multi-version steps move out of it.
- Do not silently mark the GHCR-package-visibility or branch-protection follow-ups as done — they are boss actions, listed explicitly in Final Verification.

## Verification Strategy
> ZERO HUMAN INTERVENTION for everything agent-executable; the two GitHub-settings actions above are the sole exceptions and are called out, not silently skipped.
- Test decision: tests-after, existing pytest/Node suites are the safety net for every relocation; new bash/Dockerfile logic gets bash-level QA scenarios (no new test framework introduced for shell scripts).
- QA policy: every task below has agent-executed scenarios (happy path + a failure/edge case) with concrete commands and expected output.
- Evidence: `evidence/task-{N}-{slug}.{ext}` per task (command transcripts, `docker inspect`/`docker volume ls` output, diff output).

## Execution Strategy

### Parallel Execution Waves
Wave 1: Foundation — mise.lock, dependency-strategy reconciliation, ruff config, two independent script relocations with no cross-references.
Wave 2: Remaining script relocations (each touches a different, non-overlapping set of doc/test files — safe to parallelize).
Wave 3: mise.toml task definitions (lint/shellcheck/validate/standalone-freshness/web:dev:container) — depends on Wave 1+2 files existing at their final paths.
Wave 4: Docker images + `scripts/ci/*.sh` wrappers — depends on Wave 3's task names being final (the Dockerfile/wrapper scripts reference `mise run <task>` by name).
Wave 5: CI workflow restructure + workflow-test updates — depends on Wave 4's scripts existing and being runnable.
Wave 6: aube-in-CI reproduction + fallback decision — depends on Wave 5's containerized `web` job existing to test against.
Wave 7: `.made.yml` rewrite + final doc sweep — depends on every task name/path from Waves 1-6 being final.

### Dependency Matrix (full, all tasks)
| Task | Wave | Blocked By | Blocks |
|---|---|---|---|
| 1. mise.lock + pin ruff/shellcheck | 1 | — | 10 |
| 2. Python dependency reconciliation | 1 | — | 10, 20 |
| 3. pyproject.toml ruff config | 1 | — | 10 |
| 4. Relocate import-revised-themes.py | 1 | — | 26 |
| 5. Relocate author_character_themes.py | 1 | — | 26 |
| 6. Relocate generate-previews.py | 2 | — | 10, 26 |
| 7. Relocate generate-web-theme-data.py | 2 | — | 10, 26 |
| 8. Relocate tests/test_cli.sh | 2 | — | 10, 26 |
| 9. Relocate build-standalone.py | 2 | — | 10, 26 |
| 10. mise.toml task definitions | 3 | 1,2,3,6,7,8,9 | 12,13,17 |
| 11. web:dev:container task | 3 | 10 | 17 |
| 12. docker/toolchain image | 4 | 10 | 13,16 |
| 13. docker/dev image | 4 | 10,12 | 16,17 |
| 14. docker/dev/entrypoint.sh | 4 | 13 | 17 |
| 15. dev-image-fingerprint.sh | 4 | 12,13 | 16 |
| 16. ensure-toolchain/dev-image.sh | 4 | 12,13,15 | 17,19 |
| 17. run-in-dev-container.sh | 4 | 10,11,13,14,16 | 20,21,23 |
| 18. gc-job-overlay-volumes.sh | 4 | — | 20 |
| 19. publish-images.yml | 5 | 16 | — |
| 20. verify.yml restructure | 5 | 2,17,18 | 22,23,24 |
| 21. deploy.yml restructure | 5 | 17 | — |
| 22. test_ci_workflow.py updates | 5 | 20,21 | — |
| 23. _workflow_yaml.py parser check | 5 | 20,21 | — |
| 24. aube-in-CI reproduction | 6 | 20 | 25 |
| 25. web/AGENTS.md rationale update | 6 | 24 | — |
| 26. .made.yml rewrite | 7 | 4,5,6,7,8,9 | — |
| 27. Final doc sweep | 7 | 1-26 | F1-F4 |

## TODOs

- [x] 1. Adopt mise.lock and pin ruff/shellcheck as mise tools

  **What to do**: Add `[settings] lockfile = true` to `mise.toml`. Add `ruff` and `shellcheck` to `[tools]`, resolving exact pinned versions via `mise ls-remote ruff` / `mise ls-remote shellcheck` (pick the latest stable at implementation time — do not guess a version number here). Run `mise install` to generate and commit `mise.lock`.
  **Must NOT do**: Do not pin `mise` itself here (that happens in the Dockerfile in Wave 4, via a `MISE_VERSION` build arg).

  **Parallelization**: Can Parallel: YES | Wave 1 | Blocks: [10] | Blocked By: []

  **References**:
  - `mise.toml:1-9` - current `[tools]` block to extend.
  - mise file-task docs (`mise help run`, confirmed this session) - `[settings] lockfile = true` plus `mise install --locked` is the reproducibility contract used throughout this plan.

  **Acceptance Criteria**:
  - [ ] `mise.lock` exists at repo root and is committed.
  - [ ] `ruff --version` and `shellcheck --version` both resolve via `mise exec -- <tool> --version` without any `pip`/`apt` install.
  - [ ] `mise install --locked` exits 0 on a clean checkout.

  **QA Scenarios**:
  ```
  Scenario: Locked install is reproducible
    Tool: bash
    Steps: rm -rf ~/.local/share/mise/installs/ruff* ~/.local/share/mise/installs/shellcheck*; mise install --locked
    Expected: exits 0, `mise exec -- ruff --version` and `mise exec -- shellcheck --version` both print the exact versions recorded in mise.lock
    Evidence: evidence/task-1-mise-lock.txt

  Scenario: Lockfile drift is caught
    Tool: bash
    Steps: edit mise.toml to bump ruff's version string without updating mise.lock, then run `mise install --locked`
    Expected: non-zero exit, error naming the mismatched tool/version
    Evidence: evidence/task-1-mise-lock-drift.txt
  ```

  **Commit**: YES | Message: `chore(mise): adopt mise.lock, pin ruff and shellcheck as mise tools` | Files: [mise.toml, mise.lock]

- [x] 2. Reconcile the Python dependency install strategy on `uv sync --all-extras`

  **What to do**: Trim `pyproject.toml`'s `dev` extra to `["pytest>=8"]` (drop `ruff` — it's now a mise tool, task 1). Replace every install path with `uv sync --all-extras`: the local `mise run test` composite's `uv run --with pytest pytest -q` becomes `uv sync --all-extras && uv run pytest -q`; CI's `pip install -e ".[dev]"` (`verify.yml:91`, matrix job) becomes `pip install -e . pytest` (the matrix job stays on bare `actions/setup-python`+`pip`, per the Metis resolution — it does not use `uv`/mise, since its whole point is testing across raw Python installs); `.made.yml`'s `uv run ruff` becomes `mise exec -- ruff` (paired with task 26).
  **Must NOT do**: Do not change the matrix job's `actions/setup-python` structure — only the specific `pip install` line changes.

  **Parallelization**: Can Parallel: YES | Wave 1 | Blocks: [10, 20] | Blocked By: []

  **References**:
  - `pyproject.toml:14` - `dev = ["pytest>=8", "ruff>=0.6"]`, current extras block.
  - `.github/workflows/verify.yml:91` - `pip install -e ".[dev]"` in the matrix `test` job, stays but simplified to drop ruff.
  - `mise.toml:7-9` (current `[tasks.test]`) - `uv run --with pytest pytest -q` ad hoc invocation being replaced.
  - `.made.yml:20` - `uv run ruff` needs replacing with the mise-tool invocation once task 1 lands.

  **Acceptance Criteria**:
  - [ ] `grep -n "ruff" pyproject.toml` shows no match under `[project.optional-dependencies]`.
  - [ ] `uv sync --all-extras && uv run pytest -q` passes locally.
  - [ ] CI matrix job's `pip install` step no longer references `ruff`.

  **QA Scenarios**:
  ```
  Scenario: uv sync installs exactly pytest as the dev extra
    Tool: bash
    Steps: rm -rf .venv; uv sync --all-extras; uv pip list | grep -i ruff
    Expected: uv pip list shows pytest, shows NO ruff (ruff lives in mise's tool install, not the venv)
    Evidence: evidence/task-2-uv-sync.txt

  Scenario: Matrix job still installs cleanly without the mise/uv toolchain
    Tool: bash (simulating CI's bare pip path)
    Steps: python3 -m venv /tmp/matrix-venv && /tmp/matrix-venv/bin/pip install -e . pytest && /tmp/matrix-venv/bin/pytest -q
    Expected: exits 0, no reference to mise or uv anywhere in this path
    Evidence: evidence/task-2-matrix-install.txt
  ```

  **Commit**: YES | Message: `chore(deps): standardize Python installs on uv sync, keep the matrix job on bare pip` | Files: [pyproject.toml, mise.toml, .github/workflows/verify.yml, .made.yml]

- [x] 3. Fix pyproject.toml ruff config for extensionless mise-tasks files

  **What to do**: Update `[tool.ruff] src` from `["src", "tests", "scripts"]` to `["src", "tests", "mise-tasks"]` (drop `scripts` — only `scripts/sf2`, a bash file, remains there after Wave 2). Add `extend-include = ["mise-tasks/*"]` so ruff checks the extensionless Python file-tasks (Metis verified `ruff check mise-tasks` on a directory of extensionless Python-shebang files silently finds nothing and exits 0 — this must not regress).
  **Must NOT do**: Do not remove `scripts` from `.gitignore` or delete the directory itself — `scripts/sf2` and `scripts/ci/` (Wave 4) still live there.

  **Parallelization**: Can Parallel: YES | Wave 1 | Blocks: [10] | Blocked By: []

  **References**:
  - `pyproject.toml:23-26` - current `[tool.ruff]` block.
  - Metis finding (this session) - `ruff check mise-tasks` on extensionless files exits 0 with "No Python files found" unless passed as explicit paths or `extend-include`d.

  **Acceptance Criteria**:
  - [ ] `ruff check mise-tasks` (after Wave 2 populates it) reports the same findings as `ruff check` on the old `scripts/*.py` files did before the move (zero regressions, zero silent no-ops).
  - [ ] Introducing a deliberate lint violation into a `mise-tasks/` file makes `ruff check mise-tasks` fail with a non-zero exit.

  **QA Scenarios**:
  ```
  Scenario: extend-include makes ruff see extensionless task files
    Tool: bash
    Steps: (after Wave 2) echo "import os" >> mise-tasks/build-standalone; ruff check mise-tasks
    Expected: non-zero exit, reports the unused-import (or whatever rule fires) on mise-tasks/build-standalone; revert the test edit after
    Evidence: evidence/task-3-ruff-extend-include.txt

  Scenario: Regression guard — before extend-include, ruff silently misses it
    Tool: bash
    Steps: temporarily comment out the extend-include line, repeat the same injected-violation test
    Expected: `ruff check mise-tasks` exits 0 (proves the fix is load-bearing, not decorative); restore the line after
    Evidence: evidence/task-3-ruff-regression-proof.txt
  ```

  **Commit**: YES | Message: `chore(lint): point ruff at mise-tasks and extend-include extensionless files` | Files: [pyproject.toml]

- [x] 4. Relocate scripts/import-revised-themes.py to mise-tasks/import-revised-themes

  **What to do**: `git mv scripts/import-revised-themes.py mise-tasks/import-revised-themes` (drop the `.py` extension — mise resolves the task name from the filename; keep the `#!/usr/bin/env python3` shebang; `chmod +x`). Add a `#MISE description="..."` header line matching the file's existing docstring/usage comment. Update the self-referencing usage comment inside the file (it names its own old path). Update `tests/test_import_revised_themes.py:10,40` (`IMPORTER_PATH = Path("scripts/import-revised-themes.py")` and its second use) to point at `mise-tasks/import-revised-themes`.
  **Must NOT do**: Do not change the script's argument parsing or behavior — this is a pure relocation.

  **Parallelization**: Can Parallel: YES | Wave 1 | Blocks: [26] | Blocked By: []

  **References**:
  - `scripts/import-revised-themes.py:11,13` - self-referencing usage comments to update.
  - `tests/test_import_revised_themes.py:10,40` - hardcoded `IMPORTER_PATH`, the exact regression Metis caught.
  - mise file-task docs - filename (no extension needed on macOS/Linux) becomes the task name; `#MISE description=` header syntax.

  **Acceptance Criteria**:
  - [ ] `mise run import-revised-themes --help` (or equivalent no-op invocation) resolves to the relocated file.
  - [ ] `uv run pytest tests/test_import_revised_themes.py -q` passes unchanged.
  - [ ] `git grep -n "scripts/import-revised-themes.py"` returns nothing.

  **QA Scenarios**:
  ```
  Scenario: Relocated task runs identically to the old script
    Tool: bash
    Steps: mise run import-revised-themes --dry-run (or whatever no-op flag the script supports) ; diff output against a pre-move capture
    Expected: byte-identical output/behavior, only the invocation path differs
    Evidence: evidence/task-4-relocation-diff.txt

  Scenario: Test suite still finds it at the new path
    Tool: bash
    Steps: uv run pytest tests/test_import_revised_themes.py -q
    Expected: all tests pass; a temporary revert of the IMPORTER_PATH update (pointing back at the old, now-missing path) makes the same suite fail with FileNotFoundError, proving the test genuinely exercises the path
    Evidence: evidence/task-4-test-path-proof.txt
  ```

  **Commit**: YES | Message: `chore(mise-tasks): relocate import-revised-themes to a mise file task` | Files: [mise-tasks/import-revised-themes, tests/test_import_revised_themes.py]

- [x] 5. Relocate scripts/author_character_themes.py to mise-tasks/author-character-themes

  **What to do**: `git mv scripts/author_character_themes.py mise-tasks/author-character-themes`, keep shebang, `chmod +x`, add `#MISE description=`. No test or doc hardcodes this path (verified via grep this session) — pure relocation, no downstream reference updates needed.
  **Must NOT do**: Change argument parsing or output.

  **Parallelization**: Can Parallel: YES | Wave 1 | Blocks: [26] | Blocked By: []

  **References**:
  - `scripts/author_character_themes.py` - file being moved, no other referencing files found via `grep -rn author_character_themes`.

  **Acceptance Criteria**:
  - [ ] `git grep -n "author_character_themes"` shows only the relocated file's own internal references (if any self-naming comments — update those too).
  - [ ] Script runs identically from the new path.

  **QA Scenarios**:
  ```
  Scenario: Task runs from new location
    Tool: bash
    Steps: mise run author-character-themes --help (or its documented no-op/help flag)
    Expected: same output as invoking the old scripts/ path pre-move
    Evidence: evidence/task-5-relocation.txt

  Scenario: No stale references remain
    Tool: bash
    Steps: git grep -rn "scripts/author_character_themes.py"
    Expected: zero matches
    Evidence: evidence/task-5-no-stale-refs.txt
  ```

  **Commit**: YES | Message: `chore(mise-tasks): relocate author-character-themes to a mise file task` | Files: [mise-tasks/author-character-themes]

- [x] 6. Relocate scripts/generate-previews.py to mise-tasks/generate-previews

  **What to do**: `git mv scripts/generate-previews.py mise-tasks/generate-previews`, shebang + `chmod +x` + `#MISE description=`. Update every doc reference to `python3 scripts/generate-previews.py`: `AGENTS.md:74,97` -> `mise run generate-previews`; `themes/AGENTS.md:48,49,55` (per Metis) -> same substitution, keeping the surrounding sentence's meaning ("run after theme changes") intact.
  **Must NOT do**: Change what the script generates (`docs/previews/*.svg`) or its determinism contract.

  **Parallelization**: Can Parallel: YES | Wave 2 | Blocks: [10, 26] | Blocked By: []

  **References**:
  - `AGENTS.md:74,97` - "Do not hand-edit `docs/previews/`; run `python3 scripts/generate-previews.py`" and the theme-change note.
  - `themes/AGENTS.md:48,49,55` (Metis-cited lines) - same pattern, theme-catalog-specific doc.

  **Acceptance Criteria**:
  - [ ] `mise run generate-previews` produces byte-identical `docs/previews/*.svg` output to the pre-move script.
  - [ ] `git grep -n "scripts/generate-previews.py"` returns nothing.

  **QA Scenarios**:
  ```
  Scenario: Regenerated previews are byte-identical
    Tool: bash
    Steps: git stash; python3 scripts/generate-previews.py; cp -r docs/previews /tmp/before; git stash pop; mise run generate-previews; diff -r /tmp/before docs/previews
    Expected: no diff output
    Evidence: evidence/task-6-preview-diff.txt

  Scenario: Docs point at the new invocation
    Tool: bash
    Steps: grep -n "generate-previews" AGENTS.md themes/AGENTS.md
    Expected: every match reads "mise run generate-previews", none read the old scripts/ path
    Evidence: evidence/task-6-doc-refs.txt
  ```

  **Commit**: YES | Message: `chore(mise-tasks): relocate generate-previews to a mise file task` | Files: [mise-tasks/generate-previews, AGENTS.md, themes/AGENTS.md]

- [x] 7. Relocate scripts/generate-web-theme-data.py to mise-tasks/generate-web-theme-data

  **What to do**: `git mv scripts/generate-web-theme-data.py mise-tasks/generate-web-theme-data`, shebang + `chmod +x` + `#MISE description=`. Update `web/package.json`'s five script lines (`build`, `check`, `test:unit`, `themes:generate`, `themes:check` — currently `python3 ../scripts/generate-web-theme-data.py [--check]`) to `python3 ../mise-tasks/generate-web-theme-data [--check]`. Update `web/test/theme-data.test.mjs:19` and `web/test/palette-terminal-mapping.test.mjs:46` (both hardcode `path.join(projectRoot, "scripts", "generate-web-theme-data.py")` per Metis) to the new relative segments.
  **Must NOT do**: Change the `--check` flag's semantics or the generated `web/src/data/generated-theme-data.json` shape.

  **Parallelization**: Can Parallel: YES | Wave 2 | Blocks: [10, 26] | Blocked By: []

  **References**:
  - `web/package.json:11,13,14,15,16` - the five script lines invoking this file by relative path.
  - `web/test/theme-data.test.mjs:19` - hardcoded path (Metis-cited).
  - `web/test/palette-terminal-mapping.test.mjs:46` - hardcoded path (Metis-cited).

  **Acceptance Criteria**:
  - [ ] `aube -C web run themes:check` passes unchanged.
  - [ ] `node --test web/test/theme-data.test.mjs web/test/palette-terminal-mapping.test.mjs` passes.
  - [ ] `git grep -n "scripts/generate-web-theme-data.py"` returns nothing (only `mise-tasks/generate-web-theme-data` remains referenced).

  **QA Scenarios**:
  ```
  Scenario: Web build chain still regenerates and validates theme data
    Tool: bash
    Steps: aube -C web run themes:generate && aube -C web run themes:check
    Expected: both exit 0, generated-theme-data.json unchanged if catalog unchanged
    Evidence: evidence/task-7-themes-check.txt

  Scenario: Node tests find the relocated script
    Tool: bash
    Steps: node --test web/test/theme-data.test.mjs web/test/palette-terminal-mapping.test.mjs
    Expected: all pass; reverting just the two test-file path updates (leaving package.json updated) makes them fail with ENOENT, proving they genuinely exercise the path
    Evidence: evidence/task-7-node-test-path-proof.txt
  ```

  **Commit**: YES | Message: `chore(mise-tasks): relocate generate-web-theme-data to a mise file task` | Files: [mise-tasks/generate-web-theme-data, web/package.json, web/test/theme-data.test.mjs, web/test/palette-terminal-mapping.test.mjs]

- [x] 8. Relocate tests/test_cli.sh to mise-tasks/test-cli

  **What to do**: `git mv tests/test_cli.sh mise-tasks/test-cli`, keep the `#!/bin/sh` shebang, add a `#MISE description=` line, `chmod +x`. **Keep it flat at `mise-tasks/test-cli` — do not use subdirectory grouping** (e.g. `mise-tasks/test/cli`): the script's own root-resolution (`cd -- "$(dirname -- "$0")/.."`, line 5) assumes exactly one directory level under repo root, and subdirectory grouping would silently resolve to the wrong root (Metis-verified risk). Update every reference calling it a "contract": `src/sf2_theme/AGENTS.md:54`, `src/sf2_theme/adapters/AGENTS.md:89`, `themes/AGENTS.md:58` (Metis-cited lines — all describe it as "the installed-consumer contract," update the path, keep the framing), root `AGENTS.md`'s command list, and `.github/workflows/verify.yml`'s `shellcheck tests/test_cli.sh` + `tests/test_cli.sh` steps (superseded by Wave 3/5, but note the rename here so Wave 5 doesn't hunt for it).
  **Must NOT do**: Do not change the script's own repo-root-relative path resolution logic — only its own location moves, not how it finds the repo root.

  **Parallelization**: Can Parallel: YES | Wave 2 | Blocks: [10, 26] | Blocked By: []

  **References**:
  - `tests/test_cli.sh:5` - `cd -- "$(dirname -- "$0")/.."` root resolution, the exact line that forbids subdirectory grouping here.
  - `src/sf2_theme/AGENTS.md:54`, `src/sf2_theme/adapters/AGENTS.md:89`, `themes/AGENTS.md:58` - "installed-consumer contract" mentions (Metis-cited).
  - `.github/workflows/verify.yml:102-105` - `shellcheck tests/test_cli.sh` / `tests/test_cli.sh` steps.

  **Acceptance Criteria**:
  - [ ] `shellcheck mise-tasks/test-cli` passes with the same findings (zero, per current green CI) as before the move.
  - [ ] `mise-tasks/test-cli` run directly (`./mise-tasks/test-cli`) from repo root passes identically to the pre-move script.
  - [ ] `git grep -n "tests/test_cli.sh"` returns nothing.

  **QA Scenarios**:
  ```
  Scenario: Relocated script still resolves repo root correctly
    Tool: bash
    Steps: cd /tmp && /path/to/repo/mise-tasks/test-cli
    Expected: exits 0, exercises every adapter exactly as it did at tests/test_cli.sh (compare full stdout against a pre-move capture)
    Evidence: evidence/task-8-test-cli-run.txt

  Scenario: shellcheck stays clean at the new path
    Tool: bash
    Steps: shellcheck mise-tasks/test-cli
    Expected: no warnings/errors, same as `shellcheck tests/test_cli.sh` reported pre-move
    Evidence: evidence/task-8-shellcheck.txt
  ```

  **Commit**: YES | Message: `chore(mise-tasks): relocate test_cli.sh to a flat mise file task` | Files: [mise-tasks/test-cli, src/sf2_theme/AGENTS.md, src/sf2_theme/adapters/AGENTS.md, themes/AGENTS.md, AGENTS.md]

- [x] 9. Relocate scripts/build-standalone.py to mise-tasks/build-standalone and fix its self-referencing embedded string

  **What to do**: `git mv scripts/build-standalone.py mise-tasks/build-standalone`, shebang + `chmod +x` + `#MISE description=`. This script embeds its own path into the generated output (`scripts/build-standalone.py:52`: `# Generated by scripts/build-standalone.py. Do not edit by hand.`) — update that literal string to `# Generated by mise-tasks/build-standalone. Do not edit by hand.`, then **regenerate the committed root `sf2-themes` executable in this same commit** (`mise run build-standalone` once the task exists — for this task, invoke the relocated file directly: `./mise-tasks/build-standalone`), or `verify.yml`'s freshness check (`git diff --exit-code -- sf2-themes`) fails on the next CI run over an unrelated PR. Update doc mentions: `AGENTS.md:40,87` ("Standalone generation" table row, `python3 scripts/build-standalone.py` command), `src/sf2_theme/AGENTS.md:21` ("Standalone mirror" table row).
  **Must NOT do**: Do not hand-edit the generated `sf2-themes` executable itself beyond what regeneration produces (per the repo's own existing AGENTS.md rule).

  **Parallelization**: Can Parallel: YES | Wave 2 | Blocks: [10, 26] | Blocked By: []

  **References**:
  - `scripts/build-standalone.py:52` - the exact embedded self-referencing string to update.
  - `AGENTS.md:40,87`, `src/sf2_theme/AGENTS.md:21` - doc table rows and command mentions.
  - `.github/workflows/verify.yml:96-99` - the freshness-check step this must not break (`python scripts/build-standalone.py && git diff --exit-code -- sf2-themes`, renamed in Wave 5).

  **Acceptance Criteria**:
  - [ ] Committed root `sf2-themes` file's header comment reads `# Generated by mise-tasks/build-standalone. Do not edit by hand.`.
  - [ ] `./mise-tasks/build-standalone && git diff --exit-code -- sf2-themes` exits 0 (freshness check passes with zero diff, i.e. this task's own regeneration is already the fresh output).
  - [ ] `git grep -n "scripts/build-standalone.py"` returns nothing.

  **QA Scenarios**:
  ```
  Scenario: Regeneration matches what's committed
    Tool: bash
    Steps: ./mise-tasks/build-standalone; git diff --exit-code -- sf2-themes
    Expected: exit 0, no diff
    Evidence: evidence/task-9-freshness.txt

  Scenario: Embedded self-reference string is correct
    Tool: bash
    Steps: head -5 sf2-themes | grep "Generated by"
    Expected: line reads "# Generated by mise-tasks/build-standalone. Do not edit by hand."
    Evidence: evidence/task-9-embedded-string.txt
  ```

  **Commit**: YES | Message: `chore(mise-tasks): relocate build-standalone to a mise file task, regenerate sf2-themes` | Files: [mise-tasks/build-standalone, sf2-themes, AGENTS.md, src/sf2_theme/AGENTS.md]

- [x] 10. Define mise.toml tasks: lint, shellcheck, validate-catalog, standalone-freshness

  **What to do**: Replace `[tasks.test]`'s current shim body (`run = ["uv run --with pytest pytest -q", "bash tests/test_cli.sh"]`) with `depends = ["pytest", "test-cli"]` plus a new `[tasks.pytest]` (`run = "uv sync --all-extras && uv run pytest -q"`, per task 2) and `[tasks.test-cli]` (`run = "./mise-tasks/test-cli"`, thin but pointing at the now-canonical file task rather than a `tests/` script — this is not a shim, it's mise's own task-to-file-task delegation, the intended end state). Add `[tasks.lint]` (`run = "ruff check src tests mise-tasks"`, per task 3's config), `[tasks.shellcheck]` (`run = "shellcheck mise-tasks/test-cli"` — the only shell file task), `[tasks."validate-catalog"]` (`run = "uv run --project . python -m sf2_theme validate --all"`), `[tasks."standalone-freshness"]` (`depends = ["build-standalone"]`, `run = "git diff --exit-code -- sf2-themes"`), and `[tasks."build-standalone"]` (`run = "./mise-tasks/build-standalone"`).
  **Must NOT do**: Do not make `[tasks.test]` (the composite) itself call `bash`/`python3` against a `mise-tasks/` file directly with a full relative path duplicated in two places — use `depends` so mise's own dependency graph is the single source of task-to-task wiring.

  **Parallelization**: Can Parallel: NO | Wave 3 | Blocks: [12, 13, 17] | Blocked By: [1, 2, 3, 6, 7, 8, 9]

  **References**:
  - `mise.toml:7-9` - current `[tasks.test]` shim being replaced.
  - `.github/workflows/verify.yml:80-99` - the exact CI steps (`pytest`, `validate --all`, `build-standalone.py` + diff, `ruff check`, `shellcheck`, `test_cli.sh`) each task here mirrors 1:1, so Wave 5 can shrink to `mise run <task>` calls.
  - mise `depends` field (confirmed via `mise help run` this session) - dependency-graph wiring between TOML tasks and file tasks.

  **Acceptance Criteria**:
  - [ ] `mise run test` still runs pytest + test-cli (same effective coverage as before this task, now via `depends` instead of an inline `run` array).
  - [ ] `mise run lint`, `mise run shellcheck`, `mise run validate-catalog`, `mise run standalone-freshness`, `mise run build-standalone` each exist and each does exactly what its corresponding CI step did.
  - [ ] `mise tasks ls --name-only` includes every new task name (write this criterion as "includes," never as an exact count — this repo's global mise config surfaces unrelated `macos:*`/`reverb:*`/`samples:*` tasks too, Metis-verified).

  **QA Scenarios**:
  ```
  Scenario: Each new task matches its old CI-step behavior
    Tool: bash
    Steps: mise run lint; mise run shellcheck; mise run validate-catalog; mise run standalone-freshness
    Expected: all four exit 0 on the current clean tree (same as the equivalent verify.yml steps did before this change)
    Evidence: evidence/task-10-new-tasks.txt

  Scenario: standalone-freshness catches real drift
    Tool: bash
    Steps: touch themes/characters/ryu.toml (no-op edit, bump mtime only won't trigger; instead append a harmless comment line then revert), or more reliably: hand-edit the committed sf2-themes file with one extra blank line, then run `mise run standalone-freshness`
    Expected: non-zero exit (git diff finds a difference); revert the manual edit after
    Evidence: evidence/task-10-freshness-catches-drift.txt
  ```

  **Commit**: YES | Message: `feat(mise): define lint, shellcheck, validate-catalog, and standalone-freshness tasks` | Files: [mise.toml]

- [x] 11. Add web:dev:container mise task

  **What to do**: Add `[tasks."web:dev:container"]` (`run = "aube -C web run dev -- --host 0.0.0.0 --port 4322"`) as a distinct task from the existing `web:dev` (which stays `--host 127.0.0.1`, unpublished-port, for bare-host local dev). Port 4322 is deliberately different from Playwright's default `PLAYWRIGHT_PORT` (4321, `web/playwright.config.mjs:3`) to avoid a collision if both an e2e run and a long-running dev container happen to run concurrently (Metis-flagged risk).
  **Must NOT do**: Do not change `web:dev`'s existing host/port — bare-host local dev is unaffected by this migration.

  **Parallelization**: Can Parallel: YES | Wave 3 | Blocks: [17] | Blocked By: [10]

  **References**:
  - `mise.toml` (current `[tasks."web:dev"]`) - `--host 127.0.0.1`, the task this one is a container-safe sibling of.
  - `web/playwright.config.mjs:3` - `PLAYWRIGHT_PORT` default 4321, the collision this task's port choice avoids.

  **Acceptance Criteria**:
  - [ ] `mise run web:dev:container` binds `0.0.0.0:4322` (verifiable via `curl -sSf http://127.0.0.1:4322/sf2-themes/` once the wrapper in task 17 publishes that port from the container).
  - [ ] `web:dev` (unsuffixed) is unchanged.

  **QA Scenarios**:
  ```
  Scenario: Container-safe dev task binds all interfaces
    Tool: bash
    Steps: mise run web:dev:container & sleep 3; curl -sSf http://127.0.0.1:4322/sf2-themes/ -o /dev/null -w "%{http_code}\n"; kill %1
    Expected: prints 200
    Evidence: evidence/task-11-dev-container-curl.txt

  Scenario: Bare-host dev task unaffected
    Tool: bash
    Steps: grep -A1 '"web:dev"\]' mise.toml
    Expected: still shows --host 127.0.0.1, no port argument added
    Evidence: evidence/task-11-web-dev-unchanged.txt
  ```

  **Commit**: YES | Message: `feat(mise): add web:dev:container task for the long-running container dev server` | Files: [mise.toml]

- [x] 12. Create docker/toolchain/Dockerfile (mise + Python + Node + aube, no app source)

  **What to do**: Create `docker/toolchain/Dockerfile` `FROM ubuntu:24.04` (pin by digest). Install `docker/toolchain/apt-packages.txt` (start minimal: `git curl ca-certificates build-essential python3` — Playwright's OS deps move to the `dev` image in task 13, since the toolchain has no app source to know it needs a browser yet). Install mise via a pinned version (`ARG MISE_VERSION=<latest stable at implementation time>`, official installer: `curl https://mise.run | sh` with `MISE_VERSION` env set, simpler than a bespoke asset-JSON bootstrap since this repo has no air-gapped-build constraint niceuptime's Cursor-Cloud note was working around). `COPY mise.toml mise.lock /opt/sf2-themes/toolchain/`, `WORKDIR /opt/sf2-themes/toolchain`, `RUN mise trust --yes . && mise install --locked`. Add `LABEL org.opencontainers.image.source="https://github.com/douglasjarquin/sf2-themes"`. No app source, no `dev` user (this image is a pure cache layer, always run as whatever the layering image sets).
  **Must NOT do**: Do not COPY any application source (`src/`, `web/`, `themes/`) into this image — that's what makes it reusable across future repos in the "suite," and what keeps its build-cache stable regardless of app changes.

  **Parallelization**: Can Parallel: YES | Wave 4 | Blocks: [13, 16] | Blocked By: [10]

  **References**:
  - `~/github/cofactor/niceuptime/docker/toolchain/Dockerfile` (inspiration, not verbatim) - mise-bootstrap-then-`mise install --locked` pattern, `LABEL` convention.
  - `mise.toml`, `mise.lock` (task 1) - the pinned source of truth this image installs from.
  - Ubuntu 24.04 digest - resolve current `ubuntu:24.04` digest at implementation time via `docker pull ubuntu:24.04 && docker inspect --format='{{index .RepoDigests 0}}' ubuntu:24.04`.

  **Acceptance Criteria**:
  - [ ] `docker build -f docker/toolchain/Dockerfile -t sf2-themes-toolchain:test .` succeeds for both `--platform linux/amd64` and `--platform linux/arm64`.
  - [ ] `docker run --rm sf2-themes-toolchain:test mise exec -- python --version` prints `3.11.x`.
  - [ ] `docker run --rm sf2-themes-toolchain:test mise exec -- node --version` prints `v24.x`.
  - [ ] `docker run --rm sf2-themes-toolchain:test mise exec -- aube --version`, `... ruff --version`, `... shellcheck --version` all resolve.
  - [ ] Image contains no `src/`, `web/`, or `themes/` directory (`docker run --rm sf2-themes-toolchain:test test -d /workspace && exit 1 || exit 0`).

  **QA Scenarios**:
  ```
  Scenario: Toolchain image has every pinned tool, both architectures
    Tool: bash
    Steps: for arch in amd64 arm64; do docker buildx build --platform linux/$arch -f docker/toolchain/Dockerfile -t sf2-themes-toolchain:test-$arch --load .; docker run --rm sf2-themes-toolchain:test-$arch sh -c "mise exec -- python --version && mise exec -- node --version && mise exec -- aube --version && mise exec -- ruff --version && mise exec -- shellcheck --version"; done
    Expected: both architectures build and print all five versions with no error
    Evidence: evidence/task-12-toolchain-multiarch.txt

  Scenario: No app source leaked into the image
    Tool: bash
    Steps: docker run --rm sf2-themes-toolchain:test-amd64 find / -maxdepth 3 -iname "sf2_theme" -o -iname "astro.config.mjs" 2>/dev/null
    Expected: no output
    Evidence: evidence/task-12-no-app-source.txt
  ```

  **Commit**: YES | Message: `feat(docker): add the toolchain image (mise + Python + Node + aube, no app source)` | Files: [docker/toolchain/Dockerfile, docker/toolchain/apt-packages.txt]

- [x] 13. Create docker/dev/Dockerfile (FROM toolchain, app deps prefetched, non-root user)

  **What to do**: `FROM sf2-themes-toolchain:<pinned-fingerprint-or-tag-arg>` (via `ARG TOOLCHAIN_IMAGE`, set by the build wrapper in task 16). Install `docker/dev/apt-packages.txt` (Playwright Chromium's OS libs — start from Playwright's own documented Ubuntu 24.04 dependency list, verified at implementation time via `npx playwright install-deps --dry-run` output or Playwright's own docs, since this list drifts by Playwright version) **as root, before creating the `dev` user**. Create a non-root `dev` user. `COPY --chown=dev:dev pyproject.toml uv.lock web/package.json web/package-lock.json ./` (manifests only, for layer-cache locality), then as `dev`: `mise exec -- uv sync --all-extras` and `mise exec -- aube -C web install` and `mise exec -- aube -C web exec -- playwright install chromium` (Chromium binary itself can install as `dev`; only its OS-level deps needed root, already handled above). Set `ENV PLAYWRIGHT_BROWSERS_PATH=/home/dev/.cache/ms-playwright` (or wherever `aube`/Playwright resolves by default — confirm at implementation time and pin explicitly so the fingerprint and the runtime env agree). `WORKDIR /workspace/sf2-themes`. `ENTRYPOINT ["docker/dev/entrypoint.sh"]`.
  **Must NOT do**: Do not `COPY` the full repo source into this image — only manifests, for cache locality; the full source arrives via the bind mount at container-run time (task 17).

  **Parallelization**: Can Parallel: NO | Wave 4 | Blocks: [14, 16, 17] | Blocked By: [10, 12]

  **References**:
  - `~/github/cofactor/niceuptime/docker/dev/Dockerfile` (inspiration) - manifest-first COPY for cache locality, root-then-nonroot ordering for OS deps vs. app deps, `chown` pattern.
  - `pyproject.toml`, `uv.lock`, `web/package.json`, `web/package-lock.json` - the exact manifest files to COPY first (also this image's fingerprint inputs, task 15).
  - Playwright's Ubuntu 24.04 dependency list - resolve exact package names at implementation time; do not hand-guess from an older Ubuntu version's list.

  **Acceptance Criteria**:
  - [ ] `docker build -f docker/dev/Dockerfile --build-arg TOOLCHAIN_IMAGE=sf2-themes-toolchain:test .` succeeds.
  - [ ] `docker run --rm <dev-image> mise exec -- uv run pytest -q --collect-only` succeeds without re-downloading any dependency (deps already baked).
  - [ ] `docker run --rm <dev-image> mise exec -- aube -C web exec -- playwright --version` succeeds and a Chromium launch smoke-test (`playwright install --dry-run` or a one-line launch/close script) passes.
  - [ ] `docker run --rm <dev-image> whoami` prints `dev`, not `root`.

  **QA Scenarios**:
  ```
  Scenario: Deps are baked, not fetched at run time
    Tool: bash
    Steps: docker run --rm --network none <dev-image> mise exec -- uv run pytest -q --collect-only
    Expected: succeeds with networking fully disabled, proving pytest/its deps are already present in the image
    Evidence: evidence/task-13-offline-collect.txt

  Scenario: Non-root user, correct ownership
    Tool: bash
    Steps: docker run --rm <dev-image> sh -c "whoami && ls -la /workspace"
    Expected: whoami prints dev; /workspace (once mounted in task 17) is writable by dev
    Evidence: evidence/task-13-nonroot.txt
  ```

  **Commit**: YES | Message: `feat(docker): add the dev image (toolchain + prefetched app deps, non-root)` | Files: [docker/dev/Dockerfile, docker/dev/apt-packages.txt]

- [x] 14. Create docker/dev/entrypoint.sh

  **What to do**: A short entrypoint that: (1) activates mise (`eval "$(mise activate bash)"` or ensures `MISE_DATA_DIR`'s shims are on `PATH` — required because `web/package.json`'s scripts call bare `python3`, which must resolve to mise's pinned Python inside the container, Metis-flagged); (2) if a mounted target directory the dev image baked deps into (e.g. `web/node_modules`) is empty (first run against a fresh named volume, task 17), copies the image's original baked copy in before proceeding (Metis-flagged volume-masking risk — the seed step, not the prefetch itself, is what makes prefetching load-bearing rather than decorative); (3) `exec "$@"`.
  **Must NOT do**: Do not silently swallow a seed-copy failure — if the baked source directory is itself missing or empty, fail loudly rather than proceeding with a broken environment.

  **Parallelization**: Can Parallel: YES | Wave 4 | Blocks: [17] | Blocked By: [13]

  **References**:
  - Metis finding (this session) - "an empty named volume mounted over an image path masks the baked content on first run... does the wrapper seed the volumes from the image, or is the prefetch decorative?"
  - `web/package.json:11,13,14,15,16` - the bare `python3` calls that require mise's shims already on PATH by the time these run.

  **Acceptance Criteria**:
  - [ ] First run against a brand-new empty named volume at `web/node_modules` results in the volume containing the image's baked `node_modules`, not an empty directory.
  - [ ] `python3 --version` inside the entrypoint's exec'd command resolves to the mise-pinned 3.11, not any system Python.

  **QA Scenarios**:
  ```
  Scenario: Fresh volume gets seeded from the image
    Tool: bash
    Steps: docker volume create test-seed-vol; docker run --rm --mount type=volume,src=test-seed-vol,dst=/workspace/sf2-themes/web/node_modules <dev-image> ls /workspace/sf2-themes/web/node_modules | head -3; docker volume rm test-seed-vol
    Expected: lists real package directories (astro, playwright, etc.), not empty
    Evidence: evidence/task-14-volume-seed.txt

  Scenario: Bare python3 resolves to mise's pinned version inside the container
    Tool: bash
    Steps: docker run --rm <dev-image> python3 --version
    Expected: prints Python 3.11.x
    Evidence: evidence/task-14-python-resolution.txt
  ```

  **Commit**: YES | Message: `feat(docker): add dev image entrypoint with mise activation and volume seeding` | Files: [docker/dev/entrypoint.sh]

- [x] 15. Create scripts/ci/dev-image-fingerprint.sh

  **What to do**: A script printing one deterministic hash to stdout, computed from the concatenated contents of: toolchain fingerprint inputs (`mise.toml`, `mise.lock`, `docker/toolchain/Dockerfile`, `docker/toolchain/apt-packages.txt`) for the toolchain image, and additionally `pyproject.toml`, `uv.lock`, `web/package.json`, `web/package-lock.json`, `docker/dev/Dockerfile`, `docker/dev/apt-packages.txt`, `docker/dev/entrypoint.sh` for the dev image (Metis-flagged: `web/package-lock.json` must be included since Playwright's browser binary is version-coupled to it). Accept a `--target toolchain|dev` flag selecting which input set to hash. Use `sha256sum` over the sorted, concatenated file list (stable ordering) and print the first 12 hex characters.
  **Must NOT do**: Do not include the full repo tree or any app source in the hash — only the named manifest/Dockerfile inputs, or every commit would invalidate the cache.

  **Parallelization**: Can Parallel: YES | Wave 4 | Blocks: [16] | Blocked By: [12, 13]

  **References**:
  - `~/github/cofactor/niceuptime/scripts/ci/ensure-dev-image.sh` (inspiration) - calls a sibling `dev-image-fingerprint.sh`, tags images `<prefix>:sha-<fingerprint>`.
  - Metis finding - exact fingerprint input list, including `web/package-lock.json` for Playwright version coupling.

  **Acceptance Criteria**:
  - [ ] `scripts/ci/dev-image-fingerprint.sh --target toolchain` and `--target dev` each print a stable 12-char hex string, unchanged across repeated invocations with no input changes.
  - [ ] Editing any one input file (e.g. bumping a version in `mise.toml`) changes the toolchain fingerprint; editing only `web/package-lock.json` changes the dev fingerprint but not the toolchain one.

  **QA Scenarios**:
  ```
  Scenario: Fingerprint is stable and input-sensitive
    Tool: bash
    Steps: a=$(scripts/ci/dev-image-fingerprint.sh --target toolchain); b=$(scripts/ci/dev-image-fingerprint.sh --target toolchain); echo "$a $b"; sed -i '' 's/aube = "2.1"/aube = "2.1"  # touch/' mise.toml; c=$(scripts/ci/dev-image-fingerprint.sh --target toolchain); git checkout mise.toml; echo "$c"
    Expected: $a == $b (stable), $c != $a (input-sensitive)
    Evidence: evidence/task-15-fingerprint-stability.txt

  Scenario: Dev-only input does not perturb toolchain fingerprint
    Tool: bash
    Steps: t1=$(scripts/ci/dev-image-fingerprint.sh --target toolchain); echo '{}' >> web/package-lock.json; t2=$(scripts/ci/dev-image-fingerprint.sh --target toolchain); git checkout web/package-lock.json
    Expected: t1 == t2
    Evidence: evidence/task-15-fingerprint-scope.txt
  ```

  **Commit**: YES | Message: `feat(ci): add docker image fingerprinting script` | Files: [scripts/ci/dev-image-fingerprint.sh]

- [x] 16. Create scripts/ci/ensure-toolchain-image.sh and scripts/ci/ensure-dev-image.sh

  **What to do**: Two scripts (near-identical, parameterized by target) that: compute the fingerprint (task 15), check `docker manifest inspect ghcr.io/douglasjarquin/sf2-themes-<target>:sha-<fingerprint>` (or a local `docker image inspect` first, to skip a network round-trip when already pulled); if present, `docker pull` (or confirm local) and print the resolved tag to stdout; if absent, build locally via `docker buildx build --platform linux/amd64,linux/arm64 --cache-from type=registry,ref=ghcr.io/douglasjarquin/sf2-themes-<target>:buildcache -f docker/<target>/Dockerfile -t <tag> .` (for `dev`, pass `--build-arg TOOLCHAIN_IMAGE=$(scripts/ci/ensure-toolchain-image.sh)` so the dev build always layers on an already-resolved toolchain tag) and print the freshly-built tag. This is the **cold-start safety net**: it works correctly even before `publish-images.yml` (task 19) has ever run, resolving the chicken-and-egg problem Metis raised — CI never hard-fails just because nothing's been published yet, it just builds locally that one time.
  **Must NOT do**: Do not `--push` from these scripts — publishing is `publish-images.yml`'s job (task 19) only, keeping "resolve/build a usable local tag" and "publish for others to reuse" as separate concerns.

  **Parallelization**: Can Parallel: NO | Wave 4 | Blocks: [17, 19] | Blocked By: [12, 13, 15]

  **References**:
  - `~/github/cofactor/niceuptime/scripts/ci/ensure-dev-image.sh` (inspiration) - build-or-reuse-by-fingerprint, one retry if the fingerprint mutates mid-build, `--no-cache` flag support.
  - Metis finding - "no workflow can pull an image that does not exist... needs a documented local buildx --push bootstrap or build-locally-and-proceed fallback" — this task IS that fallback.

  **Acceptance Criteria**:
  - [ ] With no published image and no local cache, `scripts/ci/ensure-toolchain-image.sh` builds one and prints a valid, `docker image inspect`-able tag.
  - [ ] With a matching local image already present, the script skips rebuilding (verify via a timestamp/build-log check that no `docker build` ran).
  - [ ] `scripts/ci/ensure-dev-image.sh` correctly chains through `ensure-toolchain-image.sh` first.

  **QA Scenarios**:
  ```
  Scenario: Cold start builds locally with no published image
    Tool: bash
    Steps: docker rmi -f $(docker images 'sf2-themes-toolchain' -q) 2>/dev/null; time scripts/ci/ensure-toolchain-image.sh
    Expected: exits 0, prints a tag, docker image inspect <tag> succeeds
    Evidence: evidence/task-16-cold-start.txt

  Scenario: Warm reuse skips the build
    Tool: bash
    Steps: tag1=$(scripts/ci/ensure-toolchain-image.sh); tag2=$(scripts/ci/ensure-toolchain-image.sh 2>build.log); grep -q "building image" build.log && echo REBUILT || echo REUSED
    Expected: prints REUSED (second call found the existing tag and did not rebuild), tag1 == tag2
    Evidence: evidence/task-16-warm-reuse.txt
  ```

  **Commit**: YES | Message: `feat(ci): add fingerprinted build-or-reuse scripts for both images` | Files: [scripts/ci/ensure-toolchain-image.sh, scripts/ci/ensure-dev-image.sh]

- [x] 17. Create scripts/ci/run-in-dev-container.sh

  **What to do**: Wrapper taking a command (`mise run <task>` or arbitrary passthrough) and running it inside the dev image (via task 16), bind-mounting the repo at `/workspace/sf2-themes`, with **named volumes** for `.venv`, `web/node_modules`, `web/.astro`, `web/dist` so those never touch the host filesystem or a bind mount. **Volume naming/lifetime differs by environment** (Metis-flagged tension between "reuse for local speed" and "no cross-run leakage in CI"): when `GITHUB_RUN_ID`/`GITHUB_JOB` are set, name volumes `sf2-themes-ci-${GITHUB_RUN_ID}-${GITHUB_JOB}-{purpose}` (ephemeral, one run's worth, GC'd by task 18); otherwise (local), name them fixed `sf2-themes-dev-{purpose}` (persistent, reused across local invocations for speed — local dev isn't the shared-runner-pollution problem this whole effort targets). Forward `CI` (if set) and `PLAYWRIGHT_PORT` into the container. Publish a port when invoked with a `--publish HOST:CONTAINER` flag (used only for the `web:dev:container` long-running case, task 11). Do the root-then-drop-privileges dance from niceuptime's script (start as root, create a passwd/group entry for the host's real `id -u`/`id -g`, `chown` the mount points, then `setpriv --reuid=$HOST_UID --regid=$HOST_GID` before `exec`ing the real command) so bind-mounted files stay host-owned on both the Linux CI runner and a macOS/OrbStack dev machine.
  **Must NOT do**: Do not recursively `chown` the entire bind-mounted repo or a populated `node_modules` volume — only the mount-point directories themselves (niceuptime's own documented reason: volume init copies the image's owner onto the mount-point dir only, recursing would walk the whole dependency tree needlessly).

  **Parallelization**: Can Parallel: NO | Wave 4 | Blocks: [20, 21, 23] | Blocked By: [10, 11, 13, 14, 16]

  **References**:
  - `~/github/cofactor/niceuptime/scripts/ci/run-in-dev-container.sh` (inspiration, full mechanism studied this session) - named-volume mounts, root-then-setpriv-drop UID/GID fixup, env-var passthrough allowlist, `--shell` mode.
  - Metis findings - job-scoped-vs-local volume naming tension (resolved above), `CI` env var must be forwarded or Playwright's `reuseExistingServer: !process.env.CI` (`web/playwright.config.mjs:14`) misbehaves in CI.

  **Acceptance Criteria**:
  - [ ] Running `scripts/ci/run-in-dev-container.sh mise run pytest` from the host produces a file (any file the test suite writes, e.g. a coverage artifact) owned by the invoking host user, not root.
  - [ ] Two consecutive local invocations reuse the same named volumes (verified via `docker volume ls` showing no new volume names between the two runs).
  - [ ] Simulating CI env vars (`GITHUB_RUN_ID=123 GITHUB_JOB=web`) produces volumes named with that run/job, distinct from a second simulated run with different values.
  - [ ] `--publish 4322:4322` mode makes `mise run web:dev:container` reachable from the host.

  **QA Scenarios**:
  ```
  Scenario: Host-owned files after a container write
    Tool: bash
    Steps: scripts/ci/run-in-dev-container.sh sh -c "touch /workspace/sf2-themes/.tmp-ownership-check"; stat -f "%Su" .tmp-ownership-check; rm .tmp-ownership-check
    Expected: prints the invoking host username, not root
    Evidence: evidence/task-17-ownership.txt

  Scenario: CI-scoped volumes are distinct per run/job and local volumes are stable
    Tool: bash
    Steps: GITHUB_RUN_ID=111 GITHUB_JOB=test scripts/ci/run-in-dev-container.sh true; GITHUB_RUN_ID=222 GITHUB_JOB=test scripts/ci/run-in-dev-container.sh true; docker volume ls --filter name=sf2-themes-ci- ; scripts/ci/run-in-dev-container.sh true; scripts/ci/run-in-dev-container.sh true; docker volume ls --filter name=sf2-themes-dev-
    Expected: two distinct sf2-themes-ci-111-test-* and sf2-themes-ci-222-test-* volume sets; exactly one sf2-themes-dev-* set regardless of repeated local calls
    Evidence: evidence/task-17-volume-scoping.txt
  ```

  **Commit**: YES | Message: `feat(ci): add run-in-dev-container wrapper with job-scoped volumes and UID fixup` | Files: [scripts/ci/run-in-dev-container.sh]

- [x] 18. Create scripts/ci/gc-job-overlay-volumes.sh

  **What to do**: Given `GITHUB_RUN_ID`/`GITHUB_JOB` (required — no-op with a clear stderr message if either is unset, matching niceuptime's own script exactly, since this only makes sense in CI), `docker volume ls -q | grep '^sf2-themes-ci-${GITHUB_RUN_ID}-${GITHUB_JOB}-'` and `docker volume rm` each match. A removal failure logs to stderr but does not fail the script (this runs in an `if: always()` CI step — it must never be the thing that turns a green run red).
  **Must NOT do**: Do not touch `sf2-themes-dev-*` (local) volumes — this script only ever targets the CI-scoped naming pattern.

  **Parallelization**: Can Parallel: YES | Wave 4 | Blocks: [20] | Blocked By: []

  **References**:
  - `~/github/cofactor/niceuptime/scripts/ci/gc-job-overlay-volumes.sh` (near-verbatim pattern is appropriate here — this script's whole job is generic cleanup, not app-specific).

  **Acceptance Criteria**:
  - [ ] After a simulated CI run (task 17's second QA scenario), running this script with the same `GITHUB_RUN_ID`/`GITHUB_JOB` removes exactly those volumes.
  - [ ] Running it with `GITHUB_RUN_ID`/`GITHUB_JOB` unset exits 0 with a stderr note, removing nothing.
  - [ ] `sf2-themes-dev-*` local volumes survive a GC run untouched.

  **QA Scenarios**:
  ```
  Scenario: GC removes exactly this job's volumes, nothing else
    Tool: bash
    Steps: GITHUB_RUN_ID=333 GITHUB_JOB=test scripts/ci/run-in-dev-container.sh true; scripts/ci/run-in-dev-container.sh true; GITHUB_RUN_ID=333 GITHUB_JOB=test scripts/ci/gc-job-overlay-volumes.sh; docker volume ls --filter name=sf2-themes-ci-333-test; docker volume ls --filter name=sf2-themes-dev-
    Expected: the sf2-themes-ci-333-test-* filter returns empty; the sf2-themes-dev-* filter still shows volumes
    Evidence: evidence/task-18-gc-scoped.txt

  Scenario: Missing env vars is a safe no-op
    Tool: bash
    Steps: unset GITHUB_RUN_ID GITHUB_JOB; scripts/ci/gc-job-overlay-volumes.sh; echo "exit: $?"
    Expected: exit 0, stderr explains why it skipped
    Evidence: evidence/task-18-noop.txt
  ```

  **Commit**: YES | Message: `feat(ci): add job-scoped volume garbage collection` | Files: [scripts/ci/gc-job-overlay-volumes.sh]

- [x] 19. Create .github/workflows/publish-images.yml

  **What to do**: New workflow, `permissions: { packages: write, contents: read }`, triggered on `push` to `main` touching `docker/**`, `mise.toml`, `mise.lock`, `pyproject.toml`, `uv.lock`, `web/package.json`, `web/package-lock.json`, plus `workflow_dispatch` for manual runs. Steps: `docker/setup-buildx-action`, `docker/login-action` against `ghcr.io` using `GITHUB_TOKEN`, then `docker buildx build --platform linux/amd64,linux/arm64 --push --cache-to type=registry,ref=ghcr.io/douglasjarquin/sf2-themes-toolchain:buildcache,mode=max --cache-from type=registry,ref=ghcr.io/douglasjarquin/sf2-themes-toolchain:buildcache -t ghcr.io/douglasjarquin/sf2-themes-toolchain:sha-$(scripts/ci/dev-image-fingerprint.sh --target toolchain) -t ghcr.io/douglasjarquin/sf2-themes-toolchain:latest .` for the toolchain image, then the equivalent for `dev` (with `--build-arg TOOLCHAIN_IMAGE=ghcr.io/douglasjarquin/sf2-themes-toolchain:latest`). This is the mechanism that keeps CI (task 20/21) fast after the first run of this workflow — task 16's local-build fallback covers the cold-start gap before this has ever run once.
  **Must NOT do**: Do not make `verify.yml`/`deploy.yml` depend on this workflow having already run — they must work standalone via task 16's fallback (this is a speed optimization, not a hard dependency).

  **Parallelization**: Can Parallel: NO | Wave 5 | Blocks: [] | Blocked By: [16]

  **References**:
  - `docker/setup-buildx-action`, `docker/login-action` - standard GitHub Actions for multi-arch buildx + GHCR auth.
  - Metis finding - "Docker layer cache on ephemeral runners... needs `--cache-from type=registry` (or GHA cache) plus a stated cold-start time budget, or CI gets slower, not faster" — this task's `--cache-to`/`--cache-from` registry buildcache is that mechanism.

  **Acceptance Criteria**:
  - [ ] A manual `workflow_dispatch` run publishes both `ghcr.io/douglasjarquin/sf2-themes-toolchain:latest` and `-dev:latest`, both pullable.
  - [ ] A second run with no Dockerfile/manifest changes completes in a small fraction of the first run's wall time (registry cache hit).
  - [ ] Both images are `linux/amd64` and `linux/arm64` (`docker manifest inspect` lists both platforms).

  **QA Scenarios**:
  ```
  Scenario: First publish succeeds and is pullable
    Tool: bash (via gh CLI, run against a test branch first)
    Steps: gh workflow run publish-images.yml; gh run watch <run-id>; docker pull ghcr.io/douglasjarquin/sf2-themes-toolchain:latest
    Expected: workflow succeeds, pull succeeds
    Evidence: evidence/task-19-first-publish.txt

  Scenario: Cache makes a no-op rerun fast
    Tool: bash
    Steps: gh workflow run publish-images.yml; note duration; gh workflow run publish-images.yml again with zero relevant file changes; compare durations
    Expected: second run is markedly faster (cache hit on every layer)
    Evidence: evidence/task-19-cache-speedup.txt
  ```

  **Commit**: YES | Message: `feat(ci): add publish-images workflow for toolchain and dev GHCR images` | Files: [.github/workflows/publish-images.yml]

- [x] 20. Restructure .github/workflows/verify.yml around containerized tasks

  **What to do**: Add `mise-tasks/**`, `docker/**`, `scripts/ci/**`, `mise.toml`, `mise.lock` to the `changes` job's path filters (Metis-flagged gap: none of these currently trigger the `test`/`web` jobs). Add `mise.toml` to the existing `python` filter too (not just `web` — Python tasks now also depend on it). **Keep the `test` job's matrix structure exactly as today** (bare `actions/setup-python`, `["3.11","3.12","3.13","3.14"]`, `allow-prereleases: true`) — per the Metis-driven resolution, only simplify its `pip install` line (task 2) and drop the ruff/shellcheck/standalone-freshness/`test_cli.sh` steps out of it. Add a new `toolchain-checks` job (`needs: changes`, `if: needs.changes.outputs.python == 'true'`, `permissions: { packages: read }`) whose steps are `scripts/ci/run-in-dev-container.sh mise run lint`, `... mise run shellcheck`, `... mise run standalone-freshness`, `... mise run validate-catalog`, `... mise run test-cli`, each followed by `scripts/ci/gc-job-overlay-volumes.sh` (`if: always()`). Restructure the `web` job (`permissions: { packages: read }`) to `scripts/ci/run-in-dev-container.sh mise run web:install`, `... mise run web:build`, `... mise run web:check`, `... mise run web:test` (dropping the raw `npm --prefix web ci`/`npx playwright install`/`npm run *` steps — see task 24 for the aube-in-CI verification this depends on), plus the same GC step. Update `gate`'s `needs` to include `toolchain-checks`.
  **Must NOT do**: Do not remove or restructure the `changes` job's job-selector logic beyond adding the new filter paths — its `always()`/fallback-to-true behavior on force-pushes and missing base SHAs must survive unchanged.

  **Parallelization**: Can Parallel: NO | Wave 5 | Blocks: [22, 23, 24] | Blocked By: [2, 17, 18]

  **References**:
  - `.github/workflows/verify.yml` (full file, read this session) - every job/step being restructured.
  - Metis findings - missing path-filter entries, missing `packages: read` permission, `gate.needs` exact-set assertion in the test suite (task 22).

  **Acceptance Criteria**:
  - [ ] A PR touching only `mise-tasks/build-standalone` triggers the `python`-gated jobs (path-filter fix verified).
  - [ ] `test` job's matrix, versions, and `allow-prereleases` are byte-identical to before this task except the simplified `pip install` line.
  - [ ] `toolchain-checks` and `web` jobs both run entirely through `scripts/ci/run-in-dev-container.sh`, with zero raw `npm`/`pip install -e ".[dev]"`/`ruff`/`shellcheck` commands left directly in the YAML.
  - [ ] `gate` fails if `toolchain-checks` fails, exactly as it does for the other required jobs today.

  **QA Scenarios**:
  ```
  Scenario: New job actually gates merges
    Tool: bash (via a real draft PR)
    Steps: open a PR that deliberately fails `mise run lint` (an introduced ruff violation); observe verify.yml
    Expected: toolchain-checks fails, gate fails, PR shows as failing required checks
    Evidence: evidence/task-20-gate-enforcement.txt

  Scenario: Path filter covers the new directories
    Tool: bash (via a real draft PR touching only mise-tasks/generate-previews)
    Steps: open a PR editing only mise-tasks/generate-previews; check which jobs ran
    Expected: toolchain-checks (python-gated) ran; before this task's filter fix it would have been skipped
    Evidence: evidence/task-20-path-filter.txt
  ```

  **Commit**: YES | Message: `ci: containerize toolchain checks and web job, keep the Python matrix untouched` | Files: [.github/workflows/verify.yml]

- [x] 21. Restructure .github/workflows/deploy.yml around the containerized build

  **What to do**: Add `permissions: { packages: read }` to the `build` job. Replace `npm --prefix web ci` + `npm --prefix web run build` with `scripts/ci/run-in-dev-container.sh mise run web:install` + `... mise run web:build`, followed by `scripts/ci/gc-job-overlay-volumes.sh` (`if: always()`). Everything after (`actions/upload-pages-artifact`, the `deploy` job) is unaffected.
  **Must NOT do**: Do not change the `deploy` job or its `if: github.event_name != 'pull_request'` guard.

  **Parallelization**: Can Parallel: NO | Wave 5 | Blocks: [] | Blocked By: [17]

  **References**:
  - `.github/workflows/deploy.yml` (full file, read this session) - the two `npm` lines being replaced.

  **Acceptance Criteria**:
  - [ ] A push to `main` still produces an uploadable `web/dist` Pages artifact via the containerized path.
  - [ ] No `npm`/`npx` string remains in this file.

  **QA Scenarios**:
  ```
  Scenario: Containerized build still produces a deployable artifact
    Tool: bash (via a real push to a test branch with workflow_dispatch, or a draft PR's build job)
    Steps: trigger deploy.yml's build job; inspect the uploaded artifact
    Expected: web/dist present and matches what `mise run web:build` produces locally byte-for-byte on the same commit
    Evidence: evidence/task-21-deploy-build.txt

  Scenario: No npm left in deploy.yml
    Tool: bash
    Steps: grep -n "npm\|npx" .github/workflows/deploy.yml
    Expected: no matches
    Evidence: evidence/task-21-no-npm.txt
  ```

  **Commit**: YES | Message: `ci: containerize the Pages build step` | Files: [.github/workflows/deploy.yml]

- [x] 22. Update tests/test_ci_workflow.py for the restructured workflows

  **What to do**: Update the hardcoded assertions Metis identified: `assert deploy_steps["Install dependencies"] == verify_steps["Install dependencies"] == "npm --prefix web ci"` and the `"Build"` line (`tests/test_ci_workflow.py:53-54`) to assert the new `scripts/ci/run-in-dev-container.sh mise run web:install`/`web:build` step commands instead. Update the `gate["needs"]` exact-set assertion (`tests/test_ci_workflow.py:22`) to include `toolchain-checks`. Audit the rest of the file for any other step-name or command-string assertion touching steps this plan changed (the `test` job's `pip install` line, the removed `ruff`/`shellcheck`/`test_cli.sh` steps that move into `toolchain-checks`).
  **Must NOT do**: Do not weaken these assertions into substring/regex matches to dodge maintenance — keep them exact-string, matching the file's existing style; update them precisely instead.

  **Parallelization**: Can Parallel: NO | Wave 5 | Blocks: [] | Blocked By: [20, 21]

  **References**:
  - `tests/test_ci_workflow.py:22,53-54` (Metis-cited exact lines) - the assertions that fail by construction against tasks 20-21's changes.

  **Acceptance Criteria**:
  - [ ] `uv run pytest tests/test_ci_workflow.py -q` passes against the restructured `verify.yml`/`deploy.yml`.
  - [ ] Every step-name/command assertion in this file corresponds to an actual step in the current workflow files (no stale assertions left pointing at removed steps).

  **QA Scenarios**:
  ```
  Scenario: Updated assertions pass against the real restructured workflows
    Tool: bash
    Steps: uv run pytest tests/test_ci_workflow.py -q -v
    Expected: all tests pass, verbose output shows each assertion's target step name matching the new YAML
    Evidence: evidence/task-22-workflow-tests.txt

  Scenario: A regression is still caught
    Tool: bash
    Steps: temporarily revert one line of verify.yml's toolchain-checks job back to a raw command; rerun the test suite
    Expected: the corresponding updated assertion fails, proving it's load-bearing not vacuous; revert the temporary edit
    Evidence: evidence/task-22-regression-proof.txt
  ```

  **Commit**: YES | Message: `test: update workflow assertions for the containerized CI steps` | Files: [tests/test_ci_workflow.py]

- [x] 23. Verify and, if needed, extend tests/_workflow_yaml.py's hand-rolled parser

  **What to do**: `tests/_workflow_yaml.py` documents itself as supporting "exactly the constructs used in `.github/workflows/*.yml`... not a general-purpose YAML parser" (Metis-cited). Run the full `tests/test_ci_workflow.py` suite against the restructured workflows (tasks 20-21) and fix any parser failure caused by new YAML shapes this plan introduces (e.g. `permissions:` blocks on individual jobs if not already parsed, multi-line `run: |` blocks with embedded `&&` chains from the container wrapper calls). Extend the parser minimally — only the specific constructs actually used, matching its existing minimal-subset philosophy.
  **Must NOT do**: Do not replace this hand-rolled parser with a general YAML library as a "fix" — that's a bigger, unrequested change; extend it narrowly for the constructs this plan actually introduces.

  **Parallelization**: Can Parallel: YES | Wave 5 | Blocks: [] | Blocked By: [20, 21]

  **References**:
  - `tests/_workflow_yaml.py:1-6` (Metis-cited) - the self-documented scope limitation.

  **Acceptance Criteria**:
  - [ ] `uv run pytest tests/test_ci_workflow.py -q` passes with zero parser-level errors (as opposed to legitimate assertion failures) against the new workflow YAML shapes.
  - [ ] Any parser extension made is the minimum needed — verified by diffing `_workflow_yaml.py` before/after this task and confirming every added construct is actually exercised by the restructured workflows.

  **QA Scenarios**:
  ```
  Scenario: Parser handles the new workflow shapes
    Tool: bash
    Steps: uv run pytest tests/test_ci_workflow.py -q
    Expected: no parser-raised exceptions (SyntaxError/KeyError from _workflow_yaml.py itself, distinct from a normal assertion failure)
    Evidence: evidence/task-23-parser-check.txt

  Scenario: Extension is minimal
    Tool: bash
    Steps: git diff tests/_workflow_yaml.py
    Expected: diff touches only the specific construct(s) needed for this plan's new YAML, no speculative generalization
    Evidence: evidence/task-23-minimal-diff.txt
  ```

  **Commit**: YES | Message: `test: extend the workflow YAML test parser for new CI constructs, if needed` | Files: [tests/_workflow_yaml.py]

- [x] 24. Reproduce the aube fresh-runner install failure inside the container; decide the fallback

  **What to do**: `web/AGENTS.md:39` records a specific prior finding: "Aube's virtual-store install fails on a fresh CI runner." This plan's assumption that containerizing fixes it (task 20's `web` job now calls `mise run web:install`, i.e. `aube -C web install`, instead of `npm ci`) is **unverified** (Metis-flagged). Reproduce it directly: run `scripts/ci/run-in-dev-container.sh mise run web:install` against a **brand-new** container instance with **no** prior aube store cache (simulate a truly fresh GitHub-hosted runner: fresh container, empty `AUBE_STORE_DIR`), at least twice in separate fresh instances. If it succeeds both times: containerizing did fix it (the baked toolchain image likely resolves whatever made the bare runner "fresh" in the failing way — e.g. a missing system dependency `aube`'s virtual store needs). If it still fails: keep `npm ci` as the install step specifically **inside the container** for `web:install` in CI (still satisfies "no npm on the host" — npm now only ever runs inside the ephemeral, always-fresh container image, never on the runner's own filesystem — document this explicitly as a scoped exception, not a silent revert).
  **Must NOT do**: Do not assume success without the two-fresh-instance reproduction — this is exactly the kind of unverified claim Metis flagged as a real risk to committing to thrust 2 fully.

  **Parallelization**: Can Parallel: NO | Wave 6 | Blocks: [25] | Blocked By: [20]

  **References**:
  - `web/AGENTS.md:39` (Metis-cited) - the exact prior finding being re-tested.
  - `.github/workflows/verify.yml`'s new `web` job (task 20) - the real invocation being verified.

  **Acceptance Criteria**:
  - [ ] Two independent fresh-container reproduction attempts are documented with their outcome (both pass, or both/either fail with the specific error captured).
  - [ ] If aube fails: `mise.toml`'s `web:install` task (or a CI-only variant) documents the npm fallback and why, and `web/AGENTS.md` is updated (task 25) to reflect the real, re-verified state rather than repeating the old unverified note.
  - [ ] If aube succeeds: `web/AGENTS.md`'s note is updated to say containerizing resolved it, with the reproduction evidence linked.

  **QA Scenarios**:
  ```
  Scenario: Fresh-instance reproduction, attempt 1
    Tool: bash
    Steps: docker volume rm sf2-themes-ci-repro-1-aube-store 2>/dev/null; GITHUB_RUN_ID=repro-1 GITHUB_JOB=aube-test scripts/ci/run-in-dev-container.sh mise run web:install
    Expected: capture exact exit code and full output either way
    Evidence: evidence/task-24-repro-1.txt

  Scenario: Fresh-instance reproduction, attempt 2 (independent volume/run id)
    Tool: bash
    Steps: GITHUB_RUN_ID=repro-2 GITHUB_JOB=aube-test scripts/ci/run-in-dev-container.sh mise run web:install
    Expected: capture exact exit code and full output; compare against attempt 1 for consistency
    Evidence: evidence/task-24-repro-2.txt
  ```

  **Commit**: YES | Message: `ci: verify (or work around) aube install on a fresh container instance` | Files: [mise.toml (only if fallback needed), .github/workflows/verify.yml (only if fallback needed)]

- [x] 25. Update web/AGENTS.md's npm-in-CI rationale note

  **What to do**: Rewrite `web/AGENTS.md:39,41` (Metis-cited) based on task 24's actual, re-verified outcome — either "containerizing resolved the fresh-runner aube failure (verified <date>, see evidence/task-24-*)" or "aube still fails on a fresh instance even containerized; CI's `web:install` step uses npm specifically inside the container as a scoped exception (verified <date>)." Remove the now-stale "matching the lockfile-backed path both verify.yml and deploy.yml run" framing since those workflows no longer run raw `npm --prefix web ci`.
  **Must NOT do**: Do not leave the old note's specific claims uncorrected even if the overall conclusion (npm needed somewhere) turns out unchanged — the doc must reflect what was actually re-verified, not just carry the old sentence forward.

  **Parallelization**: Can Parallel: YES | Wave 6 | Blocks: [] | Blocked By: [24]

  **References**:
  - `web/AGENTS.md:39,41` - the note being rewritten.

  **Acceptance Criteria**:
  - [ ] The note accurately describes the post-task-24 reality, with a pointer to the reproduction evidence.
  - [ ] No sentence in this file claims the workflows run raw `npm --prefix web ci` if task 20 removed that.

  **QA Scenarios**:
  ```
  Scenario: Doc matches current workflow reality
    Tool: bash
    Steps: grep -n "npm --prefix web ci" web/AGENTS.md .github/workflows/verify.yml .github/workflows/deploy.yml
    Expected: either zero matches everywhere (aube fully replaced it), or the doc's mention matches exactly where the fallback (task 24) actually left it in the YAML
    Evidence: evidence/task-25-doc-accuracy.txt
  ```

  **Commit**: YES | Message: `docs(web): correct the npm-in-CI rationale note after re-verification` | Files: [web/AGENTS.md]

- [ ] 26. Rewrite .made.yml to call mise tasks directly

  **What to do**: Replace `.made.yml`'s `test` and `lint` command blocks — currently hardcoded absolute host paths (`/opt/homebrew/bin`, `/Users/douglasjarquin/.local/...`) duplicating `mise run test` plus raw `npm --prefix web ci`/`run themes:check`/`run build`/`run test:e2e`, node --test loops, `uv run ruff`, and `shellcheck` — with direct calls to the canonical mise task names this plan defines: `mise run test` (now covers pytest + test-cli via `depends`, task 10), `mise run web:install`, `mise run web:build` (already runs `themes:check` internally via package.json's chain, task 7), `mise run web:test` (already runs both `test:unit` and `test:e2e` per `web/package.json`'s own `"test"` script), `mise run lint`, `mise run shellcheck` (tasks 3, 10). Run these directly on the host (not through `scripts/ci/run-in-dev-container.sh`) — `.made.yml`'s job is a fast local agent self-check, not full CI parity; requiring Docker for it would make it fail wherever Docker isn't running, a regression from today.
  **Must NOT do**: Do not reintroduce hardcoded absolute paths — rely on `mise` being on `PATH` (the same assumption every other task in this plan already makes).

  **Parallelization**: Can Parallel: NO | Wave 7 | Blocks: [] | Blocked By: [4, 5, 6, 7, 8, 9, 10]

  **References**:
  - `.made.yml` (full file, read this session) - the exact hardcoded-path blocks being replaced.
  - mise task names from tasks 3, 7, 10 - the canonical invocations this file now delegates to instead of duplicating.

  **Acceptance Criteria**:
  - [ ] `.made.yml` contains no absolute host path (`grep -n "/Users/\|/opt/homebrew" .made.yml` returns nothing).
  - [ ] `.made.yml`'s `test`/`lint` commands are each a single `mise run <task>` invocation (or a short `set -e` chain of them), never a raw duplicated command.
  - [ ] Running `made test`/`made lint` (or however this tool is actually invoked — confirm exact CLI form at implementation time) produces the same pass/fail verdict as running the equivalent `mise run` tasks directly.

  **QA Scenarios**:
  ```
  Scenario: made's verdict matches direct mise invocation
    Tool: bash
    Steps: made test; echo "made exit: $?"; mise run test; echo "mise exit: $?"
    Expected: both exit codes match on the current clean tree
    Evidence: evidence/task-26-made-parity.txt

  Scenario: No hardcoded paths remain
    Tool: bash
    Steps: grep -n "/Users/\|/opt/homebrew" .made.yml
    Expected: no matches
    Evidence: evidence/task-26-no-hardcoded-paths.txt
  ```

  **Commit**: YES | Message: `chore(made): call canonical mise tasks instead of duplicating hardcoded commands` | Files: [.made.yml]

- [ ] 27. Final documentation sweep

  **What to do**: Grep the whole repo one more time for any remaining stale reference the per-task updates above might have missed: `git grep -n "scripts/build-standalone.py\|scripts/generate-previews.py\|scripts/generate-web-theme-data.py\|scripts/author_character_themes.py\|scripts/import-revised-themes.py\|tests/test_cli.sh\|npm --prefix\|npx --prefix"`. Update root `AGENTS.md`'s "STRUCTURE" and "COMMANDS" sections to reflect the new `mise-tasks/`, `docker/`, `scripts/ci/` directories and the new `mise run <task>` command surface. Update `README.md` if it names any relocated path (excluding `scripts/sf2`, which stays).
  **Must NOT do**: Do not touch `scripts/sf2`'s own documentation in `README.md` — it is explicitly out of scope for this whole plan.

  **Parallelization**: Can Parallel: NO | Wave 7 | Blocks: [F1] | Blocked By: [1-26]

  **References**:
  - Root `AGENTS.md` "STRUCTURE" and "COMMANDS" sections - need the new directories/commands reflected.
  - `README.md` - checked for any remaining stale path mention.

  **Acceptance Criteria**:
  - [ ] The consolidated grep above returns zero matches anywhere outside this plan file itself and any Dockerfile comment intentionally referencing the old niceuptime inspiration.
  - [ ] `AGENTS.md`'s STRUCTURE table lists `mise-tasks/`, `docker/`, and `scripts/ci/`.

  **QA Scenarios**:
  ```
  Scenario: Zero stale references repo-wide
    Tool: bash
    Steps: git grep -n "scripts/build-standalone.py\|scripts/generate-previews.py\|scripts/generate-web-theme-data.py\|scripts/author_character_themes.py\|scripts/import-revised-themes.py\|tests/test_cli.sh\|npm --prefix\|npx --prefix" -- ':!plans/dev-tooling-suite.md'
    Expected: no matches
    Evidence: evidence/task-27-final-sweep.txt

  Scenario: AGENTS.md structure table is current
    Tool: bash
    Steps: grep -n "mise-tasks\|docker/\|scripts/ci" AGENTS.md
    Expected: all three appear in the STRUCTURE table
    Evidence: evidence/task-27-agents-md.txt
  ```

  **Commit**: YES | Message: `docs: final sweep for the dev-tooling-suite migration` | Files: [AGENTS.md, README.md]

## Final Verification Wave (MANDATORY - after ALL implementation tasks)
> ALL must APPROVE. Present consolidated results to user and get explicit "okay" before completing.

- [ ] F1. Plan Compliance Audit
  Verify every task's Acceptance Criteria are checked, every commit landed, and the Dependency Matrix's wave ordering was actually followed (no task started before its blockers merged).

- [ ] F2. Code Quality Review
  Re-run `mise run lint`, `mise run shellcheck`, and `docker build` for both images with `--no-cache` at least once (catches anything a warm Docker layer cache silently papered over). Confirm no task introduced a new script-shim pattern (`grep -rn "run = \[\"bash \|run = \"python3 scripts"` across `mise.toml` should return nothing).

- [ ] F3. Real Manual QA
  On a genuinely clean checkout (fresh clone, no `~/.local/share/mise` cache reused): `mise install --locked`, `mise run test`, `scripts/ci/run-in-dev-container.sh mise run web:test`, `mise run web:dev:container` + a real `curl` against the published port, then open an actual draft PR against this repo and watch `verify.yml` run end-to-end for real (not simulated) — this is the only step that proves the whole chain works together, not just each piece in isolation.

- [ ] F4. Scope Fidelity Check + Manual Boss Actions
  Confirm `scripts/sf2` is untouched and `scripts/ci/*.sh` were never converted to mise tasks (Must NOT Have). Then, explicitly, in the final report to the boss — not silently checked off — list the two actions only the boss can perform: (1) set `ghcr.io/douglasjarquin/sf2-themes-toolchain` and `-dev` package visibility to **Public** in GitHub → repo → Packages, so a fork PR's read-only `GITHUB_TOKEN` can still pull them; (2) update this repo's branch-protection required-status-check list for the renamed/added job (`toolchain-checks`), since that list lives in GitHub settings, not in the workflow YAML.

## Commit Strategy
One commit per task below (message + files listed per task). Waves 1-3 land as a single PR (pure repo reorg, no CI behavior change yet — CI still runs the old workflow against the new task names via the existing `mise run test` composite). Waves 4-6 land as a second PR (adds the container path; the old direct-on-runner steps stay as a fallback in the same PR, removed only once the containerized path is proven green on that PR's own CI run). Wave 7 lands as a third, small PR.

## Success Criteria
- Every Definition of Done command above passes.
- `git log --oneline -- scripts/` shows no Python files remaining under `scripts/` except none (directory removed) other than `scripts/sf2` and `scripts/ci/*.sh`.
- A fresh `git clone` + `mise install` + `mise run test` succeeds with zero manual `pip`/`npm`/`brew` steps.
- Boss has been told, explicitly, to flip the two GHCR packages to Public and to update branch-protection required-check names — these are not silently marked done.
