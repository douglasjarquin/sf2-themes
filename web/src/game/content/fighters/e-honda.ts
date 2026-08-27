import type { FighterDefinition } from "../../types.ts";
import { FIGHTER_PALETTE_REGIONS, createGlyphFrame, type FighterContent } from "../../render/glyph-sprite.ts";

const frames = {
  "e-honda-idle-1": createGlyphFrame([
    "   @@@   ", "  /...\\  ", " / +++ \\ ", " |#####| ", " /|###|\\ ", "  |   |  ", " _/   \\_ ", "_|_   _|_",
  ]),
  "e-honda-idle-2": createGlyphFrame([
    "   @@@   ", "  /...\\  ", " / --- \\ ", " |#####| ", " /|###|\\ ", "  |   |  ", " _/   \\_ ", "_|_   _|_",
  ]),
  "e-honda-forward": createGlyphFrame([
    "   @@@     ", "  /...\\___ ", " / --   ___)", " |#####|    ", " /|###|\\    ", "  |   |_    ", " _/     \\_ ", "_|_     _|_",
  ]),
  "e-honda-backward": createGlyphFrame([
    "     @@@   ", " ___/...\\  ", "(___  -- \\ ", "    |#####| ", "   /|###|\\ ", "  _|   |   ", "_/     \\_  ", "_|_     _|_",
  ]),
  "e-honda-jump": createGlyphFrame([
    "   @@@   ", "  /...\\  ", " / ooo \\ ", "_|#####|_", "  /|###|\\ ", "   |   |  ", " _/     \\_", "/         \\",
  ]),
  "e-honda-attack": createGlyphFrame([
    "   @@@       ", "  /...\\_____ ", " / >>>    ___)", " |#####|____  ", " /|###|/      ", "  |   |       ", " _/   \\_      ", "_|_   _|_     ",
  ]),
  "e-honda-hit": createGlyphFrame([
    " \\ @@@ / ", "  \\.../  ", "   |xxx|  ", "  /#####\\ ", " / |###| /", "  / | | /", " _/   \\_ ", "_|_   _|_",
  ]),
  "e-honda-block": createGlyphFrame([
    "   @@@   ", "  /...\\  ", " _|---|_ ", "\\|#####|\\", " \\|###|/ ", "  |   |  ", " _/   \\_ ", "_|_   _|_",
  ]),
  "e-honda-ko": createGlyphFrame([
    "         ", "   @@@   ", "__/x x\\__", "\\  ---  /", " |#####| ", "_/ |#| \\_", "         ", "         ",
  ]),
  "e-honda-victory": createGlyphFrame([
    " \\ @@@ / ", "  \\.../  ", "   |+++|  ", "  /#####\\ ", " / |###| /", "   |   |  ", "  /     \\ ", " _|_   _|_",
  ]),
};

const definition = {
  id: "e-honda",
  displayName: "E. HONDA",
  maxHealth: 112,
  walkSpeed: 94,
  jumpVelocity: -850,
  gravity: 80,
  pushbox: { x: -340, y: -720, width: 680, height: 720 },
  hurtboxes: [
    { x: -300, y: -720, width: 600, height: 360 },
    { x: -340, y: -360, width: 680, height: 360 },
  ],
  animations: {
    idle: { frames: ["e-honda-idle-1", "e-honda-idle-2"], ticksPerFrame: 22, loop: true },
    walk: { frames: ["e-honda-forward", "e-honda-backward"], ticksPerFrame: 9, loop: true },
    jump: { frames: ["e-honda-jump"], ticksPerFrame: 9, loop: false },
    attack: { frames: ["e-honda-attack"], ticksPerFrame: 5, loop: false },
    hit: { frames: ["e-honda-hit"], ticksPerFrame: 6, loop: false },
    block: { frames: ["e-honda-block"], ticksPerFrame: 6, loop: false },
    ko: { frames: ["e-honda-ko"], ticksPerFrame: 13, loop: false },
    victory: { frames: ["e-honda-victory"], ticksPerFrame: 18, loop: true },
  },
  moves: [
    {
      id: "open-hand-slap", input: "light", startupTicks: 4,
      activeFrames: [{ hitId: 0, startTick: 4, endTick: 6, hitbox: { x: 240, y: -620, width: 560, height: 260 } }],
      recoveryTicks: 8, damage: 9, blockDamage: 1, hitstunTicks: 11, blockstunTicks: 6, hitstopTicks: 3, multiHit: false,
    },
    {
      id: "ringing-stomp", input: "heavy", startupTicks: 8,
      activeFrames: [
        { hitId: 0, startTick: 8, endTick: 9, hitbox: { x: 40, y: -260, width: 640, height: 260 } },
        { hitId: 1, startTick: 10, endTick: 11, hitbox: { x: 120, y: -220, width: 720, height: 220 } },
      ],
      recoveryTicks: 15, damage: 16, blockDamage: 2, hitstunTicks: 18, blockstunTicks: 10, hitstopTicks: 5, multiHit: true,
    },
    {
      id: "lantern-orb", input: "projectile", startupTicks: 13,
      activeFrames: [{ hitId: 0, startTick: 13, endTick: 14, hitbox: { x: 300, y: -520, width: 280, height: 260 } }],
      recoveryTicks: 17, damage: 12, blockDamage: 2, hitstunTicks: 14, blockstunTicks: 8, hitstopTicks: 4, multiHit: false,
    },
  ],
  projectile: {
    moveId: "lantern-orb", spawnTick: 13, offsetX: 640, offsetY: -460, velocityX: 165,
    hitbox: { x: 0, y: -150, width: 360, height: 300 }, damage: 12, blockDamage: 2,
    hitstunTicks: 14, blockstunTicks: 8, hitstopTicks: 4,
  },
  victory: { animation: "victory", quote: "Balance starts with a steady stance." },
  aiBias: { aggression: 64, defense: 69, projectile: 32, jump: 22 },
} satisfies FighterDefinition;

export const eHonda = {
  definition,
  glyphs: { frames },
  directionalFrames: { forward: ["e-honda-forward"], backward: ["e-honda-backward"] },
  paletteRegions: FIGHTER_PALETTE_REGIONS,
  metadata: { themeId: "e-honda", artOrigin: "original-project-authored" },
} satisfies FighterContent;
