import type { FighterDefinition } from "../../types.ts";
import { FIGHTER_PALETTE_REGIONS, createGlyphFrame, type FighterContent } from "../../render/glyph-sprite.ts";

const frames = {
  "cammy-idle-1": createGlyphFrame([
    "  /@@\\  ", " /....\\ ", " | ++ | ", " /|##|\\ ", "  |  |  ", " / /\\ \\ ", "_|_  _|_",
  ]),
  "cammy-idle-2": createGlyphFrame([
    "  /@@\\  ", " /....\\ ", " | -- | ", " /|##|\\ ", "  |  |  ", "  /  \\  ", "_/    \\_",
  ]),
  "cammy-forward": createGlyphFrame([
    "  /@@\\   ", " /....\\_ ", " | --  _) ", " /|##|/   ", "  |  |    ", " / / |_    ", "_|_   _|_  ",
  ]),
  "cammy-backward": createGlyphFrame([
    "   /@@\\  ", " _/....\\ ", "(_  -- | ", "  \\|##|\\ ", "   |  |  ", " _| \\ \\ ", "|_   _|_ ",
  ]),
  "cammy-jump": createGlyphFrame([
    "  /@@\\  ", " /....\\ ", " | oo | ", "_/|##|\\_", "  |  |  ", " _/  \\_ ", "/      \\",
  ]),
  "cammy-attack": createGlyphFrame([
    "  /@@\\      ", " /....\\____  ", " | >>    __)  ", " /|##|__/     ", "  |  |        ", " / /  \\       ", "_|_   _|_      ",
  ]),
  "cammy-hit": createGlyphFrame([
    " \\@@/  ", "  \\..\\ ", "   |xx| ", "  /|##|/", " / |  | ", "  / /\\  ", " _|_ |_ ",
  ]),
  "cammy-block": createGlyphFrame([
    "  /@@\\  ", " /....\\ ", "_| -- |_", "\\|++##|\\", " \\|  |/ ", "  / /\\  ", " _|_  |_",
  ]),
  "cammy-ko": createGlyphFrame([
    "        ", "  /@@\\  ", "_/xx  \\_", "\\  --  /", " |####| ", "_/    \\_", "        ",
  ]),
  "cammy-victory": createGlyphFrame([
    " \\ /@@\\ /", "  /....\\ ", "   |++|  ", "  /|##|\\ ", "   |  |  ", "  / / \\  ", " _|_  _|_",
  ]),
};

const definition = {
  id: "cammy",
  displayName: "CAMMY",
  maxHealth: 96,
  walkSpeed: 142,
  jumpVelocity: -980,
  gravity: 82,
  pushbox: { x: -260, y: -680, width: 520, height: 680 },
  hurtboxes: [
    { x: -220, y: -680, width: 440, height: 330 },
    { x: -250, y: -350, width: 500, height: 350 },
  ],
  animations: {
    idle: { frames: ["cammy-idle-1", "cammy-idle-2"], ticksPerFrame: 18, loop: true },
    walk: { frames: ["cammy-forward", "cammy-backward"], ticksPerFrame: 6, loop: true },
    jump: { frames: ["cammy-jump"], ticksPerFrame: 7, loop: false },
    attack: { frames: ["cammy-attack"], ticksPerFrame: 4, loop: false },
    hit: { frames: ["cammy-hit"], ticksPerFrame: 5, loop: false },
    block: { frames: ["cammy-block"], ticksPerFrame: 5, loop: false },
    ko: { frames: ["cammy-ko"], ticksPerFrame: 11, loop: false },
    victory: { frames: ["cammy-victory"], ticksPerFrame: 15, loop: true },
  },
  moves: [
    {
      id: "quick-jab", input: "light", startupTicks: 3,
      activeFrames: [{ hitId: 0, startTick: 3, endTick: 4, hitbox: { x: 210, y: -570, width: 480, height: 220 } }],
      recoveryTicks: 6, damage: 7, blockDamage: 1, hitstunTicks: 8, blockstunTicks: 5, hitstopTicks: 2, multiHit: false,
    },
    {
      id: "spiral-kick", input: "heavy", startupTicks: 6,
      activeFrames: [
        { hitId: 0, startTick: 6, endTick: 7, hitbox: { x: 150, y: -600, width: 600, height: 300 } },
        { hitId: 1, startTick: 8, endTick: 9, hitbox: { x: 250, y: -500, width: 620, height: 260 } },
      ],
      recoveryTicks: 12, damage: 13, blockDamage: 2, hitstunTicks: 15, blockstunTicks: 8, hitstopTicks: 4, multiHit: true,
    },
    {
      id: "delta-shot", input: "projectile", startupTicks: 9,
      activeFrames: [{ hitId: 0, startTick: 9, endTick: 10, hitbox: { x: 280, y: -500, width: 260, height: 240 } }],
      recoveryTicks: 13, damage: 10, blockDamage: 1, hitstunTicks: 12, blockstunTicks: 7, hitstopTicks: 3, multiHit: false,
    },
  ],
  projectile: {
    moveId: "delta-shot", spawnTick: 9, offsetX: 560, offsetY: -430, velocityX: 240,
    hitbox: { x: 0, y: -140, width: 300, height: 260 }, damage: 10, blockDamage: 1,
    hitstunTicks: 12, blockstunTicks: 7, hitstopTicks: 3,
  },
  victory: { animation: "victory", quote: "Precision wins before the first strike." },
  aiBias: { aggression: 78, defense: 42, projectile: 35, jump: 64 },
} satisfies FighterDefinition;

export const cammy = {
  definition,
  glyphs: { frames },
  directionalFrames: { forward: ["cammy-forward"], backward: ["cammy-backward"] },
  paletteRegions: FIGHTER_PALETTE_REGIONS,
  metadata: { themeId: "cammy", artOrigin: "original-project-authored" },
} satisfies FighterContent;
