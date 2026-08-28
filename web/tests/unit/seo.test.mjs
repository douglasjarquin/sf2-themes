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
  SITE_BASE,
  SITE_ORIGIN,
  absoluteUrl,
  canonicalUrl,
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

test("sitemap.xml lists the five indexable trailing-slash URLs", () => {
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
    ],
  );
});

test("JSON-LD describes the visible site and software without ratings or offers", () => {
  const jsonLd = productJsonLd();
  const serialized = JSON.stringify(jsonLd);
  assert.equal(jsonLd["@context"], "https://schema.org");
  assert.deepEqual(
    jsonLd["@graph"].map((node) => node["@type"]),
    ["WebSite", "SoftwareApplication"],
  );
  for (const node of jsonLd["@graph"]) {
    assert.equal(node.name, PRODUCT_NAME);
    assert.equal(node.description, PRODUCT_DESCRIPTION);
    assert.equal(node.url, "https://douglasjarquin.github.io/sf2-themes/");
    assert.equal("aggregateRating" in node, false);
    assert.equal("offers" in node, false);
  }
  assert.doesNotMatch(serialized, /aggregateRating|offers|"ratingValue"/);
  assert.equal(absoluteUrl(CABINET_IMAGE_PATH), "https://douglasjarquin.github.io/sf2-themes/screenshots/game/ryu.png");
});
