import { expect, test } from "@playwright/test";

test("the home route presents the playable theme cabinet", async ({ page }) => {
  await page.goto("./");

  await expect(page.getByRole("heading", { level: 1 })).toHaveText(
    "FIGHT FOR YOUR TERMINAL",
  );
  await expect(page.getByRole("link", { name: "VIEW THE ROSTER" })).toHaveAttribute(
    "href",
    "/street-fighter-2-theme/themes/",
  );
  await expect(page.getByRole("link", { name: "INSTALL THE PACK" })).toHaveAttribute(
    "href",
    "/street-fighter-2-theme/install/",
  );
  await expect(page.getByText("sf2-theme apply wezterm --theme ryu")).toBeVisible();
  await expect
    .poll(() =>
      page.locator("[data-terminal-transcript]").evaluate(
        (element) => getComputedStyle(element).animationName,
      ),
    )
    .toBe("transcript-in");

  const activeChip = page.locator("[data-active-chip]");
  await expect(activeChip).toHaveText("RYU");
  await expect(page.getByRole("button", { name: "KEN" })).toHaveAttribute(
    "aria-pressed",
    "false",
  );

  await page.getByRole("button", { name: "KEN" }).click();
  await expect(activeChip).toHaveText("KEN");
  await expect(page.getByRole("button", { name: "KEN" })).toHaveAttribute(
    "aria-pressed",
    "true",
  );
  await expect(page.locator("[data-terminal-transcript]")).toContainText("KEN");

  await page.keyboard.press("ArrowRight");
  await expect(activeChip).toHaveText("CHUN-LI");
  await page.keyboard.press("ArrowLeft");
  await expect(activeChip).toHaveText("KEN");

  await page.keyboard.press("ArrowUp");
  await expect(activeChip).toHaveText("KEN");

  const coinCounter = page.locator("[data-coin-counter]");
  await expect(coinCounter).toHaveText("0 CREDITS");
  await page.getByRole("button", { name: "INSERT COIN" }).click();
  await expect(coinCounter).toHaveText("1 CREDIT");

  await page.goto("./?fighter=not-a-fighter");
  await expect(page.locator("[data-active-chip]")).toHaveText("RYU");
});

test("the cabinet disables transcript animation for reduced motion", async ({ page }) => {
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.goto("./");

  await expect
    .poll(() =>
      page.locator("[data-terminal-transcript]").evaluate(
        (element) => getComputedStyle(element).animationName,
      ),
    )
    .toBe("none");
});
