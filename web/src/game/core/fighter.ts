import { GROUND_Y, STAGE_RIGHT } from "../config.ts";
import type {
  AttackButton,
  FighterDefinition,
  FighterPose,
  FighterSnapshot,
  Facing,
  MoveDefinition,
  NormalizedPlayerInput,
} from "../types.ts";
import { clampFighterX, placeBox } from "./collision.ts";

export type MutableFighter = {
  definition: FighterDefinition;
  x: number;
  y: number;
  velocityX: number;
  velocityY: number;
  facing: Facing;
  health: number;
  pose: FighterPose;
  activeMove: MoveDefinition | null;
  moveTick: number;
  moveSerial: number;
  connectedHitIds: Set<number>;
  projectileSpawned: boolean;
  hitstunTicks: number;
  blockstunTicks: number;
  hitstopTicks: number;
};

export type FighterTick = {
  readonly frozen: boolean;
  readonly attack: AttackButton | null;
};

export function createFighter(definition: FighterDefinition, x: number, facing: Facing): MutableFighter {
  return {
    definition,
    x: clampFighterX(x, definition.pushbox),
    y: GROUND_Y,
    velocityX: 0,
    velocityY: 0,
    facing,
    health: definition.maxHealth,
    pose: "idle",
    activeMove: null,
    moveTick: 0,
    moveSerial: 0,
    connectedHitIds: new Set<number>(),
    projectileSpawned: false,
    hitstunTicks: 0,
    blockstunTicks: 0,
    hitstopTicks: 0,
  };
}

export function resetFighter(fighter: MutableFighter, x: number, facing: Facing): void {
  const replacement = createFighter(fighter.definition, x, facing);
  Object.assign(fighter, replacement);
}

export function chooseAttack(input: NormalizedPlayerInput): AttackButton | null {
  if (input.projectile) return "projectile";
  if (input.heavy) return "heavy";
  if (input.light) return "light";
  return null;
}

function startMove(fighter: MutableFighter, attack: AttackButton): void {
  const move = fighter.definition.moves.find((candidate) => candidate.input === attack);
  if (move === undefined) return;
  fighter.activeMove = move;
  fighter.moveTick = 0;
  fighter.moveSerial += 1;
  fighter.connectedHitIds.clear();
  fighter.projectileSpawned = false;
  fighter.pose = "attack";
}

export function beginFighterTick(
  fighter: MutableFighter,
  input: NormalizedPlayerInput,
  attack: AttackButton | null,
): FighterTick {
  if (fighter.hitstopTicks > 0) {
    fighter.hitstopTicks -= 1;
    return { frozen: true, attack: null };
  }
  if (fighter.health === 0) {
    fighter.pose = "ko";
    return { frozen: false, attack: null };
  }
  if (fighter.hitstunTicks > 0) {
    fighter.hitstunTicks -= 1;
    fighter.pose = "hit";
    applyGravity(fighter);
    return { frozen: false, attack: null };
  }
  if (fighter.blockstunTicks > 0) {
    fighter.blockstunTicks -= 1;
    fighter.pose = "block";
    return { frozen: false, attack: null };
  }
  if (fighter.activeMove !== null) return { frozen: false, attack: null };
  if (attack !== null) {
    startMove(fighter, attack);
    return { frozen: false, attack };
  }

  if (input.vertical === -1 && fighter.y === GROUND_Y) fighter.velocityY = fighter.definition.jumpVelocity;
  fighter.velocityX = input.horizontal * fighter.definition.walkSpeed;
  fighter.x = clampFighterX(fighter.x + fighter.velocityX, fighter.definition.pushbox);
  applyGravity(fighter);
  if (fighter.y < GROUND_Y) fighter.pose = "jump";
  else fighter.pose = input.horizontal === 0 ? "idle" : "walk";
  return { frozen: false, attack: null };
}

function applyGravity(fighter: MutableFighter): void {
  if (fighter.y < GROUND_Y || fighter.velocityY < 0) {
    fighter.y += fighter.velocityY;
    fighter.velocityY += fighter.definition.gravity;
    if (fighter.y >= GROUND_Y) {
      fighter.y = GROUND_Y;
      fighter.velocityY = 0;
    }
  }
}

export function finishFighterTick(fighter: MutableFighter): void {
  if (fighter.activeMove === null || fighter.hitstopTicks > 0) return;
  const finalActiveTick = fighter.activeMove.activeFrames.at(-1)?.endTick ?? fighter.activeMove.startupTicks;
  if (fighter.moveTick >= finalActiveTick + fighter.activeMove.recoveryTicks) {
    fighter.activeMove = null;
    fighter.moveTick = 0;
    fighter.connectedHitIds.clear();
    fighter.projectileSpawned = false;
    fighter.pose = fighter.y < GROUND_Y ? "jump" : "idle";
    return;
  }
  fighter.moveTick += 1;
}

export function updateFacing(first: MutableFighter, second: MutableFighter): void {
  if (first.x === second.x) return;
  first.facing = first.x < second.x ? 1 : -1;
  second.facing = first.x < second.x ? -1 : 1;
}

export function fighterHurtboxes(fighter: MutableFighter): readonly ReturnType<typeof placeBox>[] {
  return fighter.definition.hurtboxes.map((box) => placeBox(box, fighter, fighter.facing));
}

export function fighterSnapshot(fighter: MutableFighter): FighterSnapshot {
  return {
    id: fighter.definition.id,
    x: fighter.x,
    y: fighter.y,
    velocityX: fighter.velocityX,
    velocityY: fighter.velocityY,
    facing: fighter.facing,
    health: fighter.health,
    pose: fighter.pose,
    moveId: fighter.activeMove?.id ?? null,
    moveTick: fighter.moveTick,
    hitstunTicks: fighter.hitstunTicks,
    blockstunTicks: fighter.blockstunTicks,
    hitstopTicks: fighter.hitstopTicks,
  };
}

export function defaultFighterPositions(): readonly [number, number] {
  return [Math.floor(STAGE_RIGHT * 0.3), Math.floor(STAGE_RIGHT * 0.7)];
}
