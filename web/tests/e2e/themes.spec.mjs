import { expect, test } from "@playwright/test";

const themesPath = "/sf2-themes/themes/";

test("themes catalog truthfully lists the two supported adapters", async ({ page }) => {
  await page.goto(themesPath);

  await expect(page.getByRole("heading", { name: "THEMES" })).toBeVisible();
  await expect(page.getByTestId("themes-stats")).toHaveText("2 PORTS · 2 READY · 1 CLI");

  const cards = page.getByTestId("adapter-card");
  await expect(cards).toHaveCount(2);
  await expect(cards.nth(0)).toContainText("wezterm");
  await expect(cards.nth(0)).toContainText("READY");
  await expect(cards.nth(0).locator("code")).toHaveText("sf2-theme setup wezterm");
  await expect(cards.nth(1)).toContainText("herdr");
  await expect(cards.nth(1)).toContainText("READY");
  await expect(cards.nth(1).locator("code")).toHaveText("sf2-theme setup herdr");
  await expect(page.getByText(/neovim|tmux|alacritty|ghostty/i)).toHaveCount(0);
});

test("themes catalog filters, trims searches, and shows the empty state", async ({ page }) => {
  await page.goto(themesPath);
  const visibleCards = page.locator('[data-testid="adapter-card"]:visible');

  await page.getByRole("button", { name: "READY", exact: true }).click();
  await expect(visibleCards).toHaveCount(2);

  await page.getByRole("button", { name: "PLANNED", exact: true }).click();
  await expect(visibleCards).toHaveCount(0);
  await expect(page.getByTestId("themes-empty-state")).toHaveText(
    "K.O. - NO PORT MATCHES. TRY ANOTHER NAME.",
  );

  await page.getByRole("button", { name: "ALL", exact: true }).click();
  const search = page.getByRole("searchbox", { name: "Search ports" });
  await search.fill("  WeZTeRm  ");
  await expect(visibleCards).toHaveCount(1);
  await expect(visibleCards).toContainText("wezterm");

  await search.fill("not-a-port");
  await expect(visibleCards).toHaveCount(0);
  await expect(page.getByTestId("themes-empty-state")).toBeVisible();
});

test("copy feedback follows clipboard success and never reports a rejected write", async ({ page }) => {
  await page.addInitScript(() => {
    window.__copiedCommands = [];
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: {
        writeText: async (value) => {
          window.__copiedCommands.push(value);
        },
      },
    });
  });
  await page.goto(themesPath);

  const wezterm = page.getByTestId("adapter-card").filter({ hasText: "wezterm" });
  await wezterm.getByRole("button", { name: "COPY" }).click();
  await expect(wezterm.getByRole("button", { name: "COPIED" })).toBeVisible();
  await expect
    .poll(() => page.evaluate(() => window.__copiedCommands))
    .toEqual(["sf2-theme setup wezterm"]);

  await page.addInitScript(() => {
    navigator.clipboard.writeText = async () => Promise.reject(new DOMException("Denied", "NotAllowedError"));
  });
  await page.reload();

  const herdr = page.getByTestId("adapter-card").filter({ hasText: "herdr" });
  await herdr.getByRole("button", { name: "COPY" }).click();
  await expect(herdr.getByRole("button", { name: "COPY" })).toBeVisible();
  await expect(herdr.getByRole("button", { name: "COPIED" })).toHaveCount(0);
});
