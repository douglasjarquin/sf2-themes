import assert from "node:assert/strict";
import { mkdtemp, mkdir, readFile, readdir, rm, unlink, writeFile } from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import test from "node:test";

import { parse } from "smol-toml";

import {
  characterVariants,
  loadThemeData,
  mainCards,
  paletteVariants,
  themeTokens,
} from "../src/data/theme-data.mjs";

const UI_FIELDS = [
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
const WEB_ROLE_SOURCES = {
  cursor_bg: "cursor",
  cursor_fg: "cursor_text",
  selection_bg: "selection_background",
  selection_fg: "selection_foreground",
  panel_bg: "surface",
  sidebar_bg: "background",
  active_row_bg: "selection_background",
  navigate_row_bg: "overlay",
  surface_dim: "background",
  surface0: "surface",
  surface1: "overlay",
  overlay0: "border",
  overlay1: "muted",
  subtext: "subtle",
};
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
name = "Theme ${id}"
variant = "dark"
family = "sf2"
stage = "Fixture stage"

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

test("exports canonical designer tokens and deterministic web roles", () => {
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
  assert.deepEqual(
    characterVariants.map(({ id }) => id),
    ["ryu", "ken", "chun-li", "guile"],
  );
  assert.deepEqual(characterVariants[0].colors, {
    background: "#141a23",
    red: "#c86e6c",
    yellow: "#9e8625",
    accent: "#da6a6a",
    foreground: "#cad2df",
  });
  assert.deepEqual(Object.keys(themeTokens.main.ui), [
    ...UI_FIELDS,
    ...Object.keys(WEB_ROLE_SOURCES),
  ]);
  assert.equal(themeTokens.main.meta.name, "Main");
  assert.equal(themeTokens.main.meta.variant, "dark");
  assert.equal(themeTokens.main.meta.family, "sf2");
  assert.equal(themeTokens.main.meta.stage, "Arcade · Family baseline");
  for (const [role, source] of Object.entries(WEB_ROLE_SOURCES)) {
    assert.equal(themeTokens.main.ui[role], themeTokens.main.ui[source]);
  }
  assert.deepEqual(Object.keys(themeTokens.main.semantic), SEMANTIC_FIELDS);
  assert.deepEqual(Object.keys(themeTokens.main.ansi.normal), ANSI_FIELDS);
  assert.deepEqual(Object.keys(themeTokens.main.ansi.bright), ANSI_FIELDS);
});

test("preserves canonical source values and stable catalog order for all 36 browser themes", async () => {
  const themeRoot = path.resolve("../themes");
  const characterNames = (await readdir(path.join(themeRoot, "characters")))
    .filter((fileName) => fileName.endsWith(".toml"))
    .sort();
  const sourcePaths = [
    path.join(themeRoot, "main.toml"),
    path.join(themeRoot, "main-light.toml"),
    ...characterNames.map((fileName) => path.join(themeRoot, "characters", fileName)),
  ];
  const sources = await Promise.all(
    sourcePaths.map(async (filePath) => parse(await readFile(filePath, "utf8"))),
  );

  assert.equal(paletteVariants.length, 36);
  assert.deepEqual(
    paletteVariants.map(({ id }) => id),
    sources.map(({ meta }) => meta.id),
  );
  for (const [index, source] of sources.entries()) {
    const browser = paletteVariants[index].tokens;
    assert.deepEqual(browser.meta, source.meta);
    assert.deepEqual(
      Object.fromEntries(UI_FIELDS.map((field) => [field, browser.ui[field]])),
      source.ui,
    );
    assert.deepEqual(browser.semantic, source.semantic);
    assert.deepEqual(browser.ansi, source.ansi);
  }
});

test("loads a complete temporary theme set", async (t) => {
  const paths = await fixture(t);
  const data = loadThemeData(paths);

  assert.equal(data.mainCards[0].hex, "#000001");
  assert.equal(data.mainCards[1].hex, "#000015");
  assert.equal(data.mainCards[2].hex, "#000017");
  assert.equal(data.mainCards[3].hex, "#000008");
  assert.equal(data.mainCards[4].hex, "#000005");
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

test("rejects a theme without the canonical schema version", async (t) => {
  const paths = await fixture(t);
  const malformed = themeToml("main", { kind: "main" }).replace(
    "schema_version = 1\n\n",
    "",
  );
  await writeFile(paths.mainPath, malformed);

  assert.throws(() => loadThemeData(paths), /main\.toml\.schema_version: required integer/);
});

test("reports an invalid hex color with its path", async (t) => {
  const paths = await fixture(t);
  await writeFile(paths.mainPath, themeToml("main", { kind: "main", invalidHex: true }));

  assert.throws(
    () => loadThemeData(paths),
    /main\.toml\.ui\.accent: expected #RRGGBB, got "gold"/,
  );
});
