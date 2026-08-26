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
    colors: ["#101522", "#e24c52", "#f2b134", "#d83a3a", "#fff4d6"],
  },
  {
    name: "Ken",
    colors: ["#1a1014", "#e64d52", "#f2b134", "#e23b3b", "#fff4d6"],
  },
  {
    name: "Chun-Li",
    colors: ["#0e1530", "#e8565f", "#f2b134", "#2f5bd6", "#fff4d6"],
  },
  {
    name: "Guile",
    colors: ["#101820", "#e8565f", "#f2b134", "#537244", "#fff4d6"],
  },
];

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
    origin: "http://127.0.0.1:4321",
  });
  await page.goto("palette/");

  // When: the visitor copies Ryu's background swatch.
  await page.locator('[data-palette-id="ryu"] [data-character-swatch]').first().click();

  // Then: the clipboard and live feedback contain the exact color.
  await expect.poll(() => page.evaluate(() => navigator.clipboard.readText())).toBe("#101522");
  await expect(page.locator("[data-copy-feedback]")).toHaveText("COPIED #101522");
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
