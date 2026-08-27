import type { FighterDefinition } from "../../types.ts";
import { FIGHTER_PALETTE_REGIONS, createGlyphFrame, type FighterContent } from "../../render/glyph-sprite.ts";

const frames = {
  "ken-idle-1": createGlyphFrame([
    " ^@@@@^ ", "<.....> ", " | + |  ", " /|##|\\ ", "  |  |  ", " / /\\ \\ ", "_|_  _|_",
  ]),
  "ken-idle-2": createGlyphFrame([
    "^^@@@@^^", "<.....> ", " | - |  ", " /|##|\\ ", "  |  |  ", " / /  \\ ", "_|_  _|_",
  ]),
  "ken-forward": createGlyphFrame([
    " ^@@@@^  ", "<.....>_ ", " | -  __)", " /|##|/  ", "  |  |   ", " / / |_  ", "_|    |_ ",
  ]),
  "ken-backward": createGlyphFrame([
    "  ^@@@@^ ", " _<.....>", "(__  - |", "  \\|##|\\ ", "   |  | ", "  _| \\ |", " |_    |_",
  ]),
  "ken-jump": createGlyphFrame([
    "^^@@@@^^", "<.....> ", " | o |  ", "_/|##|\\_", "  |  |  ", " _/  \\_ ", "/      \\",
  ]),
  "ken-attack": createGlyphFrame([
    " ^@@@@^    ", "<.....>____", " | >   ___)", " /|##|_/   ", "  |  |     ", " / /  \\    ", "_|_   _|_  ",
  ]),
  "ken-hit": createGlyphFrame([
    "\\^@@@@^/", " \\<.....>", "   | x |", "  /|##|/", " / |  | ", "  / /\\  ", " _|_  |_",
  ]),
  "ken-block": createGlyphFrame([
    " ^@@@@^ ", "<.....> ", "_| - |  ", "\\|+##|\\ ", " \\|  |  ", "  / /\\  ", " _|_  |_",
  ]),
  "ken-ko": createGlyphFrame([
    "        ", " ^@@@@^ ", "_<x x  >_", "\\  -   /", " |####| ", "_/    \\_", "        ",
  ]),
  "ken-victory": createGlyphFrame([
    "  ^@@@@^/", " <.....>/ ", "  | + |   ", " _/|##|\\  ", "/  |  |   ", "  / / \\   ", " _|_  _|_ ",
  ]),
};

const definition = {
  id: "ken",
  displayName: "KEN",
  maxHealth: 100,
  walkSpeed: 125,
  jumpVelocity: -940,
  gravity: 80,
  pushbox: { x: -300, y: -700, width: 600, height: 700 },
  hurtboxes: [
    { x: -275, y: -700, width: 550, height: 350 },
    { x: -300, y: -350, width: 600, height: 350 },
  ],
  animations: {
    idle: { frames: ["ken-idle-1", "ken-idle-2"], ticksPerFrame: 20, loop: true },
    walk: { frames: ["ken-forward", "ken-backward"], ticksPerFrame: 7, loop: true },
    jump: { frames: ["ken-jump"], ticksPerFrame: 8, loop: false },
    attack: { frames: ["ken-attack"], ticksPerFrame: 5, loop: false },
    hit: { frames: ["ken-hit"], ticksPerFrame: 6, loop: false },
    block: { frames: ["ken-block"], ticksPerFrame: 6, loop: false },
    ko: { frames: ["ken-ko"], ticksPerFrame: 12, loop: false },
    victory: { frames: ["ken-victory"], ticksPerFrame: 16, loop: true },
  },
  moves: [
    {
      id: "driving-punch", input: "light", startupTicks: 3,
      activeFrames: [{ hitId: 0, startTick: 3, endTick: 5, hitbox: { x: 250, y: -575, width: 550, height: 275 } }],
      recoveryTicks: 7, damage: 8, blockDamage: 1, hitstunTicks: 9, blockstunTicks: 6, hitstopTicks: 3, multiHit: false,
    },
    {
      id: "rising-heel", input: "heavy", startupTicks: 7,
      activeFrames: [
        { hitId: 0, startTick: 7, endTick: 8, hitbox: { x: 100, y: -700, width: 500, height: 450 } },
        { hitId: 1, startTick: 9, endTick: 10, hitbox: { x: 200, y: -900, width: 450, height: 400 } },
      ],
      recoveryTicks: 14, damage: 15, blockDamage: 2, hitstunTicks: 17, blockstunTicks: 9, hitstopTicks: 5, multiHit: true,
    },
    {
      id: "ember-wave", input: "projectile", startupTicks: 10,
      activeFrames: [{ hitId: 0, startTick: 10, endTick: 11, hitbox: { x: 300, y: -500, width: 250, height: 250 } }],
      recoveryTicks: 14, damage: 11, blockDamage: 2, hitstunTicks: 13, blockstunTicks: 7, hitstopTicks: 4, multiHit: false,
    },
  ],
  projectile: {
    moveId: "ember-wave", spawnTick: 10, offsetX: 600, offsetY: -450, velocityX: 210,
    hitbox: { x: 0, y: -150, width: 350, height: 300 }, damage: 11, blockDamage: 2,
    hitstunTicks: 13, blockstunTicks: 7, hitstopTicks: 4,
  },
  victory: { animation: "victory", quote: "Momentum belongs to the bold." },
  aiBias: { aggression: 72, defense: 38, projectile: 48, jump: 56 },
} satisfies FighterDefinition;

export const ken = {
  definition,
  glyphs: { frames },
  directionalFrames: { forward: ["ken-forward"], backward: ["ken-backward"] },
  paletteRegions: FIGHTER_PALETTE_REGIONS,
  metadata: { themeId: "ken", artOrigin: "original-project-authored" },
} satisfies FighterContent;
