import { expect, test } from "@playwright/test";

const routes = [
  { label: "HOME", pathname: "/sf2-themes/" },
  { label: "THEMES", pathname: "/sf2-themes/themes/" },
  { label: "PALETTE", pathname: "/sf2-themes/palette/" },
  { label: "PREVIEW", pathname: "/sf2-themes/preview/" },
  { label: "INSTALL", pathname: "/sf2-themes/install/" },
];

const viewports = [
  { name: "desktop", width: 1440, height: 900 },
  { name: "mobile", width: 390, height: 844 },
];

for (const viewport of viewports) {
  test(`shared shell works at ${viewport.name} size`, async ({ page }) => {
    // Given: a visitor opens the statically rendered site at its configured base.
    await page.setViewportSize(viewport);
    await page.goto("./");

    // When: keyboard focus enters the page.
    await page.keyboard.press("Tab");

    // Then: the skip link is focused first, then the document does not overflow horizontally.
    const focusedLink = page.locator(":focus-visible");
    await expect(focusedLink).toHaveAttribute("href", "#main-content");
    await expect(focusedLink).toHaveText("Skip to main content");
    await expect
      .poll(() => focusedLink.evaluate((element) => getComputedStyle(element).outlineStyle))
      .not.toBe("none");
    await expect
      .poll(() =>
        focusedLink.evaluate((element) =>
          Number.parseFloat(getComputedStyle(element).outlineWidth) > 0,
        ),
      )
      .toBe(true);
    await expect
      .poll(() =>
        page.evaluate(
          () => document.documentElement.scrollWidth <= document.documentElement.clientWidth,
        ),
      )
      .toBe(true);

    for (const route of routes) {
      // When: the visitor follows each shared primary navigation link.
      await page.getByRole("navigation", { name: "Primary" }).getByRole("link", {
        name: route.label,
        exact: true,
      }).click();

      // Then: navigation keeps the base prefix and marks exactly that route active.
      await expect(page).toHaveURL(new RegExp(`${route.pathname.replaceAll("/", "\\/")}$`));
      const activeLinks = page.locator('[data-nav-link][aria-current="page"]');
      await expect(activeLinks).toHaveCount(1);
      await expect(activeLinks).toHaveText(route.label);
      await expect
        .poll(() =>
          page.evaluate(
            () => document.documentElement.scrollWidth <= document.documentElement.clientWidth,
          ),
        )
        .toBe(true);
    }
  });
}
