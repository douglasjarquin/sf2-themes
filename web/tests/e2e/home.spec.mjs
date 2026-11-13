import assert from "node:assert/strict";
import { createServer } from "node:http";
import {
  baseURL,
  by,
  click,
  eventually,
  expectAttr,
  expectContainsText,
  expectCount,
  expectHidden,
  expectLacksText,
  expectText,
  expectTitle,
  expectURLContains,
  expectVisible,
  open,
  read,
  test,
} from "./support.mjs";

const INSTALL_COMMAND =
  "curl -fsSL https://github.com/douglasjarquin/sf2-themes/releases/latest/download/install.sh | sh";

const H1 = by("h1", "Fight for your terminal.");
const APPLY_CMD = by("[data-step] [data-step-cmd]", undefined, { index: 1 });

const noOverflow = () =>
  document.documentElement.scrollWidth <= document.documentElement.clientWidth + 1;

const copied = (page) => page.evaluate(() => window.__copied);

test("home renders the hero, live preview, install steps, and ports", async (page) => {
  await open(page, "./");

  await expectTitle(page, "Street Fighter II terminal themes | sf2-themes");
  await expectVisible(page, H1);
  await expectContainsText(page, ".meta", "36 themes · 8 apps plus Slack");
  await expectContainsText(page, ".meta", "one binary for macOS and Linux");
  await expectContainsText(page, ".meta", "free and open source (MIT)");
  await expectLacksText(page, ".meta", /python|\buv\b/i);

  await expectText(page, "#preview [data-t='fileId']", "sf2-ryu");
  await expectCount(page, "[data-pane]", 7);
  await expectCount(page, "[data-tab]", 7);

  await expectCount(page, "[data-step]", 2);
  await expectText(page, "[data-step-cmd]", INSTALL_COMMAND);
  await expectLacksText(page, "#install", /python|\buv\b|uvx/i);
  await expectCount(page, "[data-port-id]", 7);
  await expectCount(page, ".port-row", 7);
});

test("theme selection rewrites the preview and install steps", async (page) => {
  await open(page, "./");

  await page.locator("[data-site-select]").selectOption("ken");
  await expectText(page, "#preview [data-t='displayName']", "Ken");
  await expectText(page, "#preview [data-t='fileId']", "sf2-ken");
  await expectText(page, APPLY_CMD, "sf2 apply wezterm --theme ken");
  await expectURLContains(page, "theme=ken");
});

test("SampleBlock tabs switch panes with roving selection state", async (page) => {
  await open(page, "./");

  await expectAttr(page, "[data-tab]", "aria-selected", "true");
  await expectVisible(page, '[data-pane="shell"]');
  await expectHidden(page, '[data-pane="nvim"]');

  await click(page, '[data-tab="ansi"]');
  await expectVisible(page, '[data-pane="ansi"]');
  await expectHidden(page, '[data-pane="shell"]');
  await expectAttr(page, '[data-tab="ansi"]', "aria-selected", "true");
  await expectAttr(page, "[data-tab]", "aria-selected", "false");
});

test("port selection rewrites the install steps", async (page) => {
  await open(page, "./");

  await click(page, '[data-port-id="nvim"]');
  await expectAttr(page, '[data-port-id="nvim"]', "aria-pressed", "true");
  await expectAttr(page, '[data-port-id="wezterm"]', "aria-pressed", "false");
  await expectText(page, APPLY_CMD, "sf2 apply nvim --theme ryu");
  await expectContainsText(
    page,
    "[data-port-note-out]",
    "The loader applies your pick every time Neovim starts.",
  );
});

test(
  "step copy buttons send the install and apply payloads",
  async (page) => {
    await open(page, "./");

    await click(page, "[data-step-copy]");
    await expectText(page, "[data-step-copy]", "copied");
    await eventually(() => copied(page), (value) => assert.equal(value, INSTALL_COMMAND));

    await click(page, "[data-step] [data-step-copy]", 1);
    await eventually(() => copied(page), (value) => assert.equal(value, "sf2 apply wezterm --theme ryu"));

    await click(page, '[data-port-id="nvim"]');
    await click(page, "[data-step] [data-step-copy]", 1);
    await eventually(() => copied(page), (value) => assert.equal(value, "sf2 apply nvim --theme ryu"));
  },
  {
    initScript: () => {
      Object.defineProperty(navigator, "clipboard", {
        configurable: true,
        value: { writeText: async (value) => { window.__copied = value; } },
      });
    },
  },
);

test(
  "copy buttons are hidden when the clipboard API is unavailable",
  async (page) => {
    await open(page, "./");

    await expectCount(page, "[data-step-copy]", 2);
    await expectCount(page, by("[data-step-copy]", undefined, { visible: true }), 0);
  },
  {
    initScript: () => {
      Object.defineProperty(navigator, "clipboard", { configurable: true, value: undefined });
    },
  },
);

// Stagehand's runtime needs script execution, so the page's own scripts are blocked with a CSP header from a
// local proxy instead of disabling JavaScript for the whole browser. The site has no <noscript> content.
test("the home page remains useful without client JavaScript", async (page) => {
  const proxy = createServer(async (request, response) => {
    const upstream = await fetch(new URL(request.url, baseURL));
    response.writeHead(upstream.status, {
      "content-type": upstream.headers.get("content-type") ?? "application/octet-stream",
      "content-security-policy": "script-src 'none'",
    });
    response.end(Buffer.from(await upstream.arrayBuffer()));
  });
  await new Promise((resolve) => proxy.listen(0, "127.0.0.1", resolve));
  try {
    await open(page, "./?theme=ken", `http://127.0.0.1:${proxy.address().port}/sf2-themes/`);

    assert.equal(await read(page, "html", "attr", "data-theme"), "ryu", "the deep-link script ran");
    await expectVisible(page, H1);
    await expectText(page, "[data-step-cmd]", INSTALL_COMMAND);
    await expectAttr(page, ".step__cmd code", "class", "language-sh");
    await expectContainsText(page, ".meta", "free and open source (MIT)");
    await expectAttr(page, by("footer a", /docs/), "href", "https://github.com/douglasjarquin/sf2-themes/tree/main/docs/sf2-themes");
    await expectCount(page, ".port-row", 7);
    await expectVisible(page, '[data-pane="shell"]');
  } finally {
    proxy.close();
  }
});

test(
  "the home column reflows without horizontal overflow",
  async (page) => {
    await open(page, "./");
    await eventually(() => page.evaluate(noOverflow), (value) => assert.equal(value, true));

    await page.setViewportSize(375, 844);
    await eventually(() => page.evaluate(noOverflow), (value) => assert.equal(value, true));
  },
  { viewport: { width: 1280, height: 900 } },
);
