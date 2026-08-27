import type { FighterDefinition } from "../../types.ts";
import { FIGHTER_PALETTE_REGIONS, createGlyphFrame, type FighterContent } from "../../render/glyph-sprite.ts";

const frames = {
  "balrog-idle-1": createGlyphFrame([
    "  @@@@@  ", " /.....\\ ", " | + + | ", " /|###|\\ ", "  | | |  ", " / / \\ \\ ", "_|_   _|_",
  ]),
  "balrog-idle-2": createGlyphFrame([
    " @@@@@@@ ", " /.....\\ ", " | - - | ", " /|###|\\ ", "  | | |  ", " / /   \\ ", "_|_   _|_",
  ]),
  "balrog-forward": createGlyphFrame([
    "  @@@@@    ", " /.....\\__ ", " | -  ____)", " /|###|/   ", "  | | |    ", " / /  |_   ", "_|_    |_  ",
  ]),
  "balrog-backward": createGlyphFrame([
    "   @@@@@  ", " __/.....\\ ", "(____  - | ", "   \\|###|\\ ", "    | | |  ", "   _|  \\ ", " _|_    |_",
  ]),
  "balrog-jump": createGlyphFrame([
    " @@@@@@@ ", " /.....\\ ", " | o o | ", "_/|###|\\_", "  | | |  ", " _/   \\_ ", "/_______\\",
  ]),
  "balrog-attack": createGlyphFrame([
    "  @@@@@      ", " /.....\\_____", " | >  _____++)", " /|###|/     ", "  | | |      ", " / /   \\     ", "_|_   _|_    ",
  ]),
  "balrog-hit": createGlyphFrame([
    "\\ @@@@@ /", " \\...../ ", "  | x x |", " /|###|/ ", " / | | | ", "  / / \\  ", " _|_  |_ ",
  ]),
  "balrog-block": createGlyphFrame([
    "  @@@@@  ", " /.....\\ ", "_| - - | ", "\\|+###|\\ ", " \\| | |  ", "  / / \\  ", " _|_  _|_",
  ]),
  "balrog-ko": createGlyphFrame([
    "         ", " @@@@@@@ ", "_/x   x\\_", "\\  ---  /", " |#####| ", "_/     \\_", "         ",
  ]),
  "balrog-victory": createGlyphFrame([
    "\\ @@@@@ /", " \\...../ ", "  | + + |", " /|###|\\ ", "  | | |  ", " / / \\ \\ ", "_|_   _|_",
  ]),
};

const definition = {
  id: "balrog",
  displayName: "BALROG",
  maxHealth: 108,
  walkSpeed: 118,
  jumpVelocity: -880,
  gravity: 82,
  pushbox: { x: -320, y: -740, width: 640, height: 740 },
  hurtboxes: [
    { x: -270, y: -740, width: 540, height: 350 },
    { x: -320, y: -390, width: 640, height: 390 },
  ],
  animations: {
    idle: { frames: ["balrog-idle-1", "balrog-idle-2"], ticksPerFrame: 18, loop: true },
    walk: { frames: ["balrog-forward", "balrog-backward"], ticksPerFrame: 7, loop: true },
    jump: { frames: ["balrog-jump"], ticksPerFrame: 8, loop: false },
    attack: { frames: ["balrog-attack"], ticksPerFrame: 4, loop: false },
    hit: { frames: ["balrog-hit"], ticksPerFrame: 6, loop: false },
    block: { frames: ["balrog-block"], ticksPerFrame: 6, loop: false },
    ko: { frames: ["balrog-ko"], ticksPerFrame: 12, loop: false },
    victory: { frames: ["balrog-victory"], ticksPerFrame: 16, loop: true },
  },
  moves: [
    { id: "body-jab", input: "light", startupTicks: 3, activeFrames: [{ hitId: 0, startTick: 3, endTick: 5, hitbox: { x: 270, y: -560, width: 510, height: 240 } }], recoveryTicks: 6, damage: 9, blockDamage: 1, hitstunTicks: 10, blockstunTicks: 6, hitstopTicks: 3, multiHit: false },
    { id: "charging-cross", input: "heavy", startupTicks: 7, activeFrames: [{ hitId: 0, startTick: 7, endTick: 9, hitbox: { x: 220, y: -620, width: 650, height: 310 } }, { hitId: 1, startTick: 10, endTick: 11, hitbox: { x: 400, y: -570, width: 520, height: 250 } }], recoveryTicks: 13, damage: 15, blockDamage: 2, hitstunTicks: 16, blockstunTicks: 9, hitstopTicks: 5, multiHit: true },
    { id: "glove-shot", input: "projectile", startupTicks: 11, activeFrames: [{ hitId: 0, startTick: 11, endTick: 12, hitbox: { x: 330, y: -520, width: 260, height: 260 } }], recoveryTicks: 15, damage: 11, blockDamage: 2, hitstunTicks: 13, blockstunTicks: 7, hitstopTicks: 4, multiHit: false },
  ],
  projectile: { moveId: "glove-shot", spawnTick: 11, offsetX: 640, offsetY: -470, velocityX: 190, hitbox: { x: 0, y: -150, width: 360, height: 300 }, damage: 11, blockDamage: 2, hitstunTicks: 13, blockstunTicks: 7, hitstopTicks: 4 },
  victory: { animation: "victory", quote: "The bell only rings for survivors." },
  aiBias: { aggression: 84, defense: 34, projectile: 26, jump: 42 },
} satisfies FighterDefinition;

export const balrog = {
  definition,
  glyphs: { frames },
  directionalFrames: { forward: ["balrog-forward"], backward: ["balrog-backward"] },
  paletteRegions: FIGHTER_PALETTE_REGIONS,
  metadata: { themeId: "balrog", artOrigin: "original-project-authored" },
} satisfies FighterContent;
