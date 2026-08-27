import { expect, test } from "@playwright/test";
import { readdirSync, readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { parse } from "smol-toml";

const projectRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../..");
const mainTheme = parse(readFileSync(path.join(projectRoot, "themes/main.toml"), "utf8"));
const paletteIds = [
  "main",
  "main-light",
  ...readdirSync(path.join(projectRoot, "themes/characters"))
    .filter((fileName) => fileName.endsWith(".toml"))
    .map((fileName) => fileName.slice(0, -5)),
];
const darkPaletteIds = paletteIds.filter((id) => !id.endsWith("-light"));
const lightPaletteIds = paletteIds.filter((id) => id.endsWith("-light"));
const mainColors = [
  mainTheme.ui.background,
  mainTheme.semantic.red,
  mainTheme.semantic.yellow,
  mainTheme.ui.accent,
  mainTheme.ui.foreground,
];
const terminalColors = [...mainColors];
const variants = [
  {
    name: "Ryu",
    colors: ["#141a23", "#c86e6c", "#9e8625", "#da6a6a", "#cad2df"],
  },
  {
    name: "Ken",
    colors: ["#221616", "#c76e67", "#a88225", "#de6767", "#decdcc"],
  },
  {
    name: "Chun-Li",
    colors: ["#131a23", "#c86e67", "#a38424", "#498ee2", "#c8d2de"],
  },
  {
    name: "Guile",
    colors: ["#151c12", "#c96e68", "#9e8726", "#769746", "#cbd4c7"],
  },
];

test("filters catalog palettes by color mode", async ({ page }) => {
  // Given: a visitor opens the palette route with all catalog entries visible.
  await page.goto("palette/");

  // When: the visitor uses the all, dark, and light controls.
  const filters = page.locator("[data-palette-filter]");
  const palettes = page.locator("[data-palette-id]");

  // Then: each mode exposes the exact catalog subset and announces its active state.
  await expect(filters).toHaveCount(3);
  await expect(palettes).toHaveCount(paletteIds.length);
  await expect(page.locator('[data-palette-filter="all"]')).toHaveAttribute("aria-pressed", "true");

  await page.locator('[data-palette-filter="dark"]').click();
  await expect(page.locator('[data-palette-filter="dark"]')).toHaveAttribute("aria-pressed", "true");
  await expect(page.locator("[data-palette-mode=dark]:visible")).toHaveCount(darkPaletteIds.length);
  await expect(page.locator("[data-palette-mode=light]:visible")).toHaveCount(0);

  await page.locator('[data-palette-filter="light"]').click();
  await expect(page.locator('[data-palette-filter="light"]')).toHaveAttribute("aria-pressed", "true");
  await expect(page.locator("[data-palette-mode=light]:visible")).toHaveCount(lightPaletteIds.length);
  await expect(page.locator("[data-palette-mode=dark]:visible")).toHaveCount(0);

  await page.locator('[data-palette-filter="all"]').click();
  await expect(page.locator('[data-palette-filter="all"]')).toHaveAttribute("aria-pressed", "true");
  await expect(page.locator("[data-palette-id]:visible")).toHaveCount(paletteIds.length);
});

test("renders every catalog palette from the theme source", async ({ page }) => {
  // Given: a visitor opens the palette route.
  await page.goto("palette/");

  // When: the complete catalog palette surface renders.
  const palettes = page.locator("[data-palette-id]");

  // Then: every source theme appears exactly once in catalog order.
  await expect(palettes).toHaveCount(paletteIds.length);
  await expect(
    palettes.evaluateAll((items) => items.map((item) => item.getAttribute("data-palette-id"))),
  ).resolves.toEqual(paletteIds);
  const screenshots = page.locator("[data-game-screenshot]");
  await expect(screenshots).toHaveCount(paletteIds.length);
  const screenshotAttributes = await screenshots.evaluateAll((items) => items.map((image) => ({
    src: image.getAttribute("src"),
    width: image.getAttribute("width"),
    height: image.getAttribute("height"),
    alt: image.getAttribute("alt"),
    loading: image.getAttribute("loading"),
  })));
  expect(screenshotAttributes.every((entry) =>
    entry.src?.includes("/sf2-themes/screenshots/game/")
    && entry.src?.endsWith(".png")
    && entry.width === "1280"
    && entry.height === "720"
    && entry.alt?.includes("gameplay")
    && entry.loading === "lazy")).toBe(true);
  await expect.poll(() => screenshots.first().evaluate((image) => image.complete && image.naturalWidth)).toBe(1280);
});

test("renders the exact TOML source cards and terminal mapping", async ({ page }) => {
  // Given: a visitor opens the palette route.
  await page.goto("palette/");

  // When: the TOML-backed color surfaces render.
  const sourceCards = page.locator("[data-source-card]");
  const terminalMappings = page.locator("[data-terminal-mapping]");

  // Then: all five source values and the documented terminal order are exact.
  await expect(sourceCards).toHaveCount(5);
  await expect(sourceCards.evaluateAll((cards) => cards.map((card) => card.dataset.color))).resolves.toEqual(
    mainColors,
  );
  await expect(sourceCards.nth(3).getByText("ui.accent", { exact: true })).toBeVisible();
  await expect(terminalMappings).toHaveCount(5);
  await expect(
    terminalMappings.evaluateAll((mappings) => mappings.map((mapping) => mapping.dataset.color)),
  ).resolves.toEqual(terminalColors);
  await expect(
    terminalMappings.evaluateAll((mappings) => mappings.map((mapping) => mapping.dataset.token)),
  ).resolves.toEqual([
    "ui.background",
    "semantic.red",
    "semantic.yellow",
    "ui.accent",
    "ui.foreground",
  ]);
});

test("renders every catalog palette with its five TOML swatches", async ({ page }) => {
  // Given: a visitor opens the palette route.
  await page.goto("palette/");

  // When: the catalog palette rows render.
  const rows = page.locator("[data-palette-id]");

  // Then: the complete catalog is present and the existing fighter colors remain exact.
  await expect(rows).toHaveCount(paletteIds.length);
  for (const variant of variants) {
    const row = page.locator(`[data-palette-id="${variant.name.toLowerCase()}"]`);
    await expect(row).toHaveCount(1);
    await expect(row.locator("[data-character-swatch]")).toHaveCount(5);
    await expect(
      row.locator("[data-character-swatch]").evaluateAll((swatches) =>
        swatches.map((swatch) => swatch.dataset.color),
      ),
    ).resolves.toEqual(variant.colors);
  }
  await expect(page.locator("[data-character-swatch]")).toHaveCount(paletteIds.length * 5);
});

test("copies the exact selected color and confirms it", async ({ context, page }) => {
  // Given: clipboard access is granted to a palette visitor.
  await context.grantPermissions(["clipboard-read", "clipboard-write"], {
    origin: `http://127.0.0.1:${process.env.PLAYWRIGHT_PORT ?? "4321"}`,
  });
  await page.goto("palette/");

  // When: the visitor copies Ryu's background swatch.
  await page.locator('[data-palette-id="ryu"] [data-character-swatch]').first().click();

  // Then: the clipboard and live feedback contain the exact color.
  await expect.poll(() => page.evaluate(() => navigator.clipboard.readText())).toBe("#141a23");
  await expect(page.locator("[data-copy-feedback]")).toHaveText("COPIED #141a23");
});

test("reports clipboard rejection without a false success message", async ({ page }) => {
  // Given: the browser clipboard rejects writes.
  await page.addInitScript(() => {
    Object.defineProperty(Navigator.prototype, "clipboard", {
      configurable: true,
      get: () => ({ writeText: () => Promise.reject(new Error("clipboard rejected")) }),
    });
  });
  await page.goto("palette/");

  // When: the visitor tries to copy a source card color.
  await page.locator("[data-source-card]").first().click();

  // Then: the route announces the rejection instead of claiming a copy.
  await expect(page.locator("[data-copy-feedback]")).toHaveText("COPY FAILED");
});
