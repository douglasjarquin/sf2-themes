import { createHash } from "node:crypto";
import { promises as fs } from "node:fs";
import path from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { inflateSync } from "node:zlib";

const projectRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const webRoot = path.join(projectRoot, "web");

export const DEFAULT_ARCHIVE = path.join(webRoot, "public", "screenshots");
export const DEFAULT_SEED = "sf2-themes-screenshots-v1";
export const RECIPE_VERSION = "sf2-game-screenshots-v1";
export const VIEWPORT = Object.freeze({ width: 1280, height: 720, deviceScaleFactor: 1 });
export const LOGICAL_GRID = Object.freeze({ columns: 96, rows: 40 });
export const MIN_OCCUPANCY = 0.01;
export const FIGHTER_IDS = Object.freeze([
  "ryu", "ken", "chun-li", "e-honda", "blanka", "zangief", "guile", "dhalsim", "balrog",
  "vega", "sagat", "m-bison", "cammy", "t-hawk", "fei-long", "dee-jay", "akuma",
]);
const MOMENT_RECIPES = Object.freeze([
  Object.freeze({ moment: "fight", tick: 900 }),
  Object.freeze({ moment: "ko", tick: 2_400 }),
  Object.freeze({ moment: "victory", tick: 3_000 }),
  Object.freeze({ moment: "intro", tick: 120 }),
]);
const MANIFEST_KEYS = Object.freeze([
  "schemaVersion", "recipeVersion", "viewport", "logicalGrid", "sourceFingerprint",
  "recipeFingerprint", "paletteHash", "seed", "entries",
]);
const ENTRY_KEYS = Object.freeze([
  "id", "name", "image", "sha256", "width", "height", "deviceScaleFactor", "logicalGrid",
  "seed", "tick", "moment", "mode", "stage", "p1", "p2", "occupancy",
]);

export class ScreenshotVerificationError extends Error {
  constructor(message) {
    super(message);
    this.name = "ScreenshotVerificationError";
  }
}

function fail(message) {
  throw new ScreenshotVerificationError(message);
}

function sha256(value) {
  return createHash("sha256").update(value).digest("hex");
}

function record(value, label) {
  if (typeof value !== "object" || value === null || Array.isArray(value)) fail(`${label} must be an object`);
  return value;
}

function exactKeys(value, expected, label) {
  const actual = Object.keys(value).sort();
  const wanted = [...expected].sort();
  if (JSON.stringify(actual) !== JSON.stringify(wanted)) {
    fail(`${label} keys must be exactly ${wanted.join(", ")}; got ${actual.join(", ")}`);
  }
}

function text(value, label) {
  if (typeof value !== "string" || value.length === 0) fail(`${label} must be a non-empty string`);
  return value;
}

function integer(value, label) {
  if (typeof value !== "number" || !Number.isSafeInteger(value) || value < 0) {
    fail(`${label} must be a non-negative safe integer`);
  }
  return value;
}

async function filesBelow(root, accept) {
  const entries = await fs.readdir(root, { withFileTypes: true });
  const files = [];
  for (const entry of entries.sort((left, right) => left.name.localeCompare(right.name))) {
    const target = path.join(root, entry.name);
    if (entry.isDirectory()) files.push(...await filesBelow(target, accept));
    else if (entry.isFile() && accept(target)) files.push(target);
  }
  return files;
}

async function fingerprint(files, prefix = "") {
  const hash = createHash("sha256").update(prefix);
  for (const file of [...files].sort()) {
    hash.update(path.relative(projectRoot, file).split(path.sep).join("/")).update("\0");
    hash.update(await fs.readFile(file)).update("\0");
  }
  return hash.digest("hex");
}

export async function loadCanonicalThemes() {
  const priorDirectory = process.cwd();
  process.chdir(webRoot);
  try {
    const moduleUrl = pathToFileURL(path.join(webRoot, "src", "data", "theme-data.mjs")).href;
    const { paletteVariants } = await import(moduleUrl);
    if (!Array.isArray(paletteVariants) || paletteVariants.length !== 36) {
      fail(`canonical palette catalog must contain 36 themes; got ${paletteVariants?.length ?? "invalid"}`);
    }
    return paletteVariants.map((theme, index) => ({
      id: text(theme?.id, `canonical theme ${index} id`),
      name: text(theme?.name, `canonical theme ${index} name`),
      catalogIndex: index,
    }));
  } finally {
    process.chdir(priorDirectory);
  }
}

export async function currentFingerprints() {
  const paletteFiles = await filesBelow(path.join(projectRoot, "themes"), (file) => file.endsWith(".toml"));
  const gameFiles = await filesBelow(path.join(webRoot, "src", "game"), (file) => file.endsWith(".ts"));
  const sourceFiles = [
    ...gameFiles,
    path.join(webRoot, "src", "components", "ArcadeGame.astro"),
    path.join(webRoot, "src", "pages", "game", "index.astro"),
    path.join(webRoot, "src", "data", "theme-data.mjs"),
  ];
  const recipeFiles = [
    path.join(projectRoot, "scripts", "capture-game-screenshots.mjs"),
    path.join(webRoot, "package.json"),
  ];
  const recipe = JSON.stringify({ recipeVersion: RECIPE_VERSION, viewport: VIEWPORT, logicalGrid: LOGICAL_GRID, moments: MOMENT_RECIPES });
  return {
    sourceFingerprint: await fingerprint(sourceFiles),
    recipeFingerprint: await fingerprint(recipeFiles, recipe),
    paletteHash: await fingerprint(paletteFiles),
  };
}

export function captureRecipe(catalogIndex, seed) {
  const moment = MOMENT_RECIPES[catalogIndex % MOMENT_RECIPES.length];
  const p1 = FIGHTER_IDS[catalogIndex % FIGHTER_IDS.length];
  const p2 = FIGHTER_IDS[(catalogIndex + 9) % FIGHTER_IDS.length];
  if (moment === undefined || p1 === undefined || p2 === undefined) fail("capture recipe index is outside the supported catalog");
  return { seed, tick: moment.tick, moment: moment.moment, mode: "attract", stage: "dojo", p1, p2 };
}

function paeth(left, above, upperLeft) {
  const prediction = left + above - upperLeft;
  const leftDistance = Math.abs(prediction - left);
  const aboveDistance = Math.abs(prediction - above);
  const upperLeftDistance = Math.abs(prediction - upperLeft);
  return leftDistance <= aboveDistance && leftDistance <= upperLeftDistance
    ? left
    : aboveDistance <= upperLeftDistance ? above : upperLeft;
}

export function inspectPng(buffer) {
  if (!Buffer.isBuffer(buffer) || buffer.length < 33 || !buffer.subarray(0, 8).equals(Buffer.from("89504e470d0a1a0a", "hex"))) {
    fail("image is not a PNG");
  }
  let offset = 8;
  let header;
  const dataChunks = [];
  let ended = false;
  while (offset + 12 <= buffer.length) {
    const length = buffer.readUInt32BE(offset);
    const type = buffer.toString("ascii", offset + 4, offset + 8);
    const dataStart = offset + 8;
    const dataEnd = dataStart + length;
    if (dataEnd + 4 > buffer.length) fail("PNG chunk exceeds file bounds");
    if (type === "IHDR") header = buffer.subarray(dataStart, dataEnd);
    if (type === "IDAT") dataChunks.push(buffer.subarray(dataStart, dataEnd));
    if (type === "IEND") ended = true;
    offset = dataEnd + 4;
    if (ended) break;
  }
  if (header?.length !== 13 || dataChunks.length === 0 || !ended) fail("PNG requires IHDR, IDAT, and IEND chunks");
  const width = header.readUInt32BE(0);
  const height = header.readUInt32BE(4);
  const bitDepth = header[8];
  const colorType = header[9];
  const interlace = header[12];
  const channels = colorType === 2 ? 3 : colorType === 6 ? 4 : 0;
  if (bitDepth !== 8 || channels === 0 || interlace !== 0) fail(`PNG format must be non-interlaced 8-bit RGB/RGBA; got depth=${bitDepth} color=${colorType} interlace=${interlace}`);
  const rowBytes = width * channels;
  const packed = inflateSync(Buffer.concat(dataChunks));
  if (packed.length !== height * (rowBytes + 1)) fail("PNG decompressed payload has the wrong length");
  const pixels = Buffer.alloc(rowBytes * height);
  let inputOffset = 0;
  for (let row = 0; row < height; row += 1) {
    const filter = packed[inputOffset];
    inputOffset += 1;
    if (filter === undefined || filter > 4) fail(`PNG row ${row} has unsupported filter ${filter ?? "missing"}`);
    for (let column = 0; column < rowBytes; column += 1) {
      const raw = packed[inputOffset];
      inputOffset += 1;
      const outputOffset = row * rowBytes + column;
      const left = column >= channels ? pixels[outputOffset - channels] : 0;
      const above = row > 0 ? pixels[outputOffset - rowBytes] : 0;
      const upperLeft = row > 0 && column >= channels ? pixels[outputOffset - rowBytes - channels] : 0;
      const predictor = filter === 0 ? 0 : filter === 1 ? left : filter === 2 ? above : filter === 3
        ? Math.floor((left + above) / 2) : paeth(left, above, upperLeft);
      pixels[outputOffset] = (raw + predictor) & 255;
    }
  }
  let occupied = 0;
  for (let pixel = channels; pixel < pixels.length; pixel += channels) {
    if (pixels[pixel] !== pixels[0] || pixels[pixel + 1] !== pixels[1] || pixels[pixel + 2] !== pixels[2]) occupied += 1;
  }
  return { width, height, occupancy: Number((occupied / (width * height)).toFixed(6)) };
}

function sameObject(actual, expected, label) {
  if (JSON.stringify(actual) !== JSON.stringify(expected)) {
    fail(`${label} must equal ${JSON.stringify(expected)}; got ${JSON.stringify(actual)}`);
  }
}

export async function verifyArchive({ archiveDirectory = DEFAULT_ARCHIVE, themes, expectedSeed } = {}) {
  const canonicalThemes = themes ?? await loadCanonicalThemes();
  const manifestPath = path.join(archiveDirectory, "manifest.json");
  let manifest;
  try {
    manifest = JSON.parse(await fs.readFile(manifestPath, "utf8"));
  } catch (error) {
    fail(`manifest.json is unreadable: ${error instanceof Error ? error.message : String(error)}`);
  }
  const root = record(manifest, "manifest");
  exactKeys(root, MANIFEST_KEYS, "manifest");
  if (root.schemaVersion !== 1) fail("manifest schemaVersion must equal 1");
  if (root.recipeVersion !== RECIPE_VERSION) fail(`manifest recipeVersion must equal ${RECIPE_VERSION}`);
  sameObject(root.viewport, VIEWPORT, "manifest viewport");
  sameObject(root.logicalGrid, LOGICAL_GRID, "manifest logicalGrid");
  const fingerprints = await currentFingerprints();
  for (const key of ["sourceFingerprint", "recipeFingerprint", "paletteHash"]) {
    if (root[key] !== fingerprints[key]) fail(`manifest ${key} is stale`);
  }
  const seed = text(root.seed, "manifest seed");
  if (expectedSeed !== undefined && seed !== expectedSeed) fail(`manifest seed must equal ${expectedSeed}`);
  if (!Array.isArray(root.entries)) fail("manifest entries must be an array");
  if (root.entries.length !== canonicalThemes.length) fail(`manifest must contain ${canonicalThemes.length} entries; got ${root.entries.length}`);
  const ids = root.entries.map((entry, index) => text(record(entry, `entry ${index}`).id, `entry ${index} id`));
  if (new Set(ids).size !== ids.length) fail("manifest contains duplicate theme entries");
  sameObject(ids, canonicalThemes.map(({ id }) => id), "manifest theme order");
  const imageDirectory = path.join(archiveDirectory, "game");
  const actualImages = (await fs.readdir(imageDirectory, { withFileTypes: true }))
    .filter((entry) => entry.isFile()).map(({ name }) => name).sort();
  const expectedImages = canonicalThemes.map(({ id }) => `${id}.png`).sort();
  for (const image of expectedImages) if (!actualImages.includes(image)) fail(`missing image game/${image}`);
  for (const image of actualImages) if (!expectedImages.includes(image)) fail(`orphan image game/${image}`);
  const fighterCoverage = new Set();
  for (const [index, theme] of canonicalThemes.entries()) {
    const entry = record(root.entries[index], `entry ${theme.id}`);
    exactKeys(entry, ENTRY_KEYS, `entry ${theme.id}`);
    const recipe = captureRecipe(theme.catalogIndex, seed);
    const expected = { id: theme.id, name: theme.name, image: `game/${theme.id}.png`, width: VIEWPORT.width, height: VIEWPORT.height, deviceScaleFactor: 1, logicalGrid: LOGICAL_GRID, ...recipe };
    for (const key of Object.keys(expected)) sameObject(entry[key], expected[key], `entry ${theme.id} ${key}`);
    if (!FIGHTER_IDS.includes(entry.p1) || !FIGHTER_IDS.includes(entry.p2) || entry.p1 === entry.p2) {
      fail(`entry ${theme.id} fighters must be distinct canonical IDs`);
    }
    fighterCoverage.add(entry.p1);
    fighterCoverage.add(entry.p2);
    if (!/^[0-9a-f]{64}$/.test(entry.sha256)) fail(`entry ${theme.id} sha256 must be 64 lowercase hex characters`);
    const image = await fs.readFile(path.join(archiveDirectory, entry.image));
    const inspection = inspectPng(image);
    if (inspection.width !== VIEWPORT.width || inspection.height !== VIEWPORT.height) {
      fail(`entry ${theme.id} image dimensions must be 1280x720; got ${inspection.width}x${inspection.height}`);
    }
    if (inspection.occupancy < MIN_OCCUPANCY) fail(`entry ${theme.id} occupancy ${inspection.occupancy} is below ${MIN_OCCUPANCY}`);
    if (entry.occupancy !== inspection.occupancy) fail(`entry ${theme.id} occupancy metadata is stale`);
    if (entry.sha256 !== sha256(image)) fail(`entry ${theme.id} sha256 does not match image bytes`);
  }
  if (canonicalThemes.length === 36) {
    const missingFighters = FIGHTER_IDS.filter((id) => !fighterCoverage.has(id));
    if (missingFighters.length > 0) fail(`manifest fighter coverage is missing ${missingFighters.join(", ")}`);
  }
  return { archiveDirectory, entryCount: canonicalThemes.length, seed, ...fingerprints };
}

function parseArguments(argv) {
  const options = { archiveDirectory: DEFAULT_ARCHIVE, theme: undefined, help: false };
  for (let index = 0; index < argv.length; index += 1) {
    const argument = argv[index];
    if (argument === "--help") options.help = true;
    else if (argument === "--archive" || argument === "--theme") {
      const value = argv[index + 1];
      if (value === undefined || value.startsWith("--")) fail(`${argument} requires a value`);
      if (argument === "--archive") options.archiveDirectory = path.resolve(value);
      else options.theme = value;
      index += 1;
    } else fail(`unknown verifier argument ${argument}`);
  }
  return options;
}

async function main() {
  const options = parseArguments(process.argv.slice(2));
  if (options.help) {
    console.log("Usage: screenshots:verify [--archive PATH] [--theme THEME_ID]");
    return;
  }
  const catalog = await loadCanonicalThemes();
  const themes = options.theme === undefined ? catalog : catalog.filter(({ id }) => id === options.theme);
  if (themes.length === 0) fail(`unknown theme ${options.theme}`);
  const result = await verifyArchive({ archiveDirectory: options.archiveDirectory, themes });
  console.log(`Verified ${result.entryCount} deterministic game screenshot(s) in ${result.archiveDirectory}; seed=${result.seed}`);
}

const entryPath = process.argv[1] === undefined ? "" : pathToFileURL(path.resolve(process.argv[1])).href;
if (entryPath === import.meta.url) {
  main().catch((error) => {
    console.error(error instanceof Error ? `${error.name}: ${error.message}` : String(error));
    process.exitCode = 1;
  });
}
