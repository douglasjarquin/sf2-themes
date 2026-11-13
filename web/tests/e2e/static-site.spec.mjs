import assert from "node:assert/strict";
import { test as nodeTest } from "node:test";
import {
  by,
  expectAttr,
  expectContainsText,
  expectCount,
  expectTitle,
  open,
  read,
  test,
  url,
} from "./support.mjs";

const origin = "https://douglasjarquin.github.io";

const stubs = [
  { path: "install/", target: "/sf2-themes/#install" },
  { path: "palette/", target: "/sf2-themes/#preview" },
  { path: "preview/", target: "/sf2-themes/#preview" },
  { path: "themes/", target: "/sf2-themes/#preview" },
  { path: "themes/ryu/", target: "/sf2-themes/?theme=ryu" },
];

const get = (path) => fetch(url(path));

test("the static package serves its foundation document", async (page) => {
  await open(page, "./");

  await expectTitle(page, "Street Fighter II terminal themes | sf2-themes");
});

test("the home page carries a self-canonical, description, and JSON-LD", async (page) => {
  await open(page, "./");

  const title = "Street Fighter II terminal themes | sf2-themes";
  const description =
    "Street Fighter II color themes for WezTerm, Herdr, Neovim, Codex, Starship, Lazygit, and Claude Code.";

  await expectAttr(page, 'meta[name="description"]', "content", description);
  await expectAttr(page, 'link[rel="canonical"]', "href", `${origin}/sf2-themes/`);
  await expectAttr(page, 'meta[property="og:title"]', "content", title);
  await expectAttr(page, 'meta[property="og:description"]', "content", description);
  await expectAttr(page, 'meta[property="og:url"]', "content", `${origin}/sf2-themes/`);
  await expectAttr(page, 'meta[name="twitter:card"]', "content", "summary");

  const jsonLd = JSON.parse(await read(page, 'script[type="application/ld+json"]', "text"));
  assert.deepEqual(
    jsonLd["@graph"].map((node) => node["@type"]),
    ["WebSite", "SoftwareApplication"],
  );
  assert.doesNotMatch(JSON.stringify(jsonLd), /aggregateRating|"offers"|ratingValue/);
  const software = jsonLd["@graph"].find((node) => node["@type"] === "SoftwareApplication");
  assert.equal(software.operatingSystem, "Linux, macOS");
  assert.equal(software.downloadUrl, "https://github.com/douglasjarquin/sf2-themes/releases/latest");
  assert.equal(software.codeRepository, "https://github.com/douglasjarquin/sf2-themes");
});

// Agents that fetch rather than render read discovery files they find through links, so the links have to be in
// the served HTML, not added by a script.
nodeTest("the raw home HTML links docs, license, llms.txt, and the sitemap", async () => {
  const response = await get("/sf2-themes/");
  const html = await response.text();

  assert.equal(response.ok, true);
  assert.ok(html.includes('<link rel="alternate" type="text/plain" title="llms.txt" href="/sf2-themes/llms.txt">'));
  assert.ok(html.includes(`<link rel="sitemap" type="application/xml" href="${origin}/sf2-themes/sitemap.xml">`));
  assert.ok(html.includes('<link rel="license" href="https://github.com/douglasjarquin/sf2-themes/blob/main/LICENSE">'));
  assert.ok(html.includes('href="https://github.com/douglasjarquin/sf2-themes/tree/main/docs/sf2-themes"'));
  assert.ok(html.includes('<a href="/sf2-themes/llms.txt">llms.txt</a>'));
  assert.match(html, /free and open source \(MIT\)/);
  assert.match(html, /<code class="language-sh"[\s>]/);

  for (const path of ["/sf2-themes/llms.txt", "/sf2-themes/sitemap.xml"]) {
    assert.equal((await get(path)).status, 200, `${path} resolves`);
  }
});

test("the footer links docs, license, and llms.txt", async (page) => {
  await open(page, "./");

  await expectAttr(page, by("footer a", /docs/), "href", "https://github.com/douglasjarquin/sf2-themes/tree/main/docs/sf2-themes");
  await expectAttr(page, by("footer a", /mit license/), "href", "https://github.com/douglasjarquin/sf2-themes/blob/main/LICENSE");
  await expectAttr(page, by("footer a", "llms.txt"), "href", "/sf2-themes/llms.txt");
});

nodeTest("robots.txt allows the project path and names the sitemap", async () => {
  const response = await get("/sf2-themes/robots.txt");
  const body = await response.text();

  assert.equal(response.ok, true);
  assert.equal(
    body,
    "User-agent: *\nAllow: /sf2-themes/\n\nSitemap: https://douglasjarquin.github.io/sf2-themes/sitemap.xml\n",
  );
});

nodeTest("llms.txt summarizes the CLI, catalog, live site, and repository", async () => {
  const response = await get("/sf2-themes/llms.txt");
  const body = await response.text();

  assert.equal(response.ok, true);
  assert.ok(body.includes("sf2-themes is a single-binary CLI named `sf2-themes`"));
  assert.ok(
    body.includes(
      "curl -fsSL https://github.com/douglasjarquin/sf2-themes/releases/latest/download/install.sh | sh",
    ),
  );
  assert.ok(!body.includes("uvx"));
  assert.ok(body.includes("36-theme TOML catalog"));
  assert.ok(body.includes("Live site: https://douglasjarquin.github.io/sf2-themes/"));
  assert.ok(body.includes("Repository: https://github.com/douglasjarquin/sf2-themes"));
  assert.ok(!body.includes("game"));
  assert.ok(body.includes("## Pricing\n\nFree and open source under the MIT License."));
  assert.ok(body.includes("## Docs\n"));
  assert.ok(body.includes("```sh\nsf2 apps\nsf2 apply wezterm --theme ryu\n"));
});

nodeTest("sitemap.xml lists only the home route", async () => {
  const response = await get("/sf2-themes/sitemap.xml");
  const body = await response.text();
  const locations = [...body.matchAll(/<loc>([^<]+)<\/loc>/g)].map((match) => match[1]);

  assert.equal(response.ok, true);
  assert.deepEqual(locations, ["https://douglasjarquin.github.io/sf2-themes/"]);
});

test("index.html duplicates canonicalise to the trailing-slash home URL", async (page) => {
  await open(page, "index.html");

  await expectAttr(page, 'link[rel="canonical"]', "href", "https://douglasjarquin.github.io/sf2-themes/");
  await expectAttr(page, 'meta[property="og:url"]', "content", "https://douglasjarquin.github.io/sf2-themes/");
});

nodeTest("retired routes are noindex meta-refresh stubs pointing home", async () => {
  for (const stub of stubs) {
    const response = await get(`/sf2-themes/${stub.path}`);
    const html = await response.text();

    assert.equal(response.ok, true);
    assert.ok(html.includes('content="noindex"'));
    assert.ok(html.includes(`url=${stub.target}`));
    assert.ok(html.includes(`<link rel="canonical" href="${origin}${stub.target}"`));
  }
});

test("the project 404 page is branded, linked, and noindexed", async (page) => {
  const missing = await get("/sf2-themes/this-path-is-not-a-route/");
  assert.equal(missing.status, 404);

  await open(page, "404.html");

  await expectTitle(page, "Page not found | sf2-themes");
  await expectAttr(page, 'meta[name="robots"]', "content", "noindex");
  await expectCount(page, 'script[type="application/ld+json"]', 0);
  await expectContainsText(page, '[data-testid="project-404"]', "This URL is not a page in sf2-themes.");
  await expectContainsText(page, "h1", "Round over.");

  const recovery = 'nav[aria-label="Known routes"] a';
  await expectAttr(page, by(recovery, /home/), "href", "/sf2-themes/");
  await expectAttr(page, by(recovery, /install/), "href", "/sf2-themes/#install");
  await expectAttr(page, by(recovery, /ports/), "href", "/sf2-themes/#ports");
  await expectAttr(page, by(recovery, /github repository/), "href", "https://github.com/douglasjarquin/sf2-themes");
});

nodeTest("the removed game route is absent from the static package", async () => {
  const response = await get("/sf2-themes/game/");

  assert.equal(response.status, 404);
});
