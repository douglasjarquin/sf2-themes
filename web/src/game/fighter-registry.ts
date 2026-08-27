import type {
  ActiveFrame,
  AiBias,
  AnimationDefinition,
  AnimationName,
  AttackButton,
  Box,
  FighterDefinition,
  FighterId,
  MoveDefinition,
  ProjectileDefinition,
  VictoryDefinition,
} from "./types.ts";

export const FIGHTER_IDS = [
  "ryu",
  "ken",
  "chun-li",
  "e-honda",
  "blanka",
  "zangief",
  "guile",
  "dhalsim",
  "balrog",
  "vega",
  "sagat",
  "m-bison",
  "cammy",
  "t-hawk",
  "fei-long",
  "dee-jay",
  "akuma",
] as const satisfies readonly FighterId[];

const ANIMATION_NAMES = ["idle", "walk", "jump", "attack", "hit", "block", "ko", "victory"] as const;
const ATTACK_INPUTS = ["light", "heavy", "projectile"] as const;

export class FighterContractError extends TypeError {
  readonly field: string;

  constructor(field: string, message: string) {
    super(`${field}: ${message}`);
    this.name = "FighterContractError";
    this.field = field;
  }
}

function isRecord(value: unknown): value is object {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function field(value: object, key: string): unknown {
  return Reflect.get(value, key);
}

function record(value: unknown, path: string): object {
  if (!isRecord(value)) throw new FighterContractError(path, "must be an object");
  return value;
}

function stringValue(value: unknown, path: string): string {
  if (typeof value !== "string" || value.length === 0) {
    throw new FighterContractError(path, "must be a non-empty string");
  }
  return value;
}

function integer(value: unknown, path: string, minimum = 0): number {
  if (!Number.isSafeInteger(value) || typeof value !== "number" || value < minimum) {
    throw new FighterContractError(path, `must be a safe integer >= ${minimum}`);
  }
  return value;
}

function booleanValue(value: unknown, path: string): boolean {
  if (typeof value !== "boolean") throw new FighterContractError(path, "must be boolean");
  return value;
}

function isAttackInput(value: unknown): value is AttackButton {
  return typeof value === "string" && ATTACK_INPUTS.some((candidate) => candidate === value);
}

function isAnimationName(value: unknown): value is AnimationName {
  return typeof value === "string" && ANIMATION_NAMES.some((name) => name === value);
}

function box(value: unknown, path: string): Box {
  const source = record(value, path);
  return {
    x: integer(field(source, "x"), `${path}.x`, Number.MIN_SAFE_INTEGER),
    y: integer(field(source, "y"), `${path}.y`, Number.MIN_SAFE_INTEGER),
    width: integer(field(source, "width"), `${path}.width`, 1),
    height: integer(field(source, "height"), `${path}.height`, 1),
  };
}

export function isFighterId(value: unknown): value is FighterId {
  return typeof value === "string" && FIGHTER_IDS.some((id) => id === value);
}

export function parseFighterId(value: unknown): FighterId {
  if (!isFighterId(value)) throw new FighterContractError("id", "is not a playable fighter ID");
  return value;
}

function animation(value: unknown, path: string): AnimationDefinition {
  const source = record(value, path);
  const frames = field(source, "frames");
  if (!Array.isArray(frames) || frames.length === 0 || !frames.every((frame) => typeof frame === "string" && frame.length > 0)) {
    throw new FighterContractError(`${path}.frames`, "must contain non-empty frame IDs");
  }
  return {
    frames,
    ticksPerFrame: integer(field(source, "ticksPerFrame"), `${path}.ticksPerFrame`, 1),
    loop: booleanValue(field(source, "loop"), `${path}.loop`),
  };
}

function animations(value: unknown): Readonly<Record<AnimationName, AnimationDefinition>> {
  const source = record(value, "animations");
  return {
    idle: animation(field(source, "idle"), "animations.idle"),
    walk: animation(field(source, "walk"), "animations.walk"),
    jump: animation(field(source, "jump"), "animations.jump"),
    attack: animation(field(source, "attack"), "animations.attack"),
    hit: animation(field(source, "hit"), "animations.hit"),
    block: animation(field(source, "block"), "animations.block"),
    ko: animation(field(source, "ko"), "animations.ko"),
    victory: animation(field(source, "victory"), "animations.victory"),
  };
}

function activeFrame(value: unknown, path: string): ActiveFrame {
  const source = record(value, path);
  const startTick = integer(field(source, "startTick"), `${path}.startTick`);
  const endTick = integer(field(source, "endTick"), `${path}.endTick`);
  if (endTick < startTick) throw new FighterContractError(path, "endTick must be >= startTick");
  return {
    hitId: integer(field(source, "hitId"), `${path}.hitId`),
    startTick,
    endTick,
    hitbox: box(field(source, "hitbox"), `${path}.hitbox`),
  };
}

function move(value: unknown, path: string): MoveDefinition {
  const source = record(value, path);
  const input = field(source, "input");
  if (!isAttackInput(input)) {
    throw new FighterContractError(`${path}.input`, "must be light, heavy, or projectile");
  }
  const startupTicks = integer(field(source, "startupTicks"), `${path}.startupTicks`);
  const rawFrames = field(source, "activeFrames");
  if (!Array.isArray(rawFrames) || rawFrames.length === 0) {
    throw new FighterContractError(`${path}.activeFrames`, "must not be empty");
  }
  const activeFrames = rawFrames.map((frame, index) => activeFrame(frame, `${path}.activeFrames[${index}]`));
  if (activeFrames.some((frame) => frame.startTick < startupTicks)) {
    throw new FighterContractError(`${path}.activeFrames`, "cannot begin before startupTicks");
  }
  for (let index = 1; index < activeFrames.length; index += 1) {
    const previous = activeFrames[index - 1];
    const current = activeFrames[index];
    if (previous === undefined || current === undefined || current.startTick <= previous.endTick) {
      throw new FighterContractError(`${path}.activeFrames`, "must be ordered and non-overlapping");
    }
  }
  return {
    id: stringValue(field(source, "id"), `${path}.id`),
    input,
    startupTicks,
    activeFrames,
    recoveryTicks: integer(field(source, "recoveryTicks"), `${path}.recoveryTicks`),
    damage: integer(field(source, "damage"), `${path}.damage`, 1),
    blockDamage: integer(field(source, "blockDamage"), `${path}.blockDamage`),
    hitstunTicks: integer(field(source, "hitstunTicks"), `${path}.hitstunTicks`, 1),
    blockstunTicks: integer(field(source, "blockstunTicks"), `${path}.blockstunTicks`, 1),
    hitstopTicks: integer(field(source, "hitstopTicks"), `${path}.hitstopTicks`),
    multiHit: booleanValue(field(source, "multiHit"), `${path}.multiHit`),
  };
}

function projectile(value: unknown): ProjectileDefinition | null {
  if (value === null) return null;
  const source = record(value, "projectile");
  return {
    moveId: stringValue(field(source, "moveId"), "projectile.moveId"),
    spawnTick: integer(field(source, "spawnTick"), "projectile.spawnTick"),
    offsetX: integer(field(source, "offsetX"), "projectile.offsetX", Number.MIN_SAFE_INTEGER),
    offsetY: integer(field(source, "offsetY"), "projectile.offsetY", Number.MIN_SAFE_INTEGER),
    velocityX: integer(field(source, "velocityX"), "projectile.velocityX", 1),
    hitbox: box(field(source, "hitbox"), "projectile.hitbox"),
    damage: integer(field(source, "damage"), "projectile.damage", 1),
    blockDamage: integer(field(source, "blockDamage"), "projectile.blockDamage"),
    hitstunTicks: integer(field(source, "hitstunTicks"), "projectile.hitstunTicks", 1),
    blockstunTicks: integer(field(source, "blockstunTicks"), "projectile.blockstunTicks", 1),
    hitstopTicks: integer(field(source, "hitstopTicks"), "projectile.hitstopTicks"),
  };
}

function victory(value: unknown): VictoryDefinition {
  const source = record(value, "victory");
  const animationName = field(source, "animation");
  if (!isAnimationName(animationName)) {
    throw new FighterContractError("victory.animation", "must name a required animation");
  }
  return { animation: animationName, quote: stringValue(field(source, "quote"), "victory.quote") };
}

function aiBias(value: unknown): AiBias {
  const source = record(value, "aiBias");
  const bias = (key: string): number => {
    const parsed = integer(field(source, key), `aiBias.${key}`);
    if (parsed > 100) throw new FighterContractError(`aiBias.${key}`, "must be <= 100");
    return parsed;
  };
  return { aggression: bias("aggression"), defense: bias("defense"), projectile: bias("projectile"), jump: bias("jump") };
}

export function validateFighterDefinition(value: unknown): FighterDefinition {
  const source = record(value, "fighter");
  const rawMoves = field(source, "moves");
  if (!Array.isArray(rawMoves) || rawMoves.length === 0) {
    throw new FighterContractError("moves", "must not be empty");
  }
  const moves = rawMoves.map((entry, index) => move(entry, `moves[${index}]`));
  const ids = new Set(moves.map((entry) => entry.id));
  if (ids.size !== moves.length) throw new FighterContractError("moves", "move IDs must be unique");
  const parsedProjectile = projectile(field(source, "projectile"));
  if (parsedProjectile !== null && !ids.has(parsedProjectile.moveId)) {
    throw new FighterContractError("projectile.moveId", "must reference an authored move");
  }
  const pushbox = box(field(source, "pushbox"), "pushbox");
  if (pushbox.width % 2 !== 0) throw new FighterContractError("pushbox.width", "must be even for integer centering");
  return {
    id: parseFighterId(field(source, "id")),
    displayName: stringValue(field(source, "displayName"), "displayName"),
    maxHealth: integer(field(source, "maxHealth"), "maxHealth", 1),
    walkSpeed: integer(field(source, "walkSpeed"), "walkSpeed", 1),
    jumpVelocity: integer(field(source, "jumpVelocity"), "jumpVelocity", Number.MIN_SAFE_INTEGER),
    gravity: integer(field(source, "gravity"), "gravity", 1),
    pushbox,
    hurtboxes: parseHurtboxes(field(source, "hurtboxes")),
    animations: animations(field(source, "animations")),
    moves,
    projectile: parsedProjectile,
    victory: victory(field(source, "victory")),
    aiBias: aiBias(field(source, "aiBias")),
  };
}

function parseHurtboxes(value: unknown): readonly Box[] {
  if (!Array.isArray(value) || value.length === 0) throw new FighterContractError("hurtboxes", "must not be empty");
  return value.map((entry, index) => box(entry, `hurtboxes[${index}]`));
}

export function createFighterRegistry(values: readonly unknown[]): ReadonlyMap<FighterId, FighterDefinition> {
  const registry = new Map<FighterId, FighterDefinition>();
  for (const value of values) {
    const definition = validateFighterDefinition(value);
    if (registry.has(definition.id)) throw new FighterContractError("id", `duplicate fighter ${definition.id}`);
    registry.set(definition.id, definition);
  }
  const missing = FIGHTER_IDS.filter((id) => !registry.has(id));
  if (missing.length > 0) throw new FighterContractError("registry", `missing fighters: ${missing.join(", ")}`);
  return registry;
}
