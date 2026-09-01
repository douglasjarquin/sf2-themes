export const SITE_THEME_STORAGE_KEY = "sf2-themes.site-theme-family.v1";
export const DEFAULT_SITE_THEME_FAMILY = "main";

const SITE_TOKEN_MAP = [
  ["--color-background", "background"],
  ["--color-foreground", "foreground"],
  ["--color-panel", "panel"],
  ["--color-surface-dim", "surfaceDim"],
  ["--color-surface-0", "surface0"],
  ["--color-surface-1", "surface1"],
  ["--color-border", "border"],
  ["--color-muted", "muted"],
  ["--color-subtle", "subtle"],
  ["--color-accent", "accent"],
  ["--color-accent-secondary", "accentSecondary"],
  ["--color-red", "red"],
  ["--color-gold", "gold"],
  ["--color-cyan", "cyan"],
  ["--color-selection-background", "selectionBackground"],
  ["--color-selection-foreground", "selectionForeground"],
];

export function projectSiteTokens(tokens) {
  const { ui, semantic } = tokens;
  const siteTokens = {
    background: ui.background,
    foreground: ui.foreground,
    panel: ui.panel_bg,
    surfaceDim: ui.surface_dim,
    surface0: ui.surface0,
    surface1: ui.surface1,
    border: ui.border,
    muted: ui.muted,
    subtle: ui.subtle,
    accent: ui.accent,
    accentSecondary: ui.accent_secondary,
    red: semantic.red,
    gold: semantic.yellow,
    cyan: semantic.cyan,
    selectionBackground: ui.selection_background,
    selectionForeground: ui.selection_foreground,
  };
  return {
    ...siteTokens,
    preview: [
      siteTokens.background,
      siteTokens.foreground,
      siteTokens.red,
      siteTokens.gold,
      siteTokens.accent,
    ],
  };
}

function familyIdForVariant(variant) {
  return variant.tokens.meta.variant === "light" && variant.id.endsWith("-light")
    ? variant.id.slice(0, -6)
    : variant.id;
}

function concreteVariant(variant) {
  return {
    id: variant.id,
    name: variant.name,
    colors: variant.colors,
    tokens: variant.tokens,
    siteTokens: projectSiteTokens(variant.tokens),
  };
}

export function validateThemeFamilies(families) {
  const familyIds = new Set();
  const concreteIds = new Set();
  for (const family of families) {
    if (familyIds.has(family.id)) throw new TypeError(`site theme family ${family.id} is duplicated`);
    familyIds.add(family.id);
    if (!family.dark) throw new TypeError(`site theme family ${family.id} is missing dark variant`);
    if (!family.light) throw new TypeError(`site theme family ${family.id} is missing light variant`);
    for (const variant of [family.dark, family.light]) {
      if (concreteIds.has(variant.id)) throw new TypeError(`site theme concrete ID ${variant.id} is duplicated`);
      concreteIds.add(variant.id);
    }
  }
  if (!familyIds.has(DEFAULT_SITE_THEME_FAMILY)) throw new TypeError("site theme Main family is required");
  return families;
}

export function createThemeFamilies(paletteVariants) {
  const familyMap = new Map();
  const concreteIds = new Set();
  for (const variant of paletteVariants) {
    if (concreteIds.has(variant.id)) throw new TypeError(`site theme concrete ID ${variant.id} is duplicated`);
    concreteIds.add(variant.id);
    const id = familyIdForVariant(variant);
    const family = familyMap.get(id) ?? {
      id,
      name: variant.tokens.meta.name,
      stage: variant.tokens.meta.stage,
      introduced_in: variant.tokens.meta.introduced_in,
    };
    const key = variant.tokens.meta.variant;
    if (family[key]) throw new TypeError(`site theme family ${id} has duplicate ${key} variant`);
    family[key] = concreteVariant(variant);
    familyMap.set(id, family);
  }
  return validateThemeFamilies([...familyMap.values()]);
}

export function normalizeStoredFamily(value, families) {
  return typeof value === "string" && families.some(({ id }) => id === value)
    ? value
    : DEFAULT_SITE_THEME_FAMILY;
}

export function resolveThemeVariant(familyId, scheme, families) {
  const family = families.find(({ id }) => id === familyId)
    ?? families.find(({ id }) => id === DEFAULT_SITE_THEME_FAMILY);
  if (!family) throw new TypeError("site theme Main family is required");
  return scheme === "light" ? family.light : family.dark;
}

export function buildSiteThemeCss(families) {
  const variants = families.flatMap(({ dark, light }) => [dark, light]);
  return variants.map((variant) => {
    const declarations = siteThemeDeclarations(variant.siteTokens);
    return `[data-site-theme-id="${variant.id}"] { ${declarations}; }`;
  }).join("\n");
}

export function siteThemeDeclarations(siteTokens) {
  return SITE_TOKEN_MAP
    .map(([property, token]) => `${property}: ${siteTokens[token]}`)
    .join("; ");
}
