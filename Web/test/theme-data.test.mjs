import assert from "node:assert/strict";
import { mkdtemp, mkdir, readFile, rm, unlink, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import test from "node:test";

import { parse } from "smol-toml";

import {
  characterVariants,
  loadThemeData,
  mainCards,
  themeTokens,
} from "../src/data/theme-data.mjs";

const UI_FIELDS = [
  "background",
  "foreground",
  "cursor_bg",
  "cursor_fg",
  "selection_bg",
  "selection_fg",
  "panel_bg",
  "sidebar_bg",
  "active_row_bg",
  "navigate_row_bg",
  "surface_dim",
  "surface0",
  "surface1",
  "overlay0",
  "overlay1",
  "subtext",
  "accent",
];
const SEMANTIC_FIELDS = [
  "red",
  "green",
  "yellow",
  "blue",
  "magenta",
  "cyan",
  "orange",
];
const ANSI_FIELDS = [
  "black",
  "red",
  "green",
  "yellow",
  "blue",
  "magenta",
  "cyan",
  "white",
];

function themeToml(id, { kind = "character", invalidHex = false, uiValues = {} } = {}) {
  const character = kind === "character" ? `character = "${id}"\n` : "";
  const ui = UI_FIELDS.map((field, index) => {
    const color = invalidHex && field === "accent"
      ? "gold"
      : (uiValues[field] ?? `#${(index + 1).toString(16).padStart(6, "0")}`);
    return `${field} = "${color}"`;
  }).join("\n");
  const semantic = SEMANTIC_FIELDS.map(
    (field, index) => `${field} = "#${(index + 21).toString(16).padStart(6, "0")}"`,
  ).join("\n");
  const ansi = ANSI_FIELDS.map(
    (field, index) => `${field} = "#${(index + 41).toString(16).padStart(6, "0")}"`,
  ).join("\n");

  return `schema_version = 1

[meta]
id = "${id}"
display_name = "Theme ${id}"
kind = "${kind}"
introduced_in = "${kind === "main" ? "main" : "world-warrior"}"
${character}aliases = []

[ui]
${ui}

[semantic]
${semantic}

[ansi.normal]
${ansi}

[ansi.bright]
${ansi}
`;
}

async function fixture(t, { mainUiValues } = {}) {
  const root = await mkdtemp(path.join(tmpdir(), "sf2-theme-data-"));
  const characterDirectory = path.join(root, "characters");
  await mkdir(characterDirectory);
  const mainPath = path.join(root, "main.toml");
  const characterPaths = ["ryu", "ken", "chun-li", "guile"].map((id) =>
    path.join(characterDirectory, `${id}.toml`),
  );
  await writeFile(mainPath, themeToml("main", { kind: "main", uiValues: mainUiValues }));
  await Promise.all(
    characterPaths.map((filePath) =>
      writeFile(filePath, themeToml(path.basename(filePath, ".toml"))),
    ),
  );
  t.after(() => rm(root, { recursive: true, force: true }));
  return { characterPaths, mainPath };
}

test("exports the committed main cards, character variants, and complete tokens", () => {
  assert.deepEqual(
    mainCards.map(({ key, token, hex }) => [key, token, hex]),
    [
      ["deep_navy", "ui.background", "#101a3a"],
      ["arcade_red", "semantic.red", "#e8565f"],
      ["gold", "semantic.yellow", "#f2b134"],
      ["accent", "ui.accent", "#f2b134"],
      ["cream", "ui.foreground", "#fff4d6"],
    ],
  );
  assert.deepEqual(
    characterVariants.map(({ id }) => id),
    ["ryu", "ken", "chun-li", "guile"],
  );
  assert.deepEqual(characterVariants[0].colors, {
    background: "#101522",
    red: "#e24c52",
    yellow: "#f2b134",
    accent: "#d83a3a",
    foreground: "#fff4d6",
  });
  assert.deepEqual(Object.keys(themeTokens.main.ui), UI_FIELDS);
  assert.deepEqual(Object.keys(themeTokens.main.semantic), SEMANTIC_FIELDS);
  assert.deepEqual(Object.keys(themeTokens.main.ansi.normal), ANSI_FIELDS);
  assert.deepEqual(Object.keys(themeTokens.main.ansi.bright), ANSI_FIELDS);
});

test("loads a complete temporary theme set", async (t) => {
  const paths = await fixture(t);
  const data = loadThemeData(paths);

  assert.equal(data.mainCards[0].hex, "#000001");
  assert.equal(data.mainCards[1].hex, "#000015");
  assert.equal(data.mainCards[2].hex, "#000017");
  assert.equal(data.mainCards[3].hex, "#000011");
  assert.equal(data.mainCards[4].hex, "#000002");
  assert.deepEqual(data.characterVariants.map(({ id }) => id), [
    "ryu",
    "ken",
    "chun-li",
    "guile",
  ]);
});

test("sources the fourth main card from fixture ui.accent, not semantic.cyan", async (t) => {
  const paths = await fixture(t, { mainUiValues: { accent: "#123456" } });
  const fixtureMain = parse(await readFile(paths.mainPath, "utf8"));
  const data = loadThemeData(paths);

  assert.notEqual(fixtureMain.ui.accent, fixtureMain.semantic.cyan);
  assert.equal(data.mainCards[3].token, "ui.accent");
  assert.equal(data.mainCards[3].hex, fixtureMain.ui.accent);
});

test("reports a missing source file with its path", async (t) => {
  const paths = await fixture(t);
  await unlink(paths.mainPath);

  assert.throws(() => loadThemeData(paths), /main\.toml: unable to read theme file/);
});

test("reports malformed TOML with its path", async (t) => {
  const paths = await fixture(t);
  await writeFile(paths.mainPath, "[ui\nbackground = '#101a3a'");

  assert.throws(() => loadThemeData(paths), /main\.toml: invalid TOML/);
});

test("reports a missing required field with its path", async (t) => {
  const paths = await fixture(t);
  const malformed = themeToml("main", { kind: "main" }).replace(
    'red = "#000015"\n',
    "",
  );
  await writeFile(paths.mainPath, malformed);

  assert.throws(() => loadThemeData(paths), /main\.toml\.semantic\.red: required/);
});

test("reports an invalid hex color with its path", async (t) => {
  const paths = await fixture(t);
  await writeFile(paths.mainPath, themeToml("main", { kind: "main", invalidHex: true }));

  assert.throws(
    () => loadThemeData(paths),
    /main\.toml\.ui\.accent: expected #RRGGBB, got "gold"/,
  );
});
