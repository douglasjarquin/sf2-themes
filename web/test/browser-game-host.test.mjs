import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";

test("homepage delegates the arcade lifecycle to the reusable browser host", async () => {
  // Given the homepage and the planned reusable browser boundary
  const [homepage, component, host] = await Promise.all([
    readFile(new URL("../src/pages/index.astro", import.meta.url), "utf8"),
    readFile(new URL("../src/components/ArcadeGame.astro", import.meta.url), "utf8"),
    readFile(new URL("../src/game/BrowserGameHost.ts", import.meta.url), "utf8"),
  ]);

  // When the source ownership boundary is inspected
  const legacyIdentifiers = /fightFrames|sequenceTimers|data-terminal-transcript/;

  // Then the homepage composes the component and the host owns fixed-tick scheduling and cleanup
  assert.match(homepage, /<ArcadeGame\b/);
  assert.doesNotMatch(homepage, legacyIdentifiers);
  assert.match(component, /data-arcade-game/);
  assert.match(host, /advanceClock/);
  assert.match(host, /ResizeObserver/);
  assert.match(host, /IntersectionObserver/);
  assert.match(host, /destroy\(\)/);
});
