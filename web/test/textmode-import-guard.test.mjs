import assert from "node:assert/strict";
import { readFileSync, readdirSync, statSync } from "node:fs";
import path from "node:path";
import test from "node:test";

function sourceFiles(directory) {
  return readdirSync(directory).flatMap((entry) => {
    const filePath = path.join(directory, entry);
    return statSync(filePath).isDirectory() ? sourceFiles(filePath) : [filePath];
  }).filter((filePath) => /\.(?:ts|tsx|astro|mjs)$/.test(filePath));
}

test("only TextmodeRenderer imports textmode.js", () => {
  const importers = sourceFiles("src").filter((filePath) => /from\s+["']textmode(?:\.js)?["']/.test(readFileSync(filePath, "utf8")));

  assert.deepEqual(importers, [path.join("src", "game", "render", "TextmodeRenderer.ts")]);
});
