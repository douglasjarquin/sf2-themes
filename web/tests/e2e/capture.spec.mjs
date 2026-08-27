import { expect, test } from "@playwright/test";

const capturePath = "game/?capture=1&theme=ryu-light&seed=abc123&mode=attract&stage=dojo&p1=chun-li&p2=akuma&tick=120&moment=ko";

async function waitForCaptureBridge(page) {
  await expect
    .poll(() => page.evaluate(() => typeof window.__SF2_GAME__))
    .toBe("object");
  await page.evaluate(async () => {
    const bridge = window.__SF2_GAME__;
    if (bridge === undefined) throw new Error("capture bridge was not installed");
    await bridge.ready;
  });
}

test("capture mode exposes deterministic state and exact requested metadata", async ({ page }, testInfo) => {
  const consoleErrors = [];
  page.on("console", (message) => {
    if (message.type() === "error") consoleErrors.push(message.text());
  });
  await page.context().tracing.start({ screenshots: true, snapshots: true });
  try {
    await page.goto(capturePath);
    await waitForCaptureBridge(page);

    const result = await page.evaluate(() => {
      const bridge = window.__SF2_GAME__;
      if (bridge === undefined) throw new Error("capture bridge was not installed");
      const initialState = bridge.getCaptureState();
      const initialSnapshot = bridge.getSnapshot();
      const advancedSnapshot = bridge.advanceTicks(30);
      const untilSnapshot = bridge.advanceUntil(150);
      const changedTheme = bridge.setTheme("ken");
      const resetSnapshot = bridge.reset("abc123");
      const firstReplay = bridge.advanceTicks(30);
      bridge.reset("abc123");
      const secondReplay = bridge.advanceTicks(30);
      const root = document.querySelector("[data-arcade-game]");
      return {
        initialState,
        initialSnapshot,
        advancedSnapshot,
        untilSnapshot,
        changedTheme,
        resetSnapshot,
        firstReplay,
        secondReplay,
        canvas: document.querySelector("[data-game-renderer] canvas")?.dataset.logicalGrid,
        rendererStatus: root instanceof HTMLElement ? root.dataset.rendererStatus : "missing",
        loopStatus: root instanceof HTMLElement ? root.dataset.loopStatus : "missing",
        captureTick: root instanceof HTMLElement ? root.dataset.captureTick : "missing",
        themeCount: document.querySelectorAll("[data-game-theme-chip]").length,
      };
    });

    expect(result.initialState).toMatchObject({
      tick: 120,
      mode: "attract",
      moment: "ko",
      transition: "none",
      visibility: "visible",
      theme: "ryu-light",
      seed: "abc123",
      stage: "dojo",
      p1: "chun-li",
      p2: "akuma",
      logicalSize: { columns: 96, rows: 40 },
    });
    expect(result.initialSnapshot.tick).toBe(120);
    expect(result.advancedSnapshot.tick).toBe(150);
    expect(result.untilSnapshot.tick).toBe(150);
    expect(result.changedTheme).toBe("ken");
    expect(result.resetSnapshot.tick).toBe(0);
    expect(result.firstReplay).toEqual(result.secondReplay);
    expect(result.canvas).toBe("96x40");
    expect(result.rendererStatus).toBe("ready");
    expect(result.loopStatus).toBe("manual");
    expect(result.captureTick).toBe("30");
    expect(result.themeCount).toBe(36);
    expect(consoleErrors).toEqual([]);

    await page.screenshot({ path: testInfo.outputPath("todo8-capture.png"), fullPage: true });
  } finally {
    await page.context().tracing.stop({ path: testInfo.outputPath("todo8-capture.zip") });
  }
});

test("capture mode normalizes unknown query values without throwing", async ({ page }) => {
  await page.goto("game/?capture=1&theme=not-a-theme&seed=&mode=invalid&stage=wrong&p1=missing&p2=missing&tick=not-a-tick&moment=invalid");
  await waitForCaptureBridge(page);
  const state = await page.evaluate(() => window.__SF2_GAME__?.getCaptureState());
  expect(state).toMatchObject({
    tick: 0,
    mode: "attract",
    moment: "fight",
    theme: "main",
    seed: "sf2-themes-capture",
    stage: "dojo",
    p1: "ryu",
    p2: "ken",
  });
});

test("capture mode hides volatile site chrome while normal pages remain bridge-free", async ({ page }) => {
  await page.goto(capturePath);
  await waitForCaptureBridge(page);
  const captureDom = await page.evaluate(() => {
    const styleOf = (selector) => {
      const element = document.querySelector(selector);
      return element instanceof HTMLElement ? getComputedStyle(element).display : "missing";
    };
    return {
      header: styleOf(".site-header"),
      footer: styleOf(".site-footer"),
      intro: styleOf(".game-intro"),
      config: styleOf(".game-config"),
      visibleNav: [...document.querySelectorAll("nav")].filter((element) => element.getClientRects().length > 0).length,
      width: [document.documentElement.scrollWidth, document.documentElement.clientWidth],
    };
  });
  expect(captureDom).toMatchObject({ header: "none", footer: "none", intro: "none", config: "none", visibleNav: 0 });
  expect(captureDom.width[0]).toBe(captureDom.width[1]);

  await page.goto("game/");
  expect(await page.evaluate(() => window.__SF2_GAME__)).toBeUndefined();
  await page.goto("./");
  expect(await page.evaluate(() => window.__SF2_GAME__)).toBeUndefined();
});

test("capture bridge destroys cleanly and does not leak across remount", async ({ page }) => {
  await page.goto(capturePath);
  await waitForCaptureBridge(page);
  const beforeDestroy = await page.evaluate(() => {
    const root = document.querySelector("[data-arcade-game]");
    root?.dispatchEvent(new Event("sf2-game:destroy"));
    return {
      bridgeType: typeof window.__SF2_GAME__,
      canvasCount: document.querySelectorAll("[data-game-renderer] canvas").length,
      rendererStatus: root instanceof HTMLElement ? root.dataset.rendererStatus : "missing",
    };
  });
  expect(beforeDestroy.bridgeType).toBe("undefined");
  expect(beforeDestroy.canvasCount).toBe(0);
  await page.reload();
  await waitForCaptureBridge(page);
  expect(await page.evaluate(() => window.__SF2_GAME__?.getCaptureState().tick)).toBe(120);
});

test("capture renderer failure is bounded by the visible static poster fallback", async ({ page }) => {
  const pageErrors = [];
  page.on("pageerror", (error) => pageErrors.push(error.message));
  await page.addInitScript(() => {
    HTMLCanvasElement.prototype.getContext = () => null;
  });
  await page.goto(capturePath);
  await expect
    .poll(() => page.locator("[data-arcade-game]").getAttribute("data-capture-error"))
    .not.toBeNull();
  const fallback = await page.evaluate(() => {
    const root = document.querySelector("[data-arcade-game]");
    const poster = document.querySelector("[data-game-poster]");
    return {
      bridgeType: typeof window.__SF2_GAME__,
      renderer: root?.getAttribute("data-renderer-status"),
      loop: root?.getAttribute("data-loop-status"),
      posterHidden: poster?.hasAttribute("hidden"),
      liveStatus: document.querySelector("[data-game-live-status]")?.textContent,
    };
  });
  expect(fallback).toMatchObject({
    bridgeType: "undefined",
    renderer: "failed",
    loop: "paused",
    posterHidden: false,
    liveStatus: "CAPTURE DISPLAY UNAVAILABLE - STATIC POSTER ACTIVE",
  });
  expect(pageErrors).toEqual([]);
});
