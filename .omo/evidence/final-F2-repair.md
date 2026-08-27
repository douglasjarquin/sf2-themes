# Final F2 Repair Evidence

## Scope and baseline

Worktree: `/Users/douglasjarquin/.codex/worktrees/3b65/sf2-themes`.

Branch: `codex/feat-sf2-ascii-arcade`.

Baseline HEAD before repair: `89504d9e991b64bfe1b8284bafbc18353ac2695c`.

The three existing phase commits were preserved, and no commit was created during this repair.

The preferred in-app browser was attempted first but was unavailable in this session, so all browser claims below use Aube-managed Playwright Chromium.

The repair was limited to `web/src/game/render/TextmodeRenderer.ts`, `web/src/game/BrowserGameHost.ts`, `web/src/game/bootstrap.ts`, `web/src/components/ArcadeGame.astro`, `web/test/renderer-contract.test.mjs`, `web/tests/e2e/game.spec.mjs`, and `web/tests/e2e/home.spec.mjs`.

## Runtime hypotheses

1. The renderer's private Ryu/Ken map drops every other canonical fighter before glyph drawing.

2. The full-game E2E polls a transient attack pose after the fixed-step scheduler may already have advanced beyond it.

3. Bootstrap constructs keyboard input before palette parsing, so malformed palette data leaves DOM listeners attached on the fallback path.

These hypotheses use independent renderer-content, browser-timing, and lifecycle-order evidence.

## RED evidence

### F2-1 renderer content dispatch

Added the all-roster renderer seam regression before the production fix.

Command: `perl -e 'alarm shift; exec @ARGV' 120 aube -C web run test:unit -- test/renderer-contract.test.mjs`.

Observed: `44 tests`, `43 pass`, `1 fail`.

Failure: `AssertionError [ERR_ASSERTION]: chun-li did not produce a player-one glyph draw`.

This failed specifically because the private renderer map only contained Ryu and Ken.

### F2-2 transient attack assertion

The retained F2 report records two unchanged-source fresh-port runs where the transient `data-player-one-pose="attack"` assertion failed once and passed once.

The durable replacement assertion was then run before the host fix.

Command: `perl -e 'alarm shift; exec @ARGV' 180 env PLAYWRIGHT_PORT=50634 aube -C web run test:e2e -- game.spec.mjs --grep 'full game route'`.

Observed: `1 failed`; expected `crescent-palm`, received `null`; `Timeout 1000ms exceeded while waiting on the predicate`.

This isolated the missing durable active-move observable without adding sleeps or retries.

### F2-3 malformed-palette listener lifecycle

Added a real browser regression that counts keyboard listener installation while the generated palette payload is corrupted.

Command: `perl -e 'alarm shift; exec @ARGV' 180 env PLAYWRIGHT_PORT=50633 aube -C web run test:e2e -- home.spec.mjs --grep 'malformed palette bootstrap'`.

Observed: expected `{ keydown: 0, keyup: 0 }`, received `{ keydown: 1, keyup: 1 }`; `1 failed`.

This confirmed keyboard input was constructed before the palette boundary threw.

## Root-cause toggle proof

F2-1: replacing the private Ryu/Ken-only lookup with `FIGHTER_ROSTER.find(...)` changed the same renderer seam from the Chun-Li failure to 44 of 44 passing tests.

F2-2: exposing `data-player-one-move-id` from the existing `FighterSnapshot.moveId` changed the durable E2E assertion from `null` and timeout to `crescent-palm` on repeated full game runs.

F2-3: moving `paletteMap(root)` before `createKeyboardInputSource(root)` changed the malformed-palette listener count from one keydown and one keyup listener to zero of each.

Each toggle changes the predicted mechanism and its direct observable, not merely an adjacent assertion.

## Implementation

`TextmodeRenderer` now resolves every selected fighter through the canonical `FIGHTER_ROSTER` content registry, with no duplicate renderer-owned fighter map.

The renderer lookup is typed as the existing `FighterContent` contract and preserves all authored animation and glyph behavior.

`BrowserGameHost` now exposes the active `FighterSnapshot.moveId` as `data-player-one-move-id`, while the existing pose and position observables remain intact.

`ArcadeGame.astro` initializes the durable move attribute to an empty value before the host mounts.

`bootstrapArcadeGame` parses and validates the palette payload before constructing keyboard input, preventing malformed-palette fallback from creating listeners.

`renderer-contract.test.mjs` now routes every canonical fighter ID through the renderer seam and requires a single-character player glyph draw.

`game.spec.mjs` now synchronizes attack verification on the durable authored move ID and adds a real WebGL `readPixels` regression for selected Chun-Li and Akuma glyph colors.

`home.spec.mjs` now proves malformed palette fallback installs zero keyboard listeners.

## GREEN automated verification

`perl -e 'alarm shift; exec @ARGV' 120 aube -C web run test:unit -- test/renderer-contract.test.mjs`: PASS, 44 tests passed.

`perl -e 'alarm shift; exec @ARGV' 180 aube -C web run check`: PASS, 78 files with 0 errors, 0 warnings, and 0 hints.

`perl -e 'alarm shift; exec @ARGV' 180 env PLAYWRIGHT_PORT=50635 aube -C web run test:e2e -- game.spec.mjs --grep 'full game route'`: PASS, 1 test passed.

`perl -e 'alarm shift; exec @ARGV' 180 env PLAYWRIGHT_PORT=50636 aube -C web run test:e2e -- home.spec.mjs --grep 'malformed palette bootstrap'`: PASS, 1 test passed.

Repeated `game.spec.mjs` on ports 50638 and 50639: PASS, 3 of 3 tests passed on each run.

`aube -C web run test:unit`: PASS, 44 of 44 tests passed.

`env PLAYWRIGHT_PORT=50640 aube -C web run test:e2e`: PASS, 32 of 32 tests passed.

Two consecutive `aube -C web run build` commands: PASS, six static routes built each time.

## Real Playwright Manual QA

Surface: Aube-managed Playwright Chromium at `http://127.0.0.1:50637/sf2-themes/game/?theme=main&p1=chun-li&p2=akuma`.

The route reported a ready renderer and accepted INSERT COIN before reading the live WebGL canvas.

The browser regression read the actual WebGL framebuffer and observed nonzero player-one and player-two palette-color pixels in the scene for Chun-Li and Akuma.

The persisted screenshot is [f2-repair-chun-li-akuma.png](/Users/douglasjarquin/.codex/worktrees/3b65/sf2-themes/web/test-results/game-live-renderer-draws-t-4c7aa--non-default-fighter-glyphs/f2-repair-chun-li-akuma.png).

The screenshot visibly contains the selected blue Chun-Li and orange Akuma multi-cell glyphs on the dojo floor.

The persisted browser trace is [f2-repair-chun-li-akuma.zip](/Users/douglasjarquin/.codex/worktrees/3b65/sf2-themes/web/test-results/game-live-renderer-draws-t-4c7aa--non-default-fighter-glyphs/f2-repair-chun-li-akuma.zip).

The repeated full game path observed the durable `crescent-palm` move ID after the Z key tap, avoiding a transient-pose race.

The malformed-palette path rendered the visible static fallback and observed zero surviving keydown or keyup listener installations.

The full browser suite reported no failing test or uncaught browser error.

## UltraQA matrix

| Class | Result | Evidence |
| --- | --- | --- |
| malformed_input | PASS | Corrupted palette payload produced the RED listener regression, then the GREEN zero-listener fallback; existing malformed input checks remained green. |
| stale_state | PASS | Every repeat used a fresh Aube preview port and fresh Playwright browser context; builds and routes were regenerated from the current source. |
| dirty_worktree | PASS | Baseline had the three committed phase changes; final diff contains only the seven scoped repair files plus temporary ignored debug artifacts pending cleanup. |
| hung_or_long_commands | PASS | All repair commands used bounded Perl alarms from 120 to 300 seconds and completed within bounds. |
| flaky_tests | PASS | The prior fail/pass transient pose history is retained; the durable move-ID assertion passed on two full repetitions and in the full 32-test suite. |
| misleading_success_output | PASS | Renderer correctness was checked through real WebGL `readPixels` and a persisted screenshot, while fallback correctness used listener counts rather than renderer status alone. |
| repeated_interruptions/cancel_resume | PASS | Preview servers were repeatedly started and torn down on ports 50630-50640; no task listener remained after each lifecycle. |
| cleanup-command robustness | PASS | The first zsh wildcard cleanup stopped on an unmatched glob; explicit registered paths were rerun, then `mise exec -- aube -C web run preview -- status` reported no preview and all task ports/processes were absent. |
| prompt_injection | NOT_APPLICABLE | No untrusted instruction-bearing page or repository content was encountered. |

## Cleanup receipt

Registered task preview ports 50620-50640 were checked after verification, and no task-owned listener remained.

`aube -C web run preview -- status` reported no preview server running.

The in-app browser was unavailable and created no controllable context; all Aube-managed Playwright contexts exited with their test runs.

The persisted screenshot and trace under `web/test-results/game-live-renderer-draws-t-4c7aa--non-default-fighter-glyphs/` were retained as durable repair evidence.

The temporary `.debug-journal.md` and registered `/tmp/f2-repair-game-run1.log` and `/tmp/f2-repair-game-run2.log` artifacts were moved to macOS Trash with `/usr/bin/trash`.

The cleanup rerun used `mise exec -- aube -C web run preview -- status` because a bare `aube` invocation was unavailable in the cleanup shell PATH.

No task-owned preview, browser, Playwright, Node, or debug process remains, and ports 50620-50640 are free.

Unrelated port 4321 and PID 71105 were preserved.

## Terminal status

DoneClaim: F2-1 canonical renderer dispatch, F2-2 deterministic durable attack evidence, and F2-3 malformed-palette listener ordering are repaired and verified without a commit.

Changed files: `web/src/game/render/TextmodeRenderer.ts`, `web/src/game/BrowserGameHost.ts`, `web/src/game/bootstrap.ts`, `web/src/components/ArcadeGame.astro`, `web/test/renderer-contract.test.mjs`, `web/tests/e2e/game.spec.mjs`, `web/tests/e2e/home.spec.mjs`, and `.omo/evidence/final-F2-repair.md`.

Risks: the retained F2 artifact records the original transient pose flake and renderer gap; no known repair regression remains after the full unit, browser, build, and cleanup checks.
