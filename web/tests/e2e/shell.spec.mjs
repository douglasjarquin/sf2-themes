import { expect, test } from "@playwright/test";

const routes = [
  { label: "HOME", pathname: "/sf2-themes/" },
  { label: "THEMES", pathname: "/sf2-themes/themes/" },
  { label: "INSTALL", pathname: "/sf2-themes/install/" },
];
const publicRoutes = [
  ...routes,
  { label: "PALETTE", pathname: "/sf2-themes/palette/" },
  { label: "PREVIEW", pathname: "/sf2-themes/preview/" },
  { label: "GAME", pathname: "/sf2-themes/game/" },
];

const viewports = [
  { name: "desktop", width: 1440, height: 900 },
  { name: "mobile", width: 375, height: 844 },
];

for (const viewport of viewports) {
  test(`shared shell works at ${viewport.name} size`, async ({ page }) => {
    // Given: a visitor opens the statically rendered site at its configured base.
    await page.setViewportSize(viewport);
    await page.goto("./");

    // When: keyboard focus enters the page.
    await page.keyboard.press("Tab");
    await page.waitForTimeout(150);

    // Then: the skip link is focused first, then the document does not overflow horizontally.
    const focusedLink = page.locator(":focus-visible");
    await expect(focusedLink).toHaveAttribute("href", "#main-content");
    await expect(focusedLink).toHaveText("Skip to main content");
    const focusedBox = await focusedLink.boundingBox();
    expect(focusedBox).not.toBeNull();
    expect(focusedBox?.y ?? -1).toBeGreaterThanOrEqual(8);
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

const readShell = () => {
  const readStyles = (selector, pseudo) => {
    const element = document.querySelector(selector);
    if (!(element instanceof Element)) throw new Error(`Missing ${selector}`);
    const styles = getComputedStyle(element, pseudo);
    return {
      backgroundColor: styles.backgroundColor,
      borderBlockEndColor: styles.borderBlockEndColor,
      borderBlockEndWidth: styles.borderBlockEndWidth,
      borderColor: styles.borderColor,
      borderRadius: styles.borderRadius,
      boxShadow: styles.boxShadow,
      color: styles.color,
      display: styles.display,
      fontFamily: styles.fontFamily,
      fontSize: styles.fontSize,
      fontWeight: styles.fontWeight,
      letterSpacing: styles.letterSpacing,
      textShadow: styles.textShadow,
    };
  };
  const current = document.querySelector('.primary-nav__link[aria-current="page"]');
  return {
    bodyBackground: readStyles("body").backgroundColor,
    bodyScanlines: readStyles("body", "::before").display,
    brand: readStyles(".site-brand"),
    footer: readStyles(".site-footer"),
    nav: readStyles(".primary-nav__link:not([aria-current])"),
    navAction: readStyles(".primary-nav__link--github"),
    navCurrent: current ? readStyles('.primary-nav__link[aria-current="page"]') : null,
  };
};

test("all public routes use the preview page shell", async ({ page }) => {
  // Given: the preview page is the visual source of truth for shared site chrome.
  await page.goto("preview/");
  const previewShell = await page.evaluate(readShell);

  // When: a visitor opens every public route directly.
  for (const route of publicRoutes) {
    await page.goto(route.pathname);

    // Then: the shared shell computes to the same values as the preview template.
    const shell = await page.evaluate(readShell);
    const expectedShell = { ...previewShell };
    delete shell.navCurrent;
    delete expectedShell.navCurrent;
    expect(shell, route.pathname).toEqual(expectedShell);
  }
});
