import type { FighterDefinition } from "../../types.ts";
import { FIGHTER_PALETTE_REGIONS, createGlyphFrame, type FighterContent } from "../../render/glyph-sprite.ts";

const frames = {
  "dee-jay-idle-1": createGlyphFrame([
    "  ~@@~  ", " /....\\ ", " | ++ | ", " /|##|\\ ", "  |  |  ", " / /\\ \\ ", "_|_  _|_",
  ]),
  "dee-jay-idle-2": createGlyphFrame([
    "  ~@@~  ", " /....\\ ", " | -- | ", " /|##|\\ ", "  |  |  ", " / /  \\ ", "_|_  _|_",
  ]),
  "dee-jay-forward": createGlyphFrame([
    "  ~@@~   ", " /....\\_ ", " | --  _) ", " /|##|/   ", "  |  |    ", " / / |_    ", "_|_   _|_  ",
  ]),
  "dee-jay-backward": createGlyphFrame([
    "   ~@@~  ", " _/....\\ ", "(_  -- | ", "  \\|##|\\ ", "   |  |  ", " _| \\ \\ ", "|_   _|_ ",
  ]),
  "dee-jay-jump": createGlyphFrame([
    "  ~@@~  ", " /....\\ ", " | oo | ", "_/|##|\\_", "  |  |  ", " _/  \\_ ", "/      \\",
  ]),
  "dee-jay-attack": createGlyphFrame([
    "  ~@@~      ", " /....\\____ ", " | >>>   ___)", " /|##|__/    ", "  |  |       ", " / /  \\      ", "_|_   _|_     ",
  ]),
  "dee-jay-hit": createGlyphFrame([
    " \\@@/  ", "  \\..\\ ", "   |xx| ", "  /|##|/", " / |  | ", "  / /\\  ", " _|_ |_ ",
  ]),
  "dee-jay-block": createGlyphFrame([
    "  ~@@~  ", " /....\\ ", "_| -- |_", "\\|++##|\\", " \\|  |/ ", "  / /\\  ", " _|_  |_",
  ]),
  "dee-jay-ko": createGlyphFrame([
    "        ", "  ~@@~  ", "_/xx  \\_", "\\  --  /", " |####| ", "_/    \\_", "        ",
  ]),
  "dee-jay-victory": createGlyphFrame([
    " \\ ~@@~ /", "  \\..../ ", "   |++|  ", "  /|##|\\ ", "   |  |  ", "  / / \\  ", " _|_  _|_",
  ]),
};

const definition = {
  id: "dee-jay",
  displayName: "DEE JAY",
  maxHealth: 100,
  walkSpeed: 128,
  jumpVelocity: -920,
  gravity: 80,
  pushbox: { x: -280, y: -700, width: 560, height: 700 },
  hurtboxes: [
    { x: -240, y: -700, width: 480, height: 350 },
    { x: -280, y: -350, width: 560, height: 350 },
  ],
  animations: {
    idle: { frames: ["dee-jay-idle-1", "dee-jay-idle-2"], ticksPerFrame: 17, loop: true },
    walk: { frames: ["dee-jay-forward", "dee-jay-backward"], ticksPerFrame: 7, loop: true },
    jump: { frames: ["dee-jay-jump"], ticksPerFrame: 8, loop: false },
    attack: { frames: ["dee-jay-attack"], ticksPerFrame: 4, loop: false },
    hit: { frames: ["dee-jay-hit"], ticksPerFrame: 5, loop: false },
    block: { frames: ["dee-jay-block"], ticksPerFrame: 5, loop: false },
    ko: { frames: ["dee-jay-ko"], ticksPerFrame: 11, loop: false },
    victory: { frames: ["dee-jay-victory"], ticksPerFrame: 14, loop: true },
  },
  moves: [
    {
      id: "rhythm-jab", input: "light", startupTicks: 3,
      activeFrames: [{ hitId: 0, startTick: 3, endTick: 5, hitbox: { x: 240, y: -590, width: 510, height: 230 } }],
      recoveryTicks: 6, damage: 8, blockDamage: 1, hitstunTicks: 9, blockstunTicks: 6, hitstopTicks: 3, multiHit: false,
    },
    {
      id: "double-beat", input: "heavy", startupTicks: 6,
      activeFrames: [
        { hitId: 0, startTick: 6, endTick: 7, hitbox: { x: 180, y: -660, width: 560, height: 280 } },
        { hitId: 1, startTick: 8, endTick: 9, hitbox: { x: 280, y: -500, width: 610, height: 240 } },
      ],
      recoveryTicks: 12, damage: 14, blockDamage: 2, hitstunTicks: 16, blockstunTicks: 9, hitstopTicks: 5, multiHit: true,
    },
    {
      id: "sonic-spark", input: "projectile", startupTicks: 9,
      activeFrames: [{ hitId: 0, startTick: 9, endTick: 10, hitbox: { x: 280, y: -500, width: 250, height: 240 } }],
      recoveryTicks: 13, damage: 10, blockDamage: 1, hitstunTicks: 12, blockstunTicks: 7, hitstopTicks: 3, multiHit: false,
    },
  ],
  projectile: {
    moveId: "sonic-spark", spawnTick: 9, offsetX: 590, offsetY: -430, velocityX: 230,
    hitbox: { x: 0, y: -140, width: 310, height: 270 }, damage: 10, blockDamage: 1,
    hitstunTicks: 12, blockstunTicks: 7, hitstopTicks: 3,
  },
  victory: { animation: "victory", quote: "Keep the rhythm and own the round." },
  aiBias: { aggression: 69, defense: 48, projectile: 58, jump: 49 },
} satisfies FighterDefinition;

export const deeJay = {
  definition,
  glyphs: { frames },
  directionalFrames: { forward: ["dee-jay-forward"], backward: ["dee-jay-backward"] },
  paletteRegions: FIGHTER_PALETTE_REGIONS,
  metadata: { themeId: "dee-jay", artOrigin: "original-project-authored" },
} satisfies FighterContent;
