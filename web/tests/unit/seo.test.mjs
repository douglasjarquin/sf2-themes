import assert from "node:assert/strict";
import { existsSync, readFileSync } from "node:fs";
import path from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";

import {
  DOCS,
  DOCS_URL,
  INDEXABLE_PATHS,
  LICENSE_URL,
  PRICING_TEXT,
  DOWNLOAD_URL,
  INSTALL_COMMAND,
  PRODUCT_DESCRIPTION,
  PRODUCT_NAME,
  PRODUCT_OPERATING_SYSTEM,
  REPOSITORY_URL,
  SITE_BASE,
  SITE_ORIGIN,
  canonicalUrl,
  docUrl,
  llmsTxt,
  llmsTxtLocation,
  productJsonLd,
  robotsTxt,
  sitemapLocation,
  sitemapXml,
  withTrailingSlash,
} from "../../src/lib/seo.mjs";

const publicDirectory = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../public");
const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../../..");

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
    ],
  );
});

test("llms.txt names the CLI, install command, catalog size, live site, and repository", () => {
  const committed = readFileSync(path.join(publicDirectory, "llms.txt"), "utf8");
  assert.equal(committed, llmsTxt());
  assert.match(committed, /^# sf2-themes\n/);
  assert.match(committed, /single-binary CLI named `sf2-themes`, also linked as `sf2`/);
  assert.ok(committed.includes(`\`\`\`sh\n${INSTALL_COMMAND}\n\`\`\``));
  assert.doesNotMatch(committed, /uvx|uv tool install|Python/);
  assert.match(committed, /36-theme TOML catalog/);
  assert.match(committed, /Live site: https:\/\/douglasjarquin\.github\.io\/sf2-themes\//);
  assert.match(committed, /Repository: https:\/\/github\.com\/douglasjarquin\/sf2-themes/);
  assert.match(committed, /WezTerm, Herdr, Neovim, Codex, Starship, Lazygit, and Claude Code/);
});

test("llms.txt states the price and indexes first-party docs that exist in the repository", () => {
  const committed = readFileSync(path.join(publicDirectory, "llms.txt"), "utf8");
  assert.match(committed, /\n## Pricing\n\nFree and open source under the MIT License\./);
  assert.ok(committed.includes(PRICING_TEXT));
  assert.ok(committed.includes(LICENSE_URL));
  assert.ok(existsSync(path.join(repositoryRoot, "LICENSE")));
  assert.match(readFileSync(path.join(repositoryRoot, "LICENSE"), "utf8"), /^MIT License\n/);
  assert.equal(LICENSE_URL, "https://github.com/douglasjarquin/sf2-themes/blob/main/LICENSE");
  assert.equal(DOCS_URL, "https://github.com/douglasjarquin/sf2-themes/tree/main/docs/sf2-themes");
  assert.ok(existsSync(path.join(repositoryRoot, "docs/sf2-themes")));
  assert.ok(DOCS.length > 0);
  for (const doc of DOCS) {
    assert.ok(existsSync(path.join(repositoryRoot, doc.path)), `${doc.path} exists`);
    assert.ok(committed.includes(`- [${doc.title}](${docUrl(doc.path)}): ${doc.summary}\n`), `${doc.path} is listed`);
  }
  assert.ok(committed.includes("\nsf2 apply wezterm --theme ryu\n"));
  assert.equal(llmsTxtLocation(), "/sf2-themes/llms.txt");
});

test("every code fence in llms.txt is language-tagged", () => {
  const committed = readFileSync(path.join(publicDirectory, "llms.txt"), "utf8");
  const fences = committed.split("\n").filter((line) => line.startsWith("```"));
  assert.ok(fences.length >= 2);
  assert.equal(fences.length % 2, 0, "fences are balanced");
  fences.forEach((line, index) => {
    if (index % 2 === 0) assert.match(line, /^```[a-z]+$/, `opening fence ${index / 2 + 1} has a language`);
    else assert.equal(line, "```");
  });
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
  assert.equal(software.downloadUrl, DOWNLOAD_URL);
  assert.equal(software.codeRepository, REPOSITORY_URL);
  assert.equal(software.license, LICENSE_URL);
  assert.equal(software.isAccessibleForFree, true);
  assert.equal("license" in website, false);
  assert.doesNotMatch(serialized, /aggregateRating|offers|"ratingValue"/);
});

test("the install command pipes the latest release's installer into sh", () => {
  assert.equal(
    INSTALL_COMMAND,
    "curl -fsSL https://github.com/douglasjarquin/sf2-themes/releases/latest/download/install.sh | sh",
  );
  assert.equal(DOWNLOAD_URL, "https://github.com/douglasjarquin/sf2-themes/releases/latest");
  assert.equal(PRODUCT_OPERATING_SYSTEM, "Linux, macOS");
});
