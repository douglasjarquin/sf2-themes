import type { FighterDefinition, FighterId } from "../types.ts";

export const FIGHTER_PALETTE_REGIONS = ["primary", "secondary", "shadow", "highlight"] as const;

export type FighterPaletteRegion = (typeof FIGHTER_PALETTE_REGIONS)[number];

export type GlyphFrame = {
  readonly rows: readonly string[];
  readonly regions: readonly string[];
  readonly width: number;
  readonly height: number;
};

export type GlyphSprite = {
  readonly frames: Readonly<Record<string, GlyphFrame>>;
};

export type FighterContent = {
  readonly definition: FighterDefinition;
  readonly glyphs: GlyphSprite;
  readonly directionalFrames: {
    readonly forward: readonly string[];
    readonly backward: readonly string[];
  };
  readonly paletteRegions: readonly FighterPaletteRegion[];
  readonly metadata: {
    readonly themeId: FighterId;
    readonly artOrigin: "original-project-authored";
  };
};

const REGION_BY_GLYPH: Readonly<Record<string, string>> = {
  "#": "d",
  "+": "s",
  "@": "h",
};

export function createGlyphFrame(rows: readonly string[]): GlyphFrame {
  const width = Math.max(...rows.map((row) => row.length));
  const normalizedRows = rows.map((row) => row.padEnd(width));
  return {
    rows: normalizedRows,
    regions: normalizedRows.map((row) =>
      Array.from(row, (glyph) => glyph === " " ? " " : (REGION_BY_GLYPH[glyph] ?? "p")).join(""),
    ),
    width,
    height: normalizedRows.length,
  };
}
