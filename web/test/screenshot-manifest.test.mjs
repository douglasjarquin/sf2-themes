import assert from "node:assert/strict";
import { readFile, stat } from "node:fs/promises";
import test from "node:test";

import { paletteVariants } from "../src/data/theme-data.mjs";

const manifestPath = new URL("../public/screenshots/manifest.json", import.meta.url);

test("screenshot manifest contains every cabinet fighter at the standard viewport", async () => {
  const manifest = JSON.parse(await readFile(manifestPath, "utf8"));

  assert.deepEqual(manifest.viewport, { width: 1280, height: 720, deviceScaleFactor: 1 });
  assert.deepEqual(
    manifest.entries.map(({ id }) => id),
    paletteVariants.map(({ id }) => id),
  );
  assert.equal(manifest.entries.length, 36);
  assert.equal(new Set(manifest.entries.flatMap(({ p1, p2 }) => [p1, p2])).size, 17);

  for (const entry of manifest.entries) {
    assert.match(entry.image, new RegExp(`^game/${entry.id}\\.png$`));
    assert.equal(typeof entry.moment, "string");
    assert.deepEqual(entry.logicalGrid, { columns: 96, rows: 40 });
    assert.equal(entry.width, 1280);
    assert.equal(entry.height, 720);
    const imageStats = await stat(new URL(`../public/screenshots/${entry.image}`, import.meta.url));
    assert.ok(imageStats.size > 0, `${entry.image} should not be empty`);
  }
});
