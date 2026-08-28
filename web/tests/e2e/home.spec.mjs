import { expect, test } from "@playwright/test";

import { paletteVariants } from "../../src/data/theme-data.mjs";

const initialPalette = paletteVariants[0];

test("the home features one canonical palette preview instead of the arcade", async ({ page }) => {
  // Given: a visitor opens the production-built home route.
  await page.goto("./");

  // When: the static featured palette preview renders.
  const preview = page.locator("[data-featured-palette-preview]");

  // Then: one canonical palette is selected with its real developer surfaces and no home arcade.
  await expect(preview).toHaveCount(1);
  const selectedId = await preview.getAttribute("data-selected-palette");
  const selectedPalette = paletteVariants.find((palette) => palette.id === selectedId);
  expect(selectedPalette).toBeDefined();
  await expect(preview.locator("[data-code-pane]")).toContainText(`id: \"${selectedPalette?.id}\"`);
  await expect(preview.locator("[data-code-pane]")).toContainText(selectedPalette?.tokens.ui.accent ?? "");
  const highlightedTokens = await preview.locator("[data-code-pane] [data-syntax-token]").evaluateAll(
    (tokens) => tokens.map((token) => ({ text: token.textContent, className: token.className })),
  );
  expect(highlightedTokens.length).toBeGreaterThan(0);
  expect(highlightedTokens.some(({ text, className }) => text === "const" && className.includes("syntax-token--keyword"))).toBe(true);
  expect(highlightedTokens.some(({ text, className }) => text === `\"${selectedPalette?.id}\"` && className.includes("syntax-token--string"))).toBe(true);
  await expect(preview.locator("[data-terminal-pane]")).toContainText(
    `sf2-themes show ${selectedPalette?.id}`,
  );
  await expect(preview.locator("[data-preview-swatch]")).toHaveCount(25);
  const hasBrightWhite = await preview.locator("[data-preview-swatch]").evaluateAll(
    (swatches, expected) => swatches.some((swatch) => swatch.textContent?.includes(expected)),
    selectedPalette?.tokens.ansi.bright.white ?? "",
  );
  expect(hasBrightWhite).toBe(true);
  await expect(page.locator("[data-arcade-game]")).toHaveCount(0);
});

test("the home preview uses the canonical code pane syntax contract", async ({ page }) => {
  // Given: the home preview and canonical Main preview are rendered from the same production build.
  await page.goto("./");
  const homePreview = page.locator("[data-featured-palette-preview]");
  const homeCodePane = homePreview.locator("[data-code-pane]");
  const homeStyles = await homeCodePane.evaluate((pane) => {
    const preview = pane.closest("[data-featured-palette-preview]");
    const plainToken = pane.querySelector(".syntax-token--plain");
    const keywordToken = pane.querySelector(".syntax-token--keyword");
    return {
      codePaneBackground: getComputedStyle(pane).backgroundColor,
      foreground: preview ? getComputedStyle(preview).color : "",
      plain: plainToken ? getComputedStyle(plainToken).color : "",
      keywordWeight: keywordToken ? getComputedStyle(keywordToken).fontWeight : "",
    };
  });

  // When: the canonical Main preview is inspected as the reference family surface.
  await page.goto("preview/");
  const referenceCard = page.locator('[data-preview-variant][data-preview-id="main"]');
  const referenceStyles = await referenceCard.locator("[data-code-pane]").evaluate((pane) => ({
    codePaneBackground: getComputedStyle(pane).backgroundColor,
    keywordWeight: getComputedStyle(pane.querySelector(".syntax-token--keyword") ?? pane).fontWeight,
  }));

  // Then: home plain text, keyword hierarchy, and code-pane surface match the canonical preview.
  expect(homeStyles.plain).toBe(homeStyles.foreground);
  expect(homeStyles.keywordWeight).toBe(referenceStyles.keywordWeight);
  expect(homeStyles.codePaneBackground).toBe(referenceStyles.codePaneBackground);
});

test("distinct controlled page loads select distinct canonical featured palettes", async ({ browser }) => {
  // Given: independent browser contexts provide distinct random draws before the route loads.
  const cases = [
    { draw: 0, palette: paletteVariants[0] },
    { draw: 0.999_999, palette: paletteVariants.at(-1) },
  ];

  // When: each visitor opens the production-built home route.
  for (const { draw, palette } of cases) {
    const context = await browser.newContext({
      baseURL: `http://127.0.0.1:${process.env.PLAYWRIGHT_PORT ?? "4321"}/sf2-themes/`,
    });
    const page = await context.newPage();
    await page.addInitScript((randomDraw) => {
      Math.random = () => randomDraw;
    }, draw);

    try {
      const preview = page.locator("[data-featured-palette-preview]");
      await page.goto("./");

      // Then: the controlled draw selects its distinct canonical palette at the real route boundary.
      await expect(preview).toHaveAttribute("data-selected-palette", palette?.id ?? "");
      const highlightedTokens = await preview.locator("[data-code-pane] [data-syntax-token]").evaluateAll(
        (tokens) => tokens.map((token) => ({ text: token.textContent, className: token.className })),
      );
      expect(highlightedTokens.length).toBeGreaterThan(0);
      expect(highlightedTokens.some(({ text, className }) => text === `\"${palette?.id}\"` && className.includes("syntax-token--string"))).toBe(true);
      const syntaxColors = await preview.locator("[data-code-pane] [data-syntax-token]").evaluateAll((tokens) => {
        const keyword = tokens.find((token) => token.classList.contains("syntax-token--keyword"));
        const string = tokens.find((token) => token.classList.contains("syntax-token--string"));
        const code = tokens[0]?.closest("code");
        return {
          keyword: keyword ? getComputedStyle(keyword).color : "",
          string: string ? getComputedStyle(string).color : "",
          code: code ? getComputedStyle(code).color : "",
        };
      });
      expect(syntaxColors.keyword).not.toBe(syntaxColors.code);
      expect(syntaxColors.string).not.toBe(syntaxColors.code);
    } finally {
      await context.close();
    }
  }
});

test("the home featured palette remains useful without client JavaScript", async ({ browser }) => {
  // Given: a browser context disables all client JavaScript.
  const context = await browser.newContext({
    baseURL: `http://127.0.0.1:${process.env.PLAYWRIGHT_PORT ?? "4321"}/sf2-themes/`,
    javaScriptEnabled: false,
  });
  const page = await context.newPage();

  try {
    // When: a visitor opens the statically generated home route.
    await page.goto("./");
    const preview = page.locator("[data-featured-palette-preview]");

    // Then: the canonical selection, code, terminal command, and swatches remain readable.
    await expect(preview).toHaveAttribute("data-selected-palette", initialPalette.id);
    await expect(preview.locator("[data-code-pane]")).toContainText(`id: \"${initialPalette.id}\"`);
    await expect(preview.locator("[data-terminal-pane]")).toContainText(
      `sf2-themes show ${initialPalette.id}`,
    );
    await expect(preview.locator("[data-code-pane] [data-syntax-token]")).not.toHaveCount(0);
    await expect(preview.locator("[data-preview-swatch]")).toHaveCount(25);
  } finally {
    await context.close();
  }
});

test("malformed featured palette data preserves the static fallback", async ({ page }) => {
  // Given: the serialized featured palette payload is malformed at its trust boundary.
  const runtimeErrors = [];
  page.on("pageerror", (error) => runtimeErrors.push(`page: ${error.message}`));
  page.on("console", (message) => {
    if (message.type() === "error") runtimeErrors.push(`console: ${message.text()}`);
  });
  await page.route("**/sf2-themes/", async (route) => {
    const response = await route.fetch();
    const body = await response.text();
    const corrupted = body.replace(
      /(<script[^>]*data-featured-palette-data[^>]*>)[\s\S]*?(<\/script>)/,
      "$1{bad json$2",
    );
    expect(corrupted).not.toBe(body);
    await route.fulfill({ response, body: corrupted });
  });

  // When: the production-built home route opens.
  await page.goto("./");
  const preview = page.locator("[data-featured-palette-preview]");

  // Then: the complete static first palette remains visible without uncaught errors.
  await expect(preview).toHaveAttribute("data-selected-palette", initialPalette.id);
  await expect(preview.locator("[data-code-pane]")).toContainText(`id: \"${initialPalette.id}\"`);
  await expect(preview.locator("[data-terminal-pane]")).toContainText(`sf2-themes show ${initialPalette.id}`);
  await expect(preview.locator("[data-preview-swatch]")).toHaveCount(25);
  expect(runtimeErrors).toEqual([]);
});

test("structurally malformed featured palette data preserves the static fallback", async ({ page }) => {
  // Given: the serialized payload is valid JSON but lacks required nested palette values.
  const runtimeErrors = [];
  page.on("pageerror", (error) => runtimeErrors.push(`page: ${error.message}`));
  page.on("console", (message) => {
    if (message.type() === "error") runtimeErrors.push(`console: ${message.text()}`);
  });
  await page.route("**/sf2-themes/", async (route) => {
    const response = await route.fetch();
    const body = await response.text();
    const corrupted = body.replace(
      /(<script[^>]*data-featured-palette-data[^>]*>)[\s\S]*?(<\/script>)/,
      '$1[{"id":"malformed","name":"Malformed","tokens":{"ui":{},"ansi":{},"semantic":{}}}]$2',
    );
    expect(corrupted).not.toBe(body);
    await route.fulfill({ response, body: corrupted });
  });

  // When: the production-built home route opens.
  await page.goto("./");
  const preview = page.locator("[data-featured-palette-preview]");

  // Then: the complete static first palette remains visible without uncaught errors.
  await expect(preview).toHaveAttribute("data-selected-palette", initialPalette.id);
  await expect(preview.locator("[data-code-pane]")).toContainText(`id: \"${initialPalette.id}\"`);
  await expect(preview.locator("[data-terminal-pane]")).toContainText(`sf2-themes show ${initialPalette.id}`);
  await expect(preview.locator("[data-preview-swatch]")).toHaveCount(25);
  expect(runtimeErrors).toEqual([]);
});

test("unexpected featured palette rendering errors are not swallowed", async ({ browser }) => {
  // Given: a DOM update fails after the payload has parsed successfully.
  const context = await browser.newContext({
    baseURL: `http://127.0.0.1:${process.env.PLAYWRIGHT_PORT ?? "4321"}/sf2-themes/`,
  });
  const page = await context.newPage();
  const runtimeErrors = [];
  page.on("pageerror", (error) => runtimeErrors.push(error.message));
  await page.addInitScript(() => {
    const replaceWith = HTMLElement.prototype.replaceWith;
    HTMLElement.prototype.replaceWith = function (...nodes) {
      if (this.matches("[data-featured-palette-preview]")) {
        throw new Error("featured preview DOM update probe");
      }
      return replaceWith.apply(this, nodes);
    };
  });

  try {
    // When: the production-built home route performs its runtime randomization.
    await page.goto("./");

    // Then: an unrelated DOM failure remains observable to browser error reporting.
    await expect(page.locator("[data-featured-palette-preview]")).toHaveCount(1);
    expect(runtimeErrors).toContain("featured preview DOM update probe");
  } finally {
    await context.close();
  }
});
