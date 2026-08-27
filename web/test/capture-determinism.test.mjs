import assert from "node:assert/strict";
import { createHash } from "node:crypto";
import { promises as fs } from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";

import { captureArchive } from "../../scripts/capture-game-screenshots.mjs";

async function sha256File(filePath) {
  return createHash("sha256").update(await fs.readFile(filePath)).digest("hex");
}

test("captureArchive renders byte-identical output across repeated runs of the same theme and seed", async () => {
  const parent = await fs.mkdtemp(path.join(os.tmpdir(), "sf2-capture-determinism-"));
  const firstOutput = path.join(parent, "first");
  const secondOutput = path.join(parent, "second");
  try {
    await captureArchive({ theme: "ryu", seed: "capture-determinism-check", outputDirectory: firstOutput });
    await captureArchive({ theme: "ryu", seed: "capture-determinism-check", outputDirectory: secondOutput });

    const firstManifest = JSON.parse(await fs.readFile(path.join(firstOutput, "manifest.json"), "utf8"));
    const secondManifest = JSON.parse(await fs.readFile(path.join(secondOutput, "manifest.json"), "utf8"));
    assert.deepEqual(
      firstManifest.entries.map(({ id, sha256, occupancy }) => ({ id, sha256, occupancy })),
      secondManifest.entries.map(({ id, sha256, occupancy }) => ({ id, sha256, occupancy })),
    );

    const [firstImageHash, secondImageHash] = await Promise.all([
      sha256File(path.join(firstOutput, "game", "ryu.png")),
      sha256File(path.join(secondOutput, "game", "ryu.png")),
    ]);
    assert.equal(firstImageHash, secondImageHash);
  } finally {
    await fs.rm(parent, { recursive: true, force: true });
  }
});
