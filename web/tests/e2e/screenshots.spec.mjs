import { expect, test } from "@playwright/test";

test("screenshots route renders the generated cabinet library", async ({ page }) => {
  await page.goto("screenshots/");

  await expect(page.getByRole("heading", { level: 1 })).toHaveText("SCREENSHOT LIBRARY");
  await expect(page.getByRole("link", { name: "SCREENSHOTS", exact: true })).toHaveAttribute(
    "aria-current",
    "page",
  );

  const cards = page.locator("[data-screenshot-card]");
  await expect(cards).toHaveCount(36);
  await expect(cards.evaluateAll((items) => items.map((item) => item.dataset.screenshotId))).resolves.toEqual([
    "main", "main-light", "akuma-light", "akuma", "balrog-light", "balrog", "blanka-light", "blanka",
    "cammy-light", "cammy", "chun-li-light", "chun-li", "dee-jay-light", "dee-jay", "dhalsim-light", "dhalsim",
    "e-honda-light", "e-honda", "fei-long-light", "fei-long", "guile-light", "guile", "ken-light", "ken",
    "m-bison-light", "m-bison", "ryu-light", "ryu", "sagat-light", "sagat", "t-hawk-light", "t-hawk",
    "vega-light", "vega", "zangief-light", "zangief",
  ]);

  const images = cards.locator("img");
  await expect(images).toHaveCount(36);
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
  await expect.poll(() => images.first().evaluate((image) => image.complete && image.naturalWidth)).toBe(1280);
});
