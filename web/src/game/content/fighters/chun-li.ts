import type { FighterDefinition } from "../../types.ts";
import { FIGHTER_PALETTE_REGIONS, createGlyphFrame, type FighterContent } from "../../render/glyph-sprite.ts";

const frames = {
  "chun-li-idle-1": createGlyphFrame([
    "  o@@o  ", " /....\\ ", " | ++ | ", " /|##|\\ ", "  |  |  ", " / /\\ \\ ", "_|_  _|_",
  ]),
  "chun-li-idle-2": createGlyphFrame([
    "  o@@o  ", " /....\\ ", " | -- | ", " /|##|\\ ", "  |  |  ", " / /  \\ ", "_|_  _|_",
  ]),
  "chun-li-forward": createGlyphFrame([
    "  o@@o    ", " /....\\__ ", " | --  __)", " /|##|/   ", "  |  |    ", " / /  |_  ", "_|_    |_ ",
  ]),
  "chun-li-backward": createGlyphFrame([
    "   o@@o  ", " __/....\\ ", "(__  -- | ", "  \\|##|\\ ", "   |  |  ", "  _|  \\ ", "_|_    |_ ",
  ]),
  "chun-li-jump": createGlyphFrame([
    "  o@@o  ", " /....\\ ", " | oo | ", "_/|##|\\_", "  |  |  ", " _/  \\_ ", "/      \\",
  ]),
  "chun-li-attack": createGlyphFrame([
    "  o@@o      ", " /....\\____ ", " | >>   ___)", " /|##|_/    ", "  |  |      ", " / /  \\     ", "_|_   _|_    ",
  ]),
  "chun-li-hit": createGlyphFrame([
    " \\o@@o/ ", "  \\....\\", "   | xx|", "  /|##|/", " / |  | ", "  / /\\  ", " _|_  |_",
  ]),
  "chun-li-block": createGlyphFrame([
    "  o@@o  ", " /....\\ ", "_| -- | ", "\\|+##|\\ ", " \\|  |  ", "  / /\\  ", " _|_  |_",
  ]),
  "chun-li-ko": createGlyphFrame([
    "        ", "  o@@o  ", "_/x xx\\_", "\\  --  /", " |####| ", "_/    \\_", "        ",
  ]),
  "chun-li-victory": createGlyphFrame([
    " \\o@@o/ ", "  \\..../ ", "   |++|  ", "  /|##|\\ ", "   |  |  ", "  / / \\ ", " _|_  _|_",
  ]),
};

const definition = {
  id: "chun-li",
  displayName: "CHUN-LI",
  maxHealth: 96,
  walkSpeed: 138,
  jumpVelocity: -970,
  gravity: 80,
  pushbox: { x: -280, y: -700, width: 560, height: 700 },
  hurtboxes: [
    { x: -240, y: -700, width: 480, height: 340 },
    { x: -280, y: -360, width: 560, height: 360 },
  ],
  animations: {
    idle: { frames: ["chun-li-idle-1", "chun-li-idle-2"], ticksPerFrame: 18, loop: true },
    walk: { frames: ["chun-li-forward", "chun-li-backward"], ticksPerFrame: 6, loop: true },
    jump: { frames: ["chun-li-jump"], ticksPerFrame: 7, loop: false },
    attack: { frames: ["chun-li-attack"], ticksPerFrame: 4, loop: false },
    hit: { frames: ["chun-li-hit"], ticksPerFrame: 5, loop: false },
    block: { frames: ["chun-li-block"], ticksPerFrame: 5, loop: false },
    ko: { frames: ["chun-li-ko"], ticksPerFrame: 12, loop: false },
    victory: { frames: ["chun-li-victory"], ticksPerFrame: 15, loop: true },
  },
  moves: [
    {
      id: "crescent-palm", input: "light", startupTicks: 3,
      activeFrames: [{ hitId: 0, startTick: 3, endTick: 5, hitbox: { x: 220, y: -600, width: 520, height: 240 } }],
      recoveryTicks: 6, damage: 7, blockDamage: 1, hitstunTicks: 9, blockstunTicks: 5, hitstopTicks: 3, multiHit: false,
    },
    {
      id: "sky-kick", input: "heavy", startupTicks: 6,
      activeFrames: [
        { hitId: 0, startTick: 6, endTick: 7, hitbox: { x: 120, y: -760, width: 480, height: 380 } },
        { hitId: 1, startTick: 8, endTick: 9, hitbox: { x: 260, y: -860, width: 460, height: 300 } },
      ],
      recoveryTicks: 12, damage: 13, blockDamage: 2, hitstunTicks: 15, blockstunTicks: 8, hitstopTicks: 4, multiHit: true,
    },
    {
      id: "jade-spark", input: "projectile", startupTicks: 9,
      activeFrames: [{ hitId: 0, startTick: 9, endTick: 10, hitbox: { x: 260, y: -500, width: 260, height: 240 } }],
      recoveryTicks: 13, damage: 10, blockDamage: 2, hitstunTicks: 12, blockstunTicks: 7, hitstopTicks: 4, multiHit: false,
    },
  ],
  projectile: {
    moveId: "jade-spark", spawnTick: 9, offsetX: 560, offsetY: -440, velocityX: 225,
    hitbox: { x: 0, y: -140, width: 320, height: 260 }, damage: 10, blockDamage: 2,
    hitstunTicks: 12, blockstunTicks: 7, hitstopTicks: 4,
  },
  victory: { animation: "victory", quote: "Precision turns speed into victory." },
  aiBias: { aggression: 68, defense: 50, projectile: 44, jump: 62 },
} satisfies FighterDefinition;

export const chunLi = {
  definition,
  glyphs: { frames },
  directionalFrames: { forward: ["chun-li-forward"], backward: ["chun-li-backward"] },
  paletteRegions: FIGHTER_PALETTE_REGIONS,
  metadata: { themeId: "chun-li", artOrigin: "original-project-authored" },
} satisfies FighterContent;
