import {
  ATTRACT_INTRO_TICKS,
  BOOT_TICKS,
  PLAYER_INACTIVITY_TICKS,
  PLAYER_INTRO_TICKS,
  RESULT_TICKS,
  ROUND_TICKS,
  TITLE_TICKS,
} from "../config.ts";
import { validateFighterDefinition } from "../fighter-registry.ts";
import type {
  FighterDefinition,
  GameInput,
  GamePhase,
  GameSnapshot,
  NormalizedGameInput,
  NormalizedPlayerInput,
  PlayerIndex,
  RoundResult,
} from "../types.ts";
import { separatePushboxes } from "./collision.ts";
import { resolveMeleeAttack } from "./combat.ts";
import {
  beginFighterTick,
  chooseAttack,
  createFighter,
  defaultFighterPositions,
  fighterSnapshot,
  finishFighterTick,
  resetFighter,
  updateFacing,
  type MutableFighter,
} from "./fighter.ts";
import {
  createAiInput,
  createInputBuffer,
  hasActiveInput,
  normalizeInput,
  pushInput,
  wasPressed,
  type InputBuffer,
} from "./input-buffer.ts";
import {
  advanceProjectile,
  projectileSnapshot,
  resolveProjectileAttack,
  spawnProjectile,
  type MutableProjectile,
} from "./projectile.ts";
import { createRng, type SeededRng } from "./rng.ts";

const ATTRACT_PHASES: readonly GamePhase[] = ["attract-intro", "attract-fight", "attract-result"];
export type GameCoreOptions = {
  readonly seed: number;
  readonly fighters: readonly [unknown, unknown];
};

export type GameCore = {
  readonly step: (input?: GameInput | unknown) => GameSnapshot;
  readonly advanceTicks: (count: number, input?: GameInput | unknown) => GameSnapshot;
  readonly insertCoin: () => GameSnapshot;
  readonly reset: (seed?: number) => GameSnapshot;
  readonly getSnapshot: () => GameSnapshot;
};

export function createGameCore(options: GameCoreOptions): GameCore {
  const definitions: readonly [FighterDefinition, FighterDefinition] = [
    validateFighterDefinition(options.fighters[0]),
    validateFighterDefinition(options.fighters[1]),
  ];
  if (definitions[0].id === definitions[1].id) throw new TypeError("fighters must use distinct IDs");
  const initialSeed = options.seed;
  const positions = defaultFighterPositions();
  const fighters: [MutableFighter, MutableFighter] = [
    createFighter(definitions[0], positions[0], 1),
    createFighter(definitions[1], positions[1], -1),
  ];
  let rng: SeededRng = createRng(initialSeed);
  let phase: GamePhase = "boot";
  let resumePhase: GamePhase = "player-fight";
  let tick = 0;
  let phaseTick = 0;
  let roundTicksRemaining: number = ROUND_TICKS;
  let inactivityTicks = 0;
  let result: RoundResult = null;
  let projectiles: MutableProjectile[] = [];
  let buffers: [InputBuffer, InputBuffer] = [createInputBuffer(), createInputBuffer()];

  const enter = (next: GamePhase): void => {
    phase = next;
    phaseTick = 0;
  };

  const resetRound = (): void => {
    resetFighter(fighters[0], positions[0], 1);
    resetFighter(fighters[1], positions[1], -1);
    projectiles = [];
    buffers = [createInputBuffer(), createInputBuffer()];
    roundTicksRemaining = ROUND_TICKS;
    inactivityTicks = 0;
    result = null;
  };

  const snapshot = (): GameSnapshot => ({
    tick,
    phase,
    phaseTick,
    roundTicksRemaining,
    rngState: rng.state,
    fighters: [fighterSnapshot(fighters[0]), fighterSnapshot(fighters[1])],
    projectiles: projectiles.map(projectileSnapshot),
    result,
  });

  const fightInputs = (input: NormalizedGameInput): readonly [NormalizedPlayerInput, NormalizedPlayerInput] => {
    if (phase === "attract-fight") {
      const first = createAiInput({ rng, fighter: fighters[0], opponent: fighters[1] });
      const second = createAiInput({ rng: first.rng, fighter: fighters[1], opponent: fighters[0] });
      rng = second.rng;
      return [first.input, second.input];
    }
    return input.players;
  };

  const bufferedAttack = (index: PlayerIndex, input: NormalizedPlayerInput): ReturnType<typeof chooseAttack> => {
    buffers[index] = pushInput(buffers[index], input);
    const choice = chooseAttack(input);
    return choice !== null && wasPressed(buffers[index], choice) ? choice : null;
  };

  const finishRound = (): void => {
    if (fighters[0].health === fighters[1].health) result = "draw";
    else {
      const winnerIndex: PlayerIndex = fighters[0].health > fighters[1].health ? 0 : 1;
      result = winnerIndex === 0 ? "player-1" : "player-2";
      if (phase === "player-fight") fighters[winnerIndex].pose = fighters[winnerIndex].definition.victory.animation;
    }
    enter(phase === "attract-fight" ? "attract-result" : "player-result");
  };

  const runFightTick = (input: NormalizedGameInput): void => {
    const activeInputs = fightInputs(input);
    const firstTick = beginFighterTick(fighters[0], activeInputs[0], bufferedAttack(0, activeInputs[0]));
    const secondTick = beginFighterTick(fighters[1], activeInputs[1], bufferedAttack(1, activeInputs[1]));
    if (!firstTick.frozen || !secondTick.frozen) {
      const separated = separatePushboxes(fighters[0], fighters[0].definition.pushbox, fighters[1], fighters[1].definition.pushbox);
      fighters[0].x = separated.firstX;
      fighters[1].x = separated.secondX;
      updateFacing(fighters[0], fighters[1]);
    }

    const firstProjectile = firstTick.frozen ? null : spawnProjectile(fighters[0], 0);
    const secondProjectile = secondTick.frozen ? null : spawnProjectile(fighters[1], 1);
    if (firstProjectile !== null) projectiles.push(firstProjectile);
    if (secondProjectile !== null) projectiles.push(secondProjectile);
    if (!firstTick.frozen) resolveMeleeAttack(fighters[0], fighters[1], activeInputs[1]);
    if (!secondTick.frozen) resolveMeleeAttack(fighters[1], fighters[0], activeInputs[0]);

    projectiles = projectiles.filter((projectile) => {
      if (!advanceProjectile(projectile)) return false;
      const defenderIndex: PlayerIndex = projectile.owner === 0 ? 1 : 0;
      const impact = resolveProjectileAttack(
        projectile,
        fighters[projectile.owner],
        fighters[defenderIndex],
        activeInputs[defenderIndex],
      );
      return impact === null;
    });
    finishFighterTick(fighters[0]);
    finishFighterTick(fighters[1]);
    if (!firstTick.frozen && !secondTick.frozen) roundTicksRemaining = Math.max(0, roundTicksRemaining - 1);
    if (fighters[0].health === 0 || fighters[1].health === 0 || roundTicksRemaining === 0) finishRound();
  };

  const advancePhase = (): void => {
    switch (phase) {
      case "boot":
        if (phaseTick >= BOOT_TICKS) enter("title");
        return;
      case "title":
        if (phaseTick >= TITLE_TICKS) enter("attract-intro");
        return;
      case "attract-intro":
        if (phaseTick >= ATTRACT_INTRO_TICKS) enter("attract-fight");
        return;
      case "attract-fight":
      case "player-fight":
      case "paused":
        return;
      case "attract-result":
        if (phaseTick >= RESULT_TICKS) {
          resetRound();
          enter("attract-intro");
        }
        return;
      case "player-intro":
        if (phaseTick >= PLAYER_INTRO_TICKS) enter("player-fight");
        return;
      case "player-result":
        if (phaseTick >= RESULT_TICKS) {
          resetRound();
          enter("player-intro");
        }
        return;
    }
  };

  const step = (rawInput: GameInput | unknown = {}): GameSnapshot => {
    const input = normalizeInput(rawInput);
    tick += 1;
    if (input.insertCoin && ATTRACT_PHASES.includes(phase)) {
      resetRound();
      enter("player-intro");
      return snapshot();
    }
    if (input.insertCoin && phase === "title") {
      resetRound();
      enter("player-intro");
      return snapshot();
    }
    if (input.pause && phase === "player-fight") {
      resumePhase = phase;
      enter("paused");
      return snapshot();
    }
    if (input.pause && phase === "paused") {
      enter(resumePhase);
      return snapshot();
    }
    phaseTick += 1;
    if (phase === "attract-fight" || phase === "player-fight") runFightTick(input);
    if (phase === "player-fight") {
      inactivityTicks = hasActiveInput(input) ? 0 : inactivityTicks + 1;
      if (inactivityTicks >= PLAYER_INACTIVITY_TICKS) {
        resetRound();
        enter("title");
      }
    }
    advancePhase();
    return snapshot();
  };

  const advanceTicks = (count: number, input: GameInput | unknown = {}): GameSnapshot => {
    if (!Number.isSafeInteger(count) || count < 0) throw new RangeError("count must be a non-negative safe integer");
    let current = snapshot();
    for (let index = 0; index < count; index += 1) current = step(input);
    return current;
  };

  const reset = (seed = initialSeed): GameSnapshot => {
    rng = createRng(seed);
    tick = 0;
    phase = "boot";
    phaseTick = 0;
    resetRound();
    return snapshot();
  };

  return {
    step,
    advanceTicks,
    insertCoin: () => step({ insertCoin: true }),
    reset,
    getSnapshot: snapshot,
  };
}
