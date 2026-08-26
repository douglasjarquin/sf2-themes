import { expect, test } from "@playwright/test";

test("screenshots route renders the generated cabinet library", async ({ page }) => {
  await page.goto("screenshots/");

  await expect(page.getByRole("heading", { level: 1 })).toHaveText("SCREENSHOT LIBRARY");
  await expect(page.getByRole("link", { name: "SCREENSHOTS", exact: true })).toHaveAttribute(
    "aria-current",
    "page",
  );

  const cards = page.locator("[data-screenshot-card]");
  await expect(cards).toHaveCount(4);
  await expect(cards.evaluateAll((items) => items.map((item) => item.dataset.screenshotId))).resolves.toEqual([
    "ryu",
    "ken",
    "chun-li",
    "guile",
  ]);

  const images = cards.locator("img");
  await expect(images).toHaveCount(4);
  await expect.poll(() => images.evaluateAll((items) => items.map((image) => image.naturalWidth))).toEqual([
    1280,
    1280,
    1280,
    1280,
  ]);
});
