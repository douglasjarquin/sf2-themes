import { expect, test } from "@playwright/test";

const installScript = `git clone https://github.com/douglasjarquin/street-fighter-2-theme.git
cd street-fighter-2-theme
install -m 755 sf2-theme "$HOME/.local/bin/sf2-theme"`;

test("install route gives the real setup then apply commands", async ({ page }) => {
  // Given: a visitor needs to install the CLI from the project site.
  await page.goto("./install/");

  // When: they read the installation route.
  const installBlock = page.getByLabel("Install script");

  // Then: it preserves the README script and presents supported adapter commands only.
  await expect(installBlock).toHaveText(installScript);
  await expect(page.getByText("sf2-theme setup wezterm", { exact: true })).toBeVisible();
  await expect(
    page.getByText("sf2-theme apply wezterm --theme ryu", { exact: true }),
  ).toBeVisible();
  await expect(page.getByText("sf2-theme setup herdr", { exact: true })).toBeVisible();
  await expect(
    page.getByText("sf2-theme apply herdr --theme chun-li", { exact: true }),
  ).toBeVisible();

  const pageText = await page.locator("body").innerText();
  expect(pageText).not.toMatch(/\bTUI\b/i);
  expect(pageText).not.toContain("sf2-theme install wezterm");
  expect(pageText).not.toContain("sf2-theme install herdr");
});

test("install script copy confirms only after clipboard success", async ({ page }) => {
  // Given: the browser clipboard accepts the exact install script.
  await page.addInitScript(() => {
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: {
        writeText: async (text) => {
          window.__sf2ThemeCopiedText = text;
        },
      },
    });
  });
  await page.goto("./install/");

  // When: the visitor copies the install script.
  const copyButton = page.locator("[data-install-copy]");
  await expect(copyButton).toHaveAccessibleName("Copy install script");
  await copyButton.click();

  // Then: the clipboard receives the exact multiline text and the button confirms it.
  await expect.poll(() => page.evaluate(() => window.__sf2ThemeCopiedText)).toBe(installScript);
  await expect(copyButton).toHaveText("✓ Copied");
});

test("install script copy stays unconfirmed when clipboard rejects", async ({ page }) => {
  // Given: the browser denies clipboard access.
  await page.addInitScript(() => {
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: {
        writeText: () => Promise.reject(new Error("clipboard access denied")),
      },
    });
  });
  await page.goto("./install/");

  // When: the visitor attempts to copy the install script.
  const copyButton = page.locator("[data-install-copy]");
  await expect(copyButton).toHaveAccessibleName("Copy install script");
  await copyButton.click();

  // Then: no successful-copy checkmark is shown.
  await expect(copyButton).toHaveText("Copy install script");
});
