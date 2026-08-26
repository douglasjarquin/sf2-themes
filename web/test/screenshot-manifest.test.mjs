import assert from "node:assert/strict";
import { readFile, stat } from "node:fs/promises";
import test from "node:test";

const manifestPath = new URL("../public/screenshots/manifest.json", import.meta.url);

test("screenshot manifest contains every cabinet fighter at the standard viewport", async () => {
  const manifest = JSON.parse(await readFile(manifestPath, "utf8"));

  assert.deepEqual(manifest.viewport, { width: 1280, height: 720 });
  assert.deepEqual(
    manifest.entries.map(({ id }) => id),
    ["ryu", "ken", "chun-li", "guile"],
  );

  for (const entry of manifest.entries) {
    assert.match(entry.image, new RegExp(`^${entry.id}\\.png$`));
    assert.equal(typeof entry.moment, "string");
    const imageStats = await stat(new URL(`../public/screenshots/${entry.image}`, import.meta.url));
    assert.ok(imageStats.size > 0, `${entry.image} should not be empty`);
  }
});
