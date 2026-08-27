import { advanceClock, createFixedStepClock, type FixedStepClock } from "./core/clock.ts";
import type { GameCore } from "./core/state-machine.ts";
import type { InputSource } from "./input/InputSource.ts";
import type { GameRenderer } from "./render/GameRenderer.ts";
import type { GamePalette } from "./render/palette.ts";
import type { GameSnapshot } from "./types.ts";

export function showArcadeFallback(root: HTMLElement, error: unknown): void {
  root.dataset.rendererStatus = "failed";
  root.dataset.loopStatus = "paused";
  const poster = root.querySelector("[data-game-poster]");
  if (poster instanceof HTMLElement) {
    poster.hidden = false;
    poster.dataset.failure = error instanceof Error ? error.name : "UnknownError";
  }
  const liveStatus = root.querySelector("[data-game-live-status]");
  if (liveStatus instanceof HTMLElement) {
    liveStatus.textContent = "ARCADE DISPLAY UNAVAILABLE - STATIC POSTER ACTIVE";
  }
  const gameStatus = root.querySelector("[data-game-status]");
  if (gameStatus instanceof HTMLElement) gameStatus.textContent = "STATIC FALLBACK";
}

export type BrowserGameHostOptions = {
  readonly root: HTMLElement;
  readonly core: GameCore;
  readonly renderer: GameRenderer;
  readonly input: InputSource;
  readonly palettes: ReadonlyMap<string, GamePalette>;
  readonly initialThemeId: string;
};

export class BrowserGameHost {
  private readonly root: HTMLElement;
  private readonly core: GameCore;
  private readonly renderer: GameRenderer;
  private readonly input: InputSource;
  private readonly palettes: ReadonlyMap<string, GamePalette>;
  private readonly renderTarget: HTMLElement;
  private readonly poster: HTMLElement;
  private readonly liveStatus: HTMLElement;
  private readonly creditStatus: HTMLElement;
  private readonly resizeObserver: ResizeObserver;
  private readonly intersectionObserver: IntersectionObserver;
  private activeThemeId: string;
  private clock: FixedStepClock = createFixedStepClock();
  private frameHandle: number | null = null;
  private lastFrameTime: number | null = null;
  private credits = 0;
  private intersecting = true;
  private rendererReady = false;
  private playerActivated = false;
  private readonly staticUntilActivation: boolean;
  private destroyed = false;

  constructor(options: BrowserGameHostOptions) {
    this.root = options.root;
    this.core = options.core;
    this.renderer = options.renderer;
    this.input = options.input;
    this.palettes = options.palettes;
    this.activeThemeId = options.initialThemeId;
    this.renderTarget = this.requiredElement("[data-game-renderer]");
    this.poster = this.requiredElement("[data-game-poster]");
    this.liveStatus = this.requiredElement("[data-game-live-status]");
    this.creditStatus = this.requiredElement("[data-game-credit-status]");
    this.staticUntilActivation = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    this.resizeObserver = new ResizeObserver((entries) => {
      const entry = entries[0];
      if (entry === undefined || !this.rendererReady) return;
      const { width, height } = entry.contentRect;
      if (width > 0 && height > 0) this.renderer.resize(width, height);
    });
    this.intersectionObserver = new IntersectionObserver((entries) => {
      const entry = entries[0];
      if (entry === undefined) return;
      this.intersecting = entry.isIntersecting;
      this.syncInput();
      this.syncScheduler();
    });
  }

  async start(): Promise<void> {
    this.root.dataset.rendererStatus = "mounting";
    this.root.dataset.loopStatus = "paused";
    this.bindListeners();
    this.resizeObserver.observe(this.renderTarget);
    this.intersectionObserver.observe(this.root);
    try {
      this.setTheme(this.activeThemeId);
      await this.renderer.mount(this.renderTarget);
      if (this.destroyed) return;
      this.rendererReady = true;
      this.root.dataset.rendererStatus = "ready";
      if (this.staticUntilActivation) {
        this.liveStatus.textContent = "PRESS INSERT COIN";
        this.setGameStatus("STATIC POSTER");
      } else {
        this.poster.hidden = true;
        this.render(this.core.advanceTicks(601));
      }
      this.syncScheduler();
    } catch (error) {
      this.failRenderer(error);
    }
  }

  setTheme(themeId: string): void {
    const palette = this.palettes.get(themeId);
    if (palette === undefined) return;
    this.activeThemeId = themeId;
    this.renderer.setPalette(palette);
    this.root.dataset.activeTheme = themeId;
    for (const chip of this.root.querySelectorAll<HTMLElement>("[data-game-theme-chip]")) {
      chip.setAttribute("aria-pressed", String(chip.dataset.themeId === themeId));
    }
    if (this.rendererReady) this.render(this.core.getSnapshot());
  }

  insertCoin(): void {
    if (!this.rendererReady || this.destroyed) return;
    this.playerActivated = true;
    this.root.focus({ preventScroll: true });
    this.input.setActive(true);
    this.poster.hidden = true;
    this.credits += 1;
    this.creditStatus.textContent = `CREDIT ${String(this.credits).padStart(2, "0")}`;
    if (this.core.getSnapshot().phase === "boot") this.core.advanceTicks(1);
    this.render(this.core.insertCoin());
    this.syncScheduler();
  }

  destroy(): void {
    if (this.destroyed) return;
    this.destroyed = true;
    this.teardown();
    this.root.dataset.rendererStatus = "destroyed";
    this.root.dataset.loopStatus = "destroyed";
  }

  private requiredElement(selector: string): HTMLElement {
    const element = this.root.querySelector(selector);
    if (!(element instanceof HTMLElement)) throw new TypeError(`ArcadeGame requires ${selector}`);
    return element;
  }

  private readonly onBlur = (): void => {
    this.input.setActive(false);
  };

  private readonly onVisibilityChange = (): void => {
    this.syncInput();
    this.syncScheduler();
  };

  private readonly onFocus = (): void => {
    this.syncInput();
  };

  private readonly onFocusOut = (): void => {
    this.input.setActive(false);
  };

  private readonly onCoinClick = (): void => {
    this.insertCoin();
  };

  private readonly onThemeClick = (event: Event): void => {
    if (!(event.currentTarget instanceof HTMLElement)) return;
    const themeId = event.currentTarget.dataset.themeId;
    if (themeId !== undefined) this.setTheme(themeId);
  };

  private readonly onDestroy = (): void => {
    this.destroy();
  };

  private bindListeners(): void {
    window.addEventListener("blur", this.onBlur);
    document.addEventListener("visibilitychange", this.onVisibilityChange);
    this.root.addEventListener("focus", this.onFocus);
    this.root.addEventListener("focusout", this.onFocusOut);
    this.root.querySelector("[data-game-coin]")?.addEventListener("click", this.onCoinClick);
    this.root.addEventListener("sf2-game:destroy", this.onDestroy);
    for (const chip of this.root.querySelectorAll("[data-game-theme-chip]")) {
      chip.addEventListener("click", this.onThemeClick);
    }
  }

  private unbindListeners(): void {
    window.removeEventListener("blur", this.onBlur);
    document.removeEventListener("visibilitychange", this.onVisibilityChange);
    this.root.removeEventListener("focus", this.onFocus);
    this.root.removeEventListener("focusout", this.onFocusOut);
    this.root.querySelector("[data-game-coin]")?.removeEventListener("click", this.onCoinClick);
    this.root.removeEventListener("sf2-game:destroy", this.onDestroy);
    for (const chip of this.root.querySelectorAll("[data-game-theme-chip]")) {
      chip.removeEventListener("click", this.onThemeClick);
    }
  }

  private syncScheduler(): void {
    const activationReady = !this.staticUntilActivation || this.playerActivated;
    const runnable = activationReady && this.rendererReady && !this.destroyed && this.intersecting && !document.hidden;
    if (!runnable) {
      if (this.frameHandle !== null) cancelAnimationFrame(this.frameHandle);
      this.frameHandle = null;
      this.lastFrameTime = null;
      if (!this.destroyed) this.root.dataset.loopStatus = "paused";
      return;
    }
    if (this.frameHandle !== null) return;
    this.root.dataset.loopStatus = "scheduled";
    this.frameHandle = requestAnimationFrame(this.onFrame);
  }

  private readonly onFrame = (time: number): void => {
    this.frameHandle = null;
    if (this.destroyed || !this.rendererReady || !this.intersecting || document.hidden) {
      this.syncScheduler();
      return;
    }
    this.root.dataset.loopStatus = "running";
    const previous = this.lastFrameTime;
    this.lastFrameTime = time;
    if (previous !== null) {
      const elapsedMicroseconds = Math.max(0, Math.round((time - previous) * 1_000));
      const advanced = advanceClock(this.clock, elapsedMicroseconds);
      this.clock = advanced.clock;
      let snapshot = this.core.getSnapshot();
      for (let tick = 0; tick < advanced.ticks; tick += 1) {
        const input = this.input.read();
        if (input.insertCoin === true) this.credits += 1;
        snapshot = this.core.step(input);
      }
      if (advanced.ticks > 0) {
        this.creditStatus.textContent = `CREDIT ${String(this.credits).padStart(2, "0")}`;
        this.render(snapshot);
      }
    }
    this.frameHandle = requestAnimationFrame(this.onFrame);
  };

  private render(snapshot: GameSnapshot): void {
    try {
      this.renderer.render(snapshot);
      this.root.dataset.gameState = snapshot.phase;
      this.root.dataset.simulationTick = String(snapshot.tick);
      this.root.dataset.playerOne = snapshot.fighters[0].id;
      this.root.dataset.playerTwo = snapshot.fighters[1].id;
      this.root.dataset.playerOnePose = snapshot.fighters[0].pose;
      this.root.dataset.playerOneMoveId = snapshot.fighters[0].moveId ?? "";
      this.root.dataset.playerOneX = String(snapshot.fighters[0].x);
      this.setGameStatus(this.playerActivated ? "PLAYER INPUT ACTIVE" : "DEMO MODE");
      this.liveStatus.textContent = snapshot.phase.startsWith("attract")
        ? "ATTRACT MODE"
        : snapshot.phase === "paused"
          ? "GAME PAUSED"
          : "PLAYER ONE READY";
    } catch (error) {
      this.failRenderer(error);
    }
  }

  private failRenderer(error: unknown): void {
    this.rendererReady = false;
    this.destroyed = true;
    this.teardown();
    showArcadeFallback(this.root, error);
  }

  private syncInput(): void {
    const active = this.playerActivated
      && this.intersecting
      && !document.hidden
      && document.activeElement === this.root;
    this.input.setActive(active);
  }

  private setGameStatus(status: string): void {
    const element = this.root.querySelector("[data-game-status]");
    if (element instanceof HTMLElement) element.textContent = status;
  }

  private teardown(): void {
    if (this.frameHandle !== null) cancelAnimationFrame(this.frameHandle);
    this.frameHandle = null;
    this.input.destroy();
    this.resizeObserver.disconnect();
    this.intersectionObserver.disconnect();
    this.unbindListeners();
    this.renderer.destroy();
  }
}
