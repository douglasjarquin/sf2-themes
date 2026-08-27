import type { FighterDefinition } from "../../types.ts";
import { FIGHTER_PALETTE_REGIONS, createGlyphFrame, type FighterContent } from "../../render/glyph-sprite.ts";

const frames = {
  "dhalsim-idle-1": createGlyphFrame([
    "   ooo   ", "  /...\\  ", "  | + |  ", " _/|#|\\_ ", "   | |   ", "  / | \\  ", "_/  |  \\_",
  ]),
  "dhalsim-idle-2": createGlyphFrame([
    "   ooo   ", "  /...\\  ", "  | - |  ", " _/|#|\\_ ", "   | |   ", " _/   \\_ ", "/_     _\\",
  ]),
  "dhalsim-forward": createGlyphFrame([
    "   ooo      ", "  /...\\____ ", "  | -  ____)", " _/|#|/     ", "   | |      ", "  /  |      ", "_/   |      ",
  ]),
  "dhalsim-backward": createGlyphFrame([
    "      ooo   ", " ____/...\\  ", "(____  - |  ", "     \\|#|\\_ ", "      | |   ", "      |  \\  ", "      |   \\_",
  ]),
  "dhalsim-jump": createGlyphFrame([
    "   ooo   ", "  /...\\  ", "  | o |  ", "_/|#|\\__ ", "  / | \\  ", " /  |  \\ ", "/___|___\\",
  ]),
  "dhalsim-attack": createGlyphFrame([
    "   ooo          ", "  /...\\________ ", "  | >   ________)", " _/|#|/         ", "   | |          ", "  / |           ", "_/  |           ",
  ]),
  "dhalsim-hit": createGlyphFrame([
    " \\ ooo / ", "  \\.../  ", "   |x |   ", "  /|#|\\  ", " / | | \\ ", "   / \\   ", " _/   \\_ ",
  ]),
  "dhalsim-block": createGlyphFrame([
    "   ooo   ", "  /...\\  ", " _| - |  ", "\\|+#|\\  ", " \\| |    ", "  / | \\  ", "_/  |  \\_",
  ]),
  "dhalsim-ko": createGlyphFrame([
    "        ", "   ooo  ", " _/x x\\_", "\\   -  /", " |#####|", "_/     \\_", "        ",
  ]),
  "dhalsim-victory": createGlyphFrame([
    "\\  ooo  /", " \\.../  ", "  | + |  ", "_/|#|\\_ ", "  | |    ", " / | \\   ", "/__|__\\  ",
  ]),
};

const definition = {
  id: "dhalsim",
  displayName: "DHALSIM",
  maxHealth: 96,
  walkSpeed: 92,
  jumpVelocity: -840,
  gravity: 78,
  pushbox: { x: -280, y: -720, width: 560, height: 720 },
  hurtboxes: [
    { x: -210, y: -720, width: 420, height: 330 },
    { x: -280, y: -390, width: 560, height: 390 },
  ],
  animations: {
    idle: { frames: ["dhalsim-idle-1", "dhalsim-idle-2"], ticksPerFrame: 26, loop: true },
    walk: { frames: ["dhalsim-forward", "dhalsim-backward"], ticksPerFrame: 10, loop: true },
    jump: { frames: ["dhalsim-jump"], ticksPerFrame: 8, loop: false },
    attack: { frames: ["dhalsim-attack"], ticksPerFrame: 5, loop: false },
    hit: { frames: ["dhalsim-hit"], ticksPerFrame: 6, loop: false },
    block: { frames: ["dhalsim-block"], ticksPerFrame: 6, loop: false },
    ko: { frames: ["dhalsim-ko"], ticksPerFrame: 12, loop: false },
    victory: { frames: ["dhalsim-victory"], ticksPerFrame: 18, loop: true },
  },
  moves: [
    { id: "stretching-jab", input: "light", startupTicks: 5, activeFrames: [{ hitId: 0, startTick: 5, endTick: 8, hitbox: { x: 260, y: -590, width: 760, height: 220 } }], recoveryTicks: 8, damage: 7, blockDamage: 1, hitstunTicks: 10, blockstunTicks: 6, hitstopTicks: 3, multiHit: false },
    { id: "flame-wheel", input: "heavy", startupTicks: 8, activeFrames: [{ hitId: 0, startTick: 8, endTick: 10, hitbox: { x: 120, y: -760, width: 620, height: 480 } }, { hitId: 1, startTick: 11, endTick: 13, hitbox: { x: 260, y: -650, width: 700, height: 300 } }], recoveryTicks: 14, damage: 14, blockDamage: 2, hitstunTicks: 17, blockstunTicks: 9, hitstopTicks: 5, multiHit: true },
    { id: "yoga-flare", input: "projectile", startupTicks: 12, activeFrames: [{ hitId: 0, startTick: 12, endTick: 13, hitbox: { x: 320, y: -500, width: 260, height: 250 } }], recoveryTicks: 17, damage: 12, blockDamage: 2, hitstunTicks: 14, blockstunTicks: 8, hitstopTicks: 4, multiHit: false },
  ],
  projectile: { moveId: "yoga-flare", spawnTick: 12, offsetX: 620, offsetY: -460, velocityX: 165, hitbox: { x: 0, y: -150, width: 380, height: 300 }, damage: 12, blockDamage: 2, hitstunTicks: 14, blockstunTicks: 8, hitstopTicks: 4 },
  victory: { animation: "victory", quote: "Stillness makes every reach possible." },
  aiBias: { aggression: 44, defense: 68, projectile: 78, jump: 24 },
} satisfies FighterDefinition;

export const dhalsim = {
  definition,
  glyphs: { frames },
  directionalFrames: { forward: ["dhalsim-forward"], backward: ["dhalsim-backward"] },
  paletteRegions: FIGHTER_PALETTE_REGIONS,
  metadata: { themeId: "dhalsim", artOrigin: "original-project-authored" },
} satisfies FighterContent;
