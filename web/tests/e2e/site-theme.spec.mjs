import { expect, test } from "@playwright/test";

import { themeFamilies } from "../../src/data/theme-data.mjs";

const storageKey = "sf2-themes.site-theme-family.v1";
const routes = ["./", "themes/", "palette/", "preview/", "install/", "game/"];
const family = (id) => themeFamilies.find((candidate) => candidate.id === id);
const rgb = (hex) => {
  const values = hex.slice(1).match(/.{2}/g)?.map((value) => Number.parseInt(value, 16));
  return `rgb(${values?.join(", ")})`;
};

async function deterministic(page, colorScheme = "dark") {
  await page.emulateMedia({ colorScheme });
  await page.addInitScript((key) => {
    if (sessionStorage.getItem("site-theme-test-initialized") === "true") return;
    localStorage.removeItem(key);
    sessionStorage.setItem("site-theme-test-initialized", "true");
  }, storageKey);
}

async function readRoot(page) {
  return page.evaluate(() => ({
    family: document.documentElement.dataset.siteThemeFamily,
    id: document.documentElement.dataset.siteThemeId,
    scheme: document.documentElement.dataset.siteColorScheme,
    body: getComputedStyle(document.body).backgroundColor,
    header: getComputedStyle(document.querySelector(".site-header")).backgroundColor,
    footer: getComputedStyle(document.querySelector(".site-footer")).backgroundColor,
    home: getComputedStyle(document.querySelector(".home-page") ?? document.body).backgroundColor,
    mode: document.querySelector("[data-site-theme-mode]")?.textContent,
    swatches: [...document.querySelectorAll("[data-site-theme-swatch]")].map((node) => getComputedStyle(node).backgroundColor),
  }));
}

test("defaults to Main dark and exposes the root theme attributes", async ({ page }) => {
  await deterministic(page, "dark");
  await page.goto("./");

  await expect(page.locator("[data-site-theme-select]")).toHaveValue("main");
  await expect(page.locator("[data-site-theme-select] option")).toHaveCount(18);
  await expect(page.locator("[data-site-theme-mode]")).toHaveText("FOLLOWS SYSTEM · DARK");
  await expect(page.locator("[data-site-theme-control] label")).toHaveText("SITE THEME");
  await expect(page.locator("[data-site-theme-swatch]")).toHaveCount(5);
  await expect(page.locator("[data-site-theme-status]")).toHaveText("");
  await expect.poll(() => readRoot(page)).toEqual({
    family: "main",
    id: "main",
    scheme: "dark",
    body: rgb(family("main").dark.siteTokens.background),
    header: rgb(family("main").dark.siteTokens.background),
    footer: rgb(family("main").dark.siteTokens.panel),
    home: rgb(family("main").dark.siteTokens.background),
    mode: "FOLLOWS SYSTEM · DARK",
    swatches: family("main").dark.siteTokens.preview.map(rgb),
  });
});

test("defaults to Main light when the system is light", async ({ page }) => {
  await deterministic(page, "light");
  await page.goto("./");

  await expect.poll(() => readRoot(page)).toMatchObject({
    family: "main",
    id: "main-light",
    scheme: "light",
    body: rgb(family("main").light.siteTokens.background),
    header: rgb(family("main").light.siteTokens.background),
    footer: rgb(family("main").light.siteTokens.panel),
    home: rgb(family("main").light.siteTokens.background),
    mode: "FOLLOWS SYSTEM · LIGHT",
  });
});

test("selects and persists Ken while tracking system mode", async ({ page }) => {
  await deterministic(page, "dark");
  await page.goto("./");

  await page.locator("[data-site-theme-select]").selectOption("ken");
  await expect.poll(() => readRoot(page)).toMatchObject({
    family: "ken",
    id: "ken",
    scheme: "dark",
    body: rgb(family("ken").dark.siteTokens.background),
    header: rgb(family("ken").dark.siteTokens.background),
    footer: rgb(family("ken").dark.siteTokens.panel),
    mode: "FOLLOWS SYSTEM · DARK",
    swatches: family("ken").dark.siteTokens.preview.map(rgb),
  });
  await expect(page.locator("[data-site-theme-status]")).toHaveText("SITE THEME: KEN · FOLLOWS SYSTEM · DARK");
  await expect.poll(() => page.evaluate((key) => localStorage.getItem(key), storageKey)).toBe("ken");

  await page.emulateMedia({ colorScheme: "light" });
  await expect.poll(() => readRoot(page)).toMatchObject({
    family: "ken",
    id: "ken-light",
    scheme: "light",
    body: rgb(family("ken").light.siteTokens.background),
    mode: "FOLLOWS SYSTEM · LIGHT",
  });
  await page.emulateMedia({ colorScheme: "dark" });
  await expect(page.locator("html")).toHaveAttribute("data-site-theme-id", "ken");
});

test("invalid stored family falls back to Main and persistence survives routes and reload", async ({ page }) => {
  await page.emulateMedia({ colorScheme: "dark" });
  await page.addInitScript((key) => {
    if (sessionStorage.getItem("site-theme-invalid-test-initialized") === "true") return;
    localStorage.setItem(key, "ken-light");
    sessionStorage.setItem("site-theme-invalid-test-initialized", "true");
  }, storageKey);
  await page.goto("./");
  await expect(page.locator("html")).toHaveAttribute("data-site-theme-family", "main");
  await expect(page.locator("html")).toHaveAttribute("data-site-theme-id", "main");

  await page.locator("[data-site-theme-select]").selectOption("chun-li");
  await page.goto("preview/");
  await expect(page.locator("[data-site-theme-select]")).toHaveValue("chun-li");
  await expect(page.locator("html")).toHaveAttribute("data-site-theme-id", "chun-li");
  await page.reload();
  await expect(page.locator("[data-site-theme-select]")).toHaveValue("chun-li");
  await expect(page.locator("html")).toHaveAttribute("data-site-theme-family", "chun-li");
});

test("keeps the native control usable when localStorage throws", async ({ page }) => {
  await page.emulateMedia({ colorScheme: "dark" });
  await page.addInitScript(() => {
    Object.defineProperty(window, "localStorage", {
      configurable: true,
      get() { throw new Error("storage unavailable"); },
    });
  });
  await page.goto("./");

  await expect(page.locator("[data-site-theme-select]")).toHaveValue("main");
  await page.locator("[data-site-theme-select]").selectOption("ken");
  await expect(page.locator("html")).toHaveAttribute("data-site-theme-family", "ken");
  await expect(page.locator("[data-site-theme-status]")).toHaveText("SITE THEME: KEN · FOLLOWS SYSTEM · DARK");
});

test("website selection leaves canonical preview and arcade palettes independent", async ({ page }) => {
  await deterministic(page, "dark");
  await page.goto("preview/");
  const preview = page.locator('[data-preview-variant][data-preview-id="ryu"]');
  const beforePreview = await preview.evaluate((node) => getComputedStyle(node).backgroundColor);
  await page.locator("[data-site-theme-select]").selectOption("ken");
  await expect(preview).toHaveCSS("background-color", beforePreview);

  await page.goto("game/");
  const cabinet = page.locator("[data-arcade-game]");
  const beforeCabinet = await cabinet.evaluate((node) => getComputedStyle(node).backgroundColor);
  await page.locator("[data-site-theme-select]").selectOption("ryu");
  await expect(cabinet).toHaveCSS("background-color", beforeCabinet);
  await expect(cabinet).toHaveAttribute("data-active-theme", "main");
});

test("native selector supports keyboard operation and has no dialog dependency", async ({ page }) => {
  await deterministic(page, "dark");
  await page.goto("./");
  const select = page.locator("[data-site-theme-select]");
  await select.focus();
  await select.press("k");
  await expect(select).toHaveValue("ken");
  await expect(page.locator("html")).toHaveAttribute("data-site-theme-family", "ken");
  await expect(page.locator('[role="dialog"]')).toHaveCount(0);
  await expect(select).toBeFocused();
});

test("keeps the selected family across every public route and fits the mobile header", async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 844 });
  await deterministic(page, "dark");
  await page.goto("./");
  await page.locator("[data-site-theme-select]").selectOption("vega");

  for (const route of routes) {
    await page.goto(route);
    await expect(page.locator("[data-site-theme-select]")).toHaveValue("vega");
    await expect(page.locator("html")).toHaveAttribute("data-site-theme-family", "vega");
    await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true);
  }

  const positions = await page.evaluate(() => [
    document.querySelector(".site-brand").getBoundingClientRect().bottom,
    document.querySelector(".site-header__utility").getBoundingClientRect().top,
    document.querySelector(".primary-nav").getBoundingClientRect().top,
  ]);
  expect(positions[0]).toBeLessThanOrEqual(positions[1]);
  expect(positions[1]).toBeLessThanOrEqual(positions[2]);
});
