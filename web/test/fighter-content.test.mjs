import assert from "node:assert/strict";
import test from "node:test";

import { FIGHTER_ROSTER, validateRosterFighterDefinition } from "../src/game/content/fighters/index.ts";
import { dojo } from "../src/game/content/stages/dojo.ts";
import { FIGHTER_IDS } from "../src/game/fighter-registry.ts";
import { FIGHTER_PALETTE_REGIONS } from "../src/game/render/glyph-sprite.ts";
import { RENDER_LAYER_ORDER } from "../src/game/render/layers.ts";

const REQUIRED_POSES = ["idle", "walk", "jump", "attack", "hit", "block", "ko", "victory"];

test("the authored roster satisfies the closed fighter content contract in catalog order", () => {
  assert.deepEqual(FIGHTER_ROSTER.map((fighter) => fighter.definition.id), FIGHTER_IDS);
  for (const fighter of FIGHTER_ROSTER) {
    assert.equal(validateRosterFighterDefinition(fighter.definition.id, fighter.definition).id, fighter.definition.id);
    assert.deepEqual(Object.keys(fighter.definition.animations), REQUIRED_POSES);
    assert.ok(fighter.definition.moves.every((move) => move.activeFrames.length > 0));
    assert.ok(fighter.definition.projectile);
    assert.deepEqual(fighter.paletteRegions, FIGHTER_PALETTE_REGIONS);
    assert.equal(fighter.metadata.themeId, fighter.definition.id);
    assert.equal(fighter.metadata.artOrigin, "original-project-authored");
    assert.ok(fighter.directionalFrames.forward.length > 0);
    assert.ok(fighter.directionalFrames.backward.length > 0);
  }
});

test("every roster fighter uses a distinct original multi-cell silhouette with mapped palette regions", () => {
  const silhouettes = new Set();
  for (const fighter of FIGHTER_ROSTER) {
    const silhouette = [];
    for (const pose of REQUIRED_POSES) {
      const frameIds = fighter.definition.animations[pose].frames;
      assert.ok(frameIds.length > 0);
      for (const frameId of frameIds) {
        const frame = fighter.glyphs.frames[frameId];
        assert.ok(frame, `${fighter.definition.id}:${frameId}`);
        assert.ok(frame.rows.length >= 5);
        assert.ok(frame.width >= 5);
        assert.equal(frame.rows.length, frame.regions.length);
        for (let rowIndex = 0; rowIndex < frame.rows.length; rowIndex += 1) {
          const row = frame.rows[rowIndex];
          const regions = frame.regions[rowIndex];
          assert.equal(regions.length, row.length);
          for (let column = 0; column < row.length; column += 1) {
            const glyph = row[column];
            const expectedRegion = glyph === " " ? " " : glyph === "#" ? "d" : glyph === "+" ? "s" : glyph === "@" ? "h" : "p";
            assert.equal(regions[column], expectedRegion, `${fighter.definition.id}:${frameId}:${rowIndex}:${column}`);
          }
        }
        silhouette.push(frame.rows.map((row) => row.replaceAll(/\S/g, "#")).join("\n"));
      }
    }
    const signature = silhouette.join("\n---\n");
    assert.equal(silhouettes.has(signature), false, `${fighter.definition.id} repeats an authored silhouette`);
    silhouettes.add(signature);
  }
  assert.equal(silhouettes.size, FIGHTER_IDS.length);
  assert.equal(JSON.stringify(FIGHTER_ROSTER.map((fighter) => fighter.glyphs)).match(/\.(png|gif|jpe?g|webp|rom)/i), null);
});

test("roster validation names the fighter, malformed field, and expected shape without mutating authored content", () => {
  const fighter = FIGHTER_ROSTER[0];
  const originalWidth = fighter.definition.projectile.hitbox.width;
  const malformed = {
    ...fighter.definition,
    projectile: {
      ...fighter.definition.projectile,
      hitbox: { ...fighter.definition.projectile.hitbox, width: 0 },
    },
  };

  assert.throws(
    () => validateRosterFighterDefinition(fighter.definition.id, malformed),
    {
      name: "FighterContractError",
      message: "ryu.projectile.hitbox.width: must be a safe integer >= 1",
    },
  );
  assert.equal(fighter.definition.projectile.hitbox.width, originalWidth);
});

test("dojo provides ordered depth, floor, foreground, and deterministic restrained animation", () => {
  assert.deepEqual(RENDER_LAYER_ORDER, ["distant", "middle", "floor", "foreground", "fighters", "effects", "hud", "overlays"]);
  assert.deepEqual(Object.keys(dojo.layers), ["distant", "middle", "floor", "foreground"]);
  assert.equal(dojo.groundRow, 35);
  assert.deepEqual(dojo.environment.frameAt(0), dojo.environment.frameAt(119));
  assert.notDeepEqual(dojo.environment.frameAt(0), dojo.environment.frameAt(120));
  assert.deepEqual(dojo.environment.frameAt(0), dojo.environment.frameAt(240));
  assert.ok(dojo.layers.floor.some((glyph) => glyph.text.includes("=")));
});
