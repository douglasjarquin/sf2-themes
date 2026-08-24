import { expect, test } from "@playwright/test";

const mainColors = ["#101a3a", "#e8565f", "#f2b134", "#35c4c2", "#fff4d6"];
const terminalColors = ["#101a3a", "#e8565f", "#f2b134", "#f2b134", "#fff4d6"];
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

test("renders twenty available character swatches from TOML variants", async ({ page }) => {
  // Given: a visitor opens the palette route.
  await page.goto("palette/");

  // When: the character variant rows render.
  const rows = page.locator("[data-character-variant]");

  // Then: Ryu, Ken, Chun-Li, and Guile are available with their five exact swatches.
  await expect(rows).toHaveCount(4);
  for (const [index, variant] of variants.entries()) {
    const row = rows.nth(index);
    await expect(row).toHaveAttribute("data-character-variant", variant.name.toLowerCase());
    await expect(row.getByText("AVAILABLE", { exact: true })).toBeVisible();
    await expect(row.locator("[data-character-swatch]")).toHaveCount(5);
    await expect(
      row.locator("[data-character-swatch]").evaluateAll((swatches) =>
        swatches.map((swatch) => swatch.dataset.color),
      ),
    ).resolves.toEqual(variant.colors);
  }
  await expect(page.locator("[data-character-swatch]")).toHaveCount(20);
});

test("copies the exact selected color and confirms it", async ({ context, page }) => {
  // Given: clipboard access is granted to a palette visitor.
  await context.grantPermissions(["clipboard-read", "clipboard-write"], {
    origin: "http://127.0.0.1:4321",
  });
  await page.goto("palette/");

  // When: the visitor copies Ryu's background swatch.
  await page.locator('[data-character-variant="ryu"] [data-character-swatch]').first().click();

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
