import type { FighterDefinition } from "../../types.ts";
import { FIGHTER_PALETTE_REGIONS, createGlyphFrame, type FighterContent } from "../../render/glyph-sprite.ts";

const frames = {
  "guile-idle-1": createGlyphFrame([
    " @@@@@@@ ", " |.....| ", " | +++ | ", " /|###|\\ ", "  | # |  ", " / / \\ \\ ", "_|_   _|_",
  ]),
  "guile-idle-2": createGlyphFrame([
    " @@@@@@@ ", " |.....| ", " | --- | ", " /|###|\\ ", "  | # |  ", " / /   \\ ", "_|_   _|_",
  ]),
  "guile-forward": createGlyphFrame([
    " @@@@@@@    ", " |.....|___ ", " | --    __)", " /|###|/    ", "  | # |     ", " / /  |_     ", "_|_    |_    ",
  ]),
  "guile-backward": createGlyphFrame([
    "    @@@@@@@ ", " ___|.....| ", "(___  -- | ", "  \\|###|\\ ", "   | # |  ", "  _|  \\ \\ ", "_|_    |_ ",
  ]),
  "guile-jump": createGlyphFrame([
    " @@@@@@@ ", " |.....| ", " | ooo | ", "_/|###|\\_", "  | # |  ", " _/   \\_ ", "/       \\",
  ]),
  "guile-attack": createGlyphFrame([
    " @@@@@@@      ", " |.....|_____ ", " | >>>    ___)", " /|###|_/     ", "  | # |       ", " / /   \\      ", "_|_   _|_     ",
  ]),
  "guile-hit": createGlyphFrame([
    "\\@@@@@@@/", " \\|.....|", "   | xxx|", "  /|###|/", " / | # | ", "  / /\\  ", " _|_  |_ ",
  ]),
  "guile-block": createGlyphFrame([
    " @@@@@@@ ", " |.....| ", "_| --- |_", "\\|+###|\\ ", " \\| # |/ ", "  / /\\   ", " _|_  |_  ",
  ]),
  "guile-ko": createGlyphFrame([
    "         ", " @@@@@@@ ", "_|x xxx|_", "\\  ---  /", " |#####| ", "_/  #  \\_", "         ",
  ]),
  "guile-victory": createGlyphFrame([
    "\\@@@@@@@/", " \\|...|/ ", "   |+++|  ", "  /|###|\\ ", "   | # |  ", "  / / \\ \\ ", " _|_   _|_",
  ]),
};

const definition = {
  id: "guile",
  displayName: "GUILE",
  maxHealth: 102,
  walkSpeed: 108,
  jumpVelocity: -890,
  gravity: 80,
  pushbox: { x: -300, y: -700, width: 600, height: 700 },
  hurtboxes: [
    { x: -260, y: -700, width: 520, height: 350 },
    { x: -300, y: -350, width: 600, height: 350 },
  ],
  animations: {
    idle: { frames: ["guile-idle-1", "guile-idle-2"], ticksPerFrame: 20, loop: true },
    walk: { frames: ["guile-forward", "guile-backward"], ticksPerFrame: 8, loop: true },
    jump: { frames: ["guile-jump"], ticksPerFrame: 8, loop: false },
    attack: { frames: ["guile-attack"], ticksPerFrame: 5, loop: false },
    hit: { frames: ["guile-hit"], ticksPerFrame: 6, loop: false },
    block: { frames: ["guile-block"], ticksPerFrame: 6, loop: false },
    ko: { frames: ["guile-ko"], ticksPerFrame: 12, loop: false },
    victory: { frames: ["guile-victory"], ticksPerFrame: 16, loop: true },
  },
  moves: [
    {
      id: "guard-jab", input: "light", startupTicks: 4,
      activeFrames: [{ hitId: 0, startTick: 4, endTick: 5, hitbox: { x: 230, y: -600, width: 540, height: 250 } }],
      recoveryTicks: 7, damage: 8, blockDamage: 1, hitstunTicks: 10, blockstunTicks: 6, hitstopTicks: 3, multiHit: false,
    },
    {
      id: "air-splitter", input: "heavy", startupTicks: 7,
      activeFrames: [
        { hitId: 0, startTick: 7, endTick: 8, hitbox: { x: 160, y: -720, width: 520, height: 400 } },
        { hitId: 1, startTick: 9, endTick: 10, hitbox: { x: 260, y: -840, width: 500, height: 330 } },
      ],
      recoveryTicks: 14, damage: 15, blockDamage: 2, hitstunTicks: 17, blockstunTicks: 9, hitstopTicks: 5, multiHit: true,
    },
    {
      id: "signal-wave", input: "projectile", startupTicks: 10,
      activeFrames: [{ hitId: 0, startTick: 10, endTick: 11, hitbox: { x: 280, y: -510, width: 260, height: 240 } }],
      recoveryTicks: 14, damage: 11, blockDamage: 2, hitstunTicks: 13, blockstunTicks: 7, hitstopTicks: 4, multiHit: false,
    },
  ],
  projectile: {
    moveId: "signal-wave", spawnTick: 10, offsetX: 610, offsetY: -450, velocityX: 205,
    hitbox: { x: 0, y: -150, width: 340, height: 290 }, damage: 11, blockDamage: 2,
    hitstunTicks: 13, blockstunTicks: 7, hitstopTicks: 4,
  },
  victory: { animation: "victory", quote: "Preparation makes the decisive second count." },
  aiBias: { aggression: 60, defense: 74, projectile: 71, jump: 35 },
} satisfies FighterDefinition;

export const guile = {
  definition,
  glyphs: { frames },
  directionalFrames: { forward: ["guile-forward"], backward: ["guile-backward"] },
  paletteRegions: FIGHTER_PALETTE_REGIONS,
  metadata: { themeId: "guile", artOrigin: "original-project-authored" },
} satisfies FighterContent;
