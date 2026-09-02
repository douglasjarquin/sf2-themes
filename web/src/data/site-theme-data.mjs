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
