import { CAPTURE_MAX_TICKS } from "../config.ts";
import { FIGHTER_ROSTER } from "../content/fighters/index.ts";
import { createGameCore, type GameCore } from "../core/state-machine.ts";
import { TextmodeRenderer } from "../render/TextmodeRenderer.ts";
import { createGamePalette, type GamePalette } from "../render/palette.ts";
import type { GameSnapshot } from "../types.ts";

const CAPTURE_MODES = ["attract", "player"] as const;
const CAPTURE_MOMENTS = ["title", "intro", "fight", "ko", "victory"] as const;
const DEFAULT_SEED = "sf2-themes-capture";

type CaptureMode = (typeof CAPTURE_MODES)[number];
type CaptureMoment = (typeof CAPTURE_MOMENTS)[number];
type CaptureVisibility = "visible" | "hidden";

export type CaptureHud = {
  readonly playerOneHealth: number;
  readonly playerTwoHealth: number;
  readonly roundSeconds: number;
};

export type CaptureState = {
  readonly tick: number;
  readonly mode: CaptureMode;
  readonly moment: CaptureMoment;
  readonly transition: "none";
  readonly visibility: CaptureVisibility;
  readonly hud: CaptureHud;
  readonly theme: string;
  readonly seed: string;
  readonly stage: "dojo";
  readonly p1: string;
  readonly p2: string;
  readonly complete: boolean;
  readonly logicalSize: { readonly columns: 96; readonly rows: 40 };
};

export type CaptureBridge = {
  readonly ready: Promise<void>;
  readonly reset: (seed?: string | number) => GameSnapshot;
  readonly setTheme: (themeId: string) => string;
  readonly advanceTicks: (count: number) => GameSnapshot;
  readonly advanceUntil: (target: number) => GameSnapshot;
  readonly getSnapshot: () => GameSnapshot;
  readonly getCaptureState: () => CaptureState;
};

declare global {
  interface Window {
    __SF2_GAME__?: CaptureBridge;
  }
}

type CaptureOptions = {
  readonly root: HTMLElement;
  readonly searchParams: URLSearchParams;
};

type ParsedCaptureOptions = {
  readonly mode: CaptureMode;
  readonly moment: CaptureMoment;
  readonly theme: string;
  readonly seed: string;
  readonly seedValue: number;
  readonly playerOne: string;
  readonly playerTwo: string;
  readonly initialTick: number;
};

class CaptureBridgeError extends TypeError {
  constructor(message: string) {
    super(message);
    this.name = "CaptureBridgeError";
  }
}

function valueIn<T extends string>(value: string | null, values: readonly T[], fallback: T): T {
  return values.find((candidate) => candidate === value) ?? fallback;
}

function hashSeed(seed: string): number {
  let hash = 2_166_136_261;
  for (const character of seed) hash = Math.imul(hash ^ character.charCodeAt(0), 16_777_619);
  return hash >>> 0;
}

function parseSeed(value: string | null): { readonly label: string; readonly value: number } {
  if (value === null || value.length === 0) return { label: DEFAULT_SEED, value: hashSeed(DEFAULT_SEED) };
  if (/^\d+$/.test(value)) {
    const numeric = Number(value);
    if (Number.isSafeInteger(numeric)) return { label: value, value: numeric >>> 0 };
  }
  return { label: value, value: hashSeed(value) };
}

function parseTick(value: string | null): number {
  if (value === null || !/^\d+$/.test(value)) return 0;
  const tick = Number(value);
  return Number.isSafeInteger(tick) ? Math.min(tick, CAPTURE_MAX_TICKS) : 0;
}

function fighterId(value: string | null, fallback: string): string {
  return FIGHTER_ROSTER.find(({ definition }) => definition.id === value)?.definition.id ?? fallback;
}

function parseOptions(root: HTMLElement, searchParams: URLSearchParams, palettes: ReadonlyMap<string, GamePalette>): ParsedCaptureOptions {
  const playerOne = fighterId(searchParams.get("p1"), root.dataset.initialPlayerOne ?? "ryu");
  const fallbackPlayerTwo = playerOne === "ken" ? "ryu" : "ken";
  const requestedPlayerTwo = fighterId(searchParams.get("p2"), root.dataset.initialPlayerTwo ?? fallbackPlayerTwo);
  const playerTwo = requestedPlayerTwo === playerOne ? fallbackPlayerTwo : requestedPlayerTwo;
  const seed = parseSeed(searchParams.get("seed"));
  const requestedTheme = searchParams.get("theme");
  const theme = requestedTheme !== null && palettes.has(requestedTheme) ? requestedTheme : "main";
  return {
    mode: valueIn(searchParams.get("mode"), CAPTURE_MODES, "attract"),
    moment: valueIn(searchParams.get("moment"), CAPTURE_MOMENTS, "fight"),
    theme,
    seed: seed.label,
    seedValue: seed.value,
    playerOne,
    playerTwo,
    initialTick: parseTick(searchParams.get("tick")),
  };
}

function paletteMap(root: HTMLElement): ReadonlyMap<string, GamePalette> {
  const payload = root.querySelector("[data-game-palettes]");
  if (!(payload instanceof HTMLScriptElement) || payload.textContent === null) {
    throw new CaptureBridgeError("capture palette payload is missing");
  }
  const parsed: unknown = JSON.parse(payload.textContent);
  if (!Array.isArray(parsed)) throw new CaptureBridgeError("capture palette payload must be an array");
  const palettes = new Map<string, GamePalette>();
  for (const entry of parsed) {
    if (typeof entry !== "object" || entry === null || Array.isArray(entry)) {
      throw new CaptureBridgeError("capture palette entry must be an object");
    }
    const id = Reflect.get(entry, "id");
    if (typeof id !== "string" || id.length === 0) throw new CaptureBridgeError("capture palette entry requires an ID");
    palettes.set(id, createGamePalette(Reflect.get(entry, "tokens")));
  }
  return palettes;
}

function fighterDefinition(id: string): unknown {
  const fighter = FIGHTER_ROSTER.find(({ definition }) => definition.id === id);
  if (fighter === undefined) throw new CaptureBridgeError(`capture fighter ${id} is missing`);
  return fighter.definition;
}

class CaptureRuntime {
  private readonly root: HTMLElement;
  private readonly renderer: TextmodeRenderer;
  private readonly core: GameCore;
  private readonly palettes: ReadonlyMap<string, GamePalette>;
  private readonly options: ParsedCaptureOptions;
  private readonly onVisibilityChange = (): void => this.render(this.core.getSnapshot());
  private activeTheme: string;
  private destroyed = false;

  constructor(options: CaptureOptions) {
    this.root = options.root;
    this.palettes = paletteMap(options.root);
    this.options = parseOptions(options.root, options.searchParams, this.palettes);
    this.activeTheme = this.options.theme;
    this.renderer = new TextmodeRenderer();
    this.core = createGameCore({
      seed: this.options.seedValue,
      fighters: [fighterDefinition(this.options.playerOne), fighterDefinition(this.options.playerTwo)],
    });
  }

  async start(): Promise<void> {
    this.root.dispatchEvent(new Event("sf2-game:destroy"));
    await document.fonts.ready;
    const palette = this.palettes.get(this.activeTheme);
    if (palette === undefined) throw new CaptureBridgeError(`capture theme ${this.activeTheme} is missing`);
    this.renderer.setPalette(palette);
    await this.renderer.mount(this.requiredElement("[data-game-renderer]"));
    const poster = this.root.querySelector("[data-game-poster]");
    if (poster instanceof HTMLElement) poster.hidden = true;
    document.addEventListener("visibilitychange", this.onVisibilityChange);
    this.core.reset(this.options.seedValue);
    if (this.options.initialTick > 0) this.core.advanceTicks(this.options.initialTick);
    this.render(this.core.getSnapshot());
  }

  reset(seed?: string | number): GameSnapshot {
    const parsed = typeof seed === "number" && Number.isSafeInteger(seed)
      ? { label: String(seed), value: seed >>> 0 }
      : parseSeed(typeof seed === "string" ? seed : this.options.seed);
    const snapshot = this.core.reset(parsed.value);
    this.render(snapshot);
    return snapshot;
  }

  setTheme(themeId: string): string {
    const nextTheme = this.palettes.has(themeId) ? themeId : "main";
    const palette = this.palettes.get(nextTheme);
    if (palette === undefined) throw new CaptureBridgeError(`capture theme ${nextTheme} is missing`);
    this.activeTheme = nextTheme;
    this.renderer.setPalette(palette);
    this.render(this.core.getSnapshot());
    return nextTheme;
  }

  advanceTicks(count: number): GameSnapshot {
    if (!Number.isSafeInteger(count) || count < 0) throw new RangeError("capture tick count must be a non-negative safe integer");
    const snapshot = this.core.advanceTicks(count);
    this.render(snapshot);
    return snapshot;
  }

  advanceUntil(target: number): GameSnapshot {
    if (!Number.isSafeInteger(target) || target < 0) throw new RangeError("capture target tick must be a non-negative safe integer");
    const current = this.core.getSnapshot().tick;
    return target <= current ? this.core.getSnapshot() : this.advanceTicks(target - current);
  }

  getSnapshot(): GameSnapshot {
    return this.core.getSnapshot();
  }

  getCaptureState(): CaptureState {
    const snapshot = this.core.getSnapshot();
    return {
      tick: snapshot.tick,
      mode: this.options.mode,
      moment: this.options.moment,
      transition: "none",
      visibility: document.hidden ? "hidden" : "visible",
      hud: {
        playerOneHealth: snapshot.fighters[0].health,
        playerTwoHealth: snapshot.fighters[1].health,
        roundSeconds: Math.ceil(snapshot.roundTicksRemaining / 60),
      },
      theme: this.activeTheme,
      seed: this.options.seed,
      stage: "dojo",
      p1: this.options.playerOne,
      p2: this.options.playerTwo,
      complete: snapshot.tick >= CAPTURE_MAX_TICKS,
      logicalSize: { columns: 96, rows: 40 },
    };
  }

  destroy(): void {
    if (this.destroyed) return;
    this.destroyed = true;
    document.removeEventListener("visibilitychange", this.onVisibilityChange);
    this.renderer.destroy();
  }

  private requiredElement(selector: string): HTMLElement {
    const element = this.root.querySelector(selector);
    if (!(element instanceof HTMLElement)) throw new CaptureBridgeError(`capture element ${selector} is missing`);
    return element;
  }

  private render(snapshot: GameSnapshot): void {
    if (this.destroyed) return;
    this.renderer.render(snapshot);
    this.root.dataset.captureTick = String(snapshot.tick);
    this.root.dataset.captureMode = this.options.mode;
    this.root.dataset.activeTheme = this.activeTheme;
    this.root.dataset.playerOne = this.options.playerOne;
    this.root.dataset.playerTwo = this.options.playerTwo;
    this.root.dataset.captureLogicalSize = "96x40";
    this.root.dataset.rendererStatus = "ready";
    this.root.dataset.loopStatus = "manual";
  }
}

export async function installCaptureBridge(root: HTMLElement, searchParams: URLSearchParams): Promise<void> {
  const runtime = new CaptureRuntime({ root, searchParams });
  const ready = runtime.start();
  const bridge: CaptureBridge = {
    ready,
    reset: (seed) => runtime.reset(seed),
    setTheme: (themeId) => runtime.setTheme(themeId),
    advanceTicks: (count) => runtime.advanceTicks(count),
    advanceUntil: (target) => runtime.advanceUntil(target),
    getSnapshot: () => runtime.getSnapshot(),
    getCaptureState: () => runtime.getCaptureState(),
  };
  window.__SF2_GAME__ = bridge;
  root.addEventListener("sf2-game:destroy", () => {
    runtime.destroy();
    if (window.__SF2_GAME__ === bridge) window.__SF2_GAME__ = undefined;
  }, { once: true });
  try {
    await ready;
  } catch (error) {
    window.__SF2_GAME__ = undefined;
    throw error;
  }
}
