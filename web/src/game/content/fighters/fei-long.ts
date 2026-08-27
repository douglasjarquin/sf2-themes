import type { FighterDefinition } from "../../types.ts";
import { FIGHTER_PALETTE_REGIONS, createGlyphFrame, type FighterContent } from "../../render/glyph-sprite.ts";

const frames = {
  "fei-long-idle-1": createGlyphFrame([
    "  {@@}  ", " /....\\ ", " | ++ | ", " /|##|\\ ", "  |  |  ", " / /\\ \\ ", "_|_  _|_",
  ]),
  "fei-long-idle-2": createGlyphFrame([
    "  {@@}  ", " /....\\ ", " | -- | ", " /|##|\\ ", "  |  |  ", " / /  \\ ", "_|_  _|_",
  ]),
  "fei-long-forward": createGlyphFrame([
    "  {@@}   ", " /....\\_ ", " | --  _) ", " /|##|/   ", "  |  |    ", " / / |_    ", "_|_   _|_  ",
  ]),
  "fei-long-backward": createGlyphFrame([
    "   {@@}  ", " _/....\\ ", "(_  -- | ", "  \\|##|\\ ", "   |  |  ", " _| \\ \\ ", "|_   _|_ ",
  ]),
  "fei-long-jump": createGlyphFrame([
    "  {@@}  ", " /....\\ ", " | oo | ", "_/|##|\\_", "  |  |  ", " _/  \\_ ", "/      \\",
  ]),
  "fei-long-attack": createGlyphFrame([
    "  {@@}       ", " /....\\_____ ", " | >>>    ___) ", " /|##|___/     ", "  |  |         ", " / /  \\        ", "_|_   _|_       ",
  ]),
  "fei-long-hit": createGlyphFrame([
    " \\@@/  ", "  \\..\\ ", "   |xx| ", "  /|##|/", " / |  | ", "  / /\\  ", " _|_ |_ ",
  ]),
  "fei-long-block": createGlyphFrame([
    "  {@@}  ", " /....\\ ", "_| -- |_", "\\|++##|\\", " \\|  |/ ", "  / /\\  ", " _|_  |_",
  ]),
  "fei-long-ko": createGlyphFrame([
    "        ", "  {@@}  ", "_/xx  \\_", "\\  --  /", " |####| ", "_/    \\_", "        ",
  ]),
  "fei-long-victory": createGlyphFrame([
    " \\ {@@} /", "  \\..../ ", "   |++|  ", "  /|##|\\ ", "   |  |  ", "  / / \\  ", " _|_  _|_",
  ]),
};

const definition = {
  id: "fei-long",
  displayName: "FEI LONG",
  maxHealth: 98,
  walkSpeed: 132,
  jumpVelocity: -930,
  gravity: 81,
  pushbox: { x: -280, y: -700, width: 560, height: 700 },
  hurtboxes: [
    { x: -240, y: -700, width: 480, height: 350 },
    { x: -280, y: -350, width: 560, height: 350 },
  ],
  animations: {
    idle: { frames: ["fei-long-idle-1", "fei-long-idle-2"], ticksPerFrame: 19, loop: true },
    walk: { frames: ["fei-long-forward", "fei-long-backward"], ticksPerFrame: 7, loop: true },
    jump: { frames: ["fei-long-jump"], ticksPerFrame: 8, loop: false },
    attack: { frames: ["fei-long-attack"], ticksPerFrame: 4, loop: false },
    hit: { frames: ["fei-long-hit"], ticksPerFrame: 5, loop: false },
    block: { frames: ["fei-long-block"], ticksPerFrame: 5, loop: false },
    ko: { frames: ["fei-long-ko"], ticksPerFrame: 11, loop: false },
    victory: { frames: ["fei-long-victory"], ticksPerFrame: 14, loop: true },
  },
  moves: [
    {
      id: "chain-fist", input: "light", startupTicks: 3,
      activeFrames: [{ hitId: 0, startTick: 3, endTick: 5, hitbox: { x: 230, y: -590, width: 520, height: 230 } }],
      recoveryTicks: 6, damage: 8, blockDamage: 1, hitstunTicks: 9, blockstunTicks: 6, hitstopTicks: 3, multiHit: false,
    },
    {
      id: "flying-heel", input: "heavy", startupTicks: 7,
      activeFrames: [
        { hitId: 0, startTick: 7, endTick: 8, hitbox: { x: 180, y: -690, width: 590, height: 300 } },
        { hitId: 1, startTick: 9, endTick: 10, hitbox: { x: 260, y: -540, width: 620, height: 250 } },
      ],
      recoveryTicks: 12, damage: 14, blockDamage: 2, hitstunTicks: 16, blockstunTicks: 9, hitstopTicks: 5, multiHit: true,
    },
    {
      id: "dragon-spark", input: "projectile", startupTicks: 10,
      activeFrames: [{ hitId: 0, startTick: 10, endTick: 11, hitbox: { x: 290, y: -520, width: 260, height: 250 } }],
      recoveryTicks: 14, damage: 11, blockDamage: 2, hitstunTicks: 13, blockstunTicks: 8, hitstopTicks: 4, multiHit: false,
    },
  ],
  projectile: {
    moveId: "dragon-spark", spawnTick: 10, offsetX: 600, offsetY: -450, velocityX: 220,
    hitbox: { x: 0, y: -150, width: 330, height: 280 }, damage: 11, blockDamage: 2,
    hitstunTicks: 13, blockstunTicks: 8, hitstopTicks: 4,
  },
  victory: { animation: "victory", quote: "Speed is the shape of discipline." },
  aiBias: { aggression: 75, defense: 45, projectile: 51, jump: 55 },
} satisfies FighterDefinition;

export const feiLong = {
  definition,
  glyphs: { frames },
  directionalFrames: { forward: ["fei-long-forward"], backward: ["fei-long-backward"] },
  paletteRegions: FIGHTER_PALETTE_REGIONS,
  metadata: { themeId: "fei-long", artOrigin: "original-project-authored" },
} satisfies FighterContent;
