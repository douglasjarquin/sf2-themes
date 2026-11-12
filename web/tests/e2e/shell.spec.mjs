import {
  by,
  click,
  expectAttr,
  expectCount,
  expectContainsText,
  expectCSS,
  expectFocused,
  expectHidden,
  expectInViewport,
  expectText,
  expectURLContains,
  expectValue,
  expectVisible,
  open,
  test,
} from "./support.mjs";

const PRIMARY = 'nav[aria-label="Primary"] a';
const PALETTE = '[role="dialog"][aria-label="Theme palette"]';
const PALETTE_TRIGGER = 'button[aria-label="Open theme palette"]';
const PALETTE_ITEMS = by("[data-palette-item]", undefined, { visible: true });

test("the header exposes brand, anchors, fighter select, modes, and the palette trigger", async (page) => {
  await open(page, "./");

  await expectText(page, ".site-brand", "sf2-themes");
  await expectAttr(page, by(PRIMARY, "install"), "href", "/sf2-themes/#install");
  await expectAttr(page, by(PRIMARY, "ports"), "href", "/sf2-themes/#ports");
  await expectAttr(page, by(PRIMARY, /github/), "href", "https://github.com/douglasjarquin/sf2-themes");

  await expectValue(page, "[data-site-select]", "ryu");
  await expectCount(page, "[data-site-select] option", 18);
  await expectAttr(page, '[data-site-mode="dark"]', "aria-pressed", "true");
  await expectAttr(page, PALETTE_TRIGGER, "aria-expanded", "false");
});

test("mode buttons switch the palette and update aria-pressed", async (page) => {
  await open(page, "./");

  await click(page, '[data-site-mode="light"]');
  await expectAttr(page, '[data-site-mode="light"]', "aria-pressed", "true");
  await expectAttr(page, '[data-site-mode="dark"]', "aria-pressed", "false");
  await expectAttr(page, "html", "data-mode", "light");
  await expectCSS(page, "html", "color-scheme", "light");
  await expectText(page, "#preview [data-t='fileId']", "sf2-ryu-light");
  await expectURLContains(page, "mode=light");
});

test("theme and mode persist through localStorage and deep links", async (page) => {
  await open(page, "./?theme=guile&mode=light");

  await expectAttr(page, "html", "data-theme", "guile");
  await expectAttr(page, "html", "data-mode", "light");
  await expectValue(page, "[data-site-select]", "guile");
  await expectText(page, "#preview [data-t='displayName']", "Guile Light");

  await page.locator("[data-site-select]").selectOption("ken");
  await expectAttr(page, "html", "data-theme", "ken");
  await open(page, "./");
  await expectAttr(page, "html", "data-theme", "ken");
  await expectAttr(page, "html", "data-mode", "light");
});

test("arrow keys step fighters and t toggles the mode", async (page) => {
  await open(page, "./?theme=ryu&mode=dark");

  await page.keyPress("ArrowRight");
  await expectAttr(page, "html", "data-theme", "sagat");
  await page.keyPress("ArrowLeft");
  await expectAttr(page, "html", "data-theme", "ryu");

  await page.type("t");
  await expectAttr(page, "html", "data-mode", "light");
});

test("the command palette filters, applies selections, and restores focus", async (page) => {
  await open(page, "./");

  await page.keyPress("Control+k");
  await expectVisible(page, PALETTE);
  await expectFocused(page, "[data-palette-input]");
  await expectCount(page, PALETTE_ITEMS, 37);

  await page.locator("[data-palette-input]").fill("boxer");
  await expectCount(page, PALETTE_ITEMS, 2);
  await page.keyPress("Enter");
  await expectHidden(page, PALETTE);
  await expectAttr(page, "html", "data-theme", "balrog");
  await expectURLContains(page, "theme=balrog");
});

test("the palette filters by mode words and shows an empty state", async (page) => {
  await open(page, "./");

  await click(page, PALETTE_TRIGGER);
  await page.locator("[data-palette-input]").fill("claw light");
  await expectCount(page, PALETTE_ITEMS, 1);
  await expectContainsText(page, PALETTE_ITEMS, "Vega Light");

  await page.locator("[data-palette-input]").fill("dan hibiki");
  await expectVisible(page, "[data-palette-empty]");
  await expectCount(page, PALETTE_ITEMS, 0);

  await page.keyPress("Escape");
  await expectHidden(page, PALETTE);
  await expectFocused(page, PALETTE_TRIGGER);
});

test("the palette mode toggle switches without leaving the overlay", async (page) => {
  await open(page, "./");

  await click(page, PALETTE_TRIGGER);
  await expectVisible(page, by('[role="option"]', /Switch to light mode/));
  await click(page, '[role="option"][data-mode-toggle]');

  await expectHidden(page, PALETTE);
  await expectAttr(page, "html", "data-mode", "light");
  await expectAttr(page, '[data-site-mode="light"]', "aria-pressed", "true");
});

test("the skip link reaches main content and the footer links externally", async (page) => {
  await open(page, "./");

  const skip = by("a", "Skip to main content");
  await page.keyPress("Tab");
  await expectFocused(page, skip);
  await page.keyPress("Enter");
  await expectInViewport(page, "#main-content");

  await expectContainsText(page, "footer", "Not affiliated with or endorsed by Capcom");
  await expectAttr(page, by("footer a", /github/i), "href", "https://github.com/douglasjarquin/sf2-themes");
});
