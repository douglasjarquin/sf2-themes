import { expect, test } from "@playwright/test";

import { dojo } from "../../src/game/content/stages/dojo.ts";
import { TextmodeRenderer } from "../../src/game/render/TextmodeRenderer.ts";

const themeIds = [
  "main",
  "main-light",
  "akuma-light",
  "akuma",
  "balrog-light",
  "balrog",
  "blanka-light",
  "blanka",
  "cammy-light",
  "cammy",
  "chun-li-light",
  "chun-li",
  "dee-jay-light",
  "dee-jay",
  "dhalsim-light",
  "dhalsim",
  "e-honda-light",
  "e-honda",
  "fei-long-light",
  "fei-long",
  "guile-light",
  "guile",
  "ken-light",
  "ken",
  "m-bison-light",
  "m-bison",
  "ryu-light",
  "ryu",
  "sagat-light",
  "sagat",
  "t-hawk-light",
  "t-hawk",
  "vega-light",
  "vega",
  "zangief-light",
  "zangief",
];

const fighterIds = [
  "ryu",
  "ken",
  "chun-li",
  "e-honda",
  "blanka",
  "zangief",
  "guile",
  "dhalsim",
  "balrog",
  "vega",
  "sagat",
  "m-bison",
  "cammy",
  "t-hawk",
  "fei-long",
  "dee-jay",
  "akuma",
];

function captureRuntimeErrors(page) {
  const errors = [];
  page.on("pageerror", (error) => errors.push(`page: ${error.message}`));
  page.on("console", (message) => {
    if (message.type() === "error") errors.push(`console: ${message.text()}`);
  });
  return errors;
}

// A fixed-tick simulation can run several ticks per animation frame, so a value this
// attribute takes on can appear and revert between two Node-side polls. Recording every
// distinct value via an in-page MutationObserver avoids racing the simulation's clock.
async function observeAttributeHistory(locator, attribute) {
  await locator.evaluate((element, attr) => {
    const store = (element.__attributeHistory ??= {});
    store[attr] = [element.getAttribute(attr) ?? ""];
    new MutationObserver((mutations) => {
      for (const mutation of mutations) {
        if (mutation.attributeName === attr) store[attr].push(element.getAttribute(attr) ?? "");
      }
    }).observe(element, { attributes: true, attributeFilter: [attr] });
  }, attribute);
}

function attributeHistoryIncludes(locator, attribute, value) {
  return locator.evaluate(
    (element, [attr, expected]) => (element.__attributeHistory?.[attr] ?? []).includes(expected),
    [attribute, value],
  );
}

async function numericAttribute(locator, name) {
  return Number(await locator.getAttribute(name));
}

async function logicalRowInk(canvas, logicalSize, row) {
  return canvas.evaluate((element, input) => {
    if (!(element instanceof HTMLCanvasElement)) throw new TypeError("renderer canvas missing");
    const scratch = document.createElement("canvas");
    scratch.width = element.width;
    scratch.height = element.height;
    const context = scratch.getContext("2d", { willReadFrequently: true });
    if (context === null) throw new TypeError("canvas pixel reader unavailable");
    context.drawImage(element, 0, 0);
    const pixels = context.getImageData(0, 0, scratch.width, scratch.height).data;
    const frequencies = new Map();
    for (let offset = 0; offset < pixels.length; offset += 4) {
      const color = pixels[offset] * 65_536 + pixels[offset + 1] * 256 + pixels[offset + 2];
      frequencies.set(color, (frequencies.get(color) ?? 0) + 1);
    }
    let background = 0;
    let backgroundCount = 0;
    for (const [color, count] of frequencies) {
      if (count > backgroundCount) {
        background = color;
        backgroundCount = count;
      }
    }
    const backgroundRed = background >> 16;
    const backgroundGreen = (background >> 8) & 255;
    const backgroundBlue = background & 255;
    const missingColumns = [];
    const inkPixels = [];
    const cellWidth = Math.max(1, Math.floor(scratch.width / input.columns));
    const cellHeight = Math.max(1, Math.floor(scratch.height / input.rows));
    const offsetX = Math.floor((scratch.width - cellWidth * input.columns) / 2);
    const offsetY = Math.floor((scratch.height - cellHeight * input.rows) / 2);
    const rowStart = Math.max(0, Math.floor(offsetY + input.row * cellHeight) - 1);
    const rowEnd = Math.min(scratch.height, rowStart + cellHeight + 2);
    for (let column = 0; column < input.columns; column += 1) {
      const columnStart = Math.max(0, offsetX + column * cellWidth - 2);
      const columnEnd = Math.min(scratch.width, offsetX + (column + 1) * cellWidth + 2);
      let ink = 0;
      for (let y = rowStart; y < rowEnd; y += 1) {
        for (let x = columnStart; x < columnEnd; x += 1) {
          const offset = (y * scratch.width + x) * 4;
          const distance = Math.abs(pixels[offset] - backgroundRed) + Math.abs(pixels[offset + 1] - backgroundGreen) + Math.abs(pixels[offset + 2] - backgroundBlue);
          if (pixels[offset + 3] > 0 && distance > 0) ink += 1;
        }
      }
      inkPixels.push(ink);
      if (ink === 0) missingColumns.push(column);
    }
    return {
      background,
      grid: { cellHeight, cellWidth, offsetX, offsetY },
      inkPixels,
      missingColumns,
    };
  }, { columns: logicalSize.columns, rows: logicalSize.rows, row });
}

test("the full game route exposes independent theme and fighter selectors", async ({ page }) => {
  const runtimeErrors = captureRuntimeErrors(page);
  await page.context().tracing.start({ screenshots: true, snapshots: true });
  await page.goto("game/");

  try {
    await expect(page.locator("[data-game-route]")).toHaveCount(1);
    const cabinet = page.locator("[data-arcade-game]");
    await expect(cabinet).toHaveCount(1);
    await expect(cabinet).toHaveAttribute("data-renderer-status", "ready");

    const theme = page.locator('select[name="game-theme"]');
    const playerOne = page.locator('select[name="player-one"]');
    const playerTwo = page.locator('select[name="player-two"]');
    await expect(theme.locator("option")).toHaveCount(36);
    await expect(playerOne.locator("option")).toHaveCount(17);
    await expect(playerTwo.locator("option")).toHaveCount(17);
    await expect(theme.locator("option").evaluateAll((options) => options.map((option) => option.value))).resolves.toEqual(themeIds);
    await expect(playerOne.locator("option").evaluateAll((options) => options.map((option) => option.value))).resolves.toEqual(fighterIds);
    await expect(playerTwo.locator("option").evaluateAll((options) => options.map((option) => option.value))).resolves.toEqual(fighterIds);

    await theme.selectOption("ryu-light");
    await expect(cabinet).toHaveAttribute("data-active-theme", "ryu-light");
    await expect(cabinet).toHaveAttribute("data-player-one", "ryu");
    await expect(cabinet).toHaveAttribute("data-player-two", "ken");

    await playerOne.selectOption("chun-li");
    await playerTwo.selectOption("akuma");
    await expect(playerOne).toHaveValue("chun-li");
    await expect(playerTwo).toHaveValue("akuma");
    await expect(page.locator("[data-game-selection-status]")).toHaveText("NEXT MATCH: CHUN-LI VS AKUMA");
    await expect(cabinet).toHaveAttribute("data-active-theme", "ryu-light");
    await expect(cabinet).toHaveAttribute("data-player-one", "ryu");
    await expect(cabinet).toHaveAttribute("data-player-two", "ken");

    await Promise.all([
      page.waitForURL("**/game/?theme=ryu-light&p1=chun-li&p2=akuma"),
      page.getByRole("button", { name: "START NEW MATCH" }).click(),
    ]);
    await expect(cabinet).toHaveAttribute("data-renderer-status", "ready");
    await expect(cabinet).toHaveAttribute("data-active-theme", "ryu-light");
    await expect(cabinet).toHaveAttribute("data-player-one", "chun-li");
    await expect(cabinet).toHaveAttribute("data-player-two", "akuma");

    await page.getByRole("button", { name: "INSERT COIN" }).click();
    await expect(cabinet).toHaveAttribute("data-game-state", "player-fight", { timeout: 5000 });
    const startX = Number(await cabinet.getAttribute("data-player-one-x"));
    await cabinet.press("ArrowRight");
    await expect
      .poll(() => cabinet.getAttribute("data-player-one-x"), { timeout: 5000 })
      .not.toBe(String(startX));
    await observeAttributeHistory(cabinet, "data-player-one-move-id");
    await cabinet.press("z");
    await expect
      .poll(() => attributeHistoryIncludes(cabinet, "data-player-one-move-id", "crescent-palm"), { timeout: 5000 })
      .toBe(true);
    await expect(page.locator("[data-game-live-status]")).toContainText("PLAYER ONE READY");
    expect(runtimeErrors).toEqual([]);
    await page.screenshot({ path: "test-results/todo7-game-route.png", fullPage: true });
    for (const capture of [
      { width: 375, height: 812, path: "test-results/todo7-game-route-375.png" },
      { width: 768, height: 900, path: "test-results/todo7-game-route-768.png" },
      { width: 1280, height: 900, path: "test-results/todo7-game-route-1280.png" },
    ]) {
      await page.setViewportSize(capture);
      await page.screenshot({ path: capture.path, fullPage: true });
    }
  } finally {
    await page.context().tracing.stop({ path: "test-results/todo7-game-route.zip" });
  }
});

test("live renderer draws the selected non-default fighter glyphs", async ({ page }, testInfo) => {
  const runtimeErrors = captureRuntimeErrors(page);
  await page.context().tracing.start({ screenshots: true, snapshots: true });
  try {
    await page.goto("game/?theme=main&p1=chun-li&p2=akuma");
    const cabinet = page.locator("[data-arcade-game]");
    await expect(cabinet).toHaveAttribute("data-renderer-status", "ready");
    await page.getByRole("button", { name: "INSERT COIN" }).click();
    await expect(cabinet).toHaveAttribute("data-game-state", "player-fight", { timeout: 5000 });

    const pixels = await page.evaluate(() => {
      const root = document.querySelector("[data-arcade-game]");
      const canvas = root?.querySelector("canvas");
      const payload = root?.querySelector("[data-game-palettes]");
      if (!(root instanceof HTMLElement) || !(canvas instanceof HTMLCanvasElement) || !(payload instanceof HTMLScriptElement)) {
        throw new Error("live renderer evidence requires cabinet canvas and palette payload");
      }
      const entries = JSON.parse(payload.textContent ?? "[]");
      const palette = entries.find((entry) => entry.id === "main")?.tokens;
      const context = canvas.getContext("webgl2") ?? canvas.getContext("webgl");
      if (palette === undefined || context === null) throw new Error("live renderer canvas context is unavailable");
      const rgb = (hex) => hex.slice(1).match(/.{2}/g)?.map((part) => Number.parseInt(part, 16)) ?? [];
      const p1 = rgb(palette.semantic.blue);
      const p2 = rgb(palette.semantic.orange);
      const image = new Uint8Array(canvas.width * canvas.height * 4);
      context.readPixels(0, 0, canvas.width, canvas.height, context.RGBA, context.UNSIGNED_BYTE, image);
      const count = (color) => {
        let matches = 0;
        for (let index = 0; index < image.length; index += 4) {
          if (image[index] === color[0] && image[index + 1] === color[1] && image[index + 2] === color[2]) matches += 1;
        }
        return matches;
      };
      return { width: canvas.width, height: canvas.height, playerOnePixels: count(p1), playerTwoPixels: count(p2) };
    });

    expect(pixels.width).toBeGreaterThan(0);
    expect(pixels.height).toBeGreaterThan(0);
    expect(pixels.playerOnePixels).toBeGreaterThan(0);
    expect(pixels.playerTwoPixels).toBeGreaterThan(0);
    expect(runtimeErrors).toEqual([]);
    await page.screenshot({ path: testInfo.outputPath("f2-repair-chun-li-akuma.png"), fullPage: true });
  } finally {
    await page.context().tracing.stop({ path: testInfo.outputPath("f2-repair-chun-li-akuma.zip") });
  }
});

test("unknown loadout values fail closed and controls remain reachable on mobile", async ({ page }) => {
  const runtimeErrors = captureRuntimeErrors(page);
  await page.setViewportSize({ width: 375, height: 812 });
  await page.goto("game/?theme=missing-theme&p1=missing-fighter&p2=missing-fighter");

  const cabinet = page.locator("[data-arcade-game]");
  await expect(cabinet).toHaveAttribute("data-renderer-status", "ready");
  await expect(page.locator('select[name="game-theme"]')).toHaveValue("main");
  await expect(page.locator('select[name="player-one"]')).toHaveValue("ryu");
  await expect(page.locator('select[name="player-two"]')).toHaveValue("ken");
  await expect(cabinet).toHaveAttribute("data-player-one", "ryu");
  await expect(cabinet).toHaveAttribute("data-player-two", "ken");
  await expect
    .poll(() => page.evaluate(() => document.documentElement.scrollWidth))
    .toBe(await page.evaluate(() => document.documentElement.clientWidth));

  await page.locator('select[name="game-theme"]').focus();
  await page.keyboard.press("Tab");
  await expect(page.locator('select[name="player-one"]')).toBeFocused();
  await page.keyboard.press("Tab");
  await expect(page.locator('select[name="player-two"]')).toBeFocused();
  await page.keyboard.press("Tab");
  await expect(page.getByRole("button", { name: "START NEW MATCH" })).toBeFocused();

  const fullscreen = page.getByRole("button", { name: "FULLSCREEN" });
  const fullscreenSupported = await page.evaluate(() => typeof HTMLElement.prototype.requestFullscreen === "function");
  if (fullscreenSupported) {
    await expect(fullscreen).toBeVisible();
    await fullscreen.click();
    await page.keyboard.press("Escape");
  } else {
    await expect(fullscreen).toBeHidden();
  }
  expect(runtimeErrors).toEqual([]);
});

test("game palette changes preserve fighters and outside text input keeps native arrows", async ({ page }) => {
  const runtimeErrors = captureRuntimeErrors(page);
  // Given: the full game route has its default loadout.
  await page.goto("game/");
  const cabinet = page.locator("[data-arcade-game]");
  await expect(cabinet).toHaveAttribute("data-active-theme", "main");
  await expect(cabinet).toHaveAttribute("data-player-one", "ryu");
  await expect(cabinet).toHaveAttribute("data-player-two", "ken");

  // When: the palette changes and focus moves to an outside text input.
  await page.locator('select[name="game-theme"]').selectOption("ken");
  await expect(cabinet).toHaveAttribute("data-active-theme", "ken");
  await expect(cabinet).toHaveAttribute("data-player-one", "ryu");
  await expect(cabinet).toHaveAttribute("data-player-two", "ken");
  await page.evaluate(() => {
    const input = document.createElement("input");
    input.value = "native";
    document.body.append(input);
    input.focus();
    input.setSelectionRange(6, 6);
  });
  await page.locator("input").press("ArrowLeft");

  // Then: the native control consumes its own keypress instead of the game.
  await expect
    .poll(() => page.locator("input").evaluate((input) => (input instanceof HTMLInputElement ? input.selectionStart : null)))
    .toBe(5);
  await expect(cabinet).not.toBeFocused();
  expect(runtimeErrors).toEqual([]);
});

test("the full game dojo grid remains visible at 375px and DPR 2", async ({ baseURL, browser }) => {
  // Given: a DPR-2 narrow browser on the full game route.
  const context = await browser.newContext({ deviceScaleFactor: 2, reducedMotion: "reduce", viewport: { width: 375, height: 812 } });
  const page = await context.newPage();
  try {
    await page.goto(new URL("game/", baseURL).toString());
    const cabinet = page.locator("[data-arcade-game]");
    await page.getByRole("button", { name: "INSERT COIN" }).click();
    await expect(cabinet).toHaveAttribute("data-game-state", "player-fight", { timeout: 5000 });
    const canvas = cabinet.locator("canvas");
    const metrics = await canvas.evaluate((element) => {
      const bounds = element.getBoundingClientRect();
      return {
        backingWidth: element.width,
        backingHeight: element.height,
        cssWidth: bounds.width,
        cssHeight: bounds.height,
        ratio: window.devicePixelRatio,
      };
    });

    // Then: the backing store has the DPR scale and the full rendered grid remains in view.
    expect(metrics.backingWidth / metrics.cssWidth).toBeCloseTo(metrics.ratio, 1);
    expect(metrics.backingHeight / metrics.cssHeight).toBeCloseTo(metrics.ratio, 1);
    expect(metrics.backingWidth).toBeGreaterThan(new TextmodeRenderer().logicalSize.columns);
    expect(metrics.backingHeight).toBeGreaterThan(new TextmodeRenderer().logicalSize.rows);
    const floorGlyph = dojo.layers.floor.reduce((longest, glyph) =>
      Array.from(glyph.text).length > Array.from(longest.text).length ? glyph : longest,
    );
    const logicalSize = new TextmodeRenderer().logicalSize;
    const ground = await logicalRowInk(canvas, logicalSize, floorGlyph.y);
    const floorEnd = floorGlyph.x + Array.from(floorGlyph.text).length;
    const missingFloorColumns = ground.missingColumns.filter(
      (column) => column >= floorGlyph.x && column < floorEnd,
    );
    expect(
      missingFloorColumns,
      `grid=${JSON.stringify(ground.grid)}; missing floor ink in authored columns ${missingFloorColumns.join(", ")}`,
    ).toEqual([]);
    await expect(canvas).toBeVisible();
    await expect
      .poll(() => page.evaluate(() => document.documentElement.scrollWidth))
      .toBe(await page.evaluate(() => document.documentElement.clientWidth));
  } finally {
    await context.close();
  }
});

test("game input clears across blur, visibility loss, offscreen pause, and destroy", async ({ page }) => {
  const runtimeErrors = captureRuntimeErrors(page);
  // Given: a focused player match.
  await page.setViewportSize({ width: 375, height: 812 });
  await page.goto("game/");
  const cabinet = page.locator("[data-arcade-game]");
  await page.getByRole("button", { name: "INSERT COIN" }).click();
  await expect(cabinet).toHaveAttribute("data-game-state", "player-fight", { timeout: 5000 });
  await cabinet.evaluate((element) => element.scrollIntoView({ block: "center" }));
  await expect(cabinet).toHaveAttribute("data-loop-status", /scheduled|running/);

  // When: movement loses focus and the document becomes hidden, offscreen, then destroyed.
  await page.keyboard.down("ArrowRight");
  await expect.poll(() => numericAttribute(cabinet, "data-player-one-x")).toBeGreaterThan(0);
  await page.locator('select[name="game-theme"]').focus();
  await page.keyboard.up("ArrowRight");
  const blurredX = await numericAttribute(cabinet, "data-player-one-x");
  await page.waitForTimeout(150);
  expect(await numericAttribute(cabinet, "data-player-one-x")).toBe(blurredX);
  await page.evaluate(() => { Object.defineProperty(document, "hidden", { configurable: true, value: true }); document.dispatchEvent(new Event("visibilitychange")); });
  await expect(cabinet).toHaveAttribute("data-loop-status", "paused");
  const hiddenTick = await numericAttribute(cabinet, "data-simulation-tick");
  await page.waitForTimeout(150);
  expect(await numericAttribute(cabinet, "data-simulation-tick")).toBe(hiddenTick);
  await page.evaluate(() => { delete document.hidden; document.dispatchEvent(new Event("visibilitychange")); window.scrollTo(0, 0); });
  await expect(cabinet).toHaveAttribute("data-loop-status", /scheduled|running/);
  await expect
    .poll(() => numericAttribute(cabinet, "data-simulation-tick"), { timeout: 1000 })
    .toBeGreaterThan(hiddenTick);

  await page.evaluate(() => window.scrollTo(0, document.documentElement.scrollHeight));
  await expect(cabinet).toHaveAttribute("data-loop-status", "paused");
  const offscreenTick = await numericAttribute(cabinet, "data-simulation-tick");
  await page.waitForTimeout(150);
  expect(await numericAttribute(cabinet, "data-simulation-tick")).toBe(offscreenTick);
  await cabinet.evaluate((element) => element.scrollIntoView({ block: "center" }));
  await expect(cabinet).toHaveAttribute("data-loop-status", /scheduled|running/);
  await expect
    .poll(() => numericAttribute(cabinet, "data-simulation-tick"), { timeout: 1000 })
    .toBeGreaterThan(offscreenTick);

  await cabinet.evaluate((element) => element.dispatchEvent(new Event("sf2-game:destroy")));

  // Then: a destroyed cabinet stops rendering and a reload remounts it.
  await expect(cabinet).toHaveAttribute("data-renderer-status", "destroyed");
  await expect(cabinet).toHaveAttribute("data-loop-status", "destroyed");
  await expect(cabinet.locator("canvas")).toHaveCount(0);
  const destroyedTick = await numericAttribute(cabinet, "data-simulation-tick");
  await page.waitForTimeout(150);
  expect(await numericAttribute(cabinet, "data-simulation-tick")).toBe(destroyedTick);
  await page.reload();
  const remounted = page.locator("[data-arcade-game]");
  await expect(remounted).toHaveAttribute("data-renderer-status", "ready");
  await expect(remounted.locator("canvas")).toHaveCount(1);
  await expect(page.locator("[data-game-credit-status]")).toHaveText("CREDIT 00");
  expect(runtimeErrors).toEqual([]);
});

test("game reduced motion keeps the static poster until activation", async ({ page }) => {
  const runtimeErrors = captureRuntimeErrors(page);
  // Given: reduced motion is requested before navigation.
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.goto("game/");
  const cabinet = page.locator("[data-arcade-game]");

  // When: the visitor has not inserted a coin.
  await expect(cabinet).toHaveAttribute("data-loop-status", "paused");
  await expect(cabinet).toHaveAttribute("data-simulation-tick", "0");
  await expect(page.locator("[data-game-poster]")).toBeVisible();
  await page.getByRole("button", { name: "INSERT COIN" }).click();

  // Then: explicit activation starts the real game and hides its poster.
  await expect(cabinet).toBeFocused();
  await expect(page.locator("[data-game-poster]")).toBeHidden();
  await expect(cabinet).toHaveAttribute("data-loop-status", /scheduled|running/);
  await expect
    .poll(() => numericAttribute(cabinet, "data-simulation-tick"), { timeout: 1000 })
    .toBeGreaterThan(0);
  expect(runtimeErrors).toEqual([]);
});

test("malformed game palettes fail closed before keyboard bootstrap", async ({ page }) => {
  // Given: the full-game palette payload is malformed at its trust boundary.
  const runtimeErrors = captureRuntimeErrors(page);
  await page.addInitScript(() => {
    window.__f2ListenerCounts = { keydown: 0, keyup: 0 };
    const addEventListener = HTMLElement.prototype.addEventListener;
    HTMLElement.prototype.addEventListener = function countGameListeners(type, listener, options) {
      if (type === "keydown" || type === "keyup") window.__f2ListenerCounts[type] += 1;
      return Reflect.apply(addEventListener, this, [type, listener, options]);
    };
  });
  await page.route("**/sf2-themes/game/", async (route) => {
    const response = await route.fetch();
    const body = await response.text();
    await route.fulfill({ response, body: body.replace(/(<script[^>]*data-game-palettes[^>]*>)[\s\S]*?(<\/script>)/, "$1{$2") });
  });

  // When: the route starts its browser adapter.
  await page.goto("game/");

  // Then: the static failure remains visible and no key listeners have been installed.
  await expect(page.locator("[data-arcade-game]")).toHaveAttribute("data-renderer-status", "failed");
  await expect(page.locator("[data-arcade-game]")).toHaveAttribute("data-loop-status", "paused");
  await expect(page.locator("[data-game-poster]")).toHaveAttribute("data-failure", "SyntaxError");
  await expect(page.locator("[data-game-status]")).toHaveText("STATIC FALLBACK");
  expect(await page.evaluate(() => window.__f2ListenerCounts)).toEqual({ keydown: 0, keyup: 0 });
  expect(runtimeErrors).toEqual([]);
});

test("a WebGL failure on the full game route leaves the visible fallback", async ({ page }) => {
  // Given: WebGL context creation fails before the game module starts.
  const runtimeErrors = captureRuntimeErrors(page);
  await page.addInitScript(() => {
    const getContext = HTMLCanvasElement.prototype.getContext;
    HTMLCanvasElement.prototype.getContext = function withoutWebGl(type, ...args) {
      if (type === "webgl" || type === "webgl2") return null;
      return Reflect.apply(getContext, this, [type, ...args]);
    };
  });

  // When: the full game route opens.
  await page.goto("game/");

  // Then: failure is bounded to the visible static poster.
  await expect(page.locator("[data-arcade-game]")).toHaveAttribute("data-renderer-status", "failed");
  await expect(page.locator("[data-arcade-game]")).toHaveAttribute("data-loop-status", "paused");
  await expect(page.locator("[data-game-poster]")).toBeVisible();
  await expect(page.locator("[data-game-status]")).toHaveText("STATIC FALLBACK");
  expect(runtimeErrors).toEqual([]);
});
