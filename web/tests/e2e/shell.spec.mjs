import { expect, test } from "@playwright/test";

test("the header exposes brand, anchors, fighter select, modes, and the palette trigger", async ({
  page,
}) => {
  await page.goto("./");

  await expect(page.locator(".site-brand")).toHaveText("sf2-themes");
  const nav = page.getByRole("navigation", { name: "Primary" });
  await expect(nav.getByRole("link", { name: "install" })).toHaveAttribute(
    "href",
    "/sf2-themes/#install",
  );
  await expect(nav.getByRole("link", { name: "ports" })).toHaveAttribute(
    "href",
    "/sf2-themes/#ports",
  );
  await expect(nav.getByRole("link", { name: /github/ })).toHaveAttribute(
    "href",
    "https://github.com/douglasjarquin/sf2-themes",
  );

  await expect(page.locator("[data-site-select]")).toHaveValue("ryu");
  await expect(page.locator("[data-site-select] option")).toHaveCount(18);
  await expect(page.locator('[data-site-mode="dark"]')).toHaveAttribute(
    "aria-pressed",
    "true",
  );
  await expect(
    page.getByRole("button", { name: "Open theme palette" }),
  ).toHaveAttribute("aria-expanded", "false");
});

test("mode buttons switch the palette and update aria-pressed", async ({ page }) => {
  await page.goto("./");

  await page.locator('[data-site-mode="light"]').click();
  await expect(page.locator('[data-site-mode="light"]')).toHaveAttribute(
    "aria-pressed",
    "true",
  );
  await expect(page.locator('[data-site-mode="dark"]')).toHaveAttribute(
    "aria-pressed",
    "false",
  );
  await expect(page.locator("html")).toHaveAttribute("data-mode", "light");
  await expect(page.locator("html")).toHaveCSS("color-scheme", "light");
  await expect(page.locator("#preview [data-t='fileId']").first()).toHaveText(
    "sf2-ryu-light",
  );
  await expect.poll(() => page.url()).toContain("mode=light");
});

test("theme and mode persist through localStorage and deep links", async ({ page }) => {
  await page.goto("./?theme=guile&mode=light");

  await expect(page.locator("html")).toHaveAttribute("data-theme", "guile");
  await expect(page.locator("html")).toHaveAttribute("data-mode", "light");
  await expect(page.locator("[data-site-select]")).toHaveValue("guile");
  await expect(page.locator("#preview [data-t='displayName']").first()).toHaveText(
    "Guile Light",
  );

  await page.locator("[data-site-select]").selectOption("ken");
  await page.goto("./");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "ken");
  await expect(page.locator("html")).toHaveAttribute("data-mode", "light");
});

test("arrow keys step fighters and t toggles the mode", async ({ page }) => {
  await page.goto("./?theme=ryu&mode=dark");

  await page.keyboard.press("ArrowRight");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "sagat");
  await page.keyboard.press("ArrowLeft");
  await expect(page.locator("html")).toHaveAttribute("data-theme", "ryu");

  await page.keyboard.press("t");
  await expect(page.locator("html")).toHaveAttribute("data-mode", "light");
});

test("the command palette filters, applies selections, and restores focus", async ({
  page,
}) => {
  await page.goto("./");

  await page.keyboard.press("Control+k");
  const dialog = page.getByRole("dialog", { name: "Theme palette" });
  await expect(dialog).toBeVisible();
  await expect(page.locator("[data-palette-input]")).toBeFocused();
  await expect(page.locator("[data-palette-item]:visible")).toHaveCount(37);

  await page.locator("[data-palette-input]").fill("boxer");
  await expect(page.locator("[data-palette-item]:visible")).toHaveCount(2);
  await page.locator("[data-palette-input]").press("Enter");
  await expect(dialog).toBeHidden();
  await expect(page.locator("html")).toHaveAttribute("data-theme", "balrog");
  await expect.poll(() => page.url()).toContain("theme=balrog");
});

test("the palette filters by mode words and shows an empty state", async ({ page }) => {
  await page.goto("./");

  await page.getByRole("button", { name: "Open theme palette" }).click();
  await page.locator("[data-palette-input]").fill("claw light");
  await expect(page.locator("[data-palette-item]:visible")).toHaveCount(1);
  await expect(page.locator("[data-palette-item]:visible")).toContainText("Vega Light");

  await page.locator("[data-palette-input]").fill("dan hibiki");
  await expect(page.locator("[data-palette-empty]")).toBeVisible();
  await expect(page.locator("[data-palette-item]:visible")).toHaveCount(0);

  await page.keyboard.press("Escape");
  await expect(page.getByRole("dialog", { name: "Theme palette" })).toBeHidden();
  await expect(page.getByRole("button", { name: "Open theme palette" })).toBeFocused();
});

test("the palette mode toggle switches without leaving the overlay", async ({ page }) => {
  await page.goto("./");

  await page.getByRole("button", { name: "Open theme palette" }).click();
  await page.getByRole("option", { name: /Switch to light mode/ }).click();

  await expect(page.getByRole("dialog", { name: "Theme palette" })).toBeHidden();
  await expect(page.locator("html")).toHaveAttribute("data-mode", "light");
  await expect(page.locator('[data-site-mode="light"]')).toHaveAttribute(
    "aria-pressed",
    "true",
  );
});

test("the skip link reaches main content and the footer links externally", async ({
  page,
}) => {
  await page.goto("./");

  const skip = page.getByRole("link", { name: "Skip to main content" });
  await page.keyboard.press("Tab");
  await expect(skip).toBeFocused();
  await skip.press("Enter");
  await expect(page.locator("#main-content")).toBeInViewport();

  const footer = page.getByRole("contentinfo");
  await expect(footer).toContainText("Not affiliated with or endorsed by Capcom");
  await expect(footer.getByRole("link", { name: /github/i }).first()).toHaveAttribute(
    "href",
    "https://github.com/douglasjarquin/sf2-themes",
  );
});
