import { expect, test } from "@playwright/test";

import { paletteVariants } from "../../src/data/theme-data.mjs";

const expectedIds = paletteVariants.map((variant) => variant.id);

test("preview route renders every canonical palette surface", async ({ page }) => {
  // Given: a visitor opens the production palette preview route.
  await page.goto("preview/");

  // When: the complete static preview surface renders.
  const cards = page.locator("[data-preview-card]");

  // Then: every canonical variant appears in order with its image and two labelled panes.
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("PALETTE PREVIEW");
  await expect(cards).toHaveCount(36);
  await expect(cards.evaluateAll((items) => items.map((item) => item.dataset.previewId))).resolves.toEqual(
    expectedIds,
  );
  await expect(cards.locator("[data-preview-image]")).toHaveCount(36);
  await expect(cards.locator("[data-code-pane] pre code")).toHaveCount(36);
  await expect(cards.locator("[data-terminal-pane] pre code")).toHaveCount(36);
  await expect(cards.locator("[data-command-copy]")).toHaveCount(36);

  for (const variant of paletteVariants) {
    const card = page.locator(`[data-preview-id="${variant.id}"]`);
    await expect(card.locator("[data-code-pane]")).toContainText(variant.tokens.ui.background);
    await expect(card.locator("[data-code-pane]")).toContainText(variant.tokens.ui.accent);
    await expect(card.locator("[data-terminal-pane]")).toContainText(`sf2-themes show ${variant.id}`);
    await expect(card.locator("[data-terminal-pane]")).toContainText(variant.id);
  }

  const images = cards.locator("[data-preview-image]");
  const imageAttributes = await images.evaluateAll((items) => items.map((image) => ({
    src: image.getAttribute("src"),
    width: image.getAttribute("width"),
    height: image.getAttribute("height"),
    alt: image.getAttribute("alt"),
    loading: image.getAttribute("loading"),
  })));
  expect(imageAttributes.every((entry) =>
    entry.src?.includes("/sf2-themes/screenshots/game/")
    && entry.src?.endsWith(".png")
    && entry.width === "1280"
    && entry.height === "720"
    && entry.alt?.includes("gameplay")
    && entry.loading === "lazy")).toBe(true);
  for (const image of await images.all()) await image.scrollIntoViewIfNeeded();
  await expect.poll(() => images.evaluateAll((items) =>
    items.filter((image) => image.complete && image.naturalWidth === 1280).length)).toBe(36);
});

test("preview route keeps truthful snippets and keyboard-visible controls", async ({ page }) => {
  // Given: a visitor opens the palette preview.
  await page.goto("preview/");

  // When: the visitor inspects the snippets and tabs to the first command control.
  const previewText = await page.locator("main").innerText();
  const firstCommand = page.locator("[data-command-copy]").first();
  for (let index = 0; index < 20 && !await firstCommand.evaluate((button) => button === document.activeElement); index += 1) {
    await page.keyboard.press("Tab");
  }

  // Then: proposal-only fiction is absent and the focused control has a visible outline.
  expect(previewText).not.toMatch(/ansi\.solve|GateError|mise run gate/);
  await expect(firstCommand).toBeFocused();
  expect(await firstCommand.evaluate((button) =>
    getComputedStyle(button).outlineStyle)).not.toBe("none");
});

test("preview content remains available without client JavaScript", async ({ browser }) => {
  // Given: a browser context has client JavaScript disabled.
  const context = await browser.newContext({
    baseURL: `http://127.0.0.1:${process.env.PLAYWRIGHT_PORT ?? "4321"}/sf2-themes/`,
    javaScriptEnabled: false,
  });
  const page = await context.newPage();

  // When: a visitor opens the statically generated preview route.
  await page.goto("preview/");

  // Then: all cards, code panes, and terminal commands remain readable.
  await expect(page.locator("[data-preview-card]")).toHaveCount(36);
  await expect(page.locator("[data-code-pane] pre code")).toHaveCount(36);
  await expect(page.locator("[data-terminal-pane] pre code")).toHaveCount(36);
  await expect(page.locator('[data-preview-id="ryu"] [data-terminal-pane]')).toContainText("sf2-themes show ryu");
  await context.close();
});

for (const viewport of [
  { name: "desktop", width: 1440, height: 900 },
  { name: "mobile", width: 390, height: 844 },
]) {
  test(`preview route has no horizontal overflow at ${viewport.name} width`, async ({ page }) => {
    // Given: the preview is rendered at the target viewport.
    await page.setViewportSize({ width: viewport.width, height: viewport.height });
    await page.goto("preview/");

    // When: the browser computes the complete static layout.
    const dimensions = await page.evaluate(() => ({
      clientWidth: document.documentElement.clientWidth,
      scrollWidth: document.documentElement.scrollWidth,
    }));

    // Then: no content extends past the viewport.
    expect(dimensions.scrollWidth).toBeLessThanOrEqual(dimensions.clientWidth);
  });
}

test("old screenshots page route is absent while gameplay assets remain available", async ({ request }) => {
  // Given: the production preview server exposes the renamed route and generated asset namespace.
  // When: a client requests the retired page and one canonical gameplay image.
  const oldRoute = await request.get("screenshots/");
  const image = await request.get("screenshots/game/ryu.png");

  // Then: the page is gone but the generated image remains available.
  expect(oldRoute.status()).toBe(404);
  expect(image.status()).toBe(200);
  expect(image.headers()["content-type"]).toBe("image/png");
});
