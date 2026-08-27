import type { FighterDefinition } from "../../types.ts";
import { FIGHTER_PALETTE_REGIONS, createGlyphFrame, type FighterContent } from "../../render/glyph-sprite.ts";

const frames = {
  "ryu-idle-1": createGlyphFrame([
    "  @@@  ", " /...\\ ", " | + | ", " /|#|\\ ", "/ | | \\", "  / \\  ", " _| |_ ",
  ]),
  "ryu-idle-2": createGlyphFrame([
    "  @@@  ", " /...\\ ", " | - | ", " /|#|\\ ", "  | |  ", " /   \\ ", "_|   |_",
  ]),
  "ryu-forward": createGlyphFrame([
    "  @@@  ", " /...\\_", " | -  _)", " /|#|/ ", "  | |  ", " /  |_ ", "_|    |",
  ]),
  "ryu-backward": createGlyphFrame([
    "  @@@  ", "_/...\\ ", "(_  - |", " \\|#|\\ ", "  | |  ", " _|  \\ ", "/    |_",
  ]),
  "ryu-jump": createGlyphFrame([
    "  @@@  ", " /...\\ ", " | o | ", "_/|#|\\_", "  | |  ", " _/ \\_ ", "/     \\",
  ]),
  "ryu-attack": createGlyphFrame([
    "  @@@     ", " /...\\____", " | >  ___)", " /|#|/    ", "  | |     ", " /   \\    ", "_|   |_   ",
  ]),
  "ryu-hit": createGlyphFrame([
    " \\@@@/ ", "  \\...\\", "   |x |", "  /|#|/", " / | | ", "   / \\ ", " _|  |_",
  ]),
  "ryu-block": createGlyphFrame([
    "  @@@  ", " /...\\ ", "_| - | ", "\\|+#|\\ ", " \\| | ", "  / \\  ", " _| |_ ",
  ]),
  "ryu-ko": createGlyphFrame([
    "       ", "  @@@  ", "_/x x\\_", "\\  -  /", " |###| ", "_/   \\_", "       ",
  ]),
  "ryu-victory": createGlyphFrame([
    " \\ @@@ /", "  \\.../ ", "   |+|  ", "  /|#|\\ ", "   | |  ", "  /   \\ ", " _|   |_",
  ]),
};

const definition = {
  id: "ryu",
  displayName: "RYU",
  maxHealth: 100,
  walkSpeed: 110,
  jumpVelocity: -900,
  gravity: 80,
  pushbox: { x: -300, y: -700, width: 600, height: 700 },
  hurtboxes: [
    { x: -250, y: -700, width: 500, height: 350 },
    { x: -300, y: -350, width: 600, height: 350 },
  ],
  animations: {
    idle: { frames: ["ryu-idle-1", "ryu-idle-2"], ticksPerFrame: 24, loop: true },
    walk: { frames: ["ryu-forward", "ryu-backward"], ticksPerFrame: 8, loop: true },
    jump: { frames: ["ryu-jump"], ticksPerFrame: 8, loop: false },
    attack: { frames: ["ryu-attack"], ticksPerFrame: 5, loop: false },
    hit: { frames: ["ryu-hit"], ticksPerFrame: 6, loop: false },
    block: { frames: ["ryu-block"], ticksPerFrame: 6, loop: false },
    ko: { frames: ["ryu-ko"], ticksPerFrame: 12, loop: false },
    victory: { frames: ["ryu-victory"], ticksPerFrame: 18, loop: true },
  },
  moves: [
    {
      id: "straight-punch", input: "light", startupTicks: 4,
      activeFrames: [{ hitId: 0, startTick: 4, endTick: 6, hitbox: { x: 250, y: -600, width: 500, height: 250 } }],
      recoveryTicks: 7, damage: 8, blockDamage: 1, hitstunTicks: 10, blockstunTicks: 6, hitstopTicks: 3, multiHit: false,
    },
    {
      id: "sweeping-kick", input: "heavy", startupTicks: 8,
      activeFrames: [{ hitId: 0, startTick: 8, endTick: 11, hitbox: { x: 150, y: -250, width: 700, height: 250 } }],
      recoveryTicks: 13, damage: 14, blockDamage: 2, hitstunTicks: 16, blockstunTicks: 9, hitstopTicks: 5, multiHit: false,
    },
    {
      id: "focus-wave", input: "projectile", startupTicks: 12,
      activeFrames: [{ hitId: 0, startTick: 12, endTick: 13, hitbox: { x: 300, y: -500, width: 250, height: 250 } }],
      recoveryTicks: 16, damage: 12, blockDamage: 2, hitstunTicks: 14, blockstunTicks: 8, hitstopTicks: 4, multiHit: false,
    },
  ],
  projectile: {
    moveId: "focus-wave", spawnTick: 12, offsetX: 600, offsetY: -450, velocityX: 180,
    hitbox: { x: 0, y: -150, width: 350, height: 300 }, damage: 12, blockDamage: 2,
    hitstunTicks: 14, blockstunTicks: 8, hitstopTicks: 4,
  },
  victory: { animation: "victory", quote: "A clear mind finds the opening." },
  aiBias: { aggression: 58, defense: 52, projectile: 66, jump: 34 },
} satisfies FighterDefinition;

export const ryu = {
  definition,
  glyphs: { frames },
  directionalFrames: { forward: ["ryu-forward"], backward: ["ryu-backward"] },
  paletteRegions: FIGHTER_PALETTE_REGIONS,
  metadata: { themeId: "ryu", artOrigin: "original-project-authored" },
} satisfies FighterContent;
