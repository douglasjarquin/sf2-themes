import assert from "node:assert/strict";
import test from "node:test";

import { loadThemeData } from "../src/data/theme-data.mjs";
import {
  buildSiteThemeCss,
  createThemeFamilies,
  normalizeStoredFamily,
  resolveThemeVariant,
  validateThemeFamilies,
} from "../src/data/site-theme.mjs";

const { themeFamilies } = loadThemeData();

test("derives the complete site theme family model from all canonical variants", () => {
  const { themeFamilies } = loadThemeData();

  assert.equal(themeFamilies.length, 18);
  assert.equal(new Set(themeFamilies.map(({ id }) => id)).size, 18);
  assert.equal(themeFamilies[0].id, "main");
  assert.equal(themeFamilies[0].dark.id, "main");
  assert.equal(themeFamilies[0].light.id, "main-light");
  assert.ok(themeFamilies.every(({ dark, light }) => dark && light));
  assert.equal(new Set(themeFamilies.flatMap(({ dark, light }) => [dark.id, light.id])).size, 36);
});

test("projects canonical UI and semantic tokens for the site root and preview strip", () => {
  const ken = themeFamilies.find(({ id }) => id === "ken");

  assert.deepEqual(ken.dark.siteTokens, {
    background: ken.dark.tokens.ui.background,
    foreground: ken.dark.tokens.ui.foreground,
    panel: ken.dark.tokens.ui.panel_bg,
    surfaceDim: ken.dark.tokens.ui.surface_dim,
    surface0: ken.dark.tokens.ui.surface0,
    surface1: ken.dark.tokens.ui.surface1,
    border: ken.dark.tokens.ui.border,
    muted: ken.dark.tokens.ui.muted,
    subtle: ken.dark.tokens.ui.subtle,
    accent: ken.dark.tokens.ui.accent,
    accentSecondary: ken.dark.tokens.ui.accent_secondary,
    red: ken.dark.tokens.semantic.red,
    gold: ken.dark.tokens.semantic.yellow,
    cyan: ken.dark.tokens.semantic.cyan,
    selectionBackground: ken.dark.tokens.ui.selection_background,
    selectionForeground: ken.dark.tokens.ui.selection_foreground,
    preview: [
      ken.dark.tokens.ui.background,
      ken.dark.tokens.ui.foreground,
      ken.dark.tokens.semantic.red,
      ken.dark.tokens.semantic.yellow,
      ken.dark.tokens.ui.accent,
    ],
  });
});

test("normalizes stored family IDs and resolves the system scheme", () => {
  const ken = themeFamilies.find(({ id }) => id === "ken");

  assert.equal(normalizeStoredFamily("ken", themeFamilies), "ken");
  assert.equal(normalizeStoredFamily("ken-light", themeFamilies), "main");
  assert.equal(normalizeStoredFamily("missing", themeFamilies), "main");
  assert.equal(resolveThemeVariant("ken", "dark", themeFamilies), ken.dark);
  assert.equal(resolveThemeVariant("ken", "light", themeFamilies), ken.light);
  assert.equal(resolveThemeVariant("missing", "dark", themeFamilies), themeFamilies[0].dark);
});

test("builds a compact CSS matrix without ANSI data", () => {
  const css = buildSiteThemeCss(themeFamilies);

  assert.match(css, /\[data-site-theme-id="ken-light"\]/);
  assert.match(css, /--color-selection-foreground: #[0-9a-f]{6}/);
  assert.doesNotMatch(css, /ansi|cursor/);
});

test("rejects duplicate, incomplete, and missing-main family contracts", () => {
  assert.throws(
    () => validateThemeFamilies([...themeFamilies, themeFamilies[0]]),
    /site theme family main is duplicated/,
  );
  assert.throws(
    () => validateThemeFamilies(themeFamilies.map((family) => family.id === "ken" ? { ...family, light: undefined } : family)),
    /site theme family ken is missing light variant/,
  );
  assert.throws(
    () => validateThemeFamilies(themeFamilies.filter(({ id }) => id !== "main")),
    /site theme Main family is required/,
  );
  assert.throws(
    () => validateThemeFamilies(themeFamilies.map((family) => family.id === "ken"
      ? { ...family, light: { ...family.light, id: family.dark.id } }
      : family)),
    /site theme concrete ID ken is duplicated/,
  );
  assert.throws(() => createThemeFamilies([]), /site theme Main family is required/);
});
