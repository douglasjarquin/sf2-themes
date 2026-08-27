import assert from "node:assert/strict";
import test from "node:test";

import { TextmodeGrid } from "textmode.js";

import { TextmodeRenderer } from "../src/game/render/TextmodeRenderer.ts";
import { FIGHTER_ROSTER } from "../src/game/content/fighters/index.ts";
import { createGamePalette } from "../src/game/render/palette.ts";
import { themeTokens } from "../src/data/theme-data.mjs";

function makeHarness({ width = 960, height = 400, realGrid = false } = {}) {
  const calls = {
    noLoop: 0,
    redraw: 0,
    resize: [],
    fontSize: [],
    destroy: 0,
    removed: 0,
    draw: null,
    options: null,
    prints: [],
  };
  const canvas = { style: {}, dataset: {}, remove() { calls.removed += 1; } };
  let activeCharColor = null;
  let translation = { x: 0, y: 0 };
  const transformStack = [];
  const context = {
    canvas,
    grid: { cols: 1, rows: 1 },
    setup(callback) { callback(); return Promise.resolve(); },
    draw(callback) { calls.draw = callback; },
    noLoop() { calls.noLoop += 1; },
    isLooping() { return false; },
    redraw(count) { calls.redraw += count; calls.draw?.(); },
    resizeCanvas(width, height) {
      calls.resize.push([width, height]);
      const density = calls.options?.pixelDensity ?? 1;
      canvas.width = width * density;
      canvas.height = height * density;
    },
    fontSize(size) {
      calls.fontSize.push(size);
      if (realGrid) {
        context.grid = new TextmodeGrid(canvas, size, size);
      }
    },
    destroy() { calls.destroy += 1; },
    background() {},
    charColor(color) { activeCharColor = color; },
    cellColor() {},
    push() { transformStack.push({ ...translation }); },
    translate(x = 0, y = 0) {
      translation = { x: translation.x + x, y: translation.y + y };
    },
    pop() {
      const previous = transformStack.pop();
      assert.ok(previous, "renderer popped an empty transform stack");
      translation = previous;
    },
    print(text, x, y) {
      calls.prints.push({
        text,
        x,
        y,
        projectedX: x + translation.x,
        projectedY: y + translation.y,
        color: activeCharColor,
      });
    },
  };
  const container = {
    ownerDocument: { createElement: () => canvas },
    append: (child) => { assert.equal(child, canvas); },
    getBoundingClientRect: () => ({ width, height }),
  };
  return {
    calls,
    canvas,
    container,
    context,
    factory: (options) => {
      calls.options = options;
      if (realGrid) context.grid = new TextmodeGrid(canvas, options.fontSize ?? 16, options.fontSize ?? 16);
      return context;
    },
  };
}

const snapshot = {
  tick: 120,
  phase: "player-fight",
  phaseTick: 12,
  roundTicksRemaining: 5_400,
  rngState: 7,
  fighters: [
    { id: "ryu", x: 2_800, y: 3_600, velocityX: 0, velocityY: 0, facing: 1, health: 90, pose: "idle", moveId: null, moveTick: 0, hitstunTicks: 0, blockstunTicks: 0, hitstopTicks: 0 },
    { id: "ken", x: 6_800, y: 3_600, velocityX: 0, velocityY: 0, facing: -1, health: 75, pose: "attack", moveId: "roundhouse", moveTick: 5, hitstunTicks: 0, blockstunTicks: 0, hitstopTicks: 0 },
  ],
  projectiles: [{ owner: 0, x: 4_800, y: 3_000, velocityX: 200 }],
  result: null,
};

test("renderer mounts a fixed 96x40 non-looping grid, resizes, redraws, and destroys", async () => {
  for (let cycle = 0; cycle < 2; cycle += 1) {
    const harness = makeHarness();
    const renderer = new TextmodeRenderer(harness.factory);
    renderer.setPalette(createGamePalette(themeTokens.main));

    await renderer.mount(harness.container);
    renderer.resize(480, 320);
    renderer.render(snapshot);
    renderer.destroy();

    assert.deepEqual(renderer.logicalSize, { columns: 96, rows: 40 });
    assert.ok(harness.calls.noLoop >= 1);
    assert.equal(harness.calls.redraw, 1);
    assert.deepEqual(harness.calls.resize.at(-1), [480, 200]);
    assert.equal(harness.calls.destroy, 1);
    assert.equal(harness.calls.removed, 1);
  }
});

test("renderer rejects malformed lifecycle and snapshot input without leaking a loop", async () => {
  const harness = makeHarness();
  const renderer = new TextmodeRenderer(harness.factory);

  assert.throws(() => renderer.render(null), /snapshot/);
  await renderer.mount(harness.container);
  await assert.rejects(() => renderer.mount(harness.container), /mounted/);
  assert.throws(() => renderer.render({ ...snapshot, fighters: [] }), /fighters/);
  renderer.destroy();
  assert.equal(harness.calls.destroy, 1);
});

test("renderer keeps the complete fight scene inside its public cell grid", async () => {
  // Given a mounted renderer with a representative attract fight snapshot
  const harness = makeHarness();
  const renderer = new TextmodeRenderer(harness.factory);
  const palette = createGamePalette(themeTokens.main);
  renderer.setPalette(palette);

  await renderer.mount(harness.container);
  renderer.render({ ...snapshot, phase: "attract-fight" });
  renderer.render({ ...snapshot, tick: snapshot.tick + 1, phase: "attract-fight" });

  // Then every authored print remains inside the public top-left logical grid
  assert.ok(harness.calls.prints.length > 0);
  const { columns, rows } = renderer.logicalSize;
  const outsideLogicalGrid = harness.calls.prints.filter(
    ({ text, x, y }) =>
      x < 0
      || y < 0
      || x + Array.from(text).length > columns
      || y >= rows,
  );
  assert.deepEqual(outsideLogicalGrid, []);

  // And repeated frames resolve inside textmode's center-origin drawing grid
  const projectedLeft = -Math.floor((columns - 1) / 2);
  const projectedTop = -Math.floor(rows / 2);
  const outsideTextmodeGrid = harness.calls.prints.filter(
    ({ text, projectedX, projectedY }) =>
      projectedX < projectedLeft
      || projectedY < projectedTop
      || projectedX + Array.from(text).length > projectedLeft + columns
      || projectedY >= projectedTop + rows,
  );
  assert.deepEqual(outsideTextmodeGrid, []);

  // And semantic layers still form a complete composition
  assert.ok(harness.calls.prints.some(({ text }) => text.includes("OPEN HAND DOJO")));
  assert.ok(harness.calls.prints.some(({ text }) => text.startsWith("P1 ")));
  assert.ok(harness.calls.prints.some(({ text }) => text === "o=>"));
  assert.ok(harness.calls.prints.some(({ text }) => text === "DEMO"));
  assert.ok(harness.calls.prints.some(({ color, text }) => color === palette.roles.playerOne && text.length === 1));
  assert.ok(harness.calls.prints.some(({ color, text }) => color === palette.roles.playerTwo && text.length === 1));
  renderer.destroy();
});

test("renderer draws every canonical fighter through the authored roster", async () => {
  const palette = createGamePalette(themeTokens.main);
  for (const [index, fighter] of FIGHTER_ROSTER.entries()) {
    const harness = makeHarness();
    const renderer = new TextmodeRenderer(harness.factory);
    renderer.setPalette(palette);
    await renderer.mount(harness.container);
    renderer.render({
      ...snapshot,
      fighters: [
        { ...snapshot.fighters[0], id: fighter.definition.id },
        { ...snapshot.fighters[1], id: index === 0 ? "ken" : "ryu" },
      ],
    });
    assert.ok(
      harness.calls.prints.some(({ color, text }) => color === palette.roles.playerOne && text.length === 1),
      `${fighter.definition.id} did not produce a player-one glyph draw`,
    );
    renderer.destroy();
  }
});

test("renderer fits a real locked 96x40 grid inside a compact canvas", async () => {
  const previousDescriptor = Object.getOwnPropertyDescriptor(globalThis, "devicePixelRatio");
  Object.defineProperty(globalThis, "devicePixelRatio", { configurable: true, value: 3 });
  try {
    const harness = makeHarness({ width: 448, height: 186, realGrid: true });
    const renderer = new TextmodeRenderer(harness.factory);

    await renderer.mount(harness.container);
    assert.equal(harness.calls.options?.pixelDensity, 2);
    assert.equal(typeof harness.calls.options?.fontSize, "number");
    const assertGridFits = () => {
      const grid = harness.context.grid;
      assert.equal(grid.cols, renderer.logicalSize.columns);
      assert.equal(grid.rows, renderer.logicalSize.rows);
      assert.ok(grid.width <= harness.canvas.width, `grid width ${grid.width} exceeds canvas width ${harness.canvas.width}`);
      assert.ok(grid.height <= harness.canvas.height, `grid height ${grid.height} exceeds canvas height ${harness.canvas.height}`);
      assert.ok(grid.offsetX >= 0, `grid offsetX ${grid.offsetX} clips the grid`);
      assert.ok(grid.offsetY >= 0, `grid offsetY ${grid.offsetY} clips the grid`);
    };
    assertGridFits();

    renderer.resize(292, 121);
    assert.equal(harness.calls.fontSize.length, 1);
    assert.ok(harness.calls.fontSize[0] > 0);
    assertGridFits();
    renderer.destroy();
  } finally {
    if (previousDescriptor === undefined) delete globalThis.devicePixelRatio;
    else Object.defineProperty(globalThis, "devicePixelRatio", previousDescriptor);
  }
});
