import { readFileSync } from "node:fs";

import { expect, test } from "@playwright/test";

const readmeInstallBlock = readFileSync(new URL("../../../README.md", import.meta.url), "utf8").match(
  /## Install[\s\S]*?```sh\n([\s\S]*?)```/,
);
if (!readmeInstallBlock) throw new Error("README install script not found");
const installScript = readmeInstallBlock[1].trim();
const uvCommand = "uv run --with git+https://github.com/douglasjarquin/sf2-themes.git sf2-themes";

test("install route gives the real setup then apply commands", async ({ page }) => {
  // Given: a visitor needs to install the CLI from the project site.
  await page.goto("./install/");

  // When: they read the installation route.
  const installBlock = page.getByLabel("Install script");
  const installHeadings = page.locator(".install-moves__heading h2, .adapter-notes h2");

  // Then: it preserves the README script and presents uv-backed adapter commands only.
  await expect(page.getByText("READY PLAYER ONE", { exact: true })).toHaveCount(0);
  await expect(installHeadings).toHaveCount(2);
  await expect(installHeadings.first()).toHaveCSS("color", "rgb(202, 209, 222)");
  await expect(installHeadings.first()).toHaveCSS("font-weight", "700");
  await expect(installHeadings.first()).toHaveCSS("text-shadow", "none");
  await expect(installBlock).toHaveText(installScript);
  await expect(installBlock).toHaveText(`${uvCommand} --version`);
  await expect(page.getByText(`${uvCommand} setup wezterm`, { exact: true })).toBeVisible();
  await expect(
    page.getByText(`${uvCommand} apply wezterm --theme ryu`, { exact: true }),
  ).toBeVisible();
  await expect(page.getByText(`${uvCommand} setup herdr`, { exact: true })).toBeVisible();
  await expect(
    page.getByText(`${uvCommand} apply herdr --theme chun-li`, { exact: true }),
  ).toBeVisible();
  await expect(page.getByText(`${uvCommand} setup nvim`, { exact: true })).toBeVisible();
  await expect(
    page.getByText(`${uvCommand} apply nvim --theme ryu-light`, { exact: true }),
  ).toBeVisible();
  await expect(page.getByText(`${uvCommand} setup codex`, { exact: true })).toBeVisible();
  await expect(
    page.getByText(`${uvCommand} apply codex --theme ryu-light`, { exact: true }),
  ).toBeVisible();

  const adapterNotesText = await page.locator(".adapter-notes").innerText();
  expect(adapterNotesText).not.toContain("street-fighter-2");
  expect(adapterNotesText).not.toMatch(/\bryu\b.*alias|alias.*\bryu\b/i);

  const pageText = await page.locator("body").innerText();
  expect(pageText).not.toMatch(/\bTUI\b/i);
  expect(pageText).not.toContain("sf2-themes install wezterm");
  expect(pageText).not.toContain("sf2-themes install herdr");
  expect(pageText).not.toContain("sf2-themes install nvim");
  expect(pageText).not.toContain("sf2-themes install codex");
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
  const copyStatus = page.locator("[data-install-status]");
  await expect(copyButton).toHaveAccessibleName("Copy install script");
  await copyButton.click();

  // Then: the clipboard receives the exact multiline text and the button confirms it.
  await expect.poll(() => page.evaluate(() => window.__sf2ThemeCopiedText)).toBe(installScript);
  await expect(copyButton).toHaveText("✓ Copied");
  await expect(copyStatus).toHaveText("Install script copied");
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
  const copyStatus = page.locator("[data-install-status]");
  await expect(copyButton).toHaveAccessibleName("Copy install script");
  await copyButton.click();

  // Then: no successful-copy checkmark is shown.
  await expect(copyButton).toHaveText("Copy install script");
  await expect(copyStatus).toHaveText("Copy failed");
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
  const copyStatus = page.locator("[data-install-status]");
  await expect(copyButton).toHaveAccessibleName("Copy install script");
  await copyButton.click();

  // Then: the failure is observable by safe error type only and never looks successful.
  await expect.poll(() => diagnostics).toContain("[sf2-themes] Clipboard write failed: Error");
  expect(diagnostics.join("\n")).not.toContain("secret clipboard payload");
  await expect(copyButton).toHaveText("Copy install script");
  await expect(copyStatus).toHaveText("Copy failed");
});
