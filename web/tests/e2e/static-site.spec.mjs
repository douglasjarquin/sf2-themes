import { expect, test } from "@playwright/test";

test("the static package serves its foundation document", async ({ page }) => {
  await page.goto("./");

  await expect(page).toHaveTitle("Street Fighter II terminal themes | sf2-theme");
});
