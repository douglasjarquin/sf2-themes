import { expect, test } from "@playwright/test";

import { dojo } from "../../src/game/content/stages/dojo.ts";
import { TextmodeRenderer } from "../../src/game/render/TextmodeRenderer.ts";

async function numericAttribute(locator, name) {
  return Number(await locator.getAttribute(name));
}

function captureRuntimeErrors(page) {
  const errors = [];
  page.on("pageerror", (error) => errors.push(`page: ${error.message}`));
  page.on("console", (message) => {
    if (message.type() === "error") errors.push(`console: ${message.text()}`);
  });
  return errors;
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
    const rowStart = offsetY + input.row * cellHeight;
    const rowEnd = rowStart + cellHeight;
    for (let column = 0; column < input.columns; column += 1) {
      const columnStart = offsetX + column * cellWidth;
      const columnEnd = columnStart + cellWidth;
      let ink = 0;
      for (let y = rowStart; y < rowEnd; y += 1) {
        for (let x = columnStart; x < columnEnd; x += 1) {
          const offset = (y * scratch.width + x) * 4;
          const distance = Math.abs(pixels[offset] - backgroundRed)
            + Math.abs(pixels[offset + 1] - backgroundGreen)
            + Math.abs(pixels[offset + 2] - backgroundBlue);
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

test("the home arcade handles focused key taps and palette-only changes", async ({ page }) => {
  // Given the production-built home route and runtime error capture
  const runtimeErrors = captureRuntimeErrors(page);
  await page.goto("./");

  await expect(page.getByRole("heading", { level: 1 })).toHaveText(
    "FIGHT FOR YOUR TERMINAL",
  );
  await expect(page.getByRole("link", { name: "VIEW THE ROSTER" })).toHaveAttribute(
    "href",
    "/sf2-themes/themes/",
  );
  await expect(page.getByRole("link", { name: "INSTALL THE PACK" })).toHaveAttribute(
    "href",
    "/sf2-themes/install/",
  );
  await expect(page.getByRole("link", { name: "OPEN FULL GAME" })).toHaveAttribute(
    "href",
    "/sf2-themes/game/",
  );

  const cabinet = page.locator("[data-arcade-game]");
  await expect(cabinet).toHaveCount(1);
  await expect(cabinet).toHaveAttribute("data-game-focus", "");
  await expect(cabinet).toHaveAttribute("data-renderer-status", "ready");
  await expect(cabinet).toHaveAttribute("data-loop-status", /scheduled|running/);
  await expect(cabinet).toHaveAttribute("data-active-theme", "ryu");
  await expect(cabinet).toHaveAttribute("data-game-state", /title|attract-/);
  await expect(cabinet).toHaveAttribute("data-simulation-tick", /[1-9]\d*/);
  await expect(cabinet).toHaveAttribute("data-player-one", "ryu");
  await expect(cabinet).toHaveAttribute("data-player-two", "ken");
  const attractTick = await numericAttribute(cabinet, "data-simulation-tick");
  await expect
    .poll(() => numericAttribute(cabinet, "data-simulation-tick"), { timeout: 1000 })
    .toBeGreaterThan(attractTick);

  // When the native coin control is activated
  await page.getByRole("button", { name: "INSERT COIN" }).click();

  // Then the cabinet owns focus and exposes the live game state
  await expect(cabinet).toBeFocused();
  await expect(page.locator("[data-game-credit-status]")).toHaveText("CREDIT 01");
  await expect(cabinet).toHaveAttribute("data-game-state", "player-fight", { timeout: 5000 });
  await expect(cabinet).toHaveAttribute("data-simulation-tick", /[1-9]\d*/);

  const playerOneX = await numericAttribute(cabinet, "data-player-one-x");
  await cabinet.press("ArrowRight");
  await expect
    .poll(() => numericAttribute(cabinet, "data-player-one-x"), { timeout: 1000 })
    .toBeGreaterThan(playerOneX);

  await page.keyboard.down("z");
  try {
    await expect
      .poll(() => cabinet.getAttribute("data-player-one-move-id"), { timeout: 1000 })
      .toBe("straight-punch");
  } finally {
    await page.keyboard.up("z");
  }

  const fighters = await cabinet.evaluate((element) => ({
    first: element.getAttribute("data-player-one"),
    second: element.getAttribute("data-player-two"),
  }));
  await page.getByRole("button", { name: "Use Ken palette" }).click();
  await expect(cabinet).toHaveAttribute("data-active-theme", "ken");
  await expect(cabinet).toHaveAttribute("data-player-one", fighters.first ?? "");
  await expect(cabinet).toHaveAttribute("data-player-two", fighters.second ?? "");

  // Then an outside text input keeps native arrow behavior and receives no game input
  await page.evaluate(() => {
    const input = document.createElement("input");
    input.id = "outside-input";
    input.value = "native";
    document.body.append(input);
  });
  const outsideInput = page.locator("#outside-input");
  await outsideInput.focus();
  await outsideInput.evaluate((element) => {
    if (!(element instanceof HTMLInputElement)) throw new TypeError("outside input missing");
    element.setSelectionRange(6, 6);
  });
  await outsideInput.press("ArrowLeft");
  await expect
    .poll(() =>
      outsideInput.evaluate((element) =>
        element instanceof HTMLInputElement ? element.selectionStart : null,
      ),
    )
    .toBe(5);
  await expect(cabinet).not.toBeFocused();
  expect(runtimeErrors).toEqual([]);
});

test("the full dojo grid remains visible at 375px and DPR 2", async ({ baseURL, browser }) => {
  // Given a DPR-2 narrow browser and the renderer's public logical/stage contracts
  const logicalSize = new TextmodeRenderer().logicalSize;
  const context = await browser.newContext({
    deviceScaleFactor: 2,
    reducedMotion: "reduce",
    viewport: { width: 375, height: 812 },
  });
  const page = await context.newPage();
  const runtimeErrors = captureRuntimeErrors(page);
  try {
    await page.goto(baseURL ?? "./");
    const cabinet = page.locator("[data-arcade-game]");
    await expect(cabinet).toHaveAttribute("data-renderer-status", "ready");

    // When the native coin control renders the deterministic player fight
    await page.getByRole("button", { name: "INSERT COIN" }).click();
    await expect(cabinet).toHaveAttribute("data-game-state", "player-fight", { timeout: 5000 });
    const canvas = cabinet.locator("canvas");
    const floorLine = dojo.layers.floor
      .filter(({ y }) => y > dojo.groundRow)
      .reduce((longest, glyph) =>
        Array.from(glyph.text).length > Array.from(longest.text).length ? glyph : longest,
      );
    const canvasMetrics = await canvas.evaluate((element) => {
      if (!(element instanceof HTMLCanvasElement)) throw new TypeError("renderer canvas missing");
      const bounds = element.getBoundingClientRect();
      return {
        backingHeight: element.height,
        backingWidth: element.width,
        cssHeight: bounds.height,
        cssWidth: bounds.width,
        devicePixelRatio: window.devicePixelRatio,
      };
    });
    const ground = await logicalRowInk(canvas, logicalSize, floorLine.y);
    const floorEnd = floorLine.x + Array.from(floorLine.text).length;
    const missingFloorColumns = ground.missingColumns.filter(
      (column) => column >= floorLine.x && column < floorEnd,
    );

    // Then the DPR backing store and every authored ground-row cell remain visible
    expect(canvasMetrics.backingWidth / canvasMetrics.cssWidth).toBeCloseTo(
      canvasMetrics.devicePixelRatio,
      1,
    );
    expect(canvasMetrics.backingHeight / canvasMetrics.cssHeight).toBeCloseTo(
      canvasMetrics.devicePixelRatio,
      1,
    );
    expect(
      missingFloorColumns,
      `grid=${JSON.stringify(ground.grid)}; missing floor ink in authored columns ${missingFloorColumns.join(", ")}`,
    ).toEqual([]);
    const widths = await page.evaluate(() => ({
      clientWidth: document.documentElement.clientWidth,
      scrollWidth: document.documentElement.scrollWidth,
    }));
    expect(widths.scrollWidth).toBe(widths.clientWidth);
    await expect(cabinet).toBeVisible();
    expect(runtimeErrors).toEqual([]);
  } finally {
    await context.close();
  }
});

test("held input clears on blur, visibility loss, offscreen pause, and destroy", async ({ page }) => {
  // Given a focused player match
  const runtimeErrors = captureRuntimeErrors(page);
  await page.goto("./");
  const cabinet = page.locator("[data-arcade-game]");
  await page.getByRole("button", { name: "INSERT COIN" }).click();
  await expect(cabinet).toHaveAttribute("data-game-state", "player-fight", { timeout: 5000 });

  // When movement is held and focus leaves the cabinet
  const startX = await numericAttribute(cabinet, "data-player-one-x");
  await page.keyboard.down("ArrowRight");
  await expect
    .poll(() => numericAttribute(cabinet, "data-player-one-x"), { timeout: 1000 })
    .toBeGreaterThan(startX);
  await page.evaluate(() => {
    const input = document.createElement("input");
    input.id = "blur-input";
    document.body.append(input);
  });
  await page.locator("#blur-input").focus();
  await page.keyboard.up("ArrowRight");
  await page.waitForTimeout(100);
  const blurredX = await numericAttribute(cabinet, "data-player-one-x");
  await page.waitForTimeout(150);
  expect(await numericAttribute(cabinet, "data-player-one-x")).toBe(blurredX);

  // Then hidden and offscreen states pause ticks and release held input
  await cabinet.focus();
  await page.keyboard.down("ArrowLeft");
  await page.evaluate(() => {
    Object.defineProperty(document, "hidden", { configurable: true, value: true });
    document.dispatchEvent(new Event("visibilitychange"));
  });
  await expect(cabinet).toHaveAttribute("data-loop-status", "paused");
  const hiddenTick = await numericAttribute(cabinet, "data-simulation-tick");
  await page.waitForTimeout(150);
  expect(await numericAttribute(cabinet, "data-simulation-tick")).toBe(hiddenTick);
  await page.keyboard.up("ArrowLeft");
  await page.evaluate(() => {
    delete document.hidden;
    document.dispatchEvent(new Event("visibilitychange"));
  });
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

  await cabinet.evaluate((element) => element.dispatchEvent(new Event("sf2-game:destroy")));
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

test("reduced motion keeps a static poster until explicit activation", async ({ page }) => {
  // Given reduced motion is requested before navigation
  const runtimeErrors = captureRuntimeErrors(page);
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.goto("./");
  const cabinet = page.locator("[data-arcade-game]");

  // Then the ready renderer remains static until INSERT COIN
  await expect(cabinet).toHaveAttribute("data-renderer-status", "ready");
  await expect(cabinet).toHaveAttribute("data-loop-status", "paused");
  await expect(cabinet).toHaveAttribute("data-game-state", "boot");
  await expect(cabinet).toHaveAttribute("data-simulation-tick", "0");
  await expect(page.locator("[data-game-poster]")).toBeVisible();
  await expect(page.getByRole("button", { name: "INSERT COIN" })).toHaveCSS(
    "transition-duration",
    "0s",
  );
  await page.waitForTimeout(150);
  await expect(cabinet).toHaveAttribute("data-simulation-tick", "0");

  await page.getByRole("button", { name: "INSERT COIN" }).click();
  await expect(cabinet).toBeFocused();
  await expect(page.locator("[data-game-poster]")).toBeHidden();
  await expect(cabinet).toHaveAttribute("data-loop-status", /scheduled|running/);
  await expect
    .poll(() => numericAttribute(cabinet, "data-simulation-tick"), { timeout: 1000 })
    .toBeGreaterThan(0);
  expect(runtimeErrors).toEqual([]);
});

test("malformed palette input fails closed to the visible static poster", async ({ page }) => {
  // Given the generated page palette payload is corrupted at its trust boundary
  const runtimeErrors = captureRuntimeErrors(page);
  await page.route("**/sf2-themes/", async (route) => {
    const response = await route.fetch();
    const body = await response.text();
    const corrupted = body.replace(
      /(<script[^>]*data-game-palettes[^>]*>)[\s\S]*?(<\/script>)/,
      "$1{$2",
    );
    expect(corrupted).not.toBe(body);
    await route.fulfill({ response, body: corrupted });
  });

  // When the production-built home route opens
  await page.goto("./");
  const cabinet = page.locator("[data-arcade-game]");

  // Then bootstrap exposes a bounded visible fallback without an uncaught error
  await expect(cabinet).toHaveAttribute("data-renderer-status", "failed");
  await expect(cabinet).toHaveAttribute("data-loop-status", "paused");
  await expect(page.locator("[data-game-poster]")).toBeVisible();
  await expect(page.locator("[data-game-poster]")).toHaveAttribute("data-failure", "SyntaxError");
  await expect(page.locator("[data-game-status]")).toHaveText("STATIC FALLBACK");
  expect(runtimeErrors).toEqual([]);
});

test("malformed palette bootstrap does not install keyboard listeners before fallback", async ({ page }) => {
  await page.addInitScript(() => {
    window.__f2ListenerCounts = { keydown: 0, keyup: 0 };
    const addEventListener = HTMLElement.prototype.addEventListener;
    HTMLElement.prototype.addEventListener = function countGameListeners(type, listener, options) {
      if (type === "keydown" || type === "keyup") window.__f2ListenerCounts[type] += 1;
      return Reflect.apply(addEventListener, this, [type, listener, options]);
    };
  });
  await page.route("**/sf2-themes/", async (route) => {
    const response = await route.fetch();
    const body = await response.text();
    const corrupted = body.replace(
      /(<script[^>]*data-game-palettes[^>]*>)[\s\S]*?(<\/script>)/,
      "$1{$2",
    );
    await route.fulfill({ response, body: corrupted });
  });

  await page.goto("./");
  await expect(page.locator("[data-arcade-game]")).toHaveAttribute("data-renderer-status", "failed");
  expect(await page.evaluate(() => window.__f2ListenerCounts)).toEqual({ keydown: 0, keyup: 0 });
});

test("a real WebGL initialization failure leaves a visible renderer fallback", async ({ page }) => {
  // Given WebGL context creation is unavailable before the game module boots
  const runtimeErrors = captureRuntimeErrors(page);
  await page.addInitScript(() => {
    const getContext = HTMLCanvasElement.prototype.getContext;
    HTMLCanvasElement.prototype.getContext = function getContextWithoutWebGl(type, ...args) {
      if (type === "webgl" || type === "webgl2") return null;
      return Reflect.apply(getContext, this, [type, ...args]);
    };
  });

  // When the production-built home route opens
  await page.goto("./");
  const cabinet = page.locator("[data-arcade-game]");

  // Then renderer startup fails visibly instead of throwing through the page
  await expect(cabinet).toHaveAttribute("data-renderer-status", "failed");
  await expect(cabinet).toHaveAttribute("data-loop-status", "paused");
  await expect(page.locator("[data-game-poster]")).toBeVisible();
  await expect(page.locator("[data-game-status]")).toHaveText("STATIC FALLBACK");
  expect(runtimeErrors).toEqual([]);
});
