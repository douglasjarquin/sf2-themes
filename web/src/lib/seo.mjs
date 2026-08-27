import astroConfig from "../../astro.config.mjs";

export const SITE_ORIGIN = astroConfig.site;
export const SITE_BASE = `${astroConfig.base.replace(/\/+$/, "")}/`;
export const PRODUCT_NAME = "sf2-themes";
export const PRODUCT_DESCRIPTION =
  "Street Fighter II color themes for WezTerm, Herdr, Neovim, and Codex.";
export const CABINET_IMAGE_PATH = `${SITE_BASE}screenshots/game/ryu.png`;

export const INDEXABLE_PATHS = Object.freeze([
  SITE_BASE,
  `${SITE_BASE}themes/`,
  `${SITE_BASE}palette/`,
  `${SITE_BASE}preview/`,
  `${SITE_BASE}install/`,
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
      },
    ],
  };
}
