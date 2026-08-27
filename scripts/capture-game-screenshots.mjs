import { spawn } from "node:child_process";
import { createHash, randomBytes } from "node:crypto";
import { existsSync, promises as fs } from "node:fs";
import http from "node:http";
import { createRequire } from "node:module";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";

import {
  DEFAULT_ARCHIVE,
  DEFAULT_SEED,
  LOGICAL_GRID,
  RECIPE_VERSION,
  VIEWPORT,
  ScreenshotVerificationError,
  captureRecipe,
  currentFingerprints,
  inspectPng,
  loadCanonicalThemes,
  verifyArchive,
} from "./verify-game-screenshots.mjs";

const projectRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const webRoot = path.join(projectRoot, "web");
const distRoot = path.join(webRoot, "dist");
const requireFromWeb = createRequire(path.join(webRoot, "package.json"));
const { chromium } = requireFromWeb("@playwright/test");
const SAFE_SEED = /^[A-Za-z0-9._:-]{1,128}$/;

class ScreenshotCaptureError extends Error {
  constructor(message) {
    super(message);
    this.name = "ScreenshotCaptureError";
  }
}

function fail(message) {
  throw new ScreenshotCaptureError(message);
}

const OUTPUT_POLICY_ERROR = "existing custom output paths are not supported; --output must be a fresh dedicated directory or the canonical archive";

function validateOutputDirectory(outputDirectory) {
  if (existsSync(outputDirectory) && outputDirectory !== DEFAULT_ARCHIVE) fail(OUTPUT_POLICY_ERROR);
}

function requiredValue(argv, index, argument) {
  const value = argv[index + 1];
  if (value === undefined || value.startsWith("--")) fail(`${argument} requires a value`);
  return value;
}

export function parseCaptureArguments(argv) {
  const options = { theme: undefined, seed: DEFAULT_SEED, seedExplicit: false, randomize: false, outputDirectory: undefined, help: false };
  for (let index = 0; index < argv.length; index += 1) {
    const argument = argv[index];
    if (argument === "--help") options.help = true;
    else if (argument === "--randomize") options.randomize = true;
    else if (argument === "--theme" || argument === "--seed" || argument === "--output") {
      const value = requiredValue(argv, index, argument);
      if (argument === "--theme") options.theme = value;
      if (argument === "--seed") {
        options.seed = value;
        options.seedExplicit = true;
      }
      if (argument === "--output") options.outputDirectory = path.resolve(value);
      index += 1;
    } else fail(`unknown capture argument ${argument}`);
  }
  if (options.randomize && options.seedExplicit) fail("--randomize and --seed are mutually exclusive");
  if (!SAFE_SEED.test(options.seed)) fail("seed must contain 1-128 letters, digits, dots, underscores, colons, or hyphens");
  if (options.theme !== undefined && options.outputDirectory === undefined) fail("--theme requires --output so a partial capture cannot replace the canonical archive");
  if (options.outputDirectory !== undefined) {
    validateOutputDirectory(options.outputDirectory);
  }
  return options;
}

function contentType(filePath) {
  const extension = path.extname(filePath);
  if (extension === ".css") return "text/css; charset=utf-8";
  if (extension === ".html") return "text/html; charset=utf-8";
  if (extension === ".js") return "text/javascript; charset=utf-8";
  if (extension === ".json") return "application/json; charset=utf-8";
  if (extension === ".png") return "image/png";
  if (extension === ".svg") return "image/svg+xml";
  if (extension === ".woff2") return "font/woff2";
  return "application/octet-stream";
}

async function staticResponse(request, response) {
  try {
    const requestUrl = new URL(request.url ?? "/", "http://127.0.0.1");
    if (!requestUrl.pathname.startsWith("/sf2-themes/")) {
      response.writeHead(404).end();
      return;
    }
    let relativePath = decodeURIComponent(requestUrl.pathname.slice("/sf2-themes/".length));
    if (relativePath.length === 0 || relativePath.endsWith("/")) relativePath += "index.html";
    const filePath = path.resolve(distRoot, relativePath);
    if (filePath !== distRoot && !filePath.startsWith(`${distRoot}${path.sep}`)) {
      response.writeHead(400).end();
      return;
    }
    const file = await fs.readFile(filePath);
    response.writeHead(200, { "Content-Type": contentType(filePath), "Cache-Control": "no-store" });
    response.end(file);
  } catch (error) {
    if (error instanceof URIError || (error instanceof Error && "code" in error && error.code === "ENOENT")) {
      response.writeHead(404).end();
      return;
    }
    response.writeHead(500).end();
  }
}

async function startStaticServer() {
  await fs.access(path.join(distRoot, "game", "index.html"));
  const server = http.createServer((request, response) => {
    void staticResponse(request, response);
  });
  await new Promise((resolve, reject) => {
    server.once("error", reject);
    server.listen(0, "127.0.0.1", resolve);
  });
  const address = server.address();
  if (address === null || typeof address === "string") fail("capture server did not expose a loopback port");
  return { server, port: address.port };
}

async function closeServer(server) {
  await new Promise((resolve, reject) => {
    server.close((error) => error === undefined ? resolve() : reject(error));
    server.closeAllConnections();
  });
}

function sha256(value) {
  return createHash("sha256").update(value).digest("hex");
}

function same(actual, expected, label) {
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    fail(`${label} mismatch: expected ${JSON.stringify(expected)}, got ${JSON.stringify(actual)}`);
  }
}

async function captureTheme(page, stageDirectory, origin, theme, seed) {
  const recipe = captureRecipe(theme.catalogIndex, seed);
  const search = new URLSearchParams({
    capture: "1",
    theme: theme.id,
    seed,
    mode: recipe.mode,
    stage: recipe.stage,
    p1: recipe.p1,
    p2: recipe.p2,
    tick: "0",
    moment: recipe.moment,
  });
  const url = `${origin}/sf2-themes/game/?${search.toString()}`;
  await page.goto(url, { waitUntil: "networkidle" });
  const observed = await page.evaluate(async ({ targetTick, requestedSeed }) => {
    await document.fonts.ready;
    const bridge = window.__SF2_GAME__;
    if (bridge === undefined) throw new Error("capture bridge was not installed");
    await bridge.ready;
    bridge.reset(requestedSeed);
    bridge.advanceUntil(targetTick);
    const canvas = document.querySelector("[data-game-renderer] canvas");
    const surface = document.querySelector("[data-game-route]");
    return {
      state: bridge.getCaptureState(),
      canvasCount: document.querySelectorAll("[data-game-renderer] canvas").length,
      logicalGrid: canvas instanceof HTMLCanvasElement ? canvas.dataset.logicalGrid : "missing",
      surface: surface instanceof HTMLElement
        ? { width: surface.getBoundingClientRect().width }
        : null,
      viewport: { width: window.innerWidth, height: window.innerHeight },
    };
  }, { targetTick: recipe.tick, requestedSeed: seed });
  same(observed.state, {
    tick: recipe.tick,
    mode: recipe.mode,
    moment: recipe.moment,
    transition: "none",
    visibility: "visible",
    hud: observed.state.hud,
    theme: theme.id,
    seed,
    stage: recipe.stage,
    p1: recipe.p1,
    p2: recipe.p2,
    complete: observed.state.complete,
    logicalSize: LOGICAL_GRID,
  }, `${theme.id} capture state`);
  same(observed.canvasCount, 1, `${theme.id} canvas count`);
  same(observed.logicalGrid, "96x40", `${theme.id} logical grid`);
  same(observed.surface, { width: VIEWPORT.width }, `${theme.id} surface width`);
  same(observed.viewport, { width: VIEWPORT.width, height: VIEWPORT.height }, `${theme.id} viewport dimensions`);
  const relativeImage = `game/${theme.id}.png`;
  const imagePath = path.join(stageDirectory, relativeImage);
  await page.screenshot({ path: imagePath, animations: "disabled", caret: "hide" });
  const image = await fs.readFile(imagePath);
  const inspection = inspectPng(image);
  same({ width: inspection.width, height: inspection.height }, { width: VIEWPORT.width, height: VIEWPORT.height }, `${theme.id} PNG dimensions`);
  return {
    id: theme.id,
    name: theme.name,
    image: relativeImage,
    sha256: sha256(image),
    width: VIEWPORT.width,
    height: VIEWPORT.height,
    deviceScaleFactor: VIEWPORT.deviceScaleFactor,
    logicalGrid: LOGICAL_GRID,
    ...recipe,
    occupancy: inspection.occupancy,
  };
}

async function promote(stageDirectory, outputDirectory) {
  if (outputDirectory !== DEFAULT_ARCHIVE) {
    await fs.mkdir(outputDirectory);
    for (const entry of await fs.readdir(stageDirectory)) {
      await fs.rename(path.join(stageDirectory, entry), path.join(outputDirectory, entry));
    }
    await fs.rmdir(stageDirectory);
    return;
  }
  const backupDirectory = `${outputDirectory}.backup-${process.pid}`;
  let previousMoved = false;
  let stageMoved = false;
  try {
    try {
      await fs.rename(outputDirectory, backupDirectory);
      previousMoved = true;
    } catch (error) {
      if (!(error instanceof Error) || !("code" in error) || error.code !== "ENOENT") throw error;
    }
    if (process.env.SF2_CAPTURE_FAIL_PROMOTION === "1") throw new Error("injected promotion failure");
    await fs.rename(stageDirectory, outputDirectory);
    stageMoved = true;
    if (previousMoved) await fs.rm(backupDirectory, { recursive: true });
  } catch (error) {
    if (previousMoved && !stageMoved) await fs.rename(backupDirectory, outputDirectory);
    throw error;
  }
}

export async function captureArchive(options) {
  const outputDirectory = options.outputDirectory ?? DEFAULT_ARCHIVE;
  validateOutputDirectory(outputDirectory);
  const catalog = await loadCanonicalThemes();
  const themes = options.theme === undefined ? catalog : catalog.filter(({ id }) => id === options.theme);
  if (themes.length === 0) fail(`unknown theme ${options.theme}`);
  const seed = options.randomize ? randomBytes(16).toString("hex") : options.seed;
  await fs.mkdir(path.dirname(outputDirectory), { recursive: true });
  const stageDirectory = await fs.mkdtemp(path.join(path.dirname(outputDirectory), `.${path.basename(outputDirectory)}.stage-`));
  await fs.mkdir(path.join(stageDirectory, "game"));
  console.log(`Capture seed=${seed}; staging=${stageDirectory}`);
  const errors = [];
  let browser;
  let context;
  let server;
  try {
    const running = await startStaticServer();
    server = running.server;
    browser = await chromium.launch({
      headless: true,
      args: ["--force-color-profile=srgb", "--disable-lcd-text", "--disable-font-subpixel-positioning", "--font-render-hinting=none"],
    });
    context = await browser.newContext({ viewport: { width: VIEWPORT.width, height: VIEWPORT.height }, deviceScaleFactor: VIEWPORT.deviceScaleFactor, reducedMotion: "reduce" });
    const page = await context.newPage();
    page.on("console", (message) => {
      if (message.type() === "error") errors.push(`console: ${message.text()}`);
    });
    page.on("pageerror", (error) => errors.push(`page: ${error.message}`));
    const origin = `http://127.0.0.1:${running.port}`;
    const entries = [];
    for (const theme of themes) {
      entries.push(await captureTheme(page, stageDirectory, origin, theme, seed));
      console.log(`Captured ${theme.id}`);
    }
    if (errors.length > 0) fail(`browser errors: ${errors.join(" | ")}`);
    const fingerprints = await currentFingerprints();
    const manifest = {
      schemaVersion: 1,
      recipeVersion: RECIPE_VERSION,
      viewport: VIEWPORT,
      logicalGrid: LOGICAL_GRID,
      ...fingerprints,
      seed,
      entries,
    };
    await fs.writeFile(path.join(stageDirectory, "manifest.json"), `${JSON.stringify(manifest, null, 2)}\n`, "utf8");
    await verifyArchive({ archiveDirectory: stageDirectory, themes, expectedSeed: seed });
    await context.close();
    context = undefined;
    await browser.close();
    browser = undefined;
    await closeServer(server);
    server = undefined;
    await promote(stageDirectory, outputDirectory);
    console.log(`Promoted ${themes.length} verified screenshot(s) to ${outputDirectory}; seed=${seed}`);
    return { outputDirectory, seed, entryCount: themes.length };
  } catch (error) {
    await context?.close();
    await browser?.close();
    if (server !== undefined) await closeServer(server);
    await fs.rm(stageDirectory, { recursive: true, force: true });
    throw error;
  }
}

async function main() {
  const options = parseCaptureArguments(process.argv.slice(2));
  if (options.help) {
    console.log("Usage: screenshots:game [--theme THEME_ID --output PATH] [--seed SEED | --randomize] [--output PATH]");
    return;
  }
  await captureArchive(options);
}

const entryPath = process.argv[1] === undefined ? "" : pathToFileURL(path.resolve(process.argv[1])).href;
if (entryPath === import.meta.url) {
  main().catch((error) => {
    const detail = error instanceof ScreenshotVerificationError || error instanceof Error
      ? `${error.name}: ${error.message}`
      : String(error);
    console.error(detail);
    process.exitCode = 1;
  });
}
