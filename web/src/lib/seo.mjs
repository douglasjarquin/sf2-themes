import astroConfig from "../../astro.config.mjs";
import { themeFamilies } from "../data/site-theme-data.mjs";

export const SITE_ORIGIN = astroConfig.site;
export const SITE_BASE = `${astroConfig.base.replace(/\/+$/, "")}/`;
export const PRODUCT_NAME = "sf2-themes";
export const PRODUCT_DESCRIPTION =
  "Street Fighter II color themes for WezTerm, Herdr, Neovim, Codex, and Starship.";
export const REPOSITORY_URL = "https://github.com/douglasjarquin/sf2-themes";
export const PRODUCT_OPERATING_SYSTEM = "Linux, macOS, Windows";
export const CABINET_IMAGE_PATH = `${SITE_BASE}screenshots/game/ryu.png`;

export const INDEXABLE_PATHS = Object.freeze([
  SITE_BASE,
  `${SITE_BASE}themes/`,
  `${SITE_BASE}palette/`,
  `${SITE_BASE}preview/`,
  `${SITE_BASE}install/`,
  `${SITE_BASE}game/`,
  ...themeFamilies.map(({ id }) => `${SITE_BASE}themes/${id}/`),
]);

export function withTrailingSlash(pathname) {
  const withoutIndex = pathname.replace(/\/index\.html$/, "/");
  return withoutIndex.endsWith("/") ? withoutIndex : `${withoutIndex}/`;
}

export function canonicalUrl(pathname) {
  return new URL(withTrailingSlash(pathname), SITE_ORIGIN).href;
}

export function absoluteUrl(pathname) {
  return new URL(pathname, SITE_ORIGIN).href;
}

export function sitemapLocation() {
  return absoluteUrl(`${SITE_BASE}sitemap.xml`);
}

export function robotsTxt() {
  return `User-agent: *\nAllow: ${SITE_BASE}\n\nSitemap: ${sitemapLocation()}\n`;
}

export function sitemapXml() {
  const entries = INDEXABLE_PATHS.map(
    (pathname) => `  <url>\n    <loc>${canonicalUrl(pathname)}</loc>\n  </url>`,
  ).join("\n");
  return `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n${entries}\n</urlset>\n`;
}

export function llmsTxt() {
  const liveUrl = canonicalUrl(SITE_BASE);
  return `# sf2-themes

> ${PRODUCT_DESCRIPTION}

sf2-themes is a Python 3.11 CLI named \`sf2-themes\`. It installs a 36-theme TOML catalog (dark and light main variants plus the Super Street Fighter II Turbo roster) into WezTerm, Herdr, Neovim, Codex, and Starship.

Live site: ${liveUrl}
Repository: ${REPOSITORY_URL}

Run the CLI from GitHub with uv:

\`\`\`
uvx --from git+https://github.com/douglasjarquin/sf2-themes.git sf2-themes --version
\`\`\`

Catalog ids stay short (\`ryu\`, \`ken-light\`). Installed identities use \`sf2-<catalog-id>\`.

## Pages

- [Home](${liveUrl})
- [Adapters](${canonicalUrl(`${SITE_BASE}themes/`)})
- [Palette catalog](${canonicalUrl(`${SITE_BASE}palette/`)})
- [Palette preview](${canonicalUrl(`${SITE_BASE}preview/`)})
- [Install](${canonicalUrl(`${SITE_BASE}install/`)})
- [Arcade game](${canonicalUrl(`${SITE_BASE}game/`)})
`;
}

export function productJsonLd() {
  const url = canonicalUrl(SITE_BASE);
  return {
    "@context": "https://schema.org",
    "@graph": [
      {
        "@type": "WebSite",
        name: PRODUCT_NAME,
        url,
        description: PRODUCT_DESCRIPTION,
      },
      {
        "@type": "SoftwareApplication",
        name: PRODUCT_NAME,
        url,
        description: PRODUCT_DESCRIPTION,
        applicationCategory: "DeveloperApplication",
        operatingSystem: PRODUCT_OPERATING_SYSTEM,
        downloadUrl: REPOSITORY_URL,
        codeRepository: REPOSITORY_URL,
      },
    ],
  };
}
