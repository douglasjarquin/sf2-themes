import { expect, test } from "@playwright/test";

test("the static package serves its foundation document", async ({ page }) => {
  await page.goto("./");

  await expect(page).toHaveTitle("sf2-theme");
});
