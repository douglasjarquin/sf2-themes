import type { GameInput } from "../types.ts";

export interface InputSource {
  read(): GameInput;
  setActive(active: boolean): void;
  releaseAll(): void;
  destroy(): void;
}
