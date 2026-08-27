export const RENDER_LAYER_ORDER = [
  "distant",
  "middle",
  "floor",
  "foreground",
  "fighters",
  "effects",
  "hud",
  "overlays",
] as const;

export type RenderLayer = (typeof RENDER_LAYER_ORDER)[number];

export type StageLayer = Extract<RenderLayer, "distant" | "middle" | "floor" | "foreground">;
