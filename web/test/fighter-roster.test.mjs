import assert from "node:assert/strict";
import test from "node:test";

import { FIGHTER_ROSTER } from "../src/game/content/fighters/index.ts";
import { createGameCore } from "../src/game/core/state-machine.ts";

function inputsFor(playerIndex, input) {
  return playerIndex === 0 ? { players: [input, {}] } : { players: [{}, input] };
}

function advanceUntil(core, predicate, label) {
  for (let tick = 0; tick < 120; tick += 1) {
    const snapshot = core.step();
    if (predicate(snapshot)) return snapshot;
  }
  assert.fail(`${label} was not observed within 120 ticks`);
}

function finishAttack(core, playerIndex) {
  return advanceUntil(
    core,
    (snapshot) => snapshot.phase !== "player-fight" || snapshot.fighters[playerIndex].moveId === null,
    `player ${playerIndex + 1} attack completion`,
  );
}

function startAttack(core, playerIndex, input) {
  const snapshot = core.step(inputsFor(playerIndex, { [input]: true }));
  core.step();
  return snapshot;
}

function makeCore(fighter, playerIndex, seed) {
  const opponent = FIGHTER_ROSTER.find((candidate) => candidate.definition.id !== fighter.definition.id);
  assert.ok(opponent);
  const fighters = playerIndex === 0
    ? [fighter.definition, opponent.definition]
    : [opponent.definition, fighter.definition];
  return createGameCore({ seed, fighters });
}

test("all 17 authored fighters run as P1 and P2 through the core combat snapshot lifecycle", () => {
  let scenarios = 0;

  for (const [rosterIndex, fighter] of FIGHTER_ROSTER.entries()) {
    for (const playerIndex of [0, 1]) {
      const defenderIndex = playerIndex === 0 ? 1 : 0;
      const core = makeCore(fighter, playerIndex, 10_000 + rosterIndex * 2 + playerIndex);

      const reset = core.reset();
      assert.equal(reset.phase, "boot");
      assert.equal(reset.fighters[playerIndex].id, fighter.definition.id);
      assert.equal(reset.fighters[playerIndex].health, fighter.definition.maxHealth);

      core.advanceTicks(1);
      const intro = core.insertCoin();
      assert.equal(intro.phase, "player-intro");
      assert.equal(core.advanceTicks(120).phase, "player-fight");

      const attack = startAttack(core, playerIndex, "light");
      assert.equal(attack.fighters[playerIndex].pose, "attack");
      assert.equal(attack.fighters[playerIndex].moveId, fighter.definition.moves.find((move) => move.input === "light")?.id);
      finishAttack(core, playerIndex);

      startAttack(core, playerIndex, "projectile");
      const projectile = advanceUntil(core, (snapshot) => snapshot.projectiles.length > 0, `${fighter.definition.id} projectile`);
      assert.equal(projectile.projectiles[0]?.owner, playerIndex);
      advanceUntil(
        core,
        (snapshot) => snapshot.projectiles.length === 0 && snapshot.fighters[playerIndex].moveId === null,
        `${fighter.definition.id} projectile completion`,
      );

      core.advanceTicks(40, { players: [{ right: true }, { left: true }] });
      const healthBeforeHit = core.getSnapshot().fighters[defenderIndex].health;
      startAttack(core, playerIndex, "light");
      const hit = advanceUntil(
        core,
        (snapshot) => snapshot.fighters[defenderIndex].health < healthBeforeHit,
        `${fighter.definition.id} authored hit`,
      );
      assert.equal(hit.fighters[defenderIndex].pose, "hit");
      finishAttack(core, playerIndex);

      for (let attackCount = 0; attackCount < 20 && core.getSnapshot().phase === "player-fight"; attackCount += 1) {
        startAttack(core, playerIndex, "heavy");
        finishAttack(core, playerIndex);
      }
      const knockout = core.getSnapshot();
      assert.equal(knockout.phase, "player-result", `${fighter.definition.id} did not reach a result as P${playerIndex + 1}`);
      assert.equal(knockout.fighters[defenderIndex].pose, "ko");

      const winner = knockout.fighters[playerIndex];
      const winnerAnimation = fighter.definition.animations[winner.pose];
      assert.equal(winner.id, fighter.definition.id);
      assert.equal(knockout.result, playerIndex === 0 ? "player-1" : "player-2");
      assert.equal(winner.pose, fighter.definition.victory.animation);
      assert.ok(winnerAnimation.frames.length > 0);
      for (const frameId of winnerAnimation.frames) {
        assert.ok(fighter.glyphs.frames[frameId]?.rows.length > 0, `${fighter.definition.id} winner frame ${frameId} is missing`);
      }
      scenarios += 1;
    }
  }

  assert.equal(scenarios, 34);
});
