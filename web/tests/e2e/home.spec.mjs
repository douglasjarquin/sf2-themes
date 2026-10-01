import { expect, test } from "@playwright/test";

const noOverflow = () =>
  document.documentElement.scrollWidth <= document.documentElement.clientWidth + 1;

test("home renders the hero, live preview, install steps, and ports", async ({ page }) => {
  await page.goto("./");

  await expect(page).toHaveTitle("Street Fighter II terminal themes | sf2-themes");
  await expect(
    page.getByRole("heading", { level: 1, name: "Fight for your terminal." }),
  ).toBeVisible();
  await expect(page.locator(".meta")).toContainText("36 themes · 7 apps");

  await expect(page.locator("#preview [data-t='fileId']").first()).toHaveText("sf2-ryu");
  await expect(page.locator("[data-pane]")).toHaveCount(7);
  await expect(page.locator("[data-tab]")).toHaveCount(7);

  await expect(page.locator("[data-step]")).toHaveCount(3);
  await expect(page.locator("[data-port-id]")).toHaveCount(7);
  await expect(page.locator(".port-row")).toHaveCount(7);
});

test("theme selection rewrites the preview and install steps", async ({ page }) => {
  await page.goto("./");

  await page.locator("[data-site-select]").selectOption("ken");
  await expect(page.locator("#preview [data-t='displayName']").first()).toHaveText("Ken");
  await expect(page.locator("#preview [data-t='fileId']").first()).toHaveText("sf2-ken");
  await expect(page.locator("[data-step]").nth(2).locator("[data-step-cmd]")).toHaveText(
    "sf2-themes apply wezterm --theme ken",
  );
  await expect.poll(() => page.url()).toContain("theme=ken");
});

test("SampleBlock tabs switch panes with roving selection state", async ({ page }) => {
  await page.goto("./");

  const tabs = page.locator("[data-tab]");
  await expect(tabs.first()).toHaveAttribute("aria-selected", "true");
  await expect(page.locator('[data-pane="shell"]')).toBeVisible();
  await expect(page.locator('[data-pane="nvim"]')).toBeHidden();

  await page.locator('[data-tab="ansi"]').click();
  await expect(page.locator('[data-pane="ansi"]')).toBeVisible();
  await expect(page.locator('[data-pane="shell"]')).toBeHidden();
  await expect(page.locator('[data-tab="ansi"]')).toHaveAttribute("aria-selected", "true");
  await expect(tabs.first()).toHaveAttribute("aria-selected", "false");
});

test("port selection rewrites the install steps", async ({ page }) => {
  await page.goto("./");

  await page.locator('[data-port-id="nvim"]').click();
  await expect(page.locator('[data-port-id="nvim"]')).toHaveAttribute("aria-pressed", "true");
  await expect(page.locator('[data-port-id="wezterm"]')).toHaveAttribute("aria-pressed", "false");
  await expect(page.locator("[data-step]").nth(1).locator("[data-step-cmd]")).toHaveText(
    "sf2-themes setup nvim",
  );
  await expect(page.locator("[data-step]").nth(2).locator("[data-step-cmd]")).toHaveText(
    "sf2-themes apply nvim --theme ryu",
  );
  await expect(page.locator("[data-port-note-out]")).toContainText(
    "The loader applies your pick every time Neovim starts.",
  );
});

test("step copy buttons send the uvx payload and confirm only after success", async ({
  page,
}) => {
  await page.addInitScript(() => {
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText: async (value) => { window.__copied = value; } },
    });
  });
  await page.goto("./");

  const firstCopy = page.locator("[data-step-copy]").first();
  await firstCopy.click();
  await expect(firstCopy).toHaveText("copied");
  await expect
    .poll(() => page.evaluate(() => window.__copied))
    .toBe(
      "uvx --from git+https://github.com/douglasjarquin/sf2-themes.git sf2-themes --version",
    );

  const applyCopy = page.locator("[data-step]").nth(2).locator("[data-step-copy]");
  await applyCopy.click();
  await expect
    .poll(() => page.evaluate(() => window.__copied))
    .toContain("sf2-themes apply wezterm --theme ryu");
});

test("copy buttons are hidden when the clipboard API is unavailable", async ({ page }) => {
  await page.addInitScript(() => {
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: undefined });
  });
  await page.goto("./");

  await expect(page.locator("[data-step-copy]")).toHaveCount(3);
  await expect(page.locator("[data-step-copy]:visible")).toHaveCount(0);
});

test("the home page remains useful without client JavaScript", async ({ browser }) => {
  const context = await browser.newContext({ javaScriptEnabled: false });
  const page = await context.newPage();
  await page.goto("./");

  await expect(
    page.getByRole("heading", { level: 1, name: "Fight for your terminal." }),
  ).toBeVisible();
  await expect(page.locator("[data-step-cmd]").first()).toHaveText("sf2-themes --version");
  await expect(page.locator(".port-row")).toHaveCount(7);
  await expect(page.locator('[data-pane="shell"]')).toBeVisible();
  await context.close();
});

test("the home column reflows without horizontal overflow", async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 900 });
  await page.goto("./");
  await expect.poll(() => page.evaluate(noOverflow)).toBe(true);

  await page.setViewportSize({ width: 375, height: 844 });
  await expect.poll(() => page.evaluate(noOverflow)).toBe(true);
});
