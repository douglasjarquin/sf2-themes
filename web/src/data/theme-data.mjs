import generatedThemeData from "./generated-theme-data.json" with { type: "json" };
import { createThemeFamilies } from "./site-theme.mjs";

const DEFAULT_CHARACTER_IDS = ["ryu", "ken", "chun-li", "guile"];

export function loadThemeData({
  themes = generatedThemeData.themes,
  characterIds = DEFAULT_CHARACTER_IDS,
} = {}) {
  const main = themes.find(({ meta }) => meta.id === "main");
  if (main === undefined) {
    throw new Error("generated theme main is required");
  }
  const characters = characterIds.map((id) => {
    const theme = themes.find(({ meta }) => meta.id === id);
    if (theme === undefined) {
      throw new Error(`generated theme ${id} is required`);
    }
    return theme;
  });
  const toPaletteVariant = (tokens) => ({
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
  });

  const paletteVariants = themes.map(toPaletteVariant);

  return {
    mainCards: [
      { name: "Deep Navy", key: "deep_navy", token: "ui.background", hex: main.ui.background },
      { name: "Arcade Red", key: "arcade_red", token: "semantic.red", hex: main.semantic.red },
      { name: "Gold", key: "gold", token: "semantic.yellow", hex: main.semantic.yellow },
      { name: "Accent", key: "accent", token: "ui.accent", hex: main.ui.accent },
      { name: "Cream", key: "cream", token: "ui.foreground", hex: main.ui.foreground },
    ],
    characterVariants: characters.map(toPaletteVariant),
    paletteVariants,
    themeFamilies: createThemeFamilies(paletteVariants),
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
export const themeFamilies = themeData.themeFamilies;
export const themeTokens = themeData.themeTokens;
