import type { FighterDefinition } from "../../types.ts";
import { FIGHTER_PALETTE_REGIONS, createGlyphFrame, type FighterContent } from "../../render/glyph-sprite.ts";

const frames = {
  "m-bison-idle-1": createGlyphFrame([
    "  @@@@@  ", " /.....\\ ", " | +++ | ", " /|###|\\ ", "  | | |  ", " / /|\\ \\ ", "_|_ | _|_",
  ]),
  "m-bison-idle-2": createGlyphFrame([
    " @@@@@@@ ", " /.....\\ ", " | --- | ", " /|###|\\ ", "  | | |  ", " / / | \\ ", "_|_  | _|_",
  ]),
  "m-bison-forward": createGlyphFrame([
    "  @@@@@     ", " /.....\\___ ", " | -  ____++)", " /|###|/    ", "  | | |     ", " / /| |_     ", "_|_ |  |_    ",
  ]),
  "m-bison-backward": createGlyphFrame([
    "    @@@@@  ", " ___/.....\\ ", "(++____ - | ", "    \\|###|\\ ", "     | | |  ", "     | \\ | ", "    _|  |_ |_",
  ]),
  "m-bison-jump": createGlyphFrame([
    " @@@@@@@ ", " /.....\\ ", " | ooo | ", "_/|###|\\_", "  | | |  ", " _/ | \\_ ", "/___|___\\",
  ]),
  "m-bison-attack": createGlyphFrame([
    "  @@@@@        ", " /.....\\_______", " | >   _______++)", " /|###|/       ", "  | | |        ", " / /|\\ \\       ", "_|_ | _|_      ",
  ]),
  "m-bison-hit": createGlyphFrame([
    "\\ @@@@@ /", " \\...../ ", "  | xxx |", " /|###|/ ", " / | | | ", "  / /|\\  ", " _|_ | |_ ",
  ]),
  "m-bison-block": createGlyphFrame([
    "  @@@@@  ", " /.....\\ ", "_| --- | ", "\\|+###|\\ ", " \\| | |  ", "  / /|\\  ", " _|_ | |_ ",
  ]),
  "m-bison-ko": createGlyphFrame([
    "         ", " @@@@@@@ ", "_/x   x\\_", "\\  ---  /", " |#####| ", "_/     \\_", "         ",
  ]),
  "m-bison-victory": createGlyphFrame([
    "\\ @@@@@ /", " \\...../ ", "  | +++ |", " /|###|\\ ", "  | | |  ", " / /|\\ \\ ", "_|_ | _|_",
  ]),
};

const definition = {
  id: "m-bison",
  displayName: "M. BISON",
  maxHealth: 104,
  walkSpeed: 116,
  jumpVelocity: -900,
  gravity: 81,
  pushbox: { x: -310, y: -730, width: 620, height: 730 },
  hurtboxes: [
    { x: -260, y: -730, width: 520, height: 350 },
    { x: -310, y: -380, width: 620, height: 380 },
  ],
  animations: {
    idle: { frames: ["m-bison-idle-1", "m-bison-idle-2"], ticksPerFrame: 19, loop: true },
    walk: { frames: ["m-bison-forward", "m-bison-backward"], ticksPerFrame: 7, loop: true },
    jump: { frames: ["m-bison-jump"], ticksPerFrame: 8, loop: false },
    attack: { frames: ["m-bison-attack"], ticksPerFrame: 4, loop: false },
    hit: { frames: ["m-bison-hit"], ticksPerFrame: 6, loop: false },
    block: { frames: ["m-bison-block"], ticksPerFrame: 6, loop: false },
    ko: { frames: ["m-bison-ko"], ticksPerFrame: 12, loop: false },
    victory: { frames: ["m-bison-victory"], ticksPerFrame: 16, loop: true },
  },
  moves: [
    { id: "command-fist", input: "light", startupTicks: 4, activeFrames: [{ hitId: 0, startTick: 4, endTick: 6, hitbox: { x: 250, y: -590, width: 530, height: 250 } }], recoveryTicks: 7, damage: 9, blockDamage: 1, hitstunTicks: 10, blockstunTicks: 6, hitstopTicks: 3, multiHit: false },
    { id: "psycho-crusher", input: "heavy", startupTicks: 7, activeFrames: [{ hitId: 0, startTick: 7, endTick: 8, hitbox: { x: 160, y: -660, width: 650, height: 360 } }, { hitId: 1, startTick: 9, endTick: 10, hitbox: { x: 310, y: -600, width: 700, height: 310 } }], recoveryTicks: 13, damage: 15, blockDamage: 2, hitstunTicks: 17, blockstunTicks: 9, hitstopTicks: 5, multiHit: true },
    { id: "psycho-orb", input: "projectile", startupTicks: 11, activeFrames: [{ hitId: 0, startTick: 11, endTick: 12, hitbox: { x: 310, y: -510, width: 260, height: 260 } }], recoveryTicks: 15, damage: 12, blockDamage: 2, hitstunTicks: 14, blockstunTicks: 8, hitstopTicks: 4, multiHit: false },
  ],
  projectile: { moveId: "psycho-orb", spawnTick: 11, offsetX: 630, offsetY: -460, velocityX: 205, hitbox: { x: 0, y: -150, width: 370, height: 300 }, damage: 12, blockDamage: 2, hitstunTicks: 14, blockstunTicks: 8, hitstopTicks: 4 },
  victory: { animation: "victory", quote: "Order is simply power made visible." },
  aiBias: { aggression: 76, defense: 49, projectile: 58, jump: 46 },
} satisfies FighterDefinition;

export const mBison = {
  definition,
  glyphs: { frames },
  directionalFrames: { forward: ["m-bison-forward"], backward: ["m-bison-backward"] },
  paletteRegions: FIGHTER_PALETTE_REGIONS,
  metadata: { themeId: "m-bison", artOrigin: "original-project-authored" },
} satisfies FighterContent;
