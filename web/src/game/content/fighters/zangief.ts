import type { FighterDefinition } from "../../types.ts";
import { FIGHTER_PALETTE_REGIONS, createGlyphFrame, type FighterContent } from "../../render/glyph-sprite.ts";

const frames = {
  "zangief-idle-1": createGlyphFrame([
    "  @@@@@  ", " /.....\\ ", " | +++ | ", "/|#####|\\", " |#####| ", " / | | \\ ", "_/  |  \\_", "_|_   _|_",
  ]),
  "zangief-idle-2": createGlyphFrame([
    "  @@@@@  ", " /.....\\ ", " | --- | ", "/|#####|\\", " |#####| ", " / | | \\ ", "_/  |  \\_", "_|_   _|_",
  ]),
  "zangief-forward": createGlyphFrame([
    "  @@@@@     ", " /.....\\___ ", " | --    __)", "/|#####|/    ", " |#####|     ", " / | | |_    ", "_/  |    \\_ ", "_|_      _|_",
  ]),
  "zangief-backward": createGlyphFrame([
    "     @@@@@  ", " ___/.....\\ ", "(___  -- |  ", "  \\|#####|\\ ", "   |#####| ", "  _| | | \\ ", "_/    |  \\_", "_|_      _|_",
  ]),
  "zangief-jump": createGlyphFrame([
    "  @@@@@  ", " /.....\\ ", " | ooo | ", "_|#####|_", "/ |#####| /", "   | |   ", " _/   \\_ ", "/       \\",
  ]),
  "zangief-attack": createGlyphFrame([
    "  @@@@@       ", " /.....\\_____ ", " | >>>     __)", "/|#####|____   ", " |#####|/      ", " / | | \\      ", "_/  |  \\_     ", "_|_   _|_      ",
  ]),
  "zangief-hit": createGlyphFrame([
    "\\ @@@@@ /", " \\...../ ", "  | xxx |", " /|#####|", "/ |#####|", "  / | | ", " _/  |  /", "_|_   _|_",
  ]),
  "zangief-block": createGlyphFrame([
    "  @@@@@  ", " /.....\\ ", "_| --- |_", "\\|#####|\\", " \\|###|/ ", "  |###|  ", " _/ | \\_ ", "_|_   _|_",
  ]),
  "zangief-ko": createGlyphFrame([
    "         ", "  @@@@@  ", "_/x xxx\\_", "\\  ---  /", " |#####| ", "_/#####\\_", "         ", "         ",
  ]),
  "zangief-victory": createGlyphFrame([
    "\\ @@@@@ /", " \\...../ ", "  | +++ |", " /|#####|\\", "  |#####| ", "  / | | /", " _/  |  \\_", "_|_   _|_",
  ]),
};

const definition = {
  id: "zangief",
  displayName: "ZANGIEF",
  maxHealth: 118,
  walkSpeed: 88,
  jumpVelocity: -820,
  gravity: 80,
  pushbox: { x: -360, y: -740, width: 720, height: 740 },
  hurtboxes: [
    { x: -320, y: -740, width: 640, height: 360 },
    { x: -360, y: -380, width: 720, height: 380 },
  ],
  animations: {
    idle: { frames: ["zangief-idle-1", "zangief-idle-2"], ticksPerFrame: 24, loop: true },
    walk: { frames: ["zangief-forward", "zangief-backward"], ticksPerFrame: 10, loop: true },
    jump: { frames: ["zangief-jump"], ticksPerFrame: 10, loop: false },
    attack: { frames: ["zangief-attack"], ticksPerFrame: 6, loop: false },
    hit: { frames: ["zangief-hit"], ticksPerFrame: 7, loop: false },
    block: { frames: ["zangief-block"], ticksPerFrame: 7, loop: false },
    ko: { frames: ["zangief-ko"], ticksPerFrame: 14, loop: false },
    victory: { frames: ["zangief-victory"], ticksPerFrame: 20, loop: true },
  },
  moves: [
    {
      id: "body-hook", input: "light", startupTicks: 5,
      activeFrames: [{ hitId: 0, startTick: 5, endTick: 7, hitbox: { x: 250, y: -580, width: 580, height: 280 } }],
      recoveryTicks: 8, damage: 10, blockDamage: 1, hitstunTicks: 12, blockstunTicks: 7, hitstopTicks: 4, multiHit: false,
    },
    {
      id: "iron-wheel", input: "heavy", startupTicks: 9,
      activeFrames: [
        { hitId: 0, startTick: 9, endTick: 10, hitbox: { x: 120, y: -740, width: 620, height: 420 } },
        { hitId: 1, startTick: 11, endTick: 12, hitbox: { x: 230, y: -860, width: 600, height: 360 } },
      ],
      recoveryTicks: 16, damage: 17, blockDamage: 3, hitstunTicks: 19, blockstunTicks: 11, hitstopTicks: 6, multiHit: true,
    },
    {
      id: "red-comet", input: "projectile", startupTicks: 14,
      activeFrames: [{ hitId: 0, startTick: 14, endTick: 15, hitbox: { x: 310, y: -540, width: 300, height: 270 } }],
      recoveryTicks: 18, damage: 13, blockDamage: 2, hitstunTicks: 15, blockstunTicks: 9, hitstopTicks: 5, multiHit: false,
    },
  ],
  projectile: {
    moveId: "red-comet", spawnTick: 14, offsetX: 670, offsetY: -470, velocityX: 155,
    hitbox: { x: 0, y: -160, width: 380, height: 310 }, damage: 13, blockDamage: 2,
    hitstunTicks: 15, blockstunTicks: 9, hitstopTicks: 5,
  },
  victory: { animation: "victory", quote: "Strength is built one hard round at a time." },
  aiBias: { aggression: 78, defense: 58, projectile: 22, jump: 30 },
} satisfies FighterDefinition;

export const zangief = {
  definition,
  glyphs: { frames },
  directionalFrames: { forward: ["zangief-forward"], backward: ["zangief-backward"] },
  paletteRegions: FIGHTER_PALETTE_REGIONS,
  metadata: { themeId: "zangief", artOrigin: "original-project-authored" },
} satisfies FighterContent;
