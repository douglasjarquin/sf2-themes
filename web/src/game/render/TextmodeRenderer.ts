import { textmode, type TextmodeOptions } from "textmode.js";

import { LOGICAL_COLUMNS, LOGICAL_ROWS, MAX_DEVICE_PIXEL_RATIO, TICKS_PER_SECOND, WORLD_UNITS_PER_CELL } from "../config.ts";
import { FIGHTER_ROSTER } from "../content/fighters/index.ts";
import { dojo, type StageGlyph } from "../content/stages/dojo.ts";
import type { FighterSnapshot, GameSnapshot } from "../types.ts";
import type { GameRenderer } from "./GameRenderer.ts";
import type { FighterContent, GlyphFrame } from "./glyph-sprite.ts";
import type { GamePalette, GamePaletteRole } from "./palette.ts";

type Grid = { cols: number; rows: number };

type TextmodeContext = {
  readonly canvas: HTMLCanvasElement;
  readonly grid: Grid | undefined;
  setup(callback: () => void): Promise<void>;
  draw(callback: () => void): void;
  noLoop(): void;
  isLooping(): boolean;
  redraw(count?: number): void;
  resizeCanvas(width: number, height: number): void;
  fontSize(size: number): void;
  destroy(): void;
  background(color: string): void;
  charColor(color: string): void;
  cellColor(color: string): void;
  push(): void;
  pop(): void;
  translate(x: number, y: number): void;
  print(text: string, x: number, y: number, options?: { markup?: boolean }): void;
};

export type TextmodeFactory = (options: TextmodeOptions) => TextmodeContext;

export class RendererContractError extends TypeError {
  constructor(message: string) {
    super(message);
    this.name = "RendererContractError";
  }
}

function nativeFactory(options: TextmodeOptions): TextmodeContext {
  return textmode.create(options);
}

function fitGrid(width: number, height: number): { readonly width: number; readonly height: number } {
  if (!Number.isFinite(width) || !Number.isFinite(height) || width <= 0 || height <= 0) {
    throw new RendererContractError("resize dimensions must be positive finite numbers");
  }
  const ratio = LOGICAL_COLUMNS / LOGICAL_ROWS;
  const fittedWidth = Math.max(LOGICAL_COLUMNS, Math.floor(Math.min(width, height * ratio)));
  return { width: fittedWidth, height: Math.max(LOGICAL_ROWS, Math.floor(fittedWidth / ratio)) };
}

function fitCellSize(width: number, height: number, density: number): number {
  return Math.max(1, Math.floor(Math.min(
    width * density / LOGICAL_COLUMNS,
    height * density / LOGICAL_ROWS,
  )));
}

function validSnapshot(value: unknown): value is GameSnapshot {
  if (typeof value !== "object" || value === null || Array.isArray(value)) return false;
  const fighters = Reflect.get(value, "fighters");
  return Number.isSafeInteger(Reflect.get(value, "tick")) && Array.isArray(fighters) && fighters.length === 2;
}

function reverse(value: string): string {
  return Array.from(value).reverse().join("");
}

function fighterContent(id: string): FighterContent | undefined {
  return FIGHTER_ROSTER.find(({ definition }) => definition.id === id);
}

export class TextmodeRenderer implements GameRenderer {
  readonly logicalSize = { columns: LOGICAL_COLUMNS, rows: LOGICAL_ROWS };
  private readonly factory: TextmodeFactory;
  private context: TextmodeContext | null = null;
  private canvas: HTMLCanvasElement | null = null;
  private palette: GamePalette | null = null;
  private snapshot: GameSnapshot | null = null;
  private density = 1;

  constructor(factory: TextmodeFactory = nativeFactory) {
    this.factory = factory;
  }

  async mount(container: HTMLElement): Promise<void> {
    if (this.context !== null) throw new RendererContractError("renderer is already mounted");
    if (typeof container !== "object" || container === null || typeof container.append !== "function") {
      throw new RendererContractError("mount container must be an element");
    }
    const bounds = container.getBoundingClientRect();
    const size = fitGrid(bounds.width || LOGICAL_COLUMNS, bounds.height || LOGICAL_ROWS);
    const density = Math.min(globalThis.devicePixelRatio || 1, MAX_DEVICE_PIXEL_RATIO);
    const canvas = container.ownerDocument.createElement("canvas");
    canvas.width = size.width * density;
    canvas.height = size.height * density;
    canvas.style.display = "block";
    canvas.style.width = "100%";
    canvas.style.aspectRatio = `${LOGICAL_COLUMNS} / ${LOGICAL_ROWS}`;
    canvas.style.imageRendering = "pixelated";
    canvas.dataset.logicalGrid = `${LOGICAL_COLUMNS}x${LOGICAL_ROWS}`;
    container.append(canvas);
    const context = this.factory({
      canvas,
      fontSize: fitCellSize(size.width, size.height, density),
      pixelDensity: density,
      loadingScreen: { transition: "none" },
    });
    this.density = density;
    this.canvas = canvas;
    this.context = context;
    context.draw(() => this.drawFrame());
    context.noLoop();
    try {
      await context.setup(() => {
        const grid = context.grid;
        if (grid === undefined) throw new RendererContractError("textmode grid unavailable during setup");
        grid.cols = LOGICAL_COLUMNS;
        grid.rows = LOGICAL_ROWS;
      });
    } catch (error) {
      if (this.context === context) this.destroy();
      throw error;
    }
    if (this.context !== context) return;
    context.noLoop();
    canvas.dataset.rendererStatus = "ready";
    canvas.dataset.loopStatus = context.isLooping() ? "running" : "stopped";
  }

  resize(width: number, height: number): void {
    const context = this.context;
    if (context === null) throw new RendererContractError("renderer must be mounted before resize");
    const size = fitGrid(width, height);
    context.resizeCanvas(size.width, size.height);
    context.fontSize(fitCellSize(size.width, size.height, this.density));
    const grid = context.grid;
    if (grid !== undefined) {
      grid.cols = LOGICAL_COLUMNS;
      grid.rows = LOGICAL_ROWS;
    }
  }

  setPalette(palette: GamePalette): void {
    this.palette = palette;
  }

  render(snapshot: GameSnapshot): void {
    if (!validSnapshot(snapshot)) throw new RendererContractError("snapshot must contain a tick and exactly two fighters");
    if (this.context === null) throw new RendererContractError("renderer must be mounted before render");
    if (this.palette === null) throw new RendererContractError("palette must be set before render");
    this.snapshot = snapshot;
    this.context.redraw(1);
  }

  destroy(): void {
    const context = this.context;
    const canvas = this.canvas;
    this.context = null;
    this.canvas = null;
    this.snapshot = null;
    if (context !== null) context.destroy();
    canvas?.remove();
  }

  private drawFrame(): void {
    const context = this.context;
    const palette = this.palette;
    const snapshot = this.snapshot;
    if (context === null || palette === null || snapshot === null) return;
    context.background(palette.roles.background);
    context.push();
    context.translate(
      -Math.floor((this.logicalSize.columns - 1) / 2),
      -Math.floor(this.logicalSize.rows / 2),
    );
    try {
      for (const glyph of dojo.layers.distant) this.printStage(glyph);
      for (const glyph of dojo.layers.middle) this.printStage(glyph);
      for (const glyph of dojo.layers.floor) this.printStage(glyph);
      for (const glyph of dojo.layers.foreground) this.printStage(glyph);
      this.printStage(dojo.environment.frameAt(snapshot.tick));
      snapshot.fighters.forEach((fighter, index) => this.printFighter(fighter, index));
      for (const projectile of snapshot.projectiles) {
        this.print("o=>", Math.floor(projectile.x / WORLD_UNITS_PER_CELL), Math.floor(projectile.y / WORLD_UNITS_PER_CELL), "energy");
      }
      this.printHud(snapshot);
      this.printOverlays(snapshot);
    } finally {
      context.pop();
    }
  }

  private printStage(glyph: StageGlyph): void {
    this.print(glyph.text, glyph.x, glyph.y, glyph.role);
  }

  private print(text: string, x: number, y: number, role: GamePaletteRole): void {
    const context = this.context;
    const palette = this.palette;
    if (context === null || palette === null) return;
    context.charColor(palette.roles[role]);
    context.cellColor(palette.roles.background);
    context.print(text, x, y, { markup: false });
  }

  private printFighter(fighter: FighterSnapshot, player: number): void {
    const content = fighterContent(fighter.id);
    if (content === undefined) return;
    const animation = content.definition.animations[fighter.pose];
    let frameIds = animation.frames;
    if (fighter.pose === "walk" && fighter.velocityX !== 0) {
      frameIds = fighter.velocityX * fighter.facing > 0 ? content.directionalFrames.forward : content.directionalFrames.backward;
    }
    const frameId = frameIds[Math.floor(fighter.moveTick / animation.ticksPerFrame) % frameIds.length];
    const frame = frameId === undefined ? undefined : content.glyphs.frames[frameId];
    if (frame === undefined) return;
    const x = Math.floor(fighter.x / WORLD_UNITS_PER_CELL - frame.width / 2);
    const y = Math.floor(fighter.y / WORLD_UNITS_PER_CELL - frame.height);
    this.printGlyphFrame(frame, x, y, fighter.facing, player);
  }

  private printGlyphFrame(frame: GlyphFrame, x: number, y: number, facing: -1 | 1, player: number): void {
    frame.rows.forEach((sourceRow, rowIndex) => {
      const sourceRegions = frame.regions[rowIndex] ?? "";
      const row = facing === 1 ? sourceRow : reverse(sourceRow);
      const regions = facing === 1 ? sourceRegions : reverse(sourceRegions);
      Array.from(row).forEach((glyph, column) => {
        if (glyph === " ") return;
        const region = regions[column];
        const role = region === "d" ? "fighterShadow" : region === "s" ? "secondary" : region === "h" ? "fighterHighlight" : player === 0 ? "playerOne" : "playerTwo";
        this.print(glyph, x + column, y + rowIndex, role);
      });
    });
  }

  private printHud(snapshot: GameSnapshot): void {
    const first = snapshot.fighters[0];
    const second = snapshot.fighters[1];
    this.print(`P1 ${"=".repeat(Math.ceil(first.health / 5)).padEnd(20)} ${first.health}`, 2, 1, "playerOne");
    this.print(`${second.health} ${"=".repeat(Math.ceil(second.health / 5)).padEnd(20)} P2`, 65, 1, "playerTwo");
    this.print(String(Math.ceil(snapshot.roundTicksRemaining / TICKS_PER_SECOND)).padStart(2, "0"), 47, 1, "hudForeground");
  }

  private printOverlays(snapshot: GameSnapshot): void {
    if (snapshot.phase === "title") this.print("SF2 THEMES - INSERT COIN", 35, 20, "coin");
    if (snapshot.phase.startsWith("attract")) this.print("DEMO", 46, 4, "muted");
    if (snapshot.result !== null) this.print(snapshot.result === "draw" ? "DRAW" : "K.O.", 45, 18, "victory");
    if (snapshot.phase === "paused") this.print("PAUSED", 45, 18, "hudForeground");
  }
}
