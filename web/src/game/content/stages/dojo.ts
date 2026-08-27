import type { GamePaletteRole } from "../../render/palette.ts";

export type StageGlyph = {
  readonly x: number;
  readonly y: number;
  readonly text: string;
  readonly role: GamePaletteRole;
};

export type StageDefinition = {
  readonly id: "dojo";
  readonly groundRow: number;
  readonly layers: {
    readonly distant: readonly StageGlyph[];
    readonly middle: readonly StageGlyph[];
    readonly floor: readonly StageGlyph[];
    readonly foreground: readonly StageGlyph[];
  };
  readonly environment: {
    readonly periodTicks: number;
    frameAt(tick: number): StageGlyph;
  };
};

const bannerFrames = ["~| FOCUS |~", "-| FOCUS |-"] as const;

export const dojo = {
  id: "dojo",
  groundRow: 35,
  layers: {
    distant: [
      { x: 0, y: 6, text: "............... distant ridge ............... distant ridge ...............", role: "distantForeground" },
      { x: 0, y: 10, text: "       /\\              /\\              /\\              /\\", role: "distantForeground" },
    ],
    middle: [
      { x: 8, y: 14, text: "+--------------------------------------------------------------+", role: "middleForeground" },
      { x: 8, y: 15, text: "| []      []       OPEN HAND DOJO       []      []            |", role: "middleForeground" },
      { x: 8, y: 16, text: "+--+------+------------------------------+------+-------------+", role: "middleForeground" },
      { x: 12, y: 17, text: "|  |      |                              |      |         |", role: "middleForeground" },
    ],
    floor: [
      { x: 0, y: 35, text: "================================================================================================", role: "foregroundForeground" },
      { x: 0, y: 36, text: "__/\\____/\\____/\\____/\\____/\\____/\\____/\\____/\\____/\\____/\\____/\\____/\\____/\\__", role: "foregroundForeground" },
      { x: 0, y: 37, text: "-----------------------------------------------------------------------------------------------", role: "muted" },
    ],
    foreground: [
      { x: 2, y: 31, text: "{==}", role: "foregroundForeground" },
      { x: 90, y: 31, text: "{==}", role: "foregroundForeground" },
      { x: 0, y: 38, text: "##", role: "fighterShadow" },
      { x: 94, y: 38, text: "##", role: "fighterShadow" },
    ],
  },
  environment: {
    periodTicks: 240,
    frameAt(tick: number) {
      const frame = bannerFrames[Math.floor(Math.max(0, tick) / 120) % bannerFrames.length] ?? bannerFrames[0];
      return { x: 42, y: 12, text: frame, role: "main" };
    },
  },
} satisfies StageDefinition;
