import type { GameSnapshot } from "../types.ts";
import type { GamePalette } from "./palette.ts";

export interface GameRenderer {
  readonly logicalSize: {
    readonly columns: 96;
    readonly rows: 40;
  };
  mount(container: HTMLElement): Promise<void>;
  resize(width: number, height: number): void;
  setPalette(palette: GamePalette): void;
  render(snapshot: GameSnapshot): void;
  destroy(): void;
}
