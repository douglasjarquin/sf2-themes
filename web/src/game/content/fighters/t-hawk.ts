import type { FighterDefinition } from "../../types.ts";
import { FIGHTER_PALETTE_REGIONS, createGlyphFrame, type FighterContent } from "../../render/glyph-sprite.ts";

const frames = {
  "t-hawk-idle-1": createGlyphFrame([
    "  [@@@]  ", " /.....\\ ", " | +++ | ", " /|###|\\ ", " / | | \\ ", "  /   \\  ", " _|   |_ ",
  ]),
  "t-hawk-idle-2": createGlyphFrame([
    "  [@@@]  ", " /.....\\ ", " | --- | ", " /|###|\\ ", "   | |   ", "  /     \\ ", "_|       |_",
  ]),
  "t-hawk-forward": createGlyphFrame([
    "  [@@@]    ", " /.....\\__ ", " | ---  ___)", " /|###|/    ", "   | |      ", "  /  |__    ", "_|      |_  ",
  ]),
  "t-hawk-backward": createGlyphFrame([
    "    [@@@]  ", " __/.....\\ ", "(___  --- | ", "   \\|###|\\ ", "     | |    ", "  __|  \\   ", "_|      |_  ",
  ]),
  "t-hawk-jump": createGlyphFrame([
    "  [@@@]  ", " /.....\\ ", " | ooo | ", "_/|###|\\_", "   | |   ", " _/   \\_ ", "/       \\",
  ]),
  "t-hawk-attack": createGlyphFrame([
    "  [@@@]       ", " /.....\\____  ", " | >>>    ___) ", " /|###|_/      ", "   | |         ", "  /   \\        ", "_|     |_       ",
  ]),
  "t-hawk-hit": createGlyphFrame([
    " \\[@@@]/ ", "  \\.....\\", "   |xxx |", "  /|###|/", " / | |  ", "   / \\  ", " _|   |_ ",
  ]),
  "t-hawk-block": createGlyphFrame([
    "  [@@@]  ", " /.....\\ ", "_| --- |_", "\\|+###|\\", " \\| | |/ ", "  /   \\  ", " _|   |_ ",
  ]),
  "t-hawk-ko": createGlyphFrame([
    "         ", "  [@@@]  ", "_/xx xx\\_", "\\  ---  /", " |#####| ", "_/     \\_", "         ",
  ]),
  "t-hawk-victory": createGlyphFrame([
    "\\ [@@@] /", " \\...../ ", "  |+++|  ", " /|###|\\ ", "   | |   ", "  /   \\  ", "_|     |_",
  ]),
};

const definition = {
  id: "t-hawk",
  displayName: "T. HAWK",
  maxHealth: 112,
  walkSpeed: 92,
  jumpVelocity: -850,
  gravity: 78,
  pushbox: { x: -360, y: -760, width: 720, height: 760 },
  hurtboxes: [
    { x: -310, y: -760, width: 620, height: 380 },
    { x: -350, y: -380, width: 700, height: 380 },
  ],
  animations: {
    idle: { frames: ["t-hawk-idle-1", "t-hawk-idle-2"], ticksPerFrame: 26, loop: true },
    walk: { frames: ["t-hawk-forward", "t-hawk-backward"], ticksPerFrame: 10, loop: true },
    jump: { frames: ["t-hawk-jump"], ticksPerFrame: 9, loop: false },
    attack: { frames: ["t-hawk-attack"], ticksPerFrame: 6, loop: false },
    hit: { frames: ["t-hawk-hit"], ticksPerFrame: 7, loop: false },
    block: { frames: ["t-hawk-block"], ticksPerFrame: 7, loop: false },
    ko: { frames: ["t-hawk-ko"], ticksPerFrame: 14, loop: false },
    victory: { frames: ["t-hawk-victory"], ticksPerFrame: 20, loop: true },
  },
  moves: [
    {
      id: "long-jab", input: "light", startupTicks: 5,
      activeFrames: [{ hitId: 0, startTick: 5, endTick: 7, hitbox: { x: 270, y: -640, width: 650, height: 260 } }],
      recoveryTicks: 8, damage: 9, blockDamage: 1, hitstunTicks: 11, blockstunTicks: 6, hitstopTicks: 3, multiHit: false,
    },
    {
      id: "condor-stomp", input: "heavy", startupTicks: 9,
      activeFrames: [
        { hitId: 0, startTick: 9, endTick: 10, hitbox: { x: 100, y: -760, width: 620, height: 420 } },
        { hitId: 1, startTick: 11, endTick: 12, hitbox: { x: 150, y: -330, width: 720, height: 260 } },
      ],
      recoveryTicks: 15, damage: 16, blockDamage: 3, hitstunTicks: 18, blockstunTicks: 10, hitstopTicks: 5, multiHit: true,
    },
    {
      id: "wind-totem", input: "projectile", startupTicks: 13,
      activeFrames: [{ hitId: 0, startTick: 13, endTick: 14, hitbox: { x: 340, y: -540, width: 300, height: 270 } }],
      recoveryTicks: 17, damage: 13, blockDamage: 2, hitstunTicks: 15, blockstunTicks: 9, hitstopTicks: 4, multiHit: false,
    },
  ],
  projectile: {
    moveId: "wind-totem", spawnTick: 13, offsetX: 680, offsetY: -460, velocityX: 165,
    hitbox: { x: 0, y: -160, width: 380, height: 320 }, damage: 13, blockDamage: 2,
    hitstunTicks: 15, blockstunTicks: 9, hitstopTicks: 4,
  },
  victory: { animation: "victory", quote: "The sky makes room for the fearless." },
  aiBias: { aggression: 61, defense: 63, projectile: 42, jump: 28 },
} satisfies FighterDefinition;

export const tHawk = {
  definition,
  glyphs: { frames },
  directionalFrames: { forward: ["t-hawk-forward"], backward: ["t-hawk-backward"] },
  paletteRegions: FIGHTER_PALETTE_REGIONS,
  metadata: { themeId: "t-hawk", artOrigin: "original-project-authored" },
} satisfies FighterContent;
