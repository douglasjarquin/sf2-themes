import { MAX_CATCH_UP_TICKS, TICKS_PER_SECOND } from "../config.ts";

const MICROSECONDS_PER_SECOND = 1_000_000;

export type FixedStepClock = {
  readonly remainder: number;
  readonly totalTicks: number;
};

export type ClockAdvance = {
  readonly clock: FixedStepClock;
  readonly ticks: number;
};

export function createFixedStepClock(): FixedStepClock {
  return { remainder: 0, totalTicks: 0 };
}

export function advanceClock(clock: FixedStepClock, elapsedMicroseconds: number): ClockAdvance {
  if (!Number.isSafeInteger(elapsedMicroseconds) || elapsedMicroseconds < 0) {
    throw new RangeError("elapsedMicroseconds must be a non-negative safe integer");
  }
  const accumulated = clock.remainder + elapsedMicroseconds * TICKS_PER_SECOND;
  const availableTicks = Math.floor(accumulated / MICROSECONDS_PER_SECOND);
  const ticks = Math.min(availableTicks, MAX_CATCH_UP_TICKS);
  return {
    clock: {
      remainder: accumulated - availableTicks * MICROSECONDS_PER_SECOND,
      totalTicks: clock.totalTicks + ticks,
    },
    ticks,
  };
}
