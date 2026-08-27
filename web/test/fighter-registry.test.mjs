import assert from "node:assert/strict";
import test from "node:test";

import {
  FIGHTER_IDS,
  FighterContractError,
  createFighterRegistry,
  parseFighterId,
  validateFighterDefinition,
} from "../src/game/fighter-registry.ts";
import { FIGHTER_IDS as EXPECTED_FIGHTER_IDS, makeFighter } from "./game-fixture.mjs";

test("fighter IDs are the exact closed arcade roster", () => {
  assert.deepEqual(FIGHTER_IDS, EXPECTED_FIGHTER_IDS);
  assert.equal(new Set(FIGHTER_IDS).size, 17);
});

test("main and light theme IDs cannot cross the fighter boundary", () => {
  for (const rejected of ["main", "main-light", "ryu-light", "akuma-light", "ken-light"]) {
    assert.throws(() => parseFighterId(rejected), FighterContractError);
  }
  assert.equal(parseFighterId("chun-li"), "chun-li");
});

test("fighter definitions require authored animation, move, projectile, victory, and AI fields", () => {
  const valid = makeFighter("ryu");
  assert.equal(validateFighterDefinition(valid).id, "ryu");
  for (const key of ["animations", "moves", "projectile", "victory", "aiBias"]) {
    const malformed = { ...valid };
    delete malformed[key];
    assert.throws(() => validateFighterDefinition(malformed), FighterContractError);
  }
});

test("malformed authored active frames fail at the registry boundary", () => {
  const reversed = makeFighter("ryu", {
    activeFrames: [{ hitId: 0, startTick: 4, endTick: 3, hitbox: { x: 0, y: 0, width: 1, height: 1 } }],
  });
  const beforeStartup = makeFighter("ryu", {
    activeFrames: [{ hitId: 0, startTick: 1, endTick: 2, hitbox: { x: 0, y: 0, width: 1, height: 1 } }],
  });
  assert.throws(() => validateFighterDefinition(reversed), /endTick/);
  assert.throws(() => validateFighterDefinition(beforeStartup), /startupTicks/);
  assert.throws(
    () => validateFighterDefinition({ ...makeFighter("ryu"), pushbox: { x: 0, y: -1_000, width: 601, height: 1_000 } }),
    /integer centering/,
  );
});

test("a registry is accepted only when all 17 fighters appear exactly once", () => {
  const complete = FIGHTER_IDS.map((id) => makeFighter(id));
  const registry = createFighterRegistry(complete);
  assert.equal(registry.size, 17);
  assert.equal(registry.get("m-bison")?.displayName, "M-BISON");
  assert.throws(() => createFighterRegistry(complete.slice(1)), /missing fighters: ryu/);
  assert.throws(() => createFighterRegistry([...complete, makeFighter("ryu")]), /duplicate fighter ryu/);
});
