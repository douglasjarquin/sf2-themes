import assert from "node:assert/strict";
import test from "node:test";

import { themeTokens } from "../src/data/theme-data.mjs";
import { GAME_PALETTE_ROLES, PaletteContractError, createGamePalette } from "../src/game/render/palette.ts";

test("palette maps every game role and retains all TOML color fields", () => {
  const palette = createGamePalette(themeTokens.main);

  assert.deepEqual(Object.keys(palette.roles), GAME_PALETTE_ROLES);
  assert.equal(palette.roles.background, themeTokens.main.ui.background);
  assert.equal(palette.roles.hudForeground, themeTokens.main.ui.foreground);
  assert.equal(palette.roles.damage, themeTokens.main.semantic.red);
  assert.equal(palette.roles.fighterHighlight, themeTokens.main.ansi.bright.white);
  assert.deepEqual(palette.source, {
    ui: themeTokens.main.ui,
    semantic: themeTokens.main.semantic,
    ansi: themeTokens.main.ansi,
  });
});

test("palette rejects malformed theme colors at the adapter boundary", () => {
  const malformed = structuredClone(themeTokens.main);
  malformed.semantic.red = "not-a-color";

  assert.throws(() => createGamePalette(malformed), PaletteContractError);
  assert.throws(() => createGamePalette(null), PaletteContractError);
});
