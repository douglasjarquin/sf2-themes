import { expect, test } from "@playwright/test";

const origin = "https://douglasjarquin.github.io";

const stubs = [
  { path: "install/", target: "/sf2-themes/#install" },
  { path: "palette/", target: "/sf2-themes/#preview" },
  { path: "preview/", target: "/sf2-themes/#preview" },
  { path: "themes/", target: "/sf2-themes/#preview" },
  { path: "themes/ryu/", target: "/sf2-themes/?theme=ryu" },
];

test("the static package serves its foundation document", async ({ page }) => {
  await page.goto("./");

  await expect(page).toHaveTitle("Street Fighter II terminal themes | sf2-themes");
});

test("the home page carries a self-canonical, description, and JSON-LD", async ({
  page,
}) => {
  await page.goto("./");

  const title = "Street Fighter II terminal themes | sf2-themes";
  const description =
    "Street Fighter II color themes for WezTerm, Herdr, Neovim, Codex, Starship, Lazygit, and Claude Code.";

  await expect(page.locator('meta[name="description"]')).toHaveAttribute(
    "content",
    description,
  );
  await expect(page.locator('link[rel="canonical"]')).toHaveAttribute(
    "href",
    `${origin}/sf2-themes/`,
  );
  await expect(page.locator('meta[property="og:title"]')).toHaveAttribute("content", title);
  await expect(page.locator('meta[property="og:description"]')).toHaveAttribute(
    "content",
    description,
  );
  await expect(page.locator('meta[property="og:url"]')).toHaveAttribute(
    "content",
    `${origin}/sf2-themes/`,
  );
  await expect(page.locator('meta[name="twitter:card"]')).toHaveAttribute(
    "content",
    "summary",
  );

  const jsonLd = JSON.parse(
    await page.locator('script[type="application/ld+json"]').innerText(),
  );
  expect(jsonLd["@graph"].map((node) => node["@type"])).toEqual([
    "WebSite",
    "SoftwareApplication",
  ]);
  expect(JSON.stringify(jsonLd)).not.toMatch(/aggregateRating|"offers"|ratingValue/);
  const software = jsonLd["@graph"].find((node) => node["@type"] === "SoftwareApplication");
  expect(software.operatingSystem).toBe("Linux, macOS, Windows");
  expect(software.downloadUrl).toBe("https://github.com/douglasjarquin/sf2-themes");
  expect(software.codeRepository).toBe("https://github.com/douglasjarquin/sf2-themes");
});

test("robots.txt allows the project path and names the sitemap", async ({ request }) => {
  const response = await request.get("/sf2-themes/robots.txt");
  const body = await response.text();

  expect(response.ok()).toBe(true);
  expect(body).toBe(
    "User-agent: *\nAllow: /sf2-themes/\n\nSitemap: https://douglasjarquin.github.io/sf2-themes/sitemap.xml\n",
  );
});

test("llms.txt summarizes the CLI, catalog, live site, and repository", async ({
  request,
}) => {
  const response = await request.get("/sf2-themes/llms.txt");
  const body = await response.text();

  expect(response.ok()).toBe(true);
  expect(body).toContain("sf2-themes is a Python 3.11 CLI named `sf2-themes`");
  expect(body).toContain("36-theme TOML catalog");
  expect(body).toContain("Live site: https://douglasjarquin.github.io/sf2-themes/");
  expect(body).toContain("Repository: https://github.com/douglasjarquin/sf2-themes");
  expect(body).not.toContain("game");
});

test("sitemap.xml lists only the home route", async ({ request }) => {
  const response = await request.get("/sf2-themes/sitemap.xml");
  const body = await response.text();
  const locations = [...body.matchAll(/<loc>([^<]+)<\/loc>/g)].map((match) => match[1]);

  expect(response.ok()).toBe(true);
  expect(locations).toEqual(["https://douglasjarquin.github.io/sf2-themes/"]);
});

test("index.html duplicates canonicalise to the trailing-slash home URL", async ({
  page,
}) => {
  await page.goto("index.html");

  await expect(page.locator('link[rel="canonical"]')).toHaveAttribute(
    "href",
    "https://douglasjarquin.github.io/sf2-themes/",
  );
  await expect(page.locator('meta[property="og:url"]')).toHaveAttribute(
    "content",
    "https://douglasjarquin.github.io/sf2-themes/",
  );
});

test("retired routes are noindex meta-refresh stubs pointing home", async ({ request }) => {
  for (const stub of stubs) {
    const response = await request.get(`/sf2-themes/${stub.path}`);
    const html = await response.text();

    expect(response.ok()).toBe(true);
    expect(html).toContain('content="noindex"');
    expect(html).toContain(`url=${stub.target}`);
    expect(html).toContain(`<link rel="canonical" href="${origin}${stub.target}"`);
  }
});

test("the project 404 page is branded, linked, and noindexed", async ({
  page,
  request,
}) => {
  const missing = await request.get("/sf2-themes/this-path-is-not-a-route/");
  expect(missing.status()).toBe(404);

  await page.goto("404.html");

  await expect(page).toHaveTitle("Page not found | sf2-themes");
  await expect(page.locator('meta[name="robots"]')).toHaveAttribute("content", "noindex");
  await expect(page.locator('script[type="application/ld+json"]')).toHaveCount(0);
  await expect(page.getByTestId("project-404")).toContainText(
    "This URL is not a page in sf2-themes.",
  );
  await expect(page.getByRole("heading", { level: 1 })).toContainText("Round over.");

  const recovery = page.getByRole("navigation", { name: "Known routes" });
  await expect(recovery.getByRole("link", { name: /home/ })).toHaveAttribute(
    "href",
    "/sf2-themes/",
  );
  await expect(recovery.getByRole("link", { name: /install/ })).toHaveAttribute(
    "href",
    "/sf2-themes/#install",
  );
  await expect(recovery.getByRole("link", { name: /ports/ })).toHaveAttribute(
    "href",
    "/sf2-themes/#ports",
  );
  await expect(
    recovery.getByRole("link", { name: /github repository/ }),
  ).toHaveAttribute("href", "https://github.com/douglasjarquin/sf2-themes");
});

test("the removed game route is absent from the static package", async ({ request }) => {
  const response = await request.get("/sf2-themes/game/");

  expect(response.status()).toBe(404);
});
