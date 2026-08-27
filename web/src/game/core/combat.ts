import type { Box, NormalizedPlayerInput } from "../types.ts";
import { intersects, placeBox } from "./collision.ts";
import { fighterHurtboxes, type MutableFighter } from "./fighter.ts";

export type Impact = {
  readonly blocked: boolean;
  readonly damage: number;
  readonly knockout: boolean;
};

export type ImpactValues = {
  readonly damage: number;
  readonly blockDamage: number;
  readonly hitstunTicks: number;
  readonly blockstunTicks: number;
  readonly hitstopTicks: number;
};

export function applyImpact(
  attacker: MutableFighter,
  defender: MutableFighter,
  defenderInput: NormalizedPlayerInput,
  values: ImpactValues,
): Impact {
  const blocked = defenderInput.block && defender.health > 0;
  const damage = blocked ? values.blockDamage : values.damage;
  defender.health = Math.max(0, defender.health - damage);
  defender.hitstopTicks = Math.max(defender.hitstopTicks, values.hitstopTicks);
  attacker.hitstopTicks = Math.max(attacker.hitstopTicks, values.hitstopTicks);
  if (defender.health === 0) {
    defender.pose = "ko";
    defender.hitstunTicks = 0;
    defender.blockstunTicks = 0;
  } else if (blocked) {
    defender.blockstunTicks = values.blockstunTicks;
    defender.pose = "block";
  } else {
    defender.hitstunTicks = values.hitstunTicks;
    defender.pose = "hit";
  }
  return { blocked, damage, knockout: defender.health === 0 };
}

function overlapsHurtbox(hitbox: Box, defender: MutableFighter): boolean {
  return fighterHurtboxes(defender).some((hurtbox) => intersects(hitbox, hurtbox));
}

export function resolveMeleeAttack(
  attacker: MutableFighter,
  defender: MutableFighter,
  defenderInput: NormalizedPlayerInput,
): Impact | null {
  if (attacker.health === 0) return null;
  const move = attacker.activeMove;
  if (move === null) return null;
  const activeFrame = move.activeFrames.find(
    (frame) => attacker.moveTick >= frame.startTick && attacker.moveTick <= frame.endTick,
  );
  if (activeFrame === undefined) return null;
  if (move.multiHit ? attacker.connectedHitIds.has(activeFrame.hitId) : attacker.connectedHitIds.size > 0) return null;
  const hitbox = placeBox(activeFrame.hitbox, attacker, attacker.facing);
  if (!overlapsHurtbox(hitbox, defender)) return null;

  attacker.connectedHitIds.add(activeFrame.hitId);
  return applyImpact(attacker, defender, defenderInput, move);
}

export function hitboxOverlapsFighter(hitbox: Box, fighter: MutableFighter): boolean {
  return overlapsHurtbox(hitbox, fighter);
}
