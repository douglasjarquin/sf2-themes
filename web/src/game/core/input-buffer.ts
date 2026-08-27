import type { GameInput, NormalizedGameInput, NormalizedPlayerInput, PlayerInput } from "../types.ts";
import type { MutableFighter } from "./fighter.ts";
import { randomInt, type SeededRng } from "./rng.ts";

const NEUTRAL_PLAYER: NormalizedPlayerInput = {
  horizontal: 0,
  vertical: 0,
  light: false,
  heavy: false,
  projectile: false,
  block: false,
};

export type InputBuffer = {
  readonly capacity: number;
  readonly frames: readonly NormalizedPlayerInput[];
};

function isRecord(value: unknown): value is object {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function readBoolean(value: object, key: string): boolean {
  if (!(key in value)) return false;
  const field = Reflect.get(value, key);
  if (typeof field !== "boolean") throw new TypeError(`${key} must be boolean`);
  return field;
}

export type AiDecision = {
  readonly input: NormalizedPlayerInput;
  readonly rng: SeededRng;
};

export type AiDecisionContext = {
  readonly rng: SeededRng;
  readonly fighter: MutableFighter;
  readonly opponent: MutableFighter;
};

export function createAiInput(context: AiDecisionContext): AiDecision {
  let rng = context.rng;
  const rolls: number[] = [];
  for (let index = 0; index < 4; index += 1) {
    const next = randomInt(rng, 100);
    rng = next.rng;
    rolls.push(next.value);
  }
  const aggression = rolls[0] ?? 0;
  const defense = rolls[1] ?? 0;
  const projectile = rolls[2] ?? 0;
  const jump = rolls[3] ?? 0;
  const horizontal = context.fighter.x === context.opponent.x ? 0 : context.fighter.x < context.opponent.x ? 1 : -1;
  return {
    rng,
    input: {
      horizontal: aggression < context.fighter.definition.aiBias.aggression ? horizontal : 0,
      vertical: jump < context.fighter.definition.aiBias.jump ? -1 : 0,
      light: aggression < Math.floor(context.fighter.definition.aiBias.aggression / 4),
      heavy: aggression >= 25 && aggression < Math.floor(context.fighter.definition.aiBias.aggression / 2),
      projectile: projectile < context.fighter.definition.aiBias.projectile,
      block: defense < context.fighter.definition.aiBias.defense,
    },
  };
}

function normalizePlayer(value: unknown): NormalizedPlayerInput {
  if (value === undefined) return NEUTRAL_PLAYER;
  if (!isRecord(value)) throw new TypeError("player input must be an object");
  const left = readBoolean(value, "left");
  const right = readBoolean(value, "right");
  const up = readBoolean(value, "up");
  const down = readBoolean(value, "down");
  return {
    horizontal: left === right ? 0 : left ? -1 : 1,
    vertical: up === down ? 0 : up ? -1 : 1,
    light: readBoolean(value, "light"),
    heavy: readBoolean(value, "heavy"),
    projectile: readBoolean(value, "projectile"),
    block: readBoolean(value, "block"),
  };
}

export function normalizeInput(value: GameInput | unknown = {}): NormalizedGameInput {
  if (!isRecord(value)) throw new TypeError("game input must be an object");
  const players = "players" in value ? value.players : undefined;
  if (players !== undefined && (!Array.isArray(players) || players.length !== 2)) {
    throw new TypeError("players must contain exactly two inputs");
  }
  const first: PlayerInput | unknown = players?.[0];
  const second: PlayerInput | unknown = players?.[1];
  return {
    players: [normalizePlayer(first), normalizePlayer(second)],
    insertCoin: readBoolean(value, "insertCoin"),
    pause: readBoolean(value, "pause"),
  };
}

export function hasActiveInput(input: NormalizedGameInput): boolean {
  return input.players.some(
    (player) =>
      player.horizontal !== 0 ||
      player.vertical !== 0 ||
      player.light ||
      player.heavy ||
      player.projectile ||
      player.block,
  );
}

export function createInputBuffer(capacity = 12): InputBuffer {
  if (!Number.isSafeInteger(capacity) || capacity <= 0) {
    throw new RangeError("capacity must be a positive safe integer");
  }
  return { capacity, frames: [] };
}

export function pushInput(buffer: InputBuffer, input: NormalizedPlayerInput): InputBuffer {
  return { capacity: buffer.capacity, frames: [...buffer.frames, input].slice(-buffer.capacity) };
}

export function wasPressed(buffer: InputBuffer, button: "light" | "heavy" | "projectile"): boolean {
  const current = buffer.frames.at(-1)?.[button] ?? false;
  const previous = buffer.frames.at(-2)?.[button] ?? false;
  return current && !previous;
}
