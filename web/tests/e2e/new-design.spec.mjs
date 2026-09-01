import { expect, test } from "@playwright/test";

const noOverflow = () => {
  const viewport = document.documentElement.clientWidth;
  return [...document.querySelectorAll("body *")].every((element) => element.getBoundingClientRect().right <= viewport + 1);
};

test("home and shared chrome", async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.goto("./");

  await expect(page.getByRole("heading", { name: "FIGHT FOR YOUR TERMINAL." })).toBeVisible();
  await expect(page.locator(".site-brand")).toHaveText("SF2·THEMES");
  await expect(page.locator('[data-nav-link][aria-current="page"]')).toHaveText("HOME");
  await expect(page.locator("[data-site-picker]")).toHaveCount(1);
  await expect(page.locator("[data-home-theme-preview]")).toHaveCount(1);
  await expect(page.locator("body")).toHaveCSS("font-family", /Archivo/);

  await page.locator("[data-site-picker-toggle]").click();
  await expect(page.locator("[data-site-picker-panel]")).toBeVisible();
  await page.getByRole("button", { name: "CHUN-LI", exact: true }).click();
  await expect(page.locator("[data-site-picker-panel]")).toBeHidden();
  await expect.poll(() => page.evaluate(() => localStorage.getItem("sf2-site-theme"))).toContain("chun-li");
  await page.locator("[data-site-picker-toggle]").click();
  await page.getByRole("button", { name: "LIGHT", exact: true }).click();
  await expect(page.locator("[data-site-picker-toggle]")).toContainText("LIGHT");
  await expect(page.locator("[data-home-theme-preview]")).toContainText("sf2-themes show chun-li");

  await expect.poll(() => page.evaluate(noOverflow)).toBe(true);
  await page.setViewportSize({ width: 375, height: 844 });
  await expect.poll(() => page.evaluate(noOverflow)).toBe(true);
  await page.screenshot({ path: "artifacts/ulw/home-shared-green.png", fullPage: true });
});

test("roster and detail", async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.addInitScript(() => {
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText: async (value) => { window.__copied = value; } },
    });
  });
  await page.goto("themes/");

  await expect(page.getByRole("heading", { name: "CHOOSE YOUR FIGHTER." })).toBeVisible();
  await expect(page.locator("[data-theme-card]")).toHaveCount(18);
  await expect(page.locator('[data-theme-card="ryu"] a')).toHaveAttribute("href", /themes\/ryu\/$/);
  await page.locator('[data-theme-card="ryu"] [data-set-theme]').click();
  await expect.poll(() => page.evaluate(() => localStorage.getItem("sf2-site-theme"))).toContain("ryu");

  await page.locator('[data-theme-card="ryu"] a[data-theme-detail]').click();
  await expect(page).toHaveURL(/\/themes\/ryu\/$/);
  await expect(page.getByRole("heading", { name: "RYU" })).toBeVisible();
  await expect(page.locator("[data-detail-swatch]")).toHaveCount(23);
  await expect(page.locator('[data-detail-mode="dark"]')).toHaveAttribute("aria-pressed", "true");
  await page.locator('[data-detail-mode="light"]').click();
  await expect(page.locator('[data-detail-mode="light"]')).toHaveAttribute("aria-pressed", "true");
  await expect(page.locator("[data-theme-detail-preview]")).toHaveAttribute("data-preview-mode", "light");
  await page.locator("[data-detail-apply]").click();
  await expect(page.locator("[data-detail-apply]")).toHaveText("SITE IS WEARING THIS ✓");
  await page.getByRole("button", { name: "CODEX", exact: true }).click();
  await page.locator("[data-detail-copy]").click();
  await expect(page.locator("[data-detail-copy]")).toHaveText("COPIED ✓");
  await expect.poll(() => page.evaluate(() => window.__copied)).toContain("apply codex --theme ryu-light");
  await expect(page.locator("[data-adjacent-navigation]")).toBeVisible();
  await expect.poll(() => page.evaluate(noOverflow)).toBe(true);
  await page.screenshot({ path: "artifacts/ulw/roster-detail-green.png", fullPage: true });
});

test("install and copy", async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.addInitScript(() => {
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText: async (value) => { window.__copied = value; } },
    });
  });
  await page.goto("install/");

  await expect(page.getByRole("heading", { name: "INSTALL" })).toBeVisible();
  await expect(page.locator("[data-install-step]")).toHaveCount(9);
  await page.locator("[data-install-copy]").first().click();
  await expect(page.locator("[data-install-copy]").first()).toHaveText("COPIED ✓");
  await expect(page.locator("[data-copy-status]")).toHaveText("Install script copied");
  await expect.poll(() => page.evaluate(() => window.__copied)).toContain("sf2-themes --version");
  await page.setViewportSize({ width: 375, height: 844 });
  await expect.poll(() => page.evaluate(noOverflow)).toBe(true);
  await page.screenshot({ path: "artifacts/ulw/install-green.png", fullPage: true });
});


test("install reports unavailable clipboard support", async ({ page }) => {
  await page.addInitScript(() => {
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: undefined });
  });
  await page.goto("install/");
  await page.locator("[data-install-copy]").first().click();
  await expect(page.locator("[data-copy-status]")).toHaveText("Copy unavailable");
});
