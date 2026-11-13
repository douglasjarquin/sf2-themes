import astroConfig from "../../astro.config.mjs";

export const SITE_ORIGIN = astroConfig.site;
export const SITE_BASE = `${astroConfig.base.replace(/\/+$/, "")}/`;
export const PRODUCT_NAME = "sf2-themes";
export const PRODUCT_DESCRIPTION =
  "Street Fighter II color themes for WezTerm, Herdr, Neovim, Codex, Starship, Lazygit, and Claude Code.";
export const REPOSITORY_URL = "https://github.com/douglasjarquin/sf2-themes";
export const PRODUCT_OPERATING_SYSTEM = "Linux, macOS";
export const DOWNLOAD_URL = `${REPOSITORY_URL}/releases/latest`;
export const INSTALL_COMMAND =
  "curl -fsSL https://github.com/douglasjarquin/sf2-themes/releases/latest/download/install.sh | sh";
export const LICENSE_URL = `${REPOSITORY_URL}/blob/main/LICENSE`;
export const DOCS_URL = `${REPOSITORY_URL}/tree/main/docs/sf2-themes`;
export const PRICING_TEXT =
  "Free and open source under the MIT License. No account, API key, or paid tier.";

// The first-party docs agents should ground answers on, in reading order. Paths are repository-relative so a
// test can prove each one exists; llms.txt links their GitHub pages.
export const DOCS = Object.freeze([
  { title: "Install", path: "docs/sf2-themes/install.md", summary: "the installer, version pins, verification, and upgrading from 1.x" },
  { title: "Apply and setup", path: "docs/sf2-themes/setup-and-apply.md", summary: "apply a theme to each app in one step, ownership rules, --adopt, and --dry-run" },
  { title: "Commands", path: "docs/sf2-themes/commands.md", summary: "apps, themes, show, validate, current, and boss aliases" },
  { title: "File writes", path: "docs/sf2-themes/file-writes.md", summary: "paths, overrides, symlinks, and backups" },
  { title: "Uninstall", path: "docs/sf2-themes/uninstall.md", summary: "files and settings to remove" },
  { title: "Changelog", path: "CHANGELOG.md", summary: "what changed in each release" },
]);

export function docUrl(path) {
  return `${REPOSITORY_URL}/blob/main/${path}`;
}

export function llmsTxtLocation() {
  return `${SITE_BASE}llms.txt`;
}

export const INDEXABLE_PATHS = Object.freeze([
  SITE_BASE,
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

sf2-themes is a single-binary CLI named \`sf2-themes\`, also linked as \`sf2\`, for macOS and Linux on arm64 and amd64. It has no runtime dependencies. It installs a 36-theme TOML catalog (dark and light main variants plus the Super Street Fighter II Turbo roster) into WezTerm, Herdr, Neovim, Codex, Starship, Lazygit, and Claude Code.

Live site: ${liveUrl}
Repository: ${REPOSITORY_URL}

Install the latest release into ~/.local/bin:

\`\`\`sh
${INSTALL_COMMAND}
\`\`\`

Pin a release by piping into \`SF2_THEMES_VERSION=2.0.0 sh\` instead. If a 1.x copy installed with uv is present, run \`uv tool uninstall sf2-themes\` first.

Catalog ids stay short (\`ryu\`, \`ken-light\`). Installed identities use \`sf2-<catalog-id>\`.

Then apply a theme to an app. \`sf2 apps\` lists the app ids, and \`--dry-run\` prints every write without making it:

\`\`\`sh
sf2 apps
sf2 apply wezterm --theme ryu
sf2 apply nvim --theme ken-light --dry-run
\`\`\`

## Pricing

${PRICING_TEXT} License: ${LICENSE_URL}

## Docs

${DOCS.map(({ title, path, summary }) => `- [${title}](${docUrl(path)}): ${summary}`).join("\n")}

## Pages

- [Home](${liveUrl}) — palette preview, install commands, and per-app ports
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
        downloadUrl: DOWNLOAD_URL,
        codeRepository: REPOSITORY_URL,
        license: LICENSE_URL,
        isAccessibleForFree: true,
      },
    ],
  };
}
