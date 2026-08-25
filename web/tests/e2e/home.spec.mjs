import { expect, test } from "@playwright/test";

test("the home route presents the playable theme cabinet", async ({ page }) => {
  await page.goto("./");

  await expect(page.getByRole("heading", { level: 1 })).toHaveText(
    "FIGHT FOR YOUR TERMINAL",
  );
  await expect(page.getByRole("link", { name: "VIEW THE ROSTER" })).toHaveAttribute(
    "href",
    "/sf2-themes/themes/",
  );
  await expect(page.getByRole("link", { name: "INSTALL THE PACK" })).toHaveAttribute(
    "href",
    "/sf2-themes/install/",
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
  for (const fighter of ["RYU", "KEN", "CHUN-LI", "GUILE"]) {
    const control = page.getByRole("button", { name: fighter });
    await expect(control).toBeVisible();
    await control.click();
    await expect(activeChip).toHaveText(fighter);
    await expect(control).toHaveAttribute("aria-pressed", "true");
    await expect(page.locator("[data-terminal-transcript]")).toContainText(fighter);
  }

  await page.getByRole("button", { name: "RYU" }).click();
  for (const fighter of ["KEN", "CHUN-LI", "GUILE", "RYU"]) {
    await page.keyboard.press("ArrowRight");
    await expect(activeChip).toHaveText(fighter);
  }

  const focusedInput = page.locator("#keyboard-guard-probe");
  await page.evaluate(() => {
    const input = document.createElement("input");
    input.id = "keyboard-guard-probe";
    input.name = "keyboard-guard-probe";
    input.value = "protected";
    document.body.append(input);
  });
  await focusedInput.focus();
  await page.keyboard.press("ArrowRight");
  await page.keyboard.press("ArrowLeft");
  await expect(focusedInput).toHaveValue("protected");
  await expect(activeChip).toHaveText("RYU");

  await page.keyboard.press("ArrowUp");
  await expect(activeChip).toHaveText("RYU");

  const coinCounter = page.locator("[data-coin-counter]");
  await expect(coinCounter).toHaveText("CREDIT 00");
  await page.getByRole("button", { name: "INSERT COIN" }).click();
  await expect(coinCounter).toHaveText("CREDIT 01");
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
