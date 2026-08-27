import { FIGHTER_ROSTER } from "./content/fighters/index.ts";
import { createGameCore } from "./core/state-machine.ts";
import { BrowserGameHost, showArcadeFallback } from "./BrowserGameHost.ts";
import { createKeyboardInputSource } from "./input/keyboard.ts";
import { TextmodeRenderer } from "./render/TextmodeRenderer.ts";
import { createGamePalette, type GamePalette } from "./render/palette.ts";

class ArcadeBootstrapError extends TypeError {
  constructor(message: string) {
    super(message);
    this.name = "ArcadeBootstrapError";
  }
}

function paletteMap(root: HTMLElement): ReadonlyMap<string, GamePalette> {
  const payload = root.querySelector("[data-game-palettes]");
  if (!(payload instanceof HTMLScriptElement) || payload.textContent === null) {
    throw new ArcadeBootstrapError("ArcadeGame palette payload is missing");
  }
  const parsed: unknown = JSON.parse(payload.textContent);
  if (!Array.isArray(parsed)) throw new ArcadeBootstrapError("ArcadeGame palette payload must be an array");
  const palettes = new Map<string, GamePalette>();
  for (const entry of parsed) {
    if (typeof entry !== "object" || entry === null || Array.isArray(entry)) {
      throw new ArcadeBootstrapError("ArcadeGame palette entry must be an object");
    }
    const id = Reflect.get(entry, "id");
    if (typeof id !== "string" || id.length === 0) {
      throw new ArcadeBootstrapError("ArcadeGame palette entry requires an ID");
    }
    palettes.set(id, createGamePalette(Reflect.get(entry, "tokens")));
  }
  return palettes;
}

function fighterDefinition(id: string, fallbackIndex: number): unknown {
  const fighter = FIGHTER_ROSTER.find(({ definition }) => definition.id === id)
    ?? FIGHTER_ROSTER[fallbackIndex];
  if (fighter === undefined) throw new ArcadeBootstrapError(`ArcadeGame fighter ${id} is missing`);
  return fighter.definition;
}

function configuredFighterId(root: HTMLElement, key: "p1" | "p2", fallback: string): string {
  if (root.dataset.configurableFighters !== "true") return fallback;
  const candidate = new URL(window.location.href).searchParams.get(key);
  return FIGHTER_ROSTER.find(({ definition }) => definition.id === candidate)?.definition.id ?? fallback;
}

export async function bootstrapArcadeGame(root: HTMLElement): Promise<BrowserGameHost> {
  const palettes = paletteMap(root);
  const initialThemeId = root.dataset.initialThemeId ?? "ryu";
  const initialPlayerOneId = configuredFighterId(root, "p1", root.dataset.initialPlayerOne ?? "ryu");
  const fallbackPlayerTwoId = initialPlayerOneId === "ken" ? "ryu" : "ken";
  const requestedPlayerTwoId = configuredFighterId(root, "p2", root.dataset.initialPlayerTwo ?? fallbackPlayerTwoId);
  const initialPlayerTwoId = requestedPlayerTwoId === initialPlayerOneId ? fallbackPlayerTwoId : requestedPlayerTwoId;
  const host = new BrowserGameHost({
    root,
    core: createGameCore({
      seed: 2_905_201,
      fighters: [
        fighterDefinition(initialPlayerOneId, 0),
        fighterDefinition(initialPlayerTwoId, 1),
      ],
    }),
    renderer: new TextmodeRenderer(),
    input: createKeyboardInputSource(root),
    palettes,
    initialThemeId,
  });
  await host.start();
  return host;
}

export async function bootstrapArcadeGames(): Promise<void> {
  const roots = document.querySelectorAll<HTMLElement>("[data-arcade-game]");
  await Promise.all(
    [...roots].map(async (root) => {
      try {
        await bootstrapArcadeGame(root);
      } catch (error) {
        showArcadeFallback(root, error);
      }
    }),
  );
}
