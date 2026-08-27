import type { FighterDefinition } from "../../types.ts";
import { FIGHTER_PALETTE_REGIONS, createGlyphFrame, type FighterContent } from "../../render/glyph-sprite.ts";

const frames = {
  "blanka-idle-1": createGlyphFrame([
    " ^@@@@@^ ", "<.......>", " | +++ | ", " /|###|\\ ", "  | # |  ", " / / \\ \\ ", "_|_   _|_",
  ]),
  "blanka-idle-2": createGlyphFrame([
    " ^@@@@@^ ", "<.......>", " | --- | ", " /|###|\\ ", "  | # |  ", " / /   \\ ", "_|_   _|_",
  ]),
  "blanka-forward": createGlyphFrame([
    " ^@@@@@^   ", "<.......>__ ", " | --   ___)", " /|###|/    ", "  | # |     ", " / /  |_     ", "_|_    |_    ",
  ]),
  "blanka-backward": createGlyphFrame([
    "   ^@@@@@^ ", " __<.......>", "(___  -- |  ", "  \\|###|\\  ", "   | # |   ", "  _|  \\ \\  ", "_|_    |_  ",
  ]),
  "blanka-jump": createGlyphFrame([
    " ^@@@@@^ ", "<.......>", " | ooo | ", "_/|###|\\_", "  | # |  ", " _/   \\_ ", "/       \\",
  ]),
  "blanka-attack": createGlyphFrame([
    " ^@@@@@^     ", "<.......>____", " | >>>    __)", " /|###|__/   ", "  | # |      ", " / /   \\     ", "_|_   _|_    ",
  ]),
  "blanka-hit": createGlyphFrame([
    "\\^@@@@@^/", " \\<.......>", "   | xxx |", "  /|###|/ ", " / | # |  ", "  / /\\   ", " _|_  |_  ",
  ]),
  "blanka-block": createGlyphFrame([
    " ^@@@@@^ ", "<.......>", "_| --- |_", "\\|+###|\\ ", " \\| # |/ ", "  / /\\   ", " _|_  |_  ",
  ]),
  "blanka-ko": createGlyphFrame([
    "         ", " ^@@@@@^ ", "_<x xxx>_", "\\  ---  /", " |#####| ", "_/  #  \\_", "         ",
  ]),
  "blanka-victory": createGlyphFrame([
    "\\^@@@@@^/", " \\<.....>/ ", "   |+++|   ", "  /|###|\\  ", "   | # |   ", "  / / \\ \\  ", " _|_   _|_ ",
  ]),
};

const definition = {
  id: "blanka",
  displayName: "BLANKA",
  maxHealth: 104,
  walkSpeed: 118,
  jumpVelocity: -930,
  gravity: 80,
  pushbox: { x: -310, y: -680, width: 620, height: 680 },
  hurtboxes: [
    { x: -270, y: -680, width: 540, height: 330 },
    { x: -310, y: -350, width: 620, height: 350 },
  ],
  animations: {
    idle: { frames: ["blanka-idle-1", "blanka-idle-2"], ticksPerFrame: 16, loop: true },
    walk: { frames: ["blanka-forward", "blanka-backward"], ticksPerFrame: 7, loop: true },
    jump: { frames: ["blanka-jump"], ticksPerFrame: 7, loop: false },
    attack: { frames: ["blanka-attack"], ticksPerFrame: 4, loop: false },
    hit: { frames: ["blanka-hit"], ticksPerFrame: 5, loop: false },
    block: { frames: ["blanka-block"], ticksPerFrame: 5, loop: false },
    ko: { frames: ["blanka-ko"], ticksPerFrame: 11, loop: false },
    victory: { frames: ["blanka-victory"], ticksPerFrame: 14, loop: true },
  },
  moves: [
    {
      id: "low-swipe", input: "light", startupTicks: 3,
      activeFrames: [{ hitId: 0, startTick: 3, endTick: 4, hitbox: { x: 190, y: -360, width: 560, height: 250 } }],
      recoveryTicks: 6, damage: 8, blockDamage: 1, hitstunTicks: 10, blockstunTicks: 5, hitstopTicks: 3, multiHit: false,
    },
    {
      id: "canopy-spin", input: "heavy", startupTicks: 7,
      activeFrames: [
        { hitId: 0, startTick: 7, endTick: 8, hitbox: { x: 100, y: -650, width: 540, height: 400 } },
        { hitId: 1, startTick: 9, endTick: 10, hitbox: { x: 250, y: -560, width: 560, height: 420 } },
      ],
      recoveryTicks: 13, damage: 14, blockDamage: 2, hitstunTicks: 16, blockstunTicks: 9, hitstopTicks: 5, multiHit: true,
    },
    {
      id: "storm-seed", input: "projectile", startupTicks: 11,
      activeFrames: [{ hitId: 0, startTick: 11, endTick: 12, hitbox: { x: 260, y: -480, width: 260, height: 250 } }],
      recoveryTicks: 15, damage: 11, blockDamage: 2, hitstunTicks: 13, blockstunTicks: 7, hitstopTicks: 4, multiHit: false,
    },
  ],
  projectile: {
    moveId: "storm-seed", spawnTick: 11, offsetX: 590, offsetY: -420, velocityX: 195,
    hitbox: { x: 0, y: -140, width: 330, height: 280 }, damage: 11, blockDamage: 2,
    hitstunTicks: 13, blockstunTicks: 7, hitstopTicks: 4,
  },
  victory: { animation: "victory", quote: "The wild path has its own rhythm." },
  aiBias: { aggression: 76, defense: 28, projectile: 38, jump: 70 },
} satisfies FighterDefinition;

export const blanka = {
  definition,
  glyphs: { frames },
  directionalFrames: { forward: ["blanka-forward"], backward: ["blanka-backward"] },
  paletteRegions: FIGHTER_PALETTE_REGIONS,
  metadata: { themeId: "blanka", artOrigin: "original-project-authored" },
} satisfies FighterContent;
