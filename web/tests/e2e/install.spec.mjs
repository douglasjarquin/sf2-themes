import { readFileSync } from "node:fs";

import { expect, test } from "@playwright/test";

const readmeInstallBlock = readFileSync(new URL("../../../README.md", import.meta.url), "utf8").match(
  /## Install[\s\S]*?```sh\n([\s\S]*?)```/,
);
if (!readmeInstallBlock) throw new Error("README install script not found");
const installScript = readmeInstallBlock[1].trim();

test("install route gives the real setup then apply commands", async ({ page }) => {
  // Given: a visitor needs to install the CLI from the project site.
  await page.goto("./install/");

  // When: they read the installation route.
  const installBlock = page.getByLabel("Install script");

  // Then: it preserves the README script and presents supported adapter commands only.
  await expect(installBlock).toHaveText(installScript);
  await expect(page.getByText("sf2-themes setup wezterm", { exact: true })).toBeVisible();
  await expect(
    page.getByText("sf2-themes apply wezterm --theme ryu", { exact: true }),
  ).toBeVisible();
  await expect(page.getByText("sf2-themes setup herdr", { exact: true })).toBeVisible();
  await expect(
    page.getByText("sf2-themes apply herdr --theme chun-li", { exact: true }),
  ).toBeVisible();

  const adapterNotesText = await page.locator(".adapter-notes").innerText();
  expect(adapterNotesText).not.toContain("street-fighter-2");
  expect(adapterNotesText).not.toMatch(/\bryu\b.*alias|alias.*\bryu\b/i);

  const pageText = await page.locator("body").innerText();
  expect(pageText).not.toMatch(/\bTUI\b/i);
  expect(pageText).not.toContain("sf2-themes install wezterm");
  expect(pageText).not.toContain("sf2-themes install herdr");
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

test("install script copy reports unexpected clipboard errors without exposing details", async ({ page }) => {
  // Given: the browser throws an unexpected error containing sensitive details.
  await page.addInitScript(() => {
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: {
        writeText: () => Promise.reject(new Error("secret clipboard payload")),
      },
    });
  });
  const diagnostics = [];
  page.on("console", (message) => {
    if (message.type() === "warning") diagnostics.push(message.text());
  });
  await page.goto("./install/");

  // When: the visitor attempts to copy the install script.
  const copyButton = page.locator("[data-install-copy]");
  await expect(copyButton).toHaveAccessibleName("Copy install script");
  await copyButton.click();

  // Then: the failure is observable by safe error type only and never looks successful.
  await expect.poll(() => diagnostics).toContain("[sf2-themes] Clipboard write failed: Error");
  expect(diagnostics.join("\n")).not.toContain("secret clipboard payload");
  await expect(copyButton).toHaveText("Copy install script");
});
