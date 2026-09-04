import assert from "node:assert/strict";
import { execFile } from "node:child_process";
import { cp, mkdtemp, readFile, rm, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { promisify } from "node:util";
import test from "node:test";

import {
  characterVariants,
  loadThemeData,
  mainCards,
  paletteVariants,
  themeTokens,
} from "../src/data/theme-data.mjs";

const execFileAsync = promisify(execFile);
const projectRoot = path.resolve("..");
const generatorPath = path.join(projectRoot, "mise-tasks", "generate-web-theme-data");
const sourceRoot = path.join(projectRoot, "themes");
const committedArtifactPath = path.join(
  projectRoot,
  "web",
  "src",
  "data",
  "generated-theme-data.json",
);
const CANONICAL_UI_FIELDS = [
  "background",
  "surface",
  "overlay",
  "border",
  "foreground",
  "muted",
  "subtle",
  "accent",
  "accent_secondary",
  "cursor",
  "cursor_text",
  "selection_background",
  "selection_foreground",
];

async function catalogFixture(t) {
  const root = await mkdtemp(path.join(tmpdir(), "sf2-canonical-theme-data-"));
  await cp(sourceRoot, root, { recursive: true });
  t.after(() => rm(root, { recursive: true, force: true }));
  return root;
}

async function generatedArtifact(t, source = sourceRoot) {
  const root = await mkdtemp(path.join(tmpdir(), "sf2-generated-theme-data-"));
  const output = path.join(root, "theme-data.json");
  t.after(() => rm(root, { recursive: true, force: true }));
  await execFileAsync("python3", [generatorPath, "--source", source, "--output", output]);
  return output;
}

async function assertGeneratorRejects(t, mutate, expected) {
  const fixtureRoot = await catalogFixture(t);
  const mainPath = path.join(fixtureRoot, "main.toml");
  const source = await readFile(mainPath, "utf8");
  await writeFile(mainPath, mutate(source));

  await assert.rejects(
    execFileAsync("python3", [
      generatorPath,
      "--source",
      fixtureRoot,
      "--output",
      path.join(fixtureRoot, "generated.json"),
    ]),
    (error) => {
      assert.match(error.stderr, expected);
      return true;
    },
  );
}

test("exports current cards, featured characters, and all designer fields", () => {
  // Given: browser exports loaded from the canonical generated artifact.
  const main = themeTokens.main;

  // When/Then: current public selections and all 36 designer colors are preserved.
  assert.deepEqual(
    mainCards.map(({ key, token, hex }) => [key, token, hex]),
    [
      ["deep_navy", "ui.background", "#131927"],
      ["arcade_red", "semantic.red", "#c86e67"],
      ["gold", "semantic.yellow", "#a58324"],
      ["accent", "ui.accent", "#ad8705"],
      ["cream", "ui.foreground", "#cad1de"],
    ],
  );
  assert.deepEqual(characterVariants.map(({ id }) => id), ["ryu", "ken", "chun-li", "guile"]);
  assert.deepEqual(Object.keys(main.ui).slice(0, CANONICAL_UI_FIELDS.length), CANONICAL_UI_FIELDS);
  for (const field of CANONICAL_UI_FIELDS) {
    assert.match(main.ui[field], /^#[0-9a-f]{6}$/);
  }
  assert.equal(
    CANONICAL_UI_FIELDS.length
      + Object.keys(main.semantic).length
      + Object.keys(main.ansi.normal).length
      + Object.keys(main.ansi.bright).length,
    36,
  );
  assert.equal(main.ui.panel_bg, main.ui.surface);
  assert.equal(main.ui.active_row_bg, main.ui.selection_background);
  assert.equal(main.ui.overlay1, main.ui.muted);
  assert.equal(main.ui.subtext, main.ui.subtle);
});

test("generated browser artifact is fresh and equals canonical source output", async (t) => {
  // Given: a fresh generation from the repository canonical catalog.
  const freshArtifactPath = await generatedArtifact(t);

  // When: fresh and committed representations are read.
  const [freshSource, committedSource] = await Promise.all([
    readFile(freshArtifactPath, "utf8"),
    readFile(committedArtifactPath, "utf8"),
  ]);
  const fresh = JSON.parse(freshSource);

  // Then: bytes, order, and browser-consumed tokens are identical.
  assert.equal(freshSource, committedSource);
  assert.equal(fresh.themes.length, 36);
  assert.deepEqual(paletteVariants.map(({ tokens }) => tokens), fresh.themes);
});

test("loads a complete canonical generated fixture", async (t) => {
  // Given: a generated artifact produced from a complete source fixture.
  const fixtureRoot = await catalogFixture(t);
  const artifactPath = await generatedArtifact(t, fixtureRoot);

  // When: the browser adapter selects its public views.
  const artifact = JSON.parse(await readFile(artifactPath, "utf8"));
  const data = loadThemeData({ themes: artifact.themes });

  // Then: the full catalog and featured character order are retained.
  assert.equal(data.paletteVariants.length, 36);
  assert.deepEqual(data.characterVariants.map(({ id }) => id), ["ryu", "ken", "chun-li", "guile"]);
});

test("rejects missing schema_version before browser generation", async (t) => {
  // Given/When/Then: the canonical generator rejects the malformed fixture.
  await assertGeneratorRejects(
    t,
    (source) => source.replace("schema_version = 1\n\n", ""),
    /main\.toml\.schema_version must be an integer/,
  );
});

test("rejects an unknown top-level key before browser generation", async (t) => {
  // Given/When/Then: the canonical generator rejects the malformed fixture.
  await assertGeneratorRejects(
    t,
    (source) => source.replace(
      "schema_version = 1\n",
      'schema_version = 1\nunexpected = "browser drift"\n',
    ),
    /main\.toml: unknown top-level keys: unexpected/,
  );
});

test("rejects malformed TOML before browser generation", async (t) => {
  // Given/When/Then: malformed source cannot produce browser data.
  await assertGeneratorRejects(
    t,
    () => "[ui\nbackground = '#101a3a'",
    /Expected '\]' at the end of a table declaration/,
  );
});

test("rejects a missing canonical field before browser generation", async (t) => {
  // Given/When/Then: canonical source validation names the missing field.
  await assertGeneratorRejects(
    t,
    (source) => source.replace('red = "#c86e67"\n', ""),
    /missing .*main\.toml\.semantic\.red/,
  );
});

test("rejects an invalid canonical color before browser generation", async (t) => {
  // Given/When/Then: canonical source validation rejects the invalid color.
  await assertGeneratorRejects(
    t,
    (source) => source.replace('accent = "#ad8705"', 'accent = "gold"'),
    /color must be #RRGGBB, got 'gold'/,
  );
});
