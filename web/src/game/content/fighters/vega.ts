import type { FighterDefinition } from "../../types.ts";
import { FIGHTER_PALETTE_REGIONS, createGlyphFrame, type FighterContent } from "../../render/glyph-sprite.ts";

const frames = {
  "vega-idle-1": createGlyphFrame([
    "   ^^^   ", "  /...\\  ", "  | + |  ", " _/|#|\\_ ", "   | |   ", "  / / \\  ", "_/ /   \\_",
  ]),
  "vega-idle-2": createGlyphFrame([
    "  ^^^^^  ", "  /...\\  ", "  | - |  ", " _/|#|\\_ ", "   | |   ", "  /   \\  ", "_/     \\_",
  ]),
  "vega-forward": createGlyphFrame([
    "   ^^^      ", "  /...\\___  ", "  | -  ___++)", " _/|#|/      ", "   | |       ", "  / / |_      ", "_/ /   |_     ",
  ]),
  "vega-backward": createGlyphFrame([
    "      ^^^   ", "  ___/...\\  ", "(++___  - |  ", "     \\|#|\\_ ", "      | |   ", "      _| \\  ", "     _|   \\_",
  ]),
  "vega-jump": createGlyphFrame([
    "  ^^^^^  ", "  /...\\  ", "  | o |  ", "_/|#|\\__ ", "  / | \\  ", " / / \ \\ ", "/_/   \\_\\",
  ]),
  "vega-attack": createGlyphFrame([
    "   ^^^       ", "  /...\\___// ", "  | >   __/// ", " _/|#|/       ", "   | |        ", "  / / \\       ", "_/ /   \\_     ",
  ]),
  "vega-hit": createGlyphFrame([
    " \\ ^^^ / ", "  \\.../  ", "   |x |   ", "  /|#|\\  ", " / | | \\ ", "  / / \\  ", "_/ /   \\_",
  ]),
  "vega-block": createGlyphFrame([
    "   ^^^   ", "  /...\\  ", " _| - |  ", "\\|+#|\\  ", " \\| |    ", "  / / \\  ", "_/ /   \\_",
  ]),
  "vega-ko": createGlyphFrame([
    "        ", "  ^^^^^ ", "_/x x  \\_", "\\  -   /", " |#####|", "_/     \\_", "        ",
  ]),
  "vega-victory": createGlyphFrame([
    "\\  ^^^  /", " \\.../   ", "  | + |   ", " _/|#|\\_ ", "   | |    ", "  / / \\   ", "_/ /   \\_ ",
  ]),
};

const definition = {
  id: "vega",
  displayName: "VEGA",
  maxHealth: 94,
  walkSpeed: 142,
  jumpVelocity: -970,
  gravity: 76,
  pushbox: { x: -270, y: -710, width: 540, height: 710 },
  hurtboxes: [
    { x: -220, y: -710, width: 440, height: 330 },
    { x: -270, y: -380, width: 540, height: 380 },
  ],
  animations: {
    idle: { frames: ["vega-idle-1", "vega-idle-2"], ticksPerFrame: 20, loop: true },
    walk: { frames: ["vega-forward", "vega-backward"], ticksPerFrame: 6, loop: true },
    jump: { frames: ["vega-jump"], ticksPerFrame: 7, loop: false },
    attack: { frames: ["vega-attack"], ticksPerFrame: 4, loop: false },
    hit: { frames: ["vega-hit"], ticksPerFrame: 6, loop: false },
    block: { frames: ["vega-block"], ticksPerFrame: 6, loop: false },
    ko: { frames: ["vega-ko"], ticksPerFrame: 12, loop: false },
    victory: { frames: ["vega-victory"], ticksPerFrame: 16, loop: true },
  },
  moves: [
    { id: "claw-flick", input: "light", startupTicks: 3, activeFrames: [{ hitId: 0, startTick: 3, endTick: 4, hitbox: { x: 260, y: -600, width: 620, height: 190 } }], recoveryTicks: 6, damage: 8, blockDamage: 1, hitstunTicks: 9, blockstunTicks: 5, hitstopTicks: 3, multiHit: false },
    { id: "scarlet-roll", input: "heavy", startupTicks: 6, activeFrames: [{ hitId: 0, startTick: 6, endTick: 7, hitbox: { x: 180, y: -660, width: 570, height: 360 } }, { hitId: 1, startTick: 8, endTick: 9, hitbox: { x: 320, y: -570, width: 580, height: 280 } }], recoveryTicks: 12, damage: 13, blockDamage: 2, hitstunTicks: 15, blockstunTicks: 8, hitstopTicks: 4, multiHit: true },
    { id: "rose-spark", input: "projectile", startupTicks: 10, activeFrames: [{ hitId: 0, startTick: 10, endTick: 11, hitbox: { x: 310, y: -510, width: 240, height: 240 } }], recoveryTicks: 14, damage: 10, blockDamage: 2, hitstunTicks: 12, blockstunTicks: 7, hitstopTicks: 4, multiHit: false },
  ],
  projectile: { moveId: "rose-spark", spawnTick: 10, offsetX: 610, offsetY: -460, velocityX: 225, hitbox: { x: 0, y: -140, width: 340, height: 280 }, damage: 10, blockDamage: 2, hitstunTicks: 12, blockstunTicks: 7, hitstopTicks: 4 },
  victory: { animation: "victory", quote: "Grace cuts deeper than steel." },
  aiBias: { aggression: 67, defense: 42, projectile: 38, jump: 74 },
} satisfies FighterDefinition;

export const vega = {
  definition,
  glyphs: { frames },
  directionalFrames: { forward: ["vega-forward"], backward: ["vega-backward"] },
  paletteRegions: FIGHTER_PALETTE_REGIONS,
  metadata: { themeId: "vega", artOrigin: "original-project-authored" },
} satisfies FighterContent;
