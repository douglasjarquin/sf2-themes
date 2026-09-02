import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

import {
  CABINET_IMAGE_PATH,
  INDEXABLE_PATHS,
  PRODUCT_DESCRIPTION,
  PRODUCT_NAME,
  PRODUCT_OPERATING_SYSTEM,
  REPOSITORY_URL,
  SITE_BASE,
  SITE_ORIGIN,
  absoluteUrl,
  canonicalUrl,
  llmsTxt,
  productJsonLd,
  robotsTxt,
  sitemapLocation,
  sitemapXml,
  withTrailingSlash,
} from "../../src/lib/seo.mjs";

const publicDirectory = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../public");

test("canonical URLs are HTTPS trailing-slash addresses without index.html", () => {
  assert.equal(SITE_ORIGIN, "https://douglasjarquin.github.io");
  assert.equal(SITE_BASE, "/sf2-themes/");
  assert.equal(withTrailingSlash("/sf2-themes/index.html"), "/sf2-themes/");
  assert.equal(withTrailingSlash("/sf2-themes/themes"), "/sf2-themes/themes/");
  assert.equal(
    canonicalUrl("/sf2-themes/index.html"),
    "https://douglasjarquin.github.io/sf2-themes/",
  );
  assert.equal(
    canonicalUrl("/sf2-themes/themes/index.html"),
    "https://douglasjarquin.github.io/sf2-themes/themes/",
  );
});

test("robots.txt allows the project path and points at the project sitemap", () => {
  const committed = readFileSync(path.join(publicDirectory, "robots.txt"), "utf8");
  assert.equal(committed, robotsTxt());
  assert.match(committed, /^User-agent: \*\nAllow: \/sf2-themes\/\n\nSitemap: /);
  assert.equal(sitemapLocation(), "https://douglasjarquin.github.io/sf2-themes/sitemap.xml");
  assert.doesNotMatch(committed, /Allow: \/\n/);
});

test("sitemap.xml lists every indexable trailing-slash URL", () => {
  const committed = readFileSync(path.join(publicDirectory, "sitemap.xml"), "utf8");
  assert.equal(committed, sitemapXml());
  assert.deepEqual(
    INDEXABLE_PATHS.map((pathname) => canonicalUrl(pathname)),
    [
      "https://douglasjarquin.github.io/sf2-themes/",
      "https://douglasjarquin.github.io/sf2-themes/themes/",
      "https://douglasjarquin.github.io/sf2-themes/palette/",
      "https://douglasjarquin.github.io/sf2-themes/preview/",
      "https://douglasjarquin.github.io/sf2-themes/install/",
      "https://douglasjarquin.github.io/sf2-themes/game/",
      ...["main", "akuma", "balrog", "blanka", "cammy", "chun-li", "dee-jay", "dhalsim", "e-honda", "fei-long", "guile", "ken", "m-bison", "ryu", "sagat", "t-hawk", "vega", "zangief"].map((id) => `https://douglasjarquin.github.io/sf2-themes/themes/${id}/`),
    ],
  );
});

test("llms.txt names the CLI, catalog size, live site, and repository", () => {
  const committed = readFileSync(path.join(publicDirectory, "llms.txt"), "utf8");
  assert.equal(committed, llmsTxt());
  assert.match(committed, /^# sf2-themes\n/);
  assert.match(committed, /Python 3\.11 CLI named `sf2-themes`/);
  assert.match(committed, /36-theme TOML catalog/);
  assert.match(committed, /Live site: https:\/\/douglasjarquin\.github\.io\/sf2-themes\//);
  assert.match(committed, /Repository: https:\/\/github\.com\/douglasjarquin\/sf2-themes/);
  assert.doesNotMatch(committed, /Lazygit/);
});

test("JSON-LD describes the visible site and software without ratings or offers", () => {
  const jsonLd = productJsonLd();
  const serialized = JSON.stringify(jsonLd);
  assert.equal(jsonLd["@context"], "https://schema.org");
  assert.deepEqual(
    jsonLd["@graph"].map((node) => node["@type"]),
    ["WebSite", "SoftwareApplication"],
  );
  const [website, software] = jsonLd["@graph"];
  for (const node of jsonLd["@graph"]) {
    assert.equal(node.name, PRODUCT_NAME);
    assert.equal(node.description, PRODUCT_DESCRIPTION);
    assert.equal(node.url, "https://douglasjarquin.github.io/sf2-themes/");
    assert.equal("aggregateRating" in node, false);
    assert.equal("offers" in node, false);
  }
  assert.equal("operatingSystem" in website, false);
  assert.equal("downloadUrl" in website, false);
  assert.equal("codeRepository" in website, false);
  assert.equal(software.operatingSystem, PRODUCT_OPERATING_SYSTEM);
  assert.equal(software.downloadUrl, REPOSITORY_URL);
  assert.equal(software.codeRepository, REPOSITORY_URL);
  assert.doesNotMatch(serialized, /aggregateRating|offers|"ratingValue"/);
  assert.equal(absoluteUrl(CABINET_IMAGE_PATH), "https://douglasjarquin.github.io/sf2-themes/screenshots/game/ryu.png");
});
