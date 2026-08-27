import { expect, test } from "@playwright/test";

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
    await cabinet.press("z");
    await expect
      .poll(() => cabinet.getAttribute("data-player-one-move-id"), { timeout: 5000 })
      .toBe("crescent-palm");
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
