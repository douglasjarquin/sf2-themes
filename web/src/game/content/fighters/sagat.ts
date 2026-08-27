import type { FighterDefinition } from "../../types.ts";
import { FIGHTER_PALETTE_REGIONS, createGlyphFrame, type FighterContent } from "../../render/glyph-sprite.ts";

const frames = {
  "sagat-idle-1": createGlyphFrame([
    "  @@@@@  ", " /.....\\ ", " | + | | ", " /|###|\\ ", "  | | |  ", " /  |  \\ ", "_/   |   \\_",
  ]),
  "sagat-idle-2": createGlyphFrame([
    " @@@@@@@ ", " /.....\\ ", " | - | | ", " /|###|\\ ", "  | | |  ", " /   |  \\ ", "_/    |  \\_",
  ]),
  "sagat-forward": createGlyphFrame([
    "  @@@@@     ", " /.....\\___ ", " | -  ____++)", " /|###|/    ", "  | | |     ", " /  | |_     ", "_/   |  |_   ",
  ]),
  "sagat-backward": createGlyphFrame([
    "    @@@@@  ", " ___/.....\\ ", "(++____ - | ", "    \\|###|\\ ", "     | | |  ", "     |  | \\ ", "    _|   |  \\_",
  ]),
  "sagat-jump": createGlyphFrame([
    " @@@@@@@ ", " /.....\\ ", " | o | | ", "_/|###|\\_", "  | | |  ", " _/ | \\_ ", "/___|___\\",
  ]),
  "sagat-attack": createGlyphFrame([
    "  @@@@@       ", " /.....\\______", " | >   ______++)", " /|###|/      ", "  | | |       ", " /  |  \\      ", "_/   |   \\_    ",
  ]),
  "sagat-hit": createGlyphFrame([
    "\\ @@@@@ /", " \\...../ ", "  | x | |", " /|###|/ ", " / | | | ", "  / | \\  ", " _/  |  \\_",
  ]),
  "sagat-block": createGlyphFrame([
    "  @@@@@  ", " /.....\\ ", "_| - | | ", "\\|+###|\\ ", " \\| | |  ", "  / | \\  ", " _/  |  \\_",
  ]),
  "sagat-ko": createGlyphFrame([
    "         ", " @@@@@@@ ", "_/x   x\\_", "\\  ---  /", " |#####| ", "_/     \\_", "         ",
  ]),
  "sagat-victory": createGlyphFrame([
    "\\ @@@@@ /", " \\...../ ", "  | + | |", " /|###|\\ ", "  | | |  ", " /  |  \\ ", "_/   |   \\_",
  ]),
};

const definition = {
  id: "sagat",
  displayName: "SAGAT",
  maxHealth: 112,
  walkSpeed: 102,
  jumpVelocity: -860,
  gravity: 84,
  pushbox: { x: -340, y: -780, width: 680, height: 780 },
  hurtboxes: [
    { x: -285, y: -780, width: 570, height: 390 },
    { x: -340, y: -390, width: 680, height: 390 },
  ],
  animations: {
    idle: { frames: ["sagat-idle-1", "sagat-idle-2"], ticksPerFrame: 22, loop: true },
    walk: { frames: ["sagat-forward", "sagat-backward"], ticksPerFrame: 8, loop: true },
    jump: { frames: ["sagat-jump"], ticksPerFrame: 8, loop: false },
    attack: { frames: ["sagat-attack"], ticksPerFrame: 5, loop: false },
    hit: { frames: ["sagat-hit"], ticksPerFrame: 6, loop: false },
    block: { frames: ["sagat-block"], ticksPerFrame: 6, loop: false },
    ko: { frames: ["sagat-ko"], ticksPerFrame: 12, loop: false },
    victory: { frames: ["sagat-victory"], ticksPerFrame: 18, loop: true },
  },
  moves: [
    { id: "long-knee", input: "light", startupTicks: 4, activeFrames: [{ hitId: 0, startTick: 4, endTick: 6, hitbox: { x: 240, y: -540, width: 580, height: 260 } }], recoveryTicks: 8, damage: 9, blockDamage: 1, hitstunTicks: 11, blockstunTicks: 6, hitstopTicks: 3, multiHit: false },
    { id: "temple-breaker", input: "heavy", startupTicks: 8, activeFrames: [{ hitId: 0, startTick: 8, endTick: 10, hitbox: { x: 170, y: -750, width: 680, height: 420 } }, { hitId: 1, startTick: 11, endTick: 12, hitbox: { x: 300, y: -600, width: 690, height: 300 } }], recoveryTicks: 15, damage: 16, blockDamage: 3, hitstunTicks: 18, blockstunTicks: 10, hitstopTicks: 5, multiHit: true },
    { id: "tiger-spark", input: "projectile", startupTicks: 13, activeFrames: [{ hitId: 0, startTick: 13, endTick: 14, hitbox: { x: 330, y: -530, width: 270, height: 250 } }], recoveryTicks: 17, damage: 13, blockDamage: 2, hitstunTicks: 15, blockstunTicks: 8, hitstopTicks: 4, multiHit: false },
  ],
  projectile: { moveId: "tiger-spark", spawnTick: 13, offsetX: 680, offsetY: -470, velocityX: 175, hitbox: { x: 0, y: -150, width: 390, height: 300 }, damage: 13, blockDamage: 2, hitstunTicks: 15, blockstunTicks: 8, hitstopTicks: 4 },
  victory: { animation: "victory", quote: "Height turns patience into power." },
  aiBias: { aggression: 62, defense: 58, projectile: 72, jump: 28 },
} satisfies FighterDefinition;

export const sagat = {
  definition,
  glyphs: { frames },
  directionalFrames: { forward: ["sagat-forward"], backward: ["sagat-backward"] },
  paletteRegions: FIGHTER_PALETTE_REGIONS,
  metadata: { themeId: "sagat", artOrigin: "original-project-authored" },
} satisfies FighterContent;
