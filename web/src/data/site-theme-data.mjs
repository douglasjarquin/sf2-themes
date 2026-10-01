import generatedThemeData from "./generated-theme-data.json" with { type: "json" };

function loadThemeFamilies(themes = generatedThemeData.themes) {
  const families = new Map();
  for (const theme of themes) {
    const familyId = theme.meta.id.replace(/-light$/, "");
    const family = families.get(familyId) ?? {
      id: familyId,
      name: theme.meta.name,
      world: theme.meta.stage,
      dark: null,
      light: null,
    };
    family[theme.meta.variant] = theme;
    families.set(familyId, family);
  }
  return [...families.values()].filter(({ dark, light }) => dark && light);
}

export const themeFamilies = loadThemeFamilies();

function toSiteVariant(theme) {
  return {
    id: theme.meta.id,
    fileId: `sf2-${theme.meta.id}`,
    bg: theme.ui.background,
    surface: theme.ui.surface,
    border: theme.ui.border,
    fg: theme.ui.foreground,
    muted: theme.ui.muted,
    subtle: theme.ui.subtle,
    accent: theme.ui.accent,
    accent2: theme.ui.accent_secondary,
    sel: theme.ui.selection_background,
    cursor: theme.ui.cursor,
    orange: theme.semantic.orange,
    n: Object.values(theme.ansi.normal),
    b: Object.values(theme.ansi.bright),
  };
}

export const siteThemeFamilies = themeFamilies.map(({ id, name, world, dark, light }) => ({
  id,
  name,
  stage: world,
  aliases: dark.meta.aliases ?? [],
  dark: toSiteVariant(dark),
  light: toSiteVariant(light),
}));
