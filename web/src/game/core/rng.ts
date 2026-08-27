const UINT32_RANGE = 0x1_0000_0000;
const NON_ZERO_FALLBACK = 0x6d2b79f5;

export type SeededRng = {
  readonly state: number;
};

export type RngValue = {
  readonly rng: SeededRng;
  readonly value: number;
};

export function createRng(seed: number): SeededRng {
  if (!Number.isSafeInteger(seed)) {
    throw new RangeError("seed must be a safe integer");
  }
  const state = seed >>> 0;
  return { state: state === 0 ? NON_ZERO_FALLBACK : state };
}

export function nextRng(rng: SeededRng): RngValue {
  let state = rng.state;
  state ^= state << 13;
  state ^= state >>> 17;
  state ^= state << 5;
  const nextState = state >>> 0;
  return { rng: { state: nextState }, value: nextState / UINT32_RANGE };
}

export function randomInt(rng: SeededRng, maximumExclusive: number): RngValue {
  if (!Number.isSafeInteger(maximumExclusive) || maximumExclusive <= 0) {
    throw new RangeError("maximumExclusive must be a positive safe integer");
  }
  const next = nextRng(rng);
  return { rng: next.rng, value: Math.floor(next.value * maximumExclusive) };
}
