import type { GameInput, PlayerInput } from "../types.ts";
import type { InputSource } from "./InputSource.ts";

type PlayerControl = keyof PlayerInput;
type PulseControl = "insertCoin" | "pause";
type Control =
  | { readonly kind: "player"; readonly control: PlayerControl }
  | { readonly kind: "pulse"; readonly control: PulseControl };

function controlForKey(key: string): Control | null {
  switch (key.toLowerCase()) {
    case "arrowleft":
    case "a":
      return { kind: "player", control: "left" };
    case "arrowright":
    case "d":
      return { kind: "player", control: "right" };
    case "arrowup":
    case "w":
      return { kind: "player", control: "up" };
    case "arrowdown":
    case "s":
      return { kind: "player", control: "down" };
    case "z":
    case "j":
      return { kind: "player", control: "light" };
    case "x":
    case "k":
      return { kind: "player", control: "heavy" };
    case "c":
      return { kind: "player", control: "projectile" };
    case "shift":
      return { kind: "player", control: "block" };
    case "enter":
      return { kind: "pulse", control: "insertCoin" };
    case "escape":
      return { kind: "pulse", control: "pause" };
    default:
      return null;
  }
}

export function createKeyboardInputSource(target: HTMLElement): InputSource {
  const held = new Set<PlayerControl>();
  const tapped = new Set<PlayerControl>();
  const pulses = new Set<PulseControl>();
  let active = false;

  const releaseAll = (): void => {
    held.clear();
    tapped.clear();
    pulses.clear();
  };

  const onKeyDown = (event: KeyboardEvent): void => {
    if (!active || event.target !== target || document.activeElement !== target) return;
    const mapped = controlForKey(event.key);
    if (mapped === null) return;
    event.preventDefault();
    if (mapped.kind === "player") {
      held.add(mapped.control);
      if (!event.repeat) tapped.add(mapped.control);
    }
    else if (!event.repeat) pulses.add(mapped.control);
  };

  const onKeyUp = (event: KeyboardEvent): void => {
    if (!active || event.target !== target || document.activeElement !== target) return;
    const mapped = controlForKey(event.key);
    if (mapped === null || mapped.kind !== "player") return;
    event.preventDefault();
    held.delete(mapped.control);
  };

  target.addEventListener("keydown", onKeyDown);
  target.addEventListener("keyup", onKeyUp);

  return {
    read(): GameInput {
      const player: PlayerInput = {
        left: held.has("left") || tapped.has("left"),
        right: held.has("right") || tapped.has("right"),
        up: held.has("up") || tapped.has("up"),
        down: held.has("down") || tapped.has("down"),
        light: held.has("light") || tapped.has("light"),
        heavy: held.has("heavy") || tapped.has("heavy"),
        projectile: held.has("projectile") || tapped.has("projectile"),
        block: held.has("block") || tapped.has("block"),
      };
      const input: GameInput = {
        players: [player, {}],
        insertCoin: pulses.has("insertCoin"),
        pause: pulses.has("pause"),
      };
      tapped.clear();
      pulses.clear();
      return input;
    },
    setActive(nextActive: boolean): void {
      active = nextActive;
      if (!active) releaseAll();
    },
    releaseAll,
    destroy(): void {
      active = false;
      releaseAll();
      target.removeEventListener("keydown", onKeyDown);
      target.removeEventListener("keyup", onKeyUp);
    },
  };
}
