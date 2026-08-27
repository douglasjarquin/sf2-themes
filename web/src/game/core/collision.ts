import { STAGE_LEFT, STAGE_RIGHT } from "../config.ts";
import type { Box, Facing } from "../types.ts";

export type Point = {
  readonly x: number;
  readonly y: number;
};

export type Separation = {
  readonly firstX: number;
  readonly secondX: number;
};

export function intersects(first: Box, second: Box): boolean {
  return (
    first.x < second.x + second.width &&
    first.x + first.width > second.x &&
    first.y < second.y + second.height &&
    first.y + first.height > second.y
  );
}

export function placeBox(box: Box, origin: Point, facing: Facing): Box {
  return {
    x: facing === 1 ? origin.x + box.x : origin.x - box.x - box.width,
    y: origin.y + box.y,
    width: box.width,
    height: box.height,
  };
}

export function clampFighterX(x: number, pushbox: Box): number {
  const leftExtent = pushbox.width / 2;
  return Math.max(STAGE_LEFT + leftExtent, Math.min(STAGE_RIGHT - leftExtent, x));
}

function centeredPushbox(x: number, y: number, box: Box): Box {
  return { x: x - box.width / 2, y: y + box.y, width: box.width, height: box.height };
}

export function separatePushboxes(
  first: Point,
  firstBox: Box,
  second: Point,
  secondBox: Box,
): Separation {
  const firstPlaced = centeredPushbox(first.x, first.y, firstBox);
  const secondPlaced = centeredPushbox(second.x, second.y, secondBox);
  if (!intersects(firstPlaced, secondPlaced)) return { firstX: first.x, secondX: second.x };

  const overlap = Math.min(firstPlaced.x + firstPlaced.width, secondPlaced.x + secondPlaced.width) -
    Math.max(firstPlaced.x, secondPlaced.x);
  const firstIsLeft = first.x <= second.x;
  const firstCandidate = first.x + (firstIsLeft ? -Math.ceil(overlap / 2) : Math.ceil(overlap / 2));
  const secondCandidate = second.x + (firstIsLeft ? Math.floor(overlap / 2) : -Math.floor(overlap / 2));
  let firstX = clampFighterX(firstCandidate, firstBox);
  let secondX = clampFighterX(secondCandidate, secondBox);

  const remaining = firstIsLeft
    ? firstX + firstBox.width / 2 - (secondX - secondBox.width / 2)
    : secondX + secondBox.width / 2 - (firstX - firstBox.width / 2);
  if (remaining > 0) {
    if (firstIsLeft) {
      if (secondX + secondBox.width / 2 < STAGE_RIGHT) secondX += remaining;
      else firstX -= remaining;
    } else if (firstX + firstBox.width / 2 < STAGE_RIGHT) firstX += remaining;
    else secondX -= remaining;
  }
  return {
    firstX: clampFighterX(firstX, firstBox),
    secondX: clampFighterX(secondX, secondBox),
  };
}
