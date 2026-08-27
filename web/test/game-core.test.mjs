import assert from "node:assert/strict";
import test from "node:test";

import { GROUND_Y, STAGE_LEFT, STAGE_RIGHT } from "../src/game/config.ts";
import { advanceClock, createFixedStepClock } from "../src/game/core/clock.ts";
import { createGameCore } from "../src/game/core/state-machine.ts";
import { closeFighters, enterPlayerFight, makeCoreOptions } from "./game-fixture.mjs";

function applyClock(core, chunks) {
  let clock = createFixedStepClock();
  for (const elapsed of chunks) {
    const advanced = advanceClock(clock, elapsed);
    clock = advanced.clock;
    core.advanceTicks(advanced.ticks);
  }
  return clock;
}

function strike(core, defender = {}) {
  core.step({ players: [{ light: true }, defender] });
  core.step({ players: [{}, defender] });
  return core.step({ players: [{}, defender] });
}

function advanceUntil(core, phase, limit) {
  for (let count = 0; count < limit; count += 1) {
    if (core.getSnapshot().phase === phase) return core.getSnapshot();
    core.step();
  }
  assert.fail(`phase ${phase} was not reached within ${limit} ticks`);
}

test("same seed and input frames produce equal snapshots and capture state", () => {
  const first = createGameCore(makeCoreOptions({ seed: 77 }));
  const second = createGameCore(makeCoreOptions({ seed: 77 }));
  const frames = Array.from({ length: 2_000 }, (_, tick) => ({
    players: [{ right: tick % 11 < 4, light: tick % 37 === 0 }, { left: tick % 13 < 5, block: tick % 41 < 3 }],
  }));
  for (const frame of frames) {
    first.step(frame);
    second.step(frame);
  }
  assert.deepEqual(first.getSnapshot(), second.getSnapshot());
  assert.deepEqual(first.getCaptureState(), second.getCaptureState());
});

test("fixed-step clock makes render chunking independent and caps catch-up", () => {
  const first = createGameCore(makeCoreOptions());
  const second = createGameCore(makeCoreOptions());
  const sixtyFrames = Array.from({ length: 60 }, (_, index) => (index < 40 ? 16_667 : 16_666));
  const hundredFrames = Array.from({ length: 100 }, () => 10_000);
  const firstClock = applyClock(first, sixtyFrames);
  const secondClock = applyClock(second, hundredFrames);
  assert.equal(firstClock.totalTicks, 60);
  assert.equal(secondClock.totalTicks, 60);
  assert.deepEqual(first.getSnapshot(), second.getSnapshot());
  assert.equal(advanceClock(createFixedStepClock(), 1_000_000).ticks, 8);
});

test("fighters remain inside stage and on integer ground coordinates", () => {
  const core = createGameCore(makeCoreOptions());
  enterPlayerFight(core);
  core.advanceTicks(100, { players: [{ left: true }, { right: true }] });
  const [first, second] = core.getSnapshot().fighters;
  assert.equal(first.x, STAGE_LEFT + 300);
  assert.equal(second.x, STAGE_RIGHT - 300);
  assert.equal(first.y, GROUND_Y);
  assert.equal(second.y, GROUND_Y);
  assert.ok(Number.isInteger(first.x) && Number.isInteger(second.x));
});

test("pushboxes separate fighters after sustained opposing movement", () => {
  const core = createGameCore(makeCoreOptions());
  enterPlayerFight(core);
  core.advanceTicks(100, { players: [{ right: true }, { left: true }] });
  const [first, second] = core.getSnapshot().fighters;
  assert.ok(first.x < second.x);
  assert.ok(second.x - first.x >= 600);
  assert.equal(first.facing, 1);
  assert.equal(second.facing, -1);
});

test("authored active frames delay damage and one move cannot damage repeatedly", () => {
  const core = createGameCore(makeCoreOptions());
  enterPlayerFight(core);
  closeFighters(core);
  const initialHealth = core.getSnapshot().fighters[1].health;
  core.step({ players: [{ light: true }, {}] });
  assert.equal(core.getSnapshot().fighters[1].health, initialHealth);
  core.step();
  assert.equal(core.getSnapshot().fighters[1].health, initialHealth);
  const impact = core.step();
  assert.equal(impact.fighters[1].health, initialHealth - 20);
  core.advanceTicks(12);
  assert.equal(core.getSnapshot().fighters[1].health, initialHealth - 20);
});

test("authored multi-hit moves can connect once per distinct hit ID", () => {
  const activeFrames = [
    { hitId: 0, startTick: 2, endTick: 2, hitbox: { x: 200, y: -800, width: 500, height: 500 } },
    { hitId: 1, startTick: 6, endTick: 6, hitbox: { x: 200, y: -800, width: 500, height: 500 } },
  ];
  const core = createGameCore(makeCoreOptions({ first: { activeFrames, multiHit: true } }));
  enterPlayerFight(core);
  closeFighters(core);
  core.step({ players: [{ light: true }, {}] });
  core.advanceTicks(12);
  assert.equal(core.getSnapshot().fighters[1].health, 60);
});

test("jump velocity and gravity return a fighter to the fixed ground plane", () => {
  const core = createGameCore(makeCoreOptions());
  enterPlayerFight(core);
  const airborne = core.step({ players: [{ up: true }, {}] });
  assert.ok(airborne.fighters[0].y < GROUND_Y);
  assert.ok(airborne.fighters[0].velocityY < 0);
  const landed = core.advanceTicks(40);
  assert.equal(landed.fighters[0].y, GROUND_Y);
  assert.equal(landed.fighters[0].velocityY, 0);
});

test("blocking applies chip damage and blockstun instead of full damage and hitstun", () => {
  const core = createGameCore(makeCoreOptions());
  enterPlayerFight(core);
  closeFighters(core);
  const impact = strike(core, { block: true });
  assert.equal(impact.fighters[1].health, 98);
  assert.equal(impact.fighters[1].blockstunTicks, 5);
  assert.equal(impact.fighters[1].hitstunTicks, 0);
});

test("hitstop preserves move and round timers until freeze expires", () => {
  const core = createGameCore(makeCoreOptions());
  enterPlayerFight(core);
  closeFighters(core);
  const impact = strike(core);
  const moveTick = impact.fighters[0].moveTick;
  const roundTicks = impact.roundTicksRemaining;
  assert.equal(impact.fighters[0].hitstopTicks, 3);
  const frozen = core.step();
  assert.equal(frozen.fighters[0].moveTick, moveTick);
  assert.equal(frozen.roundTicksRemaining, roundTicks);
  assert.equal(frozen.fighters[0].hitstopTicks, 2);
  assert.equal(frozen.fighters[1].health, impact.fighters[1].health);
});

test("projectile hitboxes travel through the same damage and hitstop rules", () => {
  const core = createGameCore(makeCoreOptions());
  enterPlayerFight(core);
  closeFighters(core);
  core.step({ players: [{ projectile: true }, {}] });
  const impact = core.step();
  assert.equal(impact.fighters[1].health, 88);
  assert.equal(impact.fighters[1].hitstopTicks, 2);
  assert.equal(impact.projectiles.length, 0);
});

test("KO enters result and result expiry resets health for the next intro", () => {
  const core = createGameCore(makeCoreOptions({ first: { damage: 100 } }));
  enterPlayerFight(core);
  closeFighters(core);
  const knockout = strike(core);
  assert.equal(knockout.phase, "player-result");
  assert.equal(knockout.result, "player-1");
  assert.equal(knockout.fighters[1].health, 0);
  const reset = core.advanceTicks(300);
  assert.equal(reset.phase, "player-intro");
  assert.equal(reset.fighters[1].health, 100);
  assert.equal(reset.result, null);
});

test("coin insertion immediately interrupts every attract state", () => {
  for (const attractPhase of ["attract-intro", "attract-fight", "attract-result"]) {
    const core = createGameCore(makeCoreOptions());
    advanceUntil(core, attractPhase, 7_000);
    const beforeTick = core.getSnapshot().tick;
    const interrupted = core.insertCoin();
    assert.equal(interrupted.phase, "player-intro");
    assert.equal(interrupted.phaseTick, 0);
    assert.equal(interrupted.tick, beforeTick + 1);
  }
});

test("attract loop resets repeatedly without stale round state", () => {
  const core = createGameCore(makeCoreOptions());
  for (let cycle = 0; cycle < 2; cycle += 1) {
    advanceUntil(core, "attract-result", 7_000);
    const nextIntro = core.advanceTicks(300);
    assert.equal(nextIntro.phase, "attract-intro");
    assert.equal(nextIntro.roundTicksRemaining, 5_940);
    assert.deepEqual(nextIntro.fighters.map((fighter) => fighter.health), [100, 100]);
  }
});

test("pause is explicit and preserves fight state until resumed", () => {
  const core = createGameCore(makeCoreOptions());
  enterPlayerFight(core);
  const paused = core.step({ pause: true });
  const timer = paused.roundTicksRemaining;
  assert.equal(paused.phase, "paused");
  core.advanceTicks(20);
  assert.equal(core.getSnapshot().roundTicksRemaining, timer);
  assert.equal(core.step({ pause: true }).phase, "player-fight");
});

test("malformed input and invalid tick counts reject without mutating state", () => {
  const core = createGameCore(makeCoreOptions());
  const initial = core.getSnapshot();
  for (const count of [-1, 1.5, Number.POSITIVE_INFINITY]) {
    assert.throws(() => core.advanceTicks(count), RangeError);
  }
  for (const input of [null, { players: [] }, { players: [{ left: "yes" }, {}] }]) {
    assert.throws(() => core.step(input), TypeError);
  }
  assert.deepEqual(core.getSnapshot(), initial);
});
