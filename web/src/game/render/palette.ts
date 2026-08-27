const HEX_COLOR = /^#[0-9a-fA-F]{6}$/;

export const GAME_PALETTE_ROLES = [
  "background",
  "foreground",
  "distantBackground",
  "distantForeground",
  "middleBackground",
  "middleForeground",
  "foregroundBackground",
  "foregroundForeground",
  "hudBackground",
  "hudForeground",
  "main",
  "muted",
  "playerOne",
  "playerTwo",
  "damage",
  "victory",
  "coin",
  "energy",
  "secondary",
  "fighterShadow",
  "fighterMidtone",
  "fighterHighlight",
] as const;

export type GamePaletteRole = (typeof GAME_PALETTE_ROLES)[number];

type UiColors = {
  readonly background: string;
  readonly foreground: string;
  readonly cursor_bg: string;
  readonly cursor_fg: string;
  readonly selection_bg: string;
  readonly selection_fg: string;
  readonly panel_bg: string;
  readonly sidebar_bg: string;
  readonly active_row_bg: string;
  readonly navigate_row_bg: string;
  readonly surface_dim: string;
  readonly surface0: string;
  readonly surface1: string;
  readonly overlay0: string;
  readonly overlay1: string;
  readonly subtext: string;
  readonly accent: string;
};

type SemanticColors = {
  readonly red: string;
  readonly green: string;
  readonly yellow: string;
  readonly blue: string;
  readonly magenta: string;
  readonly cyan: string;
  readonly orange: string;
};

type AnsiColors = {
  readonly black: string;
  readonly red: string;
  readonly green: string;
  readonly yellow: string;
  readonly blue: string;
  readonly magenta: string;
  readonly cyan: string;
  readonly white: string;
};

export type ThemeColorTokens = {
  readonly ui: UiColors;
  readonly semantic: SemanticColors;
  readonly ansi: {
    readonly normal: AnsiColors;
    readonly bright: AnsiColors;
  };
};

export type GamePalette = {
  readonly roles: Readonly<Record<GamePaletteRole, string>>;
  readonly source: ThemeColorTokens;
};

export class PaletteContractError extends TypeError {
  readonly field: string;

  constructor(field: string) {
    super(`${field}: expected #RRGGBB theme color`);
    this.name = "PaletteContractError";
    this.field = field;
  }
}

function table(value: unknown, field: string): object {
  if (typeof value !== "object" || value === null || Array.isArray(value)) throw new PaletteContractError(field);
  return value;
}

function color(source: object, field: string, prefix: string): string {
  const value = Reflect.get(source, field);
  if (typeof value !== "string" || !HEX_COLOR.test(value)) throw new PaletteContractError(`${prefix}.${field}`);
  return value;
}

function uiColors(value: unknown): UiColors {
  const source = table(value, "ui");
  return {
    background: color(source, "background", "ui"), foreground: color(source, "foreground", "ui"),
    cursor_bg: color(source, "cursor_bg", "ui"), cursor_fg: color(source, "cursor_fg", "ui"),
    selection_bg: color(source, "selection_bg", "ui"), selection_fg: color(source, "selection_fg", "ui"),
    panel_bg: color(source, "panel_bg", "ui"), sidebar_bg: color(source, "sidebar_bg", "ui"),
    active_row_bg: color(source, "active_row_bg", "ui"), navigate_row_bg: color(source, "navigate_row_bg", "ui"),
    surface_dim: color(source, "surface_dim", "ui"), surface0: color(source, "surface0", "ui"),
    surface1: color(source, "surface1", "ui"), overlay0: color(source, "overlay0", "ui"),
    overlay1: color(source, "overlay1", "ui"), subtext: color(source, "subtext", "ui"),
    accent: color(source, "accent", "ui"),
  };
}

function semanticColors(value: unknown): SemanticColors {
  const source = table(value, "semantic");
  return {
    red: color(source, "red", "semantic"), green: color(source, "green", "semantic"),
    yellow: color(source, "yellow", "semantic"), blue: color(source, "blue", "semantic"),
    magenta: color(source, "magenta", "semantic"), cyan: color(source, "cyan", "semantic"),
    orange: color(source, "orange", "semantic"),
  };
}

function ansiColors(value: unknown, prefix: string): AnsiColors {
  const source = table(value, prefix);
  return {
    black: color(source, "black", prefix), red: color(source, "red", prefix),
    green: color(source, "green", prefix), yellow: color(source, "yellow", prefix),
    blue: color(source, "blue", prefix), magenta: color(source, "magenta", prefix),
    cyan: color(source, "cyan", prefix), white: color(source, "white", prefix),
  };
}

export function createGamePalette(value: unknown): GamePalette {
  const source = table(value, "theme");
  const ui = uiColors(Reflect.get(source, "ui"));
  const semantic = semanticColors(Reflect.get(source, "semantic"));
  const ansi = table(Reflect.get(source, "ansi"), "ansi");
  const normal = ansiColors(Reflect.get(ansi, "normal"), "ansi.normal");
  const bright = ansiColors(Reflect.get(ansi, "bright"), "ansi.bright");
  return {
    roles: {
      background: ui.background, foreground: ui.foreground,
      distantBackground: ui.surface_dim, distantForeground: ui.overlay0,
      middleBackground: ui.surface0, middleForeground: ui.overlay1,
      foregroundBackground: ui.surface1, foregroundForeground: ui.foreground,
      hudBackground: ui.panel_bg, hudForeground: ui.foreground,
      main: ui.accent, muted: ui.subtext,
      playerOne: semantic.blue, playerTwo: semantic.orange,
      damage: semantic.red, victory: semantic.green, coin: semantic.yellow,
      energy: semantic.cyan, secondary: semantic.magenta,
      fighterShadow: normal.black, fighterMidtone: normal.white, fighterHighlight: bright.white,
    },
    source: { ui, semantic, ansi: { normal, bright } },
  };
}
