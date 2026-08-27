import type { FighterDefinition } from "../../types.ts";
import { FIGHTER_PALETTE_REGIONS, createGlyphFrame, type FighterContent } from "../../render/glyph-sprite.ts";

const frames = {
  "akuma-idle-1": createGlyphFrame([
    "  ^@@^  ", " /....\\ ", " | ++ | ", " /|##|\\ ", "  |  |  ", " / /\\ \\ ", "_|_  _|_",
  ]),
  "akuma-idle-2": createGlyphFrame([
    " ^^@@^^ ", " /....\\ ", " | -- | ", " /|##|\\ ", "  |  |  ", " / /  \\ ", "_|_  _|_",
  ]),
  "akuma-forward": createGlyphFrame([
    "  ^@@^    ", " /....\\_  ", " | --  __) ", " /|##|/    ", "  |  |     ", " / / |_     ", "_|_   _|_   ",
  ]),
  "akuma-backward": createGlyphFrame([
    "    ^@@^  ", "  _/....\\ ", " (__  -- | ", "  \\|##|\\  ", "    |  |   ", "  _| \\ \\  ", " |_   _|_  ",
  ]),
  "akuma-jump": createGlyphFrame([
    "  ^@@^  ", " /....\\ ", " | oo | ", "_/|##|\\_", "  |  |  ", " _/  \\_ ", "/      \\",
  ]),
  "akuma-attack": createGlyphFrame([
    "  ^@@^       ", " /....\\_____ ", " | >>>    ___) ", " /|##|___/     ", "  |  |         ", " / /  \\        ", "_|_   _|_       ",
  ]),
  "akuma-hit": createGlyphFrame([
    " \\@@/  ", "  \\..\\ ", "   |xx| ", "  /|##|/", " / |  | ", "  / /\\  ", " _|_ |_ ",
  ]),
  "akuma-block": createGlyphFrame([
    "  ^@@^  ", " /....\\ ", "_| -- |_", "\\|++##|\\", " \\|  |/ ", "  / /\\  ", " _|_  |_",
  ]),
  "akuma-ko": createGlyphFrame([
    "        ", "  ^@@^  ", "_/xx  \\_", "\\  --  /", " |####| ", "_/    \\_", "        ",
  ]),
  "akuma-victory": createGlyphFrame([
    " \\ ^@@^ /", "  \\..../ ", "   |++|  ", "  /|##|\\ ", "   |  |  ", "  / / \\  ", " _|_  _|_",
  ]),
};

const definition = {
  id: "akuma",
  displayName: "AKUMA",
  maxHealth: 94,
  walkSpeed: 135,
  jumpVelocity: -970,
  gravity: 82,
  pushbox: { x: -280, y: -710, width: 560, height: 710 },
  hurtboxes: [
    { x: -240, y: -710, width: 480, height: 355 },
    { x: -280, y: -355, width: 560, height: 355 },
  ],
  animations: {
    idle: { frames: ["akuma-idle-1", "akuma-idle-2"], ticksPerFrame: 18, loop: true },
    walk: { frames: ["akuma-forward", "akuma-backward"], ticksPerFrame: 6, loop: true },
    jump: { frames: ["akuma-jump"], ticksPerFrame: 7, loop: false },
    attack: { frames: ["akuma-attack"], ticksPerFrame: 4, loop: false },
    hit: { frames: ["akuma-hit"], ticksPerFrame: 5, loop: false },
    block: { frames: ["akuma-block"], ticksPerFrame: 5, loop: false },
    ko: { frames: ["akuma-ko"], ticksPerFrame: 11, loop: false },
    victory: { frames: ["akuma-victory"], ticksPerFrame: 14, loop: true },
  },
  moves: [
    {
      id: "demon-palm", input: "light", startupTicks: 3,
      activeFrames: [{ hitId: 0, startTick: 3, endTick: 4, hitbox: { x: 240, y: -600, width: 540, height: 240 } }],
      recoveryTicks: 6, damage: 8, blockDamage: 1, hitstunTicks: 9, blockstunTicks: 6, hitstopTicks: 3, multiHit: false,
    },
    {
      id: "raging-arc", input: "heavy", startupTicks: 6,
      activeFrames: [
        { hitId: 0, startTick: 6, endTick: 7, hitbox: { x: 160, y: -700, width: 580, height: 340 } },
        { hitId: 1, startTick: 8, endTick: 9, hitbox: { x: 260, y: -540, width: 630, height: 260 } },
      ],
      recoveryTicks: 12, damage: 15, blockDamage: 2, hitstunTicks: 17, blockstunTicks: 9, hitstopTicks: 5, multiHit: true,
    },
    {
      id: "ember-orb", input: "projectile", startupTicks: 8,
      activeFrames: [{ hitId: 0, startTick: 8, endTick: 9, hitbox: { x: 300, y: -520, width: 280, height: 250 } }],
      recoveryTicks: 12, damage: 12, blockDamage: 2, hitstunTicks: 14, blockstunTicks: 8, hitstopTicks: 4, multiHit: false,
    },
  ],
  projectile: {
    moveId: "ember-orb", spawnTick: 8, offsetX: 610, offsetY: -450, velocityX: 250,
    hitbox: { x: 0, y: -150, width: 340, height: 290 }, damage: 12, blockDamage: 2,
    hitstunTicks: 14, blockstunTicks: 8, hitstopTicks: 4,
  },
  victory: { animation: "victory", quote: "Only the relentless earn the silence." },
  aiBias: { aggression: 82, defense: 39, projectile: 71, jump: 57 },
} satisfies FighterDefinition;

export const akuma = {
  definition,
  glyphs: { frames },
  directionalFrames: { forward: ["akuma-forward"], backward: ["akuma-backward"] },
  paletteRegions: FIGHTER_PALETTE_REGIONS,
  metadata: { themeId: "akuma", artOrigin: "original-project-authored" },
} satisfies FighterContent;
