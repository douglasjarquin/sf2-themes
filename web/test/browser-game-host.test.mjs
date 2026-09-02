import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

test("homepage uses the live theme preview while the arcade host remains reusable", async () => {
  // Given the homepage, featured preview, preserved arcade component, and browser boundary
  const [homepage, featured, shared, styles, arcade, host] = await Promise.all([
    readFile(new URL("../src/pages/index.astro", import.meta.url), "utf8"),
    readFile(new URL("../src/components/FeaturedPalettePreview.astro", import.meta.url), "utf8"),
    readFile(new URL("../src/components/PalettePreview.astro", import.meta.url), "utf8"),
    readFile(new URL("../src/styles/palette-preview.css", import.meta.url), "utf8"),
    readFile(new URL("../src/components/ArcadeGame.astro", import.meta.url), "utf8"),
    readFile(new URL("../src/game/BrowserGameHost.ts", import.meta.url), "utf8"),
  ]);

  // When the source ownership boundary is inspected
  const legacyIdentifiers = /fightFrames|sequenceTimers|data-terminal-transcript/;

  // Then the homepage composes the static-first preview without mounting the live arcade
  assert.match(homepage, /<FeaturedPalettePreview\b/);
  assert.doesNotMatch(homepage, /<ArcadeGame\b|data-arcade-game/);
  assert.doesNotMatch(homepage, legacyIdentifiers);
  assert.match(featured, /data-home-theme-preview/);
  assert.doesNotMatch(featured, /data-code-pane|data-terminal-pane|data-preview-swatch|syntaxPattern/);
  assert.match(shared, /data-preview-variant/);
  assert.match(shared, /data-code-pane/);
  assert.match(shared, /data-terminal-pane/);
  assert.match(shared, /data-preview-swatch/);
  assert.match(shared, /randomize/);
  assert.match(shared, /highlightCodeLine/);
  assert.match(styles, /\.syntax-token--keyword \{[^}]*font-weight: 700/s);
  assert.match(styles, /\.syntax-token--plain \{[^}]*var\(--preview-fg\)/s);
  assert.match(styles, /\.palette-preview__pane--terminal \{[^}]*background: var\(--preview-surface\)/s);
  assert.match(styles, /\.palette-preview__pane \{[^}]*background: transparent/s);
  assert.match(arcade, /data-arcade-game/);
  assert.match(host, /advanceClock/);
  assert.match(host, /ResizeObserver/);
  assert.match(host, /IntersectionObserver/);
  assert.match(host, /destroy\(\)/);
});
