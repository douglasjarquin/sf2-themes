import { existsSync, readdirSync, readFileSync } from "node:fs";
import path from "node:path";

import { parse } from "smol-toml";

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
const META_FIELDS = [
  "id",
  "display_name",
  "kind",
  "introduced_in",
  "aliases",
  "name",
  "variant",
  "family",
  "stage",
];
const HEX_COLOR = /^#[0-9a-fA-F]{6}$/;

const DEFAULT_THEME_DIRECTORY = path.resolve(process.cwd(), "../themes");
const DEFAULT_MAIN_PATH = path.join(DEFAULT_THEME_DIRECTORY, "main.toml");
const DEFAULT_CHARACTER_PATHS = ["ryu", "ken", "chun-li", "guile"].map(
  (id) => path.join(DEFAULT_THEME_DIRECTORY, "characters", `${id}.toml`),
);

function discoverCharacterPaths(directory) {
  return readdirSync(directory)
    .filter((fileName) => fileName.endsWith(".toml"))
    .sort()
    .map((fileName) => path.join(directory, fileName));
}

function requiredTable(value, source) {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new Error(`${source}: required table`);
  }
  return value;
}

function requiredString(value, source) {
  if (typeof value !== "string" || value.length === 0) {
    throw new Error(`${source}: required string`);
  }
  return value;
}

function requiredSchemaVersion(value, source) {
  if (!Number.isInteger(value)) {
    throw new Error(`${source}: required integer`);
  }
  if (value !== 1) {
    throw new Error(`${source}: unsupported schema version ${value}`);
  }
  return value;
}

function validateColorTable(table, fields, source) {
  const colors = requiredTable(table, source);
  for (const field of fields) {
    const fieldSource = `${source}.${field}`;
    if (!(field in colors)) {
      throw new Error(`${fieldSource}: required`);
    }
    if (typeof colors[field] !== "string" || !HEX_COLOR.test(colors[field])) {
      throw new Error(
        `${fieldSource}: expected #RRGGBB, got ${JSON.stringify(colors[field])}`,
      );
    }
  }
  return colors;
}

function validateTheme(raw, source) {
  const theme = requiredTable(raw, source);
  requiredSchemaVersion(theme.schema_version, `${source}.schema_version`);
  const meta = requiredTable(theme.meta, `${source}.meta`);
  for (const field of META_FIELDS) {
    if (!(field in meta)) {
      throw new Error(`${source}.meta.${field}: required`);
    }
  }
  for (const field of META_FIELDS.filter((field) => field !== "aliases")) {
    requiredString(meta[field], `${source}.meta.${field}`);
  }
  if (!Array.isArray(meta.aliases) || meta.aliases.some((alias) => typeof alias !== "string")) {
    throw new Error(`${source}.meta.aliases: required string array`);
  }
  if (meta.kind === "character") {
    requiredString(meta.character, `${source}.meta.character`);
  }

  validateColorTable(theme.ui, UI_FIELDS, `${source}.ui`);
  validateColorTable(theme.semantic, SEMANTIC_FIELDS, `${source}.semantic`);
  const ansi = requiredTable(theme.ansi, `${source}.ansi`);
  validateColorTable(ansi.normal, ANSI_FIELDS, `${source}.ansi.normal`);
  validateColorTable(ansi.bright, ANSI_FIELDS, `${source}.ansi.bright`);
  return theme;
}

function projectWebRoles(ui) {
  return Object.fromEntries(
    Object.entries(WEB_ROLE_SOURCES).map(([role, source]) => [role, ui[source]]),
  );
}

function readTheme(filePath) {
  let source;
  try {
    source = readFileSync(filePath, "utf8");
  } catch (error) {
    throw new Error(`${filePath}: unable to read theme file`, { cause: error });
  }

  let parsed;
  try {
    parsed = parse(source);
  } catch (error) {
    throw new Error(`${filePath}: invalid TOML`, { cause: error });
  }
  const theme = validateTheme(parsed, filePath);
  return {
    ...theme,
    ui: {
      ...theme.ui,
      ...projectWebRoles(theme.ui),
    },
  };
}

function mainCard(name, key, token, hex) {
  return { name, key, token, hex };
}

function paletteVariant(tokens) {
  return {
    id: tokens.meta.id,
    name: tokens.meta.display_name,
    character: tokens.meta.character ?? "Main",
    colors: {
      background: tokens.ui.background,
      red: tokens.semantic.red,
      yellow: tokens.semantic.yellow,
      accent: tokens.ui.accent,
      foreground: tokens.ui.foreground,
    },
    tokens,
  };
}

export function loadThemeData({
  mainPath = DEFAULT_MAIN_PATH,
  characterPaths = DEFAULT_CHARACTER_PATHS,
  catalogCharacterPaths = discoverCharacterPaths(path.join(path.dirname(mainPath), "characters")),
} = {}) {
  const main = readTheme(mainPath);
  const mainLightPath = path.join(path.dirname(mainPath), "main-light.toml");
  const mainLight = existsSync(mainLightPath)
    ? readTheme(mainLightPath)
    : null;
  const characters = characterPaths.map(readTheme);
  const catalogCharacters = catalogCharacterPaths.map(readTheme);

  const cards = [
    mainCard("Deep Navy", "deep_navy", "ui.background", main.ui.background),
    mainCard("Arcade Red", "arcade_red", "semantic.red", main.semantic.red),
    mainCard("Gold", "gold", "semantic.yellow", main.semantic.yellow),
    mainCard("Accent", "accent", "ui.accent", main.ui.accent),
    mainCard("Cream", "cream", "ui.foreground", main.ui.foreground),
  ];
  const variants = characters.map(paletteVariant);
  const palettes = [main, ...(mainLight ? [mainLight] : []), ...catalogCharacters].map(paletteVariant);

  return {
    mainCards: cards,
    characterVariants: variants,
    paletteVariants: palettes,
    themeTokens: {
      main,
      characters: Object.fromEntries(
        characters.map((tokens) => [tokens.meta.id, tokens]),
      ),
    },
  };
}

const themeData = loadThemeData();

export const mainCards = themeData.mainCards;
export const characterVariants = themeData.characterVariants;
export const paletteVariants = themeData.paletteVariants;
export const themeTokens = themeData.themeTokens;
