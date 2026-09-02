import { expect, test } from "@playwright/test";

test("the home features the live theme preview instead of the arcade", async ({ page }) => {
  await page.goto("./");
  const preview = page.locator("[data-home-theme-preview]");
  await expect(preview).toHaveCount(1);
  await expect(preview).toContainText("LIVE PREVIEW - THIS SITE WEARS THE THEME YOU PICK");
  await expect(preview).toContainText("sf2-themes show main");
  await expect(preview.locator("[data-home-ansi]")).toHaveCount(16);
  await expect(page.locator("[data-arcade-game]")).toHaveCount(0);
});

test("the home preview keeps its terminal hierarchy", async ({ page }) => {
  await page.goto("./");
  const preview = page.locator("[data-home-theme-preview]");
  await expect(preview.locator(".home-preview__window-header")).toContainText("TERMINAL / main");
  await expect(preview.locator("pre")).toContainText("mode");
  await expect(preview.locator(".home-preview__ramp")).toBeVisible();
});

test("the site theme updates the home live preview", async ({ page }) => {
  await page.goto("./");
  const preview = page.locator("[data-home-theme-preview]");
  await page.locator("[data-site-picker-toggle]").click();
  await page.getByRole("button", { name: "CHUN-LI", exact: true }).click();
  await expect(preview.locator("[data-home-theme-label]")).toContainText("CHUN-LI");
  await expect(preview).toContainText("sf2-themes show chun-li");
});

test("the home preview labels the canonical light theme id", async ({ page }) => {
  await page.goto("./");
  const preview = page.locator("[data-home-theme-preview]");
  await page.locator("[data-site-picker-toggle]").click();
  await page.locator('[data-site-mode="light"]').click();

  await expect(preview).toContainText("sf2-themes show main-light");
  await expect(preview.locator("[data-home-terminal-label]")).toHaveText("TERMINAL / main-light");
});

test("the home live preview remains useful without client JavaScript", async ({ browser }) => {
  const context = await browser.newContext({
    baseURL: `http://127.0.0.1:${process.env.PLAYWRIGHT_PORT ?? "4321"}/sf2-themes/`,
    javaScriptEnabled: false,
  });
  const page = await context.newPage();

  try {
    await page.goto("./");
    const preview = page.locator("[data-home-theme-preview]");
    await expect(preview).toContainText("sf2-themes show main");
    await expect(preview.locator("[data-home-ansi]")).toHaveCount(16);
    await expect(page.locator("html")).toHaveAttribute("style", /--bg:/);
  } finally {
    await context.close();
  }
});

test("malformed site theme data preserves the static home preview", async ({ page }) => {
  const runtimeErrors = [];
  page.on("pageerror", (error) => runtimeErrors.push(error.message));
  await page.route("**/sf2-themes/", async (route) => {
    const response = await route.fetch();
    const body = await response.text();
    const corrupted = body.replace(
      /(<script[^>]*id=\"site-theme-data\"[^>]*>)[\s\S]*?(<\/script>)/,
      "$1{bad json$2",
    );
    expect(corrupted).not.toBe(body);
    await route.fulfill({ response, body: corrupted });
  });
  await page.goto("./");
  await expect(page.locator("[data-home-theme-preview]")).toContainText("sf2-themes show main");
  expect(runtimeErrors).toEqual([]);
});

test("structurally malformed site theme data preserves the static home preview", async ({ page }) => {
  const runtimeErrors = [];
  page.on("pageerror", (error) => runtimeErrors.push(error.message));
  await page.route("**/sf2-themes/", async (route) => {
    const response = await route.fetch();
    const body = await response.text();
    const corrupted = body.replace(
      /(<script[^>]*id=\"site-theme-data\"[^>]*>)[\s\S]*?(<\/script>)/,
      '$1[{"id":"malformed"}]$2',
    );
    expect(corrupted).not.toBe(body);
    await route.fulfill({ response, body: corrupted });
  });
  await page.goto("./");
  await expect(page.locator("[data-home-theme-preview]")).toContainText("sf2-themes show main");
  await page.locator("[data-site-picker-toggle]").click();
  await expect(page.locator("[data-site-picker-toggle]")).toHaveAttribute("aria-expanded", "false");
  expect(runtimeErrors).toEqual([]);
});

test("empty site theme data preserves the static home preview", async ({ page }) => {
  const runtimeErrors = [];
  page.on("pageerror", (error) => runtimeErrors.push(error.message));
  await page.route("**/sf2-themes/", async (route) => {
    const response = await route.fetch();
    const body = await response.text();
    const corrupted = body.replace(
      /(<script[^>]*id=\"site-theme-data\"[^>]*>)[\s\S]*?(<\/script>)/,
      "$1[]$2",
    );
    expect(corrupted).not.toBe(body);
    await route.fulfill({ response, body: corrupted });
  });
  await page.goto("./");
  await expect(page.locator("[data-home-theme-preview]")).toContainText("sf2-themes show main");
  await page.locator("[data-site-picker-toggle]").click();
  await expect(page.locator("[data-site-picker-toggle]")).toHaveAttribute("aria-expanded", "false");
  expect(runtimeErrors).toEqual([]);
});

test("object site theme data preserves the static home preview", async ({ page }) => {
  const runtimeErrors = [];
  page.on("pageerror", (error) => runtimeErrors.push(error.message));
  await page.route("**/sf2-themes/", async (route) => {
    const response = await route.fetch();
    const body = await response.text();
    const marker = "id=\"site-theme-data\"";
    const markerStart = body.indexOf(marker);
    const openEnd = body.indexOf(">", markerStart) + 1;
    const close = body.indexOf("</script>", openEnd);
    await route.fulfill({ response, body: body.slice(0, openEnd) + "{}" + body.slice(close) });
  });
  await page.goto("./");
  await expect(page.locator("[data-home-theme-preview]")).toContainText("sf2-themes show main");
  await page.locator("[data-site-picker-toggle]").click();
  await expect(page.locator("[data-site-picker-toggle]")).toHaveAttribute("aria-expanded", "false");
  expect(runtimeErrors).toEqual([]);
});

test("selected malformed site theme data preserves the static home preview", async ({ page }) => {
  const runtimeErrors = [];
  page.on("pageerror", (error) => runtimeErrors.push(error.message));
  await page.route("**/sf2-themes/", async (route) => {
    const response = await route.fetch();
    const body = await response.text();
    const marker = "id=\"site-theme-data\"";
    const markerStart = body.indexOf(marker);
    const openEnd = body.indexOf(">", markerStart) + 1;
    const close = body.indexOf("</script>", openEnd);
    await route.fulfill({ response, body: body.slice(0, openEnd) + "[{\"id\":\"main\",\"name\":\"MAIN\",\"dark\":{}}]" + body.slice(close) });
  });
  await page.goto("./");
  await expect(page.locator("[data-home-theme-preview]")).toContainText("sf2-themes show main");
  await page.locator("[data-site-picker-toggle]").click();
  await expect(page.locator("[data-site-picker-toggle]")).toHaveAttribute("aria-expanded", "false");
  expect(runtimeErrors).toEqual([]);
});
