import { expect, test } from "@playwright/test";

const origin = "https://douglasjarquin.github.io";
const routes = [
  {
    path: "./",
    title: "Street Fighter II terminal themes | sf2-themes",
    description: "Street Fighter II color themes for WezTerm, Herdr, Neovim, Codex, and Starship.",
    canonical: `${origin}/sf2-themes/`,
    heading: "Fight for your terminal.",
  },
  {
    path: "themes/",
    title: "The roster - 18 theme families | sf2-themes",
    description:
      "Browse all 18 Street Fighter II theme families, each with dark and light modes for WezTerm, Herdr, Neovim, Codex, and Starship.",
    canonical: `${origin}/sf2-themes/themes/`,
    heading: "CHOOSE YOUR FIGHTER.",
  },
  {
    path: "palette/",
    title: "Street Fighter II color catalog | sf2-themes",
    description:
      "Browse all 36 TOML-backed Street Fighter II palettes, including dark and light main variants and the Super Street Fighter II Turbo roster.",
    canonical: `${origin}/sf2-themes/palette/`,
    heading: "PALETTE",
  },
  {
    path: "preview/",
    title: "Street Fighter II palette preview | sf2-themes",
    description: "Explore all 36 canonical sf2-themes palettes through code, terminal, neutral, accent, and ANSI previews.",
    canonical: `${origin}/sf2-themes/preview/`,
    heading: "PREVIEW",
  },
  {
    path: "install/",
    title: "Install Street Fighter II terminal themes | sf2-themes",
    description: "Install sf2-themes, then set up and apply Street Fighter II terminal colors.",
    canonical: `${origin}/sf2-themes/install/`,
    heading: "INSTALL",
  },
  {
    path: "game/",
    title: "Street Fighter II terminal arcade game | sf2-themes",
    description:
      "Play the deterministic SF2 Themes terminal arcade game with the full palette and fighter catalog.",
    canonical: `${origin}/sf2-themes/game/`,
    heading: "ENTER THE ARCADE",
  },
];

test("the static package serves its foundation document", async ({ page }) => {
  await page.goto("./");

  await expect(page).toHaveTitle("Street Fighter II terminal themes | sf2-themes");
});

test("JSON-LD is in the static HTML, not injected after render", async ({ request }) => {
  const html = await (await request.get("/sf2-themes/")).text();
  const jsonLd = html.match(
    /<script type="application\/ld\+json"[^>]*>([\s\S]*?)<\/script>/,
  );

  expect(jsonLd?.[1]).toBeTruthy();
  const parsed = JSON.parse(jsonLd[1]);
  expect(parsed["@graph"].map((node) => node["@type"])).toEqual([
    "WebSite",
    "SoftwareApplication",
  ]);
});

test("robots.txt allows the project path and names the sitemap", async ({ request }) => {
  const response = await request.get("/sf2-themes/robots.txt");
  const body = await response.text();

  expect(response.ok()).toBe(true);
  expect(body).toBe(
    "User-agent: *\nAllow: /sf2-themes/\n\nSitemap: https://douglasjarquin.github.io/sf2-themes/sitemap.xml\n",
  );
});

test("llms.txt summarizes the CLI, catalog, live site, and repository", async ({ request }) => {
  const response = await request.get("/sf2-themes/llms.txt");
  const body = await response.text();

  expect(response.ok()).toBe(true);
  expect(body).toContain("sf2-themes is a Python 3.11 CLI named `sf2-themes`");
  expect(body).toContain("36-theme TOML catalog");
  expect(body).toContain("Live site: https://douglasjarquin.github.io/sf2-themes/");
  expect(body).toContain("Repository: https://github.com/douglasjarquin/sf2-themes");
  expect(body).not.toContain("Lazygit");
});

test("the project 404 page is branded, linked, and noindexed", async ({ page, request }) => {
  const missing = await request.get("/sf2-themes/this-path-is-not-a-route/");
  expect(missing.status()).toBe(404);

  await page.goto("404.html");

  await expect(page).toHaveTitle("Page not found | sf2-themes");
  await expect(page.locator('meta[name="robots"]')).toHaveAttribute("content", "noindex");
  await expect(page.locator('script[type="application/ld+json"]')).toHaveCount(0);
  await expect(page.getByTestId("project-404")).toContainText("This URL is not a page in sf2-themes.");
  await expect(page.getByRole("heading", { level: 1 })).toHaveText("CONTINUE?");

  const recovery = page.getByRole("navigation", { name: "Known routes" });
  await expect(recovery.getByRole("link", { name: "Home", exact: true })).toHaveAttribute(
    "href",
    "/sf2-themes/",
  );
  await expect(recovery.getByRole("link", { name: "Themes", exact: true })).toHaveAttribute(
    "href",
    "/sf2-themes/themes/",
  );
  await expect(recovery.getByRole("link", { name: "Palette", exact: true })).toHaveAttribute(
    "href",
    "/sf2-themes/palette/",
  );
  await expect(recovery.getByRole("link", { name: "Preview", exact: true })).toHaveAttribute(
    "href",
    "/sf2-themes/preview/",
  );
  await expect(recovery.getByRole("link", { name: "Install", exact: true })).toHaveAttribute(
    "href",
    "/sf2-themes/install/",
  );
  await expect(recovery.getByRole("link", { name: "Arcade", exact: true })).toHaveAttribute(
    "href",
    "/sf2-themes/game/",
  );
  await expect(recovery.getByRole("link", { name: /GitHub repository/ })).toHaveAttribute(
    "href",
    "https://github.com/douglasjarquin/sf2-themes",
  );
});

test("sitemap.xml lists every indexable trailing-slash URL", async ({ request }) => {
  const response = await request.get("/sf2-themes/sitemap.xml");
  const body = await response.text();
  const locations = [...body.matchAll(/<loc>([^<]+)<\/loc>/g)].map((match) => match[1]);

  expect(response.ok()).toBe(true);
  expect(locations).toEqual([
    "https://douglasjarquin.github.io/sf2-themes/",
    "https://douglasjarquin.github.io/sf2-themes/themes/",
    "https://douglasjarquin.github.io/sf2-themes/palette/",
    "https://douglasjarquin.github.io/sf2-themes/preview/",
    "https://douglasjarquin.github.io/sf2-themes/install/",
    "https://douglasjarquin.github.io/sf2-themes/game/",
    ...["main", "akuma", "balrog", "blanka", "cammy", "chun-li", "dee-jay", "dhalsim", "e-honda", "fei-long", "guile", "ken", "m-bison", "ryu", "sagat", "t-hawk", "vega", "zangief"].map((id) => `https://douglasjarquin.github.io/sf2-themes/themes/${id}/`),
  ]);
  expect(locations.every((location) => location.endsWith("/"))).toBe(true);
});

test("public preview route is separate from the preserved gameplay asset namespace", async ({
  page,
  request,
}) => {
  await page.goto("./");

  const primaryPreview = page
    .getByRole("navigation", { name: "Primary" })
    .getByRole("link", { name: "PREVIEW", exact: true });
  await expect(primaryPreview).toHaveCount(0);

  const oldPageRoute = await request.get("screenshots/");
  const gameplayAsset = await request.get("screenshots/game/ryu.png");

  expect(oldPageRoute.status()).toBe(404);
  expect(gameplayAsset.status()).toBe(200);
  expect(gameplayAsset.headers()["content-type"]).toMatch(/image\/png/);
});

test("every indexable route has unique metadata, a self-canonical, and JSON-LD", async ({
  page,
  request,
}) => {
  const titles = [];
  const descriptions = [];

  for (const route of routes) {
    await page.goto(route.path);

    await expect(page).toHaveTitle(route.title);
    await expect(page.locator('meta[name="description"]')).toHaveAttribute(
      "content",
      route.description,
    );
    await expect(page.locator('link[rel="canonical"]')).toHaveAttribute("href", route.canonical);
    await expect(page.getByRole("heading", { level: 1 })).toHaveText(route.heading);

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

    const ogImage = page.locator('meta[property="og:image"]');
    const twitterImage = page.locator('meta[name="twitter:image"]');
    const icon = page.locator('link[rel="icon"]');
    await expect(page.locator('meta[property="og:title"]')).toHaveAttribute("content", route.title);
    await expect(page.locator('meta[property="og:description"]')).toHaveAttribute(
      "content",
      route.description,
    );
    await expect(page.locator('meta[property="og:url"]')).toHaveAttribute(
      "content",
      route.canonical,
    );
    await expect(ogImage).toHaveAttribute(
      "content",
      "https://douglasjarquin.github.io/sf2-themes/screenshots/game/ryu.png",
    );
    await expect(page.locator('meta[name="twitter:card"]')).toHaveAttribute(
      "content",
      "summary_large_image",
    );
    await expect(twitterImage).toHaveAttribute(
      "content",
      "https://douglasjarquin.github.io/sf2-themes/screenshots/game/ryu.png",
    );
    await expect(icon).toHaveAttribute("href", "/sf2-themes/screenshots/game/ryu.png");

    const imageResponse = await request.get("/sf2-themes/screenshots/game/ryu.png");
    expect(imageResponse.ok()).toBe(true);
    expect(imageResponse.headers()["content-type"]).toMatch(/image\/png/);

    titles.push(route.title);
    descriptions.push(route.description);
  }

  expect(new Set(titles).size).toBe(routes.length);
  expect(new Set(descriptions).size).toBe(routes.length);
});

test("index.html duplicates canonicalise to the trailing-slash home URL", async ({ page }) => {
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

test("the footer exposes internal IA and the skip link reaches main content", async ({ page }) => {
  await page.goto("./");

  const skip = page.getByRole("link", { name: "Skip to main content" });
  await page.keyboard.press("Tab");
  await expect(skip).toBeFocused();
  await skip.press("Enter");
  await expect(page.locator("#main-content")).toBeInViewport();

  const footer = page.getByRole("contentinfo");
  await expect(footer.getByRole("link", { name: "Home", exact: true })).toHaveAttribute(
    "href",
    "/sf2-themes/",
  );
  await expect(footer.getByRole("link", { name: "Themes", exact: true })).toHaveAttribute(
    "href",
    "/sf2-themes/themes/",
  );
  await expect(footer.getByRole("link", { name: "Install", exact: true })).toHaveAttribute(
    "href",
    "/sf2-themes/install/",
  );
  await expect(footer.getByRole("link", { name: /GitHub/ })).toHaveAttribute(
    "href",
    "https://github.com/douglasjarquin/sf2-themes",
  );
});
