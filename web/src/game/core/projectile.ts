import { STAGE_LEFT, STAGE_RIGHT } from "../config.ts";
import type { Box, Facing, PlayerIndex, ProjectileDefinition, ProjectileSnapshot } from "../types.ts";
import { placeBox } from "./collision.ts";
import { type Impact, applyImpact, hitboxOverlapsFighter } from "./combat.ts";
import type { MutableFighter } from "./fighter.ts";
import type { NormalizedPlayerInput } from "../types.ts";

export type MutableProjectile = {
  owner: PlayerIndex;
  x: number;
  y: number;
  velocityX: number;
  facing: Facing;
  definition: ProjectileDefinition;
};

export function spawnProjectile(fighter: MutableFighter, owner: PlayerIndex): MutableProjectile | null {
  const definition = fighter.definition.projectile;
  if (
    definition === null ||
    fighter.activeMove?.id !== definition.moveId ||
    fighter.moveTick !== definition.spawnTick ||
    fighter.projectileSpawned
  ) {
    return null;
  }
  fighter.projectileSpawned = true;
  return {
    owner,
    x: fighter.x + definition.offsetX * fighter.facing,
    y: fighter.y + definition.offsetY,
    velocityX: definition.velocityX * fighter.facing,
    facing: fighter.facing,
    definition,
  };
}

export function advanceProjectile(projectile: MutableProjectile): boolean {
  projectile.x += projectile.velocityX;
  return projectile.x >= STAGE_LEFT && projectile.x <= STAGE_RIGHT;
}

function projectileHitbox(projectile: MutableProjectile): Box {
  return placeBox(projectile.definition.hitbox, projectile, projectile.facing);
}

export function resolveProjectileAttack(
  projectile: MutableProjectile,
  owner: MutableFighter,
  defender: MutableFighter,
  defenderInput: NormalizedPlayerInput,
): Impact | null {
  if (!hitboxOverlapsFighter(projectileHitbox(projectile), defender)) return null;
  return applyImpact(owner, defender, defenderInput, projectile.definition);
}

export function projectileSnapshot(projectile: MutableProjectile): ProjectileSnapshot {
  return {
    owner: projectile.owner,
    x: projectile.x,
    y: projectile.y,
    velocityX: projectile.velocityX,
  };
}
