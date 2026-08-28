import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

test("homepage uses the featured palette preview while the arcade host remains reusable", async () => {
  // Given the homepage, featured preview, preserved arcade component, and browser boundary
  const [homepage, preview, arcade, host] = await Promise.all([
    readFile(new URL("../src/pages/index.astro", import.meta.url), "utf8"),
    readFile(new URL("../src/components/FeaturedPalettePreview.astro", import.meta.url), "utf8"),
    readFile(new URL("../src/components/ArcadeGame.astro", import.meta.url), "utf8"),
    readFile(new URL("../src/game/BrowserGameHost.ts", import.meta.url), "utf8"),
  ]);

  // When the source ownership boundary is inspected
  const legacyIdentifiers = /fightFrames|sequenceTimers|data-terminal-transcript/;

  // Then the homepage composes the static-first preview without mounting the live arcade
  assert.match(homepage, /<FeaturedPalettePreview\b/);
  assert.doesNotMatch(homepage, /<ArcadeGame\b|data-arcade-game/);
  assert.doesNotMatch(homepage, legacyIdentifiers);
  assert.match(preview, /import \{ paletteVariants \}/);
  assert.match(preview, /data-featured-palette-preview/);
  assert.match(preview, /data-code-pane/);
  assert.match(preview, /data-terminal-pane/);
  assert.match(preview, /data-preview-swatch/);
  assert.match(preview, /paletteVariants\[0\]/);
  assert.match(arcade, /data-arcade-game/);
  assert.match(host, /advanceClock/);
  assert.match(host, /ResizeObserver/);
  assert.match(host, /IntersectionObserver/);
  assert.match(host, /destroy\(\)/);
});
