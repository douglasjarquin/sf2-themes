import assert from "node:assert/strict";
import { cp, mkdtemp, mkdir, readFile, rm, symlink, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { execFile } from "node:child_process";
import { fileURLToPath } from "node:url";
import { promisify } from "node:util";
import test from "node:test";

import { parse } from "smol-toml";

const execFileAsync = promisify(execFile);
const webRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const projectRoot = path.resolve(webRoot, "..");

test("renders the ui.accent terminal row from divergent fixture data", async (t) => {
  const root = await mkdtemp(path.join(tmpdir(), "sf2-theme-palette-"));
  const fixtureWebRoot = path.join(root, "web");
  const fixtureThemesRoot = path.join(root, "themes");
  const mainSource = await readFile(path.join(projectRoot, "themes/main.toml"), "utf8");
  const sourceTokens = parse(mainSource);
  const fixtureSource = mainSource.replace(
    `accent = "${sourceTokens.ui.accent}"`,
    `accent = "${sourceTokens.semantic.blue}"`,
  );
  const fixtureTokens = parse(fixtureSource);

  t.after(() => rm(root, { recursive: true, force: true }));
  assert.notEqual(fixtureTokens.ui.accent, fixtureTokens.semantic.yellow);

  await mkdir(fixtureWebRoot, { recursive: true });
  await Promise.all([
    cp(path.join(webRoot, "src"), path.join(fixtureWebRoot, "src"), { recursive: true }),
    cp(path.join(projectRoot, "themes/characters"), path.join(fixtureThemesRoot, "characters"), {
      recursive: true,
    }),
    cp(path.join(webRoot, "astro.config.mjs"), path.join(fixtureWebRoot, "astro.config.mjs")),
    cp(path.join(webRoot, "package.json"), path.join(fixtureWebRoot, "package.json")),
    symlink(path.join(webRoot, "node_modules"), path.join(fixtureWebRoot, "node_modules")),
  ]);
  await writeFile(path.join(fixtureThemesRoot, "main.toml"), fixtureSource);

  await execFileAsync(process.execPath, [path.join(webRoot, "node_modules/astro/bin/astro.mjs"), "build"], {
    cwd: fixtureWebRoot,
  });

  const html = await readFile(path.join(fixtureWebRoot, "dist/palette/index.html"), "utf8");
  const row = html.match(/<li[^>]*data-token="ui\.accent"[^>]*>[\s\S]*?<\/li>/)?.[0];

  assert.ok(row, "renders a ui.accent terminal row");
  assert.match(row, new RegExp(`data-color="${fixtureTokens.ui.accent}"`));
});
