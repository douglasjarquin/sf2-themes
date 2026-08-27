import assert from "node:assert/strict";
import { watch } from "node:fs";
import { promises as fs } from "node:fs";
import os from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";
import test from "node:test";

import { captureArchive, parseCaptureArguments } from "../../scripts/capture-game-screenshots.mjs";

test("parseCaptureArguments rejects an existing non-archive output with a sentinel", async () => {
  const fixture = await fs.mkdtemp(path.join(os.tmpdir(), "sf2-sec1-output-boundary-"));
  const sentinel = path.join(fixture, "sentinel.txt");
  await fs.writeFile(sentinel, "preserve-me\n", "utf8");
  try {
    assert.throws(
      () => parseCaptureArguments(["--theme", "ryu", "--output", fixture]),
      /existing custom output paths are not supported/,
    );
    assert.equal(await fs.readFile(sentinel, "utf8"), "preserve-me\n");
  } finally {
    await fs.rm(fixture, { recursive: true, force: true });
  }
});

test("parseCaptureArguments rejects a forged archive with a non-PNG image and preserves its sentinel", async () => {
  const fixture = await fs.mkdtemp(path.join(os.tmpdir(), "sf2-sec1-v2-forged-"));
  const gameDirectory = path.join(fixture, "game");
  await fs.mkdir(gameDirectory);
  await fs.writeFile(path.join(fixture, "sentinel.txt"), "do-not-delete\n", "utf8");
  await fs.writeFile(path.join(gameDirectory, "main.png"), "not-a-png\n", "utf8");
  await fs.writeFile(path.join(fixture, "manifest.json"), JSON.stringify({
    schemaVersion: 1,
    recipeVersion: "sf2-game-screenshots-v1",
    entries: [{ image: "game/main.png" }],
  }), "utf8");
  try {
    assert.throws(
      () => parseCaptureArguments(["--theme", "ryu", "--output", fixture]),
      /existing custom output paths are not supported/,
    );
    assert.equal(await fs.readFile(path.join(fixture, "sentinel.txt"), "utf8"), "do-not-delete\n");
  } finally {
    await fs.rm(fixture, { recursive: true, force: true });
  }
});

test("captureArchive rejects an existing custom output before promotion and preserves its sentinel", async () => {
  const fixture = await fs.mkdtemp(path.join(os.tmpdir(), "sf2-sec1-v3-direct-"));
  const sentinel = path.join(fixture, "sentinel.txt");
  await fs.writeFile(sentinel, "direct-preserve\n", "utf8");
  const previousPromotionFailure = process.env.SF2_CAPTURE_FAIL_PROMOTION;
  process.env.SF2_CAPTURE_FAIL_PROMOTION = "1";
  try {
    await assert.rejects(
      () => captureArchive({ theme: "ryu", seed: "boundary-v3-red", outputDirectory: fixture }),
      /existing custom output paths are not supported/,
    );
    assert.equal(await fs.readFile(sentinel, "utf8"), "direct-preserve\n");
  } finally {
    if (previousPromotionFailure === undefined) delete process.env.SF2_CAPTURE_FAIL_PROMOTION;
    else process.env.SF2_CAPTURE_FAIL_PROMOTION = previousPromotionFailure;
    await fs.rm(fixture, { recursive: true, force: true });
  }
});

test("captureArchive refuses a custom target created after preflight without deleting its sentinel", async () => {
  const parent = await fs.mkdtemp(path.join(os.tmpdir(), "sf2-sec1-v4-race-"));
  const output = path.join(parent, "archive");
  const targetCreated = new Promise((resolve, reject) => {
    const watcher = watch(parent, { persistent: false }, async (_event, filename) => {
      if (!String(filename).startsWith(".archive.stage-")) return;
      watcher.close();
      try {
        await fs.mkdir(output);
        await fs.writeFile(path.join(output, "sentinel.txt"), "race-preserve\n", "utf8");
        resolve();
      } catch (error) {
        reject(error);
      }
    });
  });
  try {
    const capture = captureArchive({ theme: "ryu", seed: "boundary-v4-race", outputDirectory: output });
    await targetCreated;
    await assert.rejects(capture, /EEXIST|ENOTEMPTY|not empty|already exists/);
    assert.equal(await fs.readFile(path.join(output, "sentinel.txt"), "utf8"), "race-preserve\n");
  } finally {
    await fs.rm(parent, { recursive: true, force: true });
  }
});

test("captureArchive refuses an empty custom target created after preflight", async () => {
  const parent = await fs.mkdtemp(path.join(os.tmpdir(), "sf2-sec1-v5-empty-"));
  const output = path.join(parent, "archive");
  const targetCreated = new Promise((resolve, reject) => {
    const watcher = watch(parent, { persistent: false }, async (_event, filename) => {
      if (!String(filename).startsWith(".archive.stage-")) return;
      watcher.close();
      try {
        await fs.mkdir(output);
        resolve();
      } catch (error) {
        reject(error);
      }
    });
  });
  try {
    const capture = captureArchive({ theme: "ryu", seed: "boundary-v5-empty", outputDirectory: output });
    await targetCreated;
    await assert.rejects(capture, /EEXIST|ENOTEMPTY|not empty|already exists/);
    assert.deepEqual(await fs.readdir(output), []);
  } finally {
    await fs.rm(parent, { recursive: true, force: true });
  }
});

test("parseCaptureArguments rejects an existing ancestor but accepts canonical and fresh outputs", async () => {
  const freshParent = await fs.mkdtemp(path.join(os.tmpdir(), "sf2-sec1-output-boundary-"));
  const freshOutput = path.join(freshParent, "archive");
  const canonicalArchive = fileURLToPath(new URL("../public/screenshots", import.meta.url));
  try {
    for (const broadPath of [os.homedir(), process.cwd(), path.dirname(process.cwd())]) {
      assert.throws(() => parseCaptureArguments(["--output", broadPath]), /existing custom output paths are not supported/);
    }
    assert.equal(parseCaptureArguments(["--output", freshOutput]).outputDirectory, freshOutput);
    assert.equal(parseCaptureArguments(["--output", canonicalArchive]).outputDirectory, canonicalArchive);
  } finally {
    await fs.rm(freshParent, { recursive: true, force: true });
  }
});
