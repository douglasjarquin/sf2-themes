import { expect, test } from "@playwright/test";

test("themes route lists generated roster families", async ({ page }) => {
  await page.goto("./themes/");

  await expect(page.getByRole("heading", { name: "CHOOSE YOUR FIGHTER." })).toBeVisible();
  await expect(page.locator("[data-theme-card]")).toHaveCount(18);
  await expect(page.locator('[data-theme-card="ryu"]')).toContainText("Ryu");
  await expect(page.locator('[data-theme-card="ryu"]')).toContainText("Japan · Suzaku Castle rooftop");
  await expect(page.locator('[data-theme-card="ryu"] [data-theme-detail]')).toHaveAttribute("href", /themes\/ryu\/$/);
});

test("theme card site action persists the selected family", async ({ page }) => {
  await page.goto("./themes/");
  await page.locator('[data-theme-card="chun-li"] [data-set-theme]').click();

  await expect(page.locator('[data-theme-card="chun-li"] [data-set-theme]')).toHaveText("ACTIVE");
  await expect(page.locator('[data-theme-card="chun-li"] [data-theme-card-tag]')).toHaveText("WEARING");
  await expect.poll(() => page.evaluate(() => localStorage.getItem("sf2-site-theme"))).toContain('"id":"chun-li"');
});

test("theme detail switches preview mode and copies its adapter command", async ({ page }) => {
  await page.addInitScript(() => {
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText: async (value) => { window.__copiedCommand = value; } },
    });
  });
  await page.goto("./themes/ryu/");

  await expect(page.getByRole("heading", { name: "RYU" })).toBeVisible();
  await expect(page.locator("[data-detail-swatch]")).toHaveCount(23);
  await page.locator('[data-detail-mode="light"]').click();
  await expect(page.locator('[data-detail-mode="light"]')).toHaveAttribute("aria-pressed", "true");
  await expect(page.locator("[data-theme-detail-preview]")).toHaveAttribute("data-preview-mode", "light");
  await page.locator("[data-detail-apply]").click();
  await expect(page.locator("[data-detail-apply]")).toHaveText("SITE IS WEARING THIS ✓");
  await page.getByRole("button", { name: "CODEX", exact: true }).click();
  await page.locator("[data-detail-copy]").click();

  await expect(page.locator("[data-detail-copy]")).toHaveText("COPIED ✓");
  await expect.poll(() => page.evaluate(() => window.__copiedCommand)).toContain("apply codex --theme ryu-light");
  await expect(page.locator("[data-adjacent-navigation]")).toBeVisible();
  await expect(page.locator(".detail-screenshot img")).toHaveAttribute("src", /screenshots\/game\/ryu\.png$/);
  await expect.poll(() => page.locator(".detail-screenshot img").evaluate((image) => image.complete && image.naturalWidth > 0)).toBe(true);
});
